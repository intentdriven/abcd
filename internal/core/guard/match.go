package guard

import (
	"path"
	"strings"
)

// wrappers are commands that RUN another command with the same intent, so the
// hazard sits one token further along: `sudo rm -rf *` is an `rm`. A wrapper's
// OWN arguments are stepped over with it (wrapperValueFlags, wrapperOperands),
// because a wrapper that defangs an entry as soon as it carries a flag is worse
// than one the set never knew — the registry looks armed and is not.
// The set is an UPGRADE, not the safety property. Since adr-42 a hazard behind a
// launcher this map does not name is a loud Tier 2 warn rather than a silent
// allow, so being incomplete here costs precision, not coverage — which is what
// makes it safe to add names without pretending the list is finished. It never
// can be: any binary that execs its arguments belongs here, and a repository adds
// one with a line in a Makefile.
//
// Every entry below is Linux/util-linux/coreutils, and every flag list is derived
// by PROBING the installed binary (wrappers_test.go), never by reading --help —
// nsenter's help contradicts its own parser on `-S`, and gh-299 shipped a list
// that a size assertion certified as complete while it was wrong three ways.
var wrappers = map[string]bool{
	"sudo":    true,
	"doas":    true,
	"command": true,
	"env":     true,
	"nohup":   true,
	"time":    true,
	"xargs":   true,
	"timeout": true,
	"exec":    true,

	// Scheduling, buffering and namespace shims: each runs the command that
	// follows it, and each was a silent allow for every bundled blocker (iss-272).
	"nice":        true,
	"setsid":      true,
	"stdbuf":      true,
	"ionice":      true,
	"eatmydata":   true,
	"proxychains": true,
	"chrt":        true,
	"taskset":     true,
	"unshare":     true,
	"nsenter":     true,
	"flock":       true,
	"chroot":      true,
	// `runuser -u <user> [--] <cmd>` is a direct-exec grammar. Its OTHER grammar,
	// `runuser -c <string>`, hands the string to a shell and belongs to the
	// execute-a-string family, not here.
	"runuser": true,
	// A multiplexer: its first operand is the applet, so stepping it leaves
	// `sh -c …` in command position and the interpreter path takes over.
	"busybox": true,

	// zsh precommand modifiers: `noglob <cmd>` and `nocorrect <cmd>` run the
	// following command with globbing / spelling-correction turned off, exactly
	// like the exec/command/nohup family above. Each takes NO options of its own,
	// so the token right after it is command position — no wrapperValueFlags or
	// wrapperOperands entry is needed. Missing here, a Tier-1 blocker behind
	// `noglob rm -rf *` never reached command position and evaded to a mere
	// Tier-2 warn (iss-2608270655497992). zsh-specific, like the zsh `-c`
	// payload scope.
	"noglob":    true,
	"nocorrect": true,

	// bash's `builtin <name>` runs the shell builtin of that name: `builtin cd`
	// is the directory change `command cd` is, and fails the same way
	// (review2-guard finding 7). It takes no options.
	"builtin": true,
}

// wrapperValueFlags names, per wrapper, that wrapper's OWN flags which consume
// the FOLLOWING token, so the walk to command position steps over the value
// rather than reading it as the command (`sudo -u bob rm -rf *` is an `rm`).
//
// The table is deliberately explicit and small: only wrappers in the set above,
// only flags those wrappers actually document, and only the separate-token form
// — a `--flag=value` carries its value in the same token and needs no entry.
// Booleans (`sudo -n`, `env -i`, `time -p`) need no entry either; they are
// stepped over as flags. A value flag the table does not name is NOT stepped
// over, so its value is read as the command and the entry misses: a miss is a
// non-match, never a false block, the same trade the operand walk already makes.
var wrapperValueFlags = map[string][]string{
	"sudo": {
		"-C", "--close-from", "-D", "--chdir", "-g", "--group", "-h", "--host",
		"-p", "--prompt", "-R", "--chroot", "-r", "--role", "-T",
		"--command-timeout", "-t", "--type", "-U", "--other-user", "-u", "--user",
	},
	"doas": {"-a", "-C", "-u"},
	// `env`'s -S/--split-string are deliberately NOT here: the execute-a-string
	// pre-pass (payload.go) owns them on the raw token chain, because the wrapper
	// walk would otherwise consume and discard the value it needs to inspect.
	"env":     {"-u", "--unset", "-C", "--chdir"},
	"time":    {"-f", "--format", "-o", "--output"},
	"xargs":   {"-a", "--arg-file", "-d", "--delimiter", "-E", "-I", "-J", "-L", "-n", "--max-args", "-P", "--max-procs", "-R", "-s", "-S", "--max-chars", "--process-slot-var"},
	"timeout": {"-k", "--kill-after", "-s", "--signal"},
	"exec":    {"-a"},
	// `command` and `nohup` take no value flags at all. xargs's `-J`, `-R` and
	// `-S` are BSD's (macOS xargs): the replacement string, the most
	// replacements, and the replacement size.

	// Probed on util-linux 2.39.3 / coreutils 9.4 (wrappers_test.go). Two of these
	// are traps a document would have got wrong:
	//   - `taskset -c` is a BOOLEAN format switch, not a value flag. Listing it
	//     would make the walk step over the COMMAND and create a miss that does
	//     not exist today.
	//   - `setsid -c` is `--ctty`, not a payload flag, and takes nothing.
	"nice":   {"-n", "--adjustment"},
	"stdbuf": {"-i", "-o", "-e", "--input", "--output", "--error"},
	"ionice": {"-c", "-n", "--class", "--classdata"},
	"chrt":   {"-T", "-P", "-D", "--sched-runtime", "--sched-period", "--sched-deadline"},
	"unshare": {
		"-S", "-G", "-w", "-R", "--setuid", "--setgid", "--wd", "--root",
		"--map-user", "--map-group", "--map-users", "--map-groups",
		"--propagation", "--setgroups", "--monotonic", "--boottime",
	},
	// NOT -W/--wd: nsenter's is optional-argument and consumes nothing, unlike
	// unshare's -w/--wd, which is required-argument. Same package, same version,
	// same letter, opposite grammar — verified by probe, not by --help.
	"nsenter": {"-t", "-S", "-G", "--target", "--setuid", "--setgid"},
	"chroot":  {"--userspec", "--groups"},
	"flock":   {"-w", "-E", "--timeout", "--wait", "--conflict-exit-code"},
	"runuser": {"-u", "-g", "-G", "-s", "-w", "--user", "--group", "--supp-group", "--shell", "--whitelist-environment"},
	// `setsid`, `eatmydata`, `proxychains` and `taskset` take no value flags.
}

// wrapperOperands names wrappers whose grammar puts a mandatory OPERAND between
// the flags and the command: `timeout [OPTIONS] DURATION COMMAND...`. The
// duration is not a flag, so no amount of flag stepping reaches past it — read
// as command position, `timeout 30 rm -rf /` is a command called `30`.
var wrapperOperands = map[string]int{
	"timeout": 1,
	// `chrt [OPTIONS] PRIORITY COMMAND`, `taskset [OPTIONS] MASK COMMAND`,
	// `flock [OPTIONS] FILE COMMAND`, `chroot [OPTIONS] DIR COMMAND`. Each was
	// verified by running it: `chrt -f /bin/echo hi` answers
	// "invalid priority argument: '/bin/echo'".
	"chrt":    1,
	"taskset": 1,
	"flock":   1,
	"chroot":  1,
}

// reserved are shell keywords and grouping tokens that PRECEDE a command rather
// than being one: without stepping over them, `if cd scratch; then rm -rf *; fi`
// reads `then` as argv[0] and every hazard inside a conditional or a loop body
// escapes command position.
//
// `coproc` belongs to this family — `coproc [NAME] command` runs the following
// command exactly like `{`, `!`, or the `time` wrapper — but it is NOT a member
// of this map, because its optional coprocess NAME has to be stepped over
// conditionally (only in the `coproc NAME { …; }` compound form). commandOf
// handles it in its own branch via skipCoproc. It is a bash/zsh/ksh keyword;
// POSIX sh/dash treats `coproc` as an ordinary not-found command, so the wrapped
// hazard never runs there — the same shell-specific scope accepted for the
// `zsh -c`/`ksh -c` payload bypass (gh-318, gh-297).
var reserved = map[string]bool{
	"if":    true,
	"then":  true,
	"elif":  true,
	"else":  true,
	"do":    true,
	"while": true,
	"until": true,
	"{":     true,
	"!":     true,
}

// precededByCD reports whether an earlier command in the SAME chain changes
// directory. A cd on a previous logical line does not chain: a new line is a
// new shell command, and its failure cannot redirect this one. Every place the
// earlier command can sit is read, and a name a substitution prints can be a
// directory change (unknown.go).
func precededByCD(before []segment, chain int) bool {
	for _, s := range before {
		if s.chain != chain {
			continue
		}
		for _, a := range commandSites(s) {
			if nameCouldBeAny(s.tokens[a.idx], directoryChanges) {
				return true
			}
		}
	}
	return false
}

// directoryChanges names the builtins an `after_cd` entry reads as the
// directory change a command is chained after. `pushd` and `popd` change it
// exactly as `cd` does and fail the same way, leaving the shell where it was
// for the command that follows (iss-2609251640464735).
var directoryChanges = []string{"cd", "pushd", "popd"}

// skipCoproc advances past the `coproc` keyword's own tokens, from pos (the
// token just after `coproc`), and returns the index where the command it launches
// begins. `coproc <simple-command>` has no coprocess NAME — the token at pos is
// the command. `coproc NAME { …; }` names the coprocess: the NAME must be stepped
// over so the command inside the `{ }` body reaches command position. The NAME is
// only present when the token at pos is a shell identifier AND the token after it
// opens the compound body with `{` (which reserved then steps over); a lone
// `coproc word` leaves `word` in command position, matching bash's own parse.
func skipCoproc(tokens []string, pos int) int {
	if pos+1 < len(tokens) && isShellName(tokens[pos]) && tokens[pos+1] == "{" {
		return pos + 1 // step over the coprocess NAME; `{` is stepped over as reserved
	}
	return pos
}

// isShellName reports whether a token is a shell identifier — a letter or
// underscore followed by letters, digits, or underscores — the only shape a
// coprocess NAME may take.
func isShellName(tok string) bool {
	if tok == "" {
		return false
	}
	for i := 0; i < len(tok); i++ {
		c := tok[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c == '_':
		case c >= '0' && c <= '9':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// steppedBeforeCommand reports whether a token precedes the command rather than
// being it: an environment assignment, a reserved word, or a word that is
// nothing but a substitution's output, which an empty output leaves no word
// for. Tier 2 reads it to skip a start no entry's command can be; the walk to
// command position reads the same three shapes in commandArrivals, where a
// substitution is also a program of unknown name.
func steppedBeforeCommand(tok string) bool {
	return isAssignment(tok) || reserved[tok] || vanishable(tok)
}

// isAssignment reports whether a token is a NAME=VALUE environment prefix,
// which precedes the command rather than being one.
func isAssignment(tok string) bool {
	eq := strings.IndexByte(tok, '=')
	if eq <= 0 {
		return false
	}
	for i := 0; i < eq; i++ {
		c := tok[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '_':
		default:
			return false
		}
	}
	return !(tok[0] >= '0' && tok[0] <= '9')
}

// matchSegment applies the pattern's command, subcommand, and flag constraints
// to one command-position segment.
//
// Every compare at a CONSTRAINED position — the command name, operands 0/1 when
// the entry names a subcommand, each flag and flag-value alternative — reads a
// globbed token as the pattern it is: bash expands an unquoted `*`, `?` or
// `[…]` against the working directory before exec, so `pus?` is `push` whenever
// a file called push exists, and a file named for a hazard is the cheapest
// condition an author can arrange in a directory the guard cannot see. A
// literal the pattern CAN produce is treated as produced (path.Match, decidable
// from the string alone). Unconstrained positions are never compared, so `ls *`
// and `git add *.md` are untouched (GHSA-3w99-pgv4-8g55). Behind zsh's `noglob`
// nothing expands, and the compares fall back to literal.
//
// A FLAG constraint is not positional: every argument is offered to it, so the
// globbed compare there is narrowed twice. The token must be FLAG-SHAPED — its
// own literal prefix begins with `-` — because otherwise a bare `*` operand
// satisfied every long alternative at once (`path.Match("*", "--force")` is
// true), and `rm *`, `git push origin *`, `git commit -m msg *` all became
// blocks. And the scan stops at a `--` operand terminator, since after it a
// word is an operand however it is spelled. What that EXCLUDES, deliberately,
// is a pattern whose dash is itself globbed (`[-]-force`, `?-force`): it is
// left to the literal compare, the same floor `flagMatches` names below.
// `--forc?` and `--force*` spell the dash and still fire.
func matchSegment(p Pattern, s segment) bool {
	hit, _ := matchSegmentNamed(p, s)
	return hit
}

// matchSegmentNamed is matchSegment, and whether the entry fired at a place
// whose command word fixes some of the program's name. A match only at words
// whose basename ends in a substitution (anyProgram) is a match because the
// name is unknown, not because the line names the entry's program, and Check
// reports it as the substitution's, not the entry's (review4-guard finding 4).
func matchSegmentNamed(p Pattern, s segment) (hit, named bool) {
	// The places a command can sit are looked through for every entry; the
	// words after them are walked only for an entry whose command stands at
	// one, which is where the count below charges them. Charging every word
	// to every entry counted a walk no entry without a named site makes, and
	// made each entry added to the registry raise the constant the cost guards
	// hold (work_test.go) on lines that never name it.
	tally(len(arrivalsOf(s)))
	// Every place the command can sit is read (commandArrivals): an unknown word
	// before it is read every way it can be, and an unknown word in command
	// position is every program its tail allows. The command NAME is folded
	// (nameCouldBe): a case-insensitive filesystem resolves `GIT`, `RM`, `GH`
	// to the real binaries and executes the hazard (gh-315). Only the name is
	// folded — subcommands, flags and values stay case-sensitive below, because
	// git/gh/rm parse THOSE case-sensitively, so a case-varied subcommand does
	// not run the hazard and must not be blocked.
	sites := sitesNamed(s, p.Command)
	if namesOnlyItsProgram(p) {
		// A variable's value as the program name fires no such entry
		// (variableCarried).
		kept := sites[:0:0]
		for _, a := range sites {
			if !(anyProgram(s.tokens[a.idx]) && variableCarried(s, a.idx)) {
				kept = append(kept, a)
			}
		}
		sites = kept
	}
	need := operandNeed(p)
	for _, noglob := range []bool{false, true} {
		var group []arrival
		for _, a := range sites {
			// A place with fewer words after it than the entry needs operands
			// is no match in any reading, and costs nothing to rule out.
			if a.noglob == noglob && len(s.tokens)-(a.idx+1) >= need {
				group = append(group, a)
			}
		}
		if len(group) == 0 {
			continue
		}
		tally(len(s.tokens))
		// glob reports, per TOKEN index, whether bash would expand that token.
		glob := func(i int) bool { return !noglob && s.globAt(i) }
		m := newEntryMatcher(p, s.tokens, s.spelled, glob)
		for _, a := range group {
			if m.matchesAfter(a.idx) && argsFed(p, s, a.idx) {
				hit = true
				if !anyProgram(s.tokens[a.idx]) {
					return true, true
				}
			}
		}
	}
	return hit, false
}

// argsFed reports whether the arguments of the command at site come from a
// command p.ArgsFrom names, or true when the entry names none. Two places hand
// a command its arguments from another command's output: a command
// substitution in a word after it (`kill $(pgrep -f make)`), and, where xargs
// runs it, the pipeline feeding xargs (`pgrep -f make | xargs kill`) or a word
// of xargs's own (`xargs -a <(pgrep make) kill`). What a substitution or a
// pipeline ran is recorded by the tokenizer (segment.feeds, segment.piped), so
// the question is only whether any of those commands matches.
//
// A command of a command string is also handed what reaches the string from
// outside it (segment.argsIn, segment.stdinIn): the input of the xargs that
// runs the shell, and the shell's own standard input, which an xargs inside
// the string reads.
func argsFed(p Pattern, s segment, site int) bool {
	if len(p.ArgsFrom) == 0 {
		return true
	}
	if anyHits(s.argsIn, p.ArgsFrom) {
		return true
	}
	from := site + 1
	if x, ok := xargsBefore(s, site); ok {
		if s.piped.hits(p.ArgsFrom) || anyHits(s.stdinIn, p.ArgsFrom) {
			return true
		}
		from = x + 1
	}
	for i, fs := range s.feeds {
		if i < from {
			continue
		}
		for _, f := range fs {
			if f.hits(p.ArgsFrom) {
				return true
			}
		}
	}
	return false
}

// xargsBefore returns the earliest place before site the walk arrives at that
// can be xargs, which runs the command at site with arguments it reads from its
// input. A name a substitution prints can be xargs too (`"$(true)"xargs kill`),
// and the walk steps it as a wrapper of unknown grammar, so it is read as one.
func xargsBefore(s segment, site int) (int, bool) {
	for _, a := range arrivalsOf(s) {
		if a.idx >= site {
			break
		}
		if commandNamed(s, a, "xargs") {
			return a.idx, true
		}
	}
	return 0, false
}

// argsReader answers, for places in one segment read left to right, what
// reaches a command there as its arguments from outside its own words: what
// reached the segment itself (segment.argsIn) and, past an xargs, the input
// xargs hands the command it runs — its standard input (the pipe into it, and
// segment.stdinIn) and the output of the substitutions in its own words
// (`xargs -a <(pgrep make)`). The words are read once however many places are
// asked about, so a launcher's windows cost what the line's words do.
//
// The words' substitutions are held as one run: every command the tokenizer
// emitted while it read a segment's words is a substitution in one of them,
// in word order, so the commands of the words between two places are one run
// of the segment's list. What a place is handed therefore stays as short as
// the nesting of strings is deep, however many words it spans.
type argsReader struct {
	s     segment
	x     int
	ok    bool
	next  int
	base  []feed
	words feed
}

func newArgsReader(s segment) *argsReader {
	r := &argsReader{s: s}
	r.x, r.ok = xargsBefore(s, len(s.tokens))
	if r.ok {
		r.base = append([]feed(nil), s.argsIn...)
		if s.piped.list != nil {
			r.base = append(r.base, s.piped)
		}
		r.base = append(r.base, s.stdinIn...)
		r.next = r.x + 1
	}
	return r
}

// before returns what reaches a command at site, for a site at or after every
// one asked before.
func (r *argsReader) before(site int) []feed {
	if !r.ok || site <= r.x {
		return r.s.argsIn
	}
	for ; r.next < site && r.next < len(r.s.tokens); r.next++ {
		tally(1)
		for _, f := range r.s.feeds[r.next] {
			switch {
			case r.words.list == nil:
				r.words = f
			case f.list == r.words.list:
				r.words.lo, r.words.hi = min(r.words.lo, f.lo), max(r.words.hi, f.hi)
			default:
				// Not reached: a word's feeds name its own tokenize call. Kept
				// whole rather than dropped, should that ever change.
				r.base = append(r.base, f)
			}
		}
	}
	out := append([]feed(nil), r.base...)
	if r.words.list != nil {
		out = append(out, r.words)
	}
	return out
}

// anyHits reports whether any command in any of the runs matches one of srcs.
func anyHits(fs []feed, srcs []Pattern) bool {
	for _, f := range fs {
		if f.hits(srcs) {
			return true
		}
	}
	return false
}

// segmentHits reports whether a command matches one of srcs, or any command of
// a command string it runs does (segList.payloads), however deep they nest.
func segmentHits(s segment, srcs []Pattern) bool {
	for _, src := range srcs {
		if matchSegment(src, s) {
			return true
		}
	}
	if s.home == nil {
		return false
	}
	for _, ps := range s.home.payloads[s.at] {
		if segmentHits(ps, srcs) {
			return true
		}
	}
	return false
}

// hits reports whether any command in the run matches one of srcs. The first
// question asked of a list counts, once, how many of its commands match, so
// every later question about any run in it — however many runs nest inside
// one another — is two lookups.
func (f feed) hits(srcs []Pattern) bool {
	if f.list == nil || f.lo >= f.hi || len(srcs) == 0 {
		return false
	}
	key := &srcs[0]
	counts, ok := f.list.hits[key]
	if !ok {
		counts = make([]int, len(f.list.segs)+1)
		for i, s := range f.list.segs {
			counts[i+1] = counts[i]
			if segmentHits(s, srcs) {
				counts[i+1]++
			}
		}
		if f.list.hits == nil {
			f.list.hits = map[*Pattern][]int{}
		}
		f.list.hits[key] = counts
	}
	hi := f.hi
	if hi > len(f.list.segs) {
		hi = len(f.list.segs)
	}
	return f.lo < hi && counts[hi] > counts[f.lo]
}

// entryMatcher answers, for any place a command can sit in one segment, whether
// the arguments after it meet an entry's operand and flag constraints. It reads
// the tokens once however many places there are — a line whose command an
// unknown word puts in several places costs what a line with one does.
type entryMatcher struct {
	// accept[i] reports whether tokens[i:], as a command's arguments, meet the
	// operand constraints in some reading (operandAcceptance).
	accept []bool
	// nextStop[i] is the first index at or after i holding the `--` operand
	// terminator, or len(tokens): no flag is read past it.
	nextStop []int
	// nextHit holds, per flag clause (each flag group, then each flag-value
	// constraint), the first index at or after i whose token satisfies it.
	nextHit [][]int
	// groups is how many of the clauses are flag groups, the ones a signal
	// word is no answer to.
	groups int
	// nextSig[i] is the first index at or after i holding a word the command
	// reads as its signal (signalWordCommands), or len(tokens); nil for a
	// command that reads none.
	nextSig []int
}

// newEntryMatcher reads the tokens for one entry. One operand walk reads every
// word, an unknown one every way it can be read (unknown.go): as a value flag's
// value it fills the slot, so the operands after it keep their positions (`git
// -C $(pwd) push` is a push); an unknown dash-word both stands alone and takes
// a value (`git -$(x) /tmp push`); a word that may print nothing both is and is
// not an operand (`git $(true) push`). The subcommands, the count, the prefix
// and the path are all met by one reading. spelled is the segment's
// segment.spelled, which only the arg_values clause reads (writtenMatches).
func newEntryMatcher(p Pattern, tokens []string, spelled map[int][]string, glob func(int) bool) entryMatcher {
	n := len(tokens)
	want := operandWant{
		sub: p.Subcommand, sub2: p.Subcommand2, min: p.MinOperands,
		prefixes: p.ArgPrefixes, paths: p.ArgPaths, values: p.ArgValues,
	}
	m := entryMatcher{accept: operandAcceptance(tokens, spelled, p.ValueFlags, want, glob), nextStop: make([]int, n+1)}
	m.nextStop[n] = n
	for i := n - 1; i >= 0; i-- {
		m.nextStop[i] = m.nextStop[i+1]
		if tokens[i] == "--" {
			m.nextStop[i] = i
		}
	}
	opts := gitOptionTable(p)
	next := func(hit func(i int) bool) []int {
		nh := make([]int, n+1)
		nh[n] = n
		for i := n - 1; i >= 0; i-- {
			nh[i] = nh[i+1]
			if hit(i) {
				nh[i] = i
			}
		}
		return nh
	}
	for _, group := range p.Flags {
		alts := strings.Split(group, "|")
		m.nextHit = append(m.nextHit, next(func(i int) bool { return flagGroupHit(alts, tokens, i, glob, opts) }))
	}
	m.groups = len(p.Flags)
	if signalWordCommands[strings.ToLower(p.Command)] && len(p.Flags) > 0 {
		m.nextSig = next(func(i int) bool { return isSignalWord(tokens[i]) })
	}
	for _, fv := range p.FlagValues {
		fv := fv
		m.nextHit = append(m.nextHit, next(func(i int) bool { return flagValueHit(fv, tokens, i, glob) }))
	}
	return m
}

// matchesAfter reports whether the arguments after a command at site meet the
// entry: the operands in some reading, and every flag clause before the first
// `--` after it.
func (m entryMatcher) matchesAfter(site int) bool {
	start := site + 1
	if !m.accept[start] {
		return false
	}
	stop := m.nextStop[start]
	for c, nh := range m.nextHit {
		h := nh[start]
		// The command's first signal word is its signal, never a flag.
		if c < m.groups && m.nextSig != nil && h < len(nh)-1 && h == m.nextSig[start] {
			h = nh[h+1]
		}
		if h >= stop {
			return false
		}
	}
	return true
}

// globMatches reports whether the shell pattern can produce the literal.
// path.Match decides a pattern whose one bracket expression is a plain set
// (plainBrackets), which it reads as bash does. Any other bracket expression
// is one this compare cannot decide (iss-2609291233468970): path.Match has no
// POSIX, equivalence or collating class, refuses a set whose first member is
// `]` or `-`, and reads bash's negation `[!…]` as a set holding `!`, while
// bash expands `r[[:lower:]]` to `rm` and `ba[].s]h` to `bash`; and the
// tokenizer removes a backslash before the compare, so `clea[\!n]` and
// `r[m\]]`, which every shell expands to `clean` and `rm`, arrive as the
// negation `clea[!n]` and the set `r[m]` followed by `]`. Such a pattern is
// compared with its bracket expressions read as any run of characters
// (bracketsAsStar), which matches whatever any reading of them can produce.
// An unterminated `[` is a literal in bash and a syntax error to path.Match;
// no literal compared here holds a `[`, so neither produces one.
func globMatches(pattern, literal string) bool {
	if !plainBrackets(pattern) {
		pattern = bracketsAsStar(pattern)
	}
	ok, err := path.Match(pattern, literal)
	return err == nil && ok
}

// plainBrackets reports whether a pattern's bracket expression is a set whose
// meaning no removed backslash can change and path.Match reads as bash does:
// it is not empty, its members are bytes other than `!`, `^`, `-`, `[` and
// `\`, and no `]` follows its close, where an escaped `]` would have kept it
// open or a second set would close. A pattern with no `[` that a later `]`
// closes holds no bracket expression: every shell reads an unterminated `[`
// as a literal.
func plainBrackets(pattern string) bool {
	first := strings.IndexByte(pattern, '[')
	if first < 0 {
		return true
	}
	end := strings.IndexByte(pattern[first+1:], ']')
	if end < 0 {
		return true
	}
	tally(len(pattern))
	end += first + 1
	return end > first+1 && !strings.ContainsAny(pattern[first+1:end], `!^-[\`) &&
		strings.IndexByte(pattern[end+1:], ']') < 0
}

// bracketsAsStar is a pattern with the span from its first `[` to its last
// `]` written as one `*`, a backslash directly before that `[` taken in with
// it. The span holds every bracket expression the pattern has, however a
// removed backslash would have opened or closed them, and each matches one
// character that is never a slash, so `*` matches whatever the span can
// produce. A `[` after the span has no `]` to close it and is kept as the
// literal it is. Its one pass is counted with plainBrackets', which every
// call follows.
func bracketsAsStar(pattern string) string {
	first, last := strings.IndexByte(pattern, '['), strings.LastIndexByte(pattern, ']')
	if first < 0 || last < first {
		return pattern
	}
	if first > 0 && pattern[first-1] == '\\' {
		first--
	}
	return pattern[:first] + "*" + strings.ReplaceAll(pattern[last+1:], "[", `\[`)
}

// argPrefixMatches reports whether some operand carries the prefix. Only
// operands are considered, so a prefix like "+" can never be satisfied by an
// option token: the constraint describes an argument (`git push origin
// +main:main`), not a flag. An unknown operand is read by its known text: a
// refspec a substitution prints whole is how an everyday push names its branch
// (unknown.go), and `"$(true)"+main:main` is still `+main:main`.
func argPrefixMatches(prefix string, ops []string) bool {
	for _, op := range ops {
		if strings.HasPrefix(knownText(op), prefix) {
			return true
		}
	}
	return false
}

// argValueMatches reports whether an operand, as writtenMatches reads each
// text of its written spelling, is one of the words. Only operands are
// considered, and a substitution's output is taken as empty, as argPrefixMatches reads a prefix: a word that is wholly
// a substitution is how an everyday delete names its target (`rm -rf
// "$(mktemp -d)"`), so reading it as every target would refuse them all
// (unknown.go's operand residual). A variable is compared as the line wrote
// it, so `$HOME` names the home and `"$OUT"/` names no root; one whose text is
// not known names nothing. A spelling holding fieldMark is the fields bash
// splits it into, and each is compared on its own; quotedFieldMark is the
// space a quoted word keeps. Each field is also compared with its redundant
// separators taken out (cleanSeparators), as the kernel reads the path.
func argValueMatches(values []string, written string) bool {
	written = strings.ReplaceAll(written, quotedFieldText, " ")
	for _, field := range strings.Split(written, fieldText) {
		if field == "" || isUnknown(field) {
			continue
		}
		clean, glob := cleanSeparators(field), globReading(field)
		for _, v := range values {
			if field == v || clean == v || glob == v {
				return true
			}
		}
		if lead := strings.TrimLeft(field, "/"); lead != field && absoluteName(lead) && argValueMatches(values, lead) {
			return true
		}
	}
	return false
}

// absoluteName reports whether p begins with the home or the working
// directory written as a variable (`$HOME`, `${HOME}`, `$PWD`, `${PWD}`),
// whose value is an absolute path: a run of `/` written before it names the
// same directory, so argValueMatches also reads the field without that run
// (`rm -rf /$HOME` deletes the home; iss-2609300057462186).
func absoluteName(p string) bool {
	for _, name := range []string{"HOME", "PWD"} {
		if strings.HasPrefix(p, "${"+name+"}") {
			return true
		}
		if strings.HasPrefix(p, "$"+name) && (len(p) == len(name)+1 || !isNameByte(p[len(name)+1])) {
			return true
		}
	}
	return false
}

// globReading is a path read the way a shell's glob can expand it, beside
// the path as written (argValueMatches compares both, so it can only add a
// hit): a run of `*` is one `*` (collapseStars), and a segment written with a
// leading `.` that can match the name `..` is that `..` where a further
// segment follows it (dotParents), its redundant separators then taken out
// and its `..` folded (cleanSeparators). It is "" where the glob reading is
// the path as written, which no value is.
func globReading(p string) string {
	g := dotParents(collapseStars(p))
	if g == p {
		return ""
	}
	return cleanSeparators(g)
}

// collapseStars is a glob with each run of `*` written as one `*`
// (iss-2609290925320493). A run matches what one `*` matches, so every shell
// without globstar expands `/**` as `/*`; with globstar set, `**` matches
// every path beneath as well, never fewer. The tokenizer removes a backslash
// before the operand reaches here, so an escaped `*` or `?` is read as the
// glob it would otherwise be: `~/*\*` reads as `~/*` and `~/.\?/*` as
// `~/.?/*`, and both block though the shell expands neither that far.
func collapseStars(p string) string {
	if !strings.Contains(p, "**") {
		return p
	}
	tally(len(p))
	b := make([]byte, 0, len(p))
	for i := 0; i < len(p); i++ {
		if p[i] == '*' && len(b) > 0 && b[len(b)-1] == '*' {
			continue
		}
		b = append(b, p[i])
	}
	return string(b)
}

// dotParents is a path with each segment that can match the name `..`
// written as `..`, where a further segment follows it (iss-2609290925333487).
// bash 3.2 and /bin/sh, the shells macOS runs, have no globskipdots, so a
// segment whose leading `.` is written lets its glob match `..`: `~/.?/*`,
// `~/.[.]/*` and `~/.*/*` expand to include `~/../*`, and `/.?/*` to the
// root's entries. A glob whose leading `.` is not written (`??`, `?.`,
// `[.]?`) never matches a dot name, and a final segment is left as it is:
// rm refuses an operand whose last segment is `..`. Reading every such
// segment as `..` climbs at least as far as any mix of the names it
// matches, so it is the reading that reaches the root or the home first.
func dotParents(p string) string {
	if !strings.Contains(p, "/.") && !strings.HasPrefix(p, ".") {
		return p
	}
	segs := strings.Split(p, "/")
	last := len(segs) - 1
	for last > 0 && segs[last] == "" {
		last--
	}
	changed := false
	for i := 0; i < last; i++ {
		tally(len(segs[i]))
		if dotGlob(segs[i]) {
			segs[i] = ".."
			changed = true
		}
	}
	if !changed {
		return p
	}
	return strings.Join(segs, "/")
}

// dotGlob reports whether a path segment is a glob with a written leading
// `.` that can match the name `..` (`.?`, `.*`, `.[.]`, `.[!x]`, `..*`), as
// globMatches decides it: a bracket expression it cannot decide
// (`.[[:punct:]]`, `.[].]`, `.[--.]`, and `.[!.]`, which is also what
// `.[\!.]` arrives as) reads as able to match `..`, and so does `.[a-z]`,
// which no shell matches with `..` (iss-2609291233390280).
func dotGlob(seg string) bool {
	return len(seg) >= 2 && seg[0] == '.' && strings.ContainsAny(seg[1:], "*?[") &&
		literalSegment(seg) && globMatches(seg, "..")
}

// cleanSeparators is a path with what the kernel reads as nothing taken
// out (iss-2609290625482831): a run of slashes is one separator (`//*` is
// `/*`, `$HOME//` is `$HOME/`), a `.` segment between two slashes is the
// directory itself (`/./*` is `/*`), and a `..` segment directly under the
// root is the root, its own parent (`/../*` is `/*`). A trailing `.` or `..`
// is kept here: rm refuses an operand whose last segment is one. A path's
// `..` segments are then folded (foldParents), a trailing `.` after one
// included, so a path ending in `/.` is handed to the fold as well: `../.`
// holds none of the three marks above, and returning it early left it
// allowed while `./../.` warned (iss-2609302306019245).
func cleanSeparators(p string) string {
	if !strings.Contains(p, "//") && !strings.Contains(p, "/./") && !strings.Contains(p, "/..") && !strings.HasSuffix(p, "/.") {
		return p
	}
	tally(len(p))
	b := make([]byte, 0, len(p))
	for i := 0; i < len(p); i++ {
		if p[i] == '/' && len(b) > 0 && b[len(b)-1] == '/' {
			continue
		}
		b = append(b, p[i])
	}
	out := string(b)
	for strings.Contains(out, "/./") {
		out = strings.ReplaceAll(out, "/./", "/")
	}
	for strings.HasPrefix(out, "/../") {
		out = out[3:]
	}
	return foldParents(out)
}

// homePrefixes are the spellings of the home a folded path may begin with.
var homePrefixes = []string{"~", "$HOME", "${HOME}"}

// pwdPrefixes are the spellings of the working directory a folded path may
// begin with.
var pwdPrefixes = []string{"$PWD", "${PWD}"}

// foldBase is where a folded path begins.
type foldBase int

const (
	foldRoot foldBase = iota
	foldHome
	foldPWD
	foldRelative
)

// foldParents folds each `..` segment of a path into the segment before it,
// as the path reads lexically (iss-2609290745243990): `/tmp/../*` is `/*`,
// `~/x/..` is `~`. The kernel reads `..` otherwise only where the segment
// before it is a symlink, and the lexical reading is the one that blocks. A
// `..` past the home climbs to a directory that holds the home, so the path
// is the home where each segment it then descends through is `*`, which
// matches the home's own name among the rest: `~/../*` and `~/..` are `~`,
// `~/../*/*` is `~/*`, and `~/../x` stays a sibling. A trailing `.` or `..`
// is folded as well, since what it names is the root or holds the home,
// though rm refuses it: `../.` and `./../.` are `..`, and `../../.` is
// `../..`. A path that begins at the working directory, `$PWD`
// or a relative name, is folded the same way (iss-2609290925346181): a `..`
// past its beginning is the working directory's parent, so `$PWD/../*`,
// `./../*` and `x/../../*` are `../*`, while a relative path that stays
// inside the working directory is returned as it is (`x/../*`). A segment
// holding a variable or a substitution is not a known count of directories,
// so nothing is folded across one, and a path of any other beginning (a
// `~user`) is returned as it is.
func foldParents(p string) string {
	if !strings.Contains(p, "..") {
		return p
	}
	base, prefix, rest := foldRelative, "", p
	switch {
	case strings.HasPrefix(p, "/"):
		base, rest = foldRoot, p[1:]
	case strings.HasPrefix(p, "~") || strings.HasPrefix(p, "$"):
		found := false
		for _, h := range homePrefixes {
			if p == h || strings.HasPrefix(p, h+"/") {
				base, prefix, found = foldHome, h, true
				break
			}
		}
		for _, w := range pwdPrefixes {
			if !found && (p == w || strings.HasPrefix(p, w+"/")) {
				base, prefix, found = foldPWD, w, true
			}
		}
		if !found {
			return p
		}
		rest = strings.TrimPrefix(p[len(prefix):], "/")
	}
	trailing := strings.HasSuffix(rest, "/")
	var stack []string
	climb := 0
	for _, seg := range strings.Split(strings.TrimSuffix(rest, "/"), "/") {
		tally(len(seg) + 1)
		switch seg {
		case "", ".":
		case "..":
			switch {
			case len(stack) > 0:
				if !literalSegment(stack[len(stack)-1]) {
					return p
				}
				stack = stack[:len(stack)-1]
			case base != foldRoot:
				climb++
			}
		default:
			stack = append(stack, seg)
		}
	}
	switch base {
	case foldRoot:
		if len(stack) == 0 {
			return "/"
		}
		return joinFolded("", stack, trailing)
	case foldPWD, foldRelative:
		if climb == 0 {
			if base == foldRelative {
				return p
			}
			if len(stack) > 0 {
				return joinFolded(prefix, stack, trailing)
			}
			if trailing {
				return prefix + "/"
			}
			return prefix
		}
		up := make([]string, climb, climb+len(stack))
		for i := range up {
			up[i] = ".."
		}
		return strings.TrimPrefix(joinFolded("", append(up, stack...), trailing), "/")
	}
	for i := 0; i < climb && i < len(stack); i++ {
		if stack[i] != "*" {
			return p
		}
	}
	if climb >= len(stack) {
		stack = nil
	} else {
		stack = stack[climb:]
	}
	if len(stack) == 0 {
		if trailing {
			return prefix + "/"
		}
		return prefix
	}
	return joinFolded(prefix, stack, trailing)
}

// joinFolded writes a folded path back: its beginning, each segment after a
// slash, and the trailing slash the operand was written with.
func joinFolded(prefix string, segs []string, trailing bool) string {
	out := prefix + "/" + strings.Join(segs, "/")
	if trailing {
		out += "/"
	}
	return out
}

// literalSegment reports whether a path segment is one directory whatever
// the shell does with it: its text holds no variable and no mark (a
// substitution is spelled as one), only name bytes and glob characters,
// which never match a slash.
func literalSegment(seg string) bool {
	tally(len(seg) + 1)
	for i := 0; i < len(seg); i++ {
		if c := seg[i]; c == '$' || c < 0x20 || c == 0x7f {
			return false
		}
	}
	return true
}

// flagGroupHit reports whether the token at i is an alternative of one "a|b"
// flag group. glob reports, per token index, whether bash would expand that
// token. The caller reads the tokens only up to `--` (entryMatcher): after the
// terminator every word is an operand, so `git push -- --force origin main` pushes a refspec
// called `--force` and is not a force push. opts, when the
// entry names a subcommand whose options are modelled (gitOptionTable), is read
// for the abbreviations git accepts of a long alternative
// (abbreviatesAlternative).
func flagGroupHit(alts []string, tokens []string, i int, glob func(int) bool, opts []string) bool {
	arg := tokens[i]
	if arg == "--" {
		return false
	}
	// The known text is the word with every substitution printing nothing; an
	// unknown dash-word is also every flag it can still become (unknown.go).
	k := knownText(arg)
	for _, alt := range alts {
		if alt == "" {
			continue
		}
		if flagMatches(alt, k, glob(i)) || unknownFlagCouldBe(arg, alt) {
			return true
		}
		if opts != nil && abbreviatesAlternative(k, alt, alts, opts) {
			return true
		}
	}
	return false
}

// flagShaped reports whether a token can be a flag at all — whether its own
// literal prefix begins with `-`. It gates the globbed compare at every flag
// position: a glob denotes a set of words, but a flag constraint is offered
// every argument, so without this an operand pattern (`*`, `*e`, `?`) matched
// each long alternative and blocked the ordinary lines it appears in. A dash
// spelled behind a glob (`[-]-force`) is not flag-shaped and is left to the
// literal compare — the floor flagMatches names.
func flagShaped(tok string) bool { return strings.HasPrefix(tok, "-") }

// flagMatches compares one flag alternative with one argument token. A long
// flag also matches its --flag=value form; a single-letter short flag also
// matches inside a bundled cluster, so -rf satisfies both -r and -f.
//
// A globbed token is compared as the pattern it is, but ONLY when it is
// flag-shaped: bash expands `*` against the working directory, and a word that
// does not start with a dash is an operand, not an option. For a long
// alternative the compare is then path.Match on the token (and on its part
// before an `=`, for the --flag=value form). For a short alternative it is the
// cluster rule read fail-closed: a pattern beginning `-` followed by anything
// but a second dash expands to some short cluster, and that cluster contains
// the letter if the pattern spells it literally OR carries a wildcard that can
// stand for it — `-r?` can be `-rf`. What is NOT modelled, and stays literal,
// is a glob inside an attached short value, extended globs (extglob, `**`),
// and a pattern whose leading dash is itself globbed (`[-]-force`): a floor,
// named in the guard's scope statement.
func flagMatches(alt, arg string, glob bool) bool {
	if alt == arg {
		return true
	}
	if strings.HasPrefix(alt, "--") {
		if strings.HasPrefix(arg, alt+"=") {
			return true
		}
		if !glob || !flagShaped(arg) {
			return false
		}
		if globMatches(arg, alt) {
			return true
		}
		if eq := strings.IndexByte(arg, '='); eq > 0 {
			return globMatches(arg[:eq], alt)
		}
		return false
	}
	if len(alt) == 2 && alt[0] == '-' {
		if isShortCluster(arg) && strings.ContainsRune(arg[1:], rune(alt[1])) {
			return true
		}
		// The first letter after a single dash is always an option, and a
		// byte no option letter can be (`/`, `.`) marks what follows as its
		// value — or the word as one the program refuses — so `-tpts/3` is
		// `-t pts/3` and `-ubob.smith` is `-u bob.smith`. Only that first
		// letter is read: the rest may be the value (`-m"new feature"`).
		if len(arg) > 2 && arg[0] == '-' && arg[1] == alt[1] && !isShortCluster(arg) {
			return true
		}
		if !glob || len(arg) < 2 || arg[0] != '-' || arg[1] == '-' {
			return false
		}
		return strings.ContainsRune(arg[1:], rune(alt[1])) || strings.ContainsAny(arg[1:], "*?[")
	}
	return false
}

// flagValueHit reports whether the token at i SETS one of the flag
// alternatives to one of the accepted values. All three spellings a shell user
// reaches for are read — `-X DELETE`, `-XDELETE`, `--method=DELETE` — because a
// constraint another spelling of the same call steps past is not one. A
// globbed flag or value token is compared as a pattern in the separate-token
// form; the attached forms stay literal (the floor flagMatches names). The
// FLAG half carries the same two narrowings flagGroupHit does — the token
// must be flag-shaped, and the caller reads only up to `--` — because this is a
// flag position too, and the rule cannot hold at two of its three sites. The
// VALUE half is a different position: a globbed value is an ordinary word, and
// `-X DELET?` is compared as the pattern it is.
func flagValueHit(fv FlagValue, tokens []string, i int, glob func(int) bool) bool {
	arg := tokens[i]
	if arg == "--" {
		return false
	}
	for _, alt := range strings.Split(fv.Flag, "|") {
		if alt == "" {
			continue
		}
		// An unknown word is read as unknown.go says: by its known text,
		// and, written with a dash, as any flag it can still become —
		// which, with its value attached, is a setting the constraint
		// accepts.
		if unknownFlagCouldBe(arg, alt) {
			return true
		}
		k := knownText(arg)
		switch {
		case k == alt || (glob(i) && flagShaped(k) && globMatches(k, alt)):
			// The separate-token form: the value is the next argument.
			if i+1 < len(tokens) && acceptsValue(fv.Values, tokens[i+1], glob(i+1)) {
				return true
			}
		case strings.HasPrefix(arg, alt+"="):
			if acceptsValue(fv.Values, arg[len(alt)+1:], false) {
				return true
			}
		case strings.HasPrefix(k, alt+"="):
			if acceptsValue(fv.Values, k[len(alt)+1:], false) {
				return true
			}
		case isShortFlag(alt) && len(arg) > len(alt) && strings.HasPrefix(arg, alt):
			// A short flag's value may be attached with no separator at all.
			if acceptsValue(fv.Values, arg[len(alt):], false) {
				return true
			}
		case isShortFlag(alt) && len(k) > len(alt) && strings.HasPrefix(k, alt):
			if acceptsValue(fv.Values, k[len(alt):], false) {
				return true
			}
		}
	}
	return false
}

// acceptsValue reports whether a setting is one the constraint accepts, ignoring
// case: what makes `-X DELETE` destructive is the request it sends, and that
// does not turn on how the word was typed. A globbed setting accepts any value
// its pattern can produce.
func acceptsValue(values []string, got string, glob bool) bool {
	// A setting a substitution prints is any setting (unknown.go).
	if isUnknown(got) {
		return true
	}
	for _, want := range values {
		if want == "" {
			continue
		}
		if strings.EqualFold(want, got) {
			return true
		}
		if glob && globMatches(strings.ToLower(got), strings.ToLower(want)) {
			return true
		}
	}
	return false
}

// isShortFlag reports whether an alternative is a single-letter short flag
// (`-X`), the only shape that can carry an attached value.
func isShortFlag(alt string) bool {
	return len(alt) == 2 && alt[0] == '-' && alt[1] != '-'
}

// pathArgMatches reports whether some operand is a resource path rooted at
// pa.Root and exactly pa.Segments segments deep. The depth limit is the entry's
// scope, not a detail: `repos/{owner}/{repo}` IS the repository, while a deeper
// path under it names something inside the repository and stays allowed.
//
// A leading or trailing slash is ignored (`gh api /repos/owner/repo` is the same
// call), and every remaining segment must be non-empty. An operand written as a
// fully-qualified URL is normalised to its path first: `gh` passes an absolute
// URL through to the API unchanged, so spelling the host out is the same call
// and must not be a way around the same entry.
//
// An unknown operand matches when the path it spells can still be the one
// constrained (unknownOperandOnPath): `repos/$(gh repo view …)` can print the
// repository, `repos/o/r/git/refs/heads/$(…)` cannot.
func pathArgMatches(pa PathArg, ops []string) bool {
	for _, op := range ops {
		if isUnknown(op) {
			// The path it can print, and the one its known text spells when
			// every substitution in it prints nothing (unknown.go).
			if unknownOperandOnPath(pa, op) || (!vanishable(op) && pathArgMatches(pa, []string{knownText(op)})) {
				return true
			}
			continue
		}
		segs := strings.Split(strings.Trim(pathOf(op), "/"), "/")
		if len(segs) != pa.Segments || segs[0] != pa.Root {
			continue
		}
		empty := false
		for _, seg := range segs {
			if seg == "" {
				empty = true
				break
			}
		}
		if !empty {
			return true
		}
	}
	return false
}

// pathOf reduces an operand to the path part a resource constraint reads: a
// leading `scheme://host` is dropped, and so is anything from the first `?` or
// `#`. Both are normalisation, not interpretation — `https://api.github.com/
// repos/owner/repo` and `repos/owner/repo?` are the same API call as
// `repos/owner/repo`, and an entry a change of spelling walks past is not one.
//
// The host itself is deliberately NOT inspected. Reading it would mean deciding
// which hostnames are the real API, which is a lookalike-domain problem this
// guard has no way to settle; normalising every authority away can only ever
// make the depth check see a path it would otherwise have missed.
func pathOf(op string) string {
	if q := strings.IndexAny(op, "?#"); q >= 0 {
		op = op[:q]
	}
	i := strings.Index(op, "://")
	if i <= 0 {
		return op
	}
	// A scheme is letters, digits, `+`, `-`, `.`, starting with a letter —
	// anything else before the `://` is ordinary path text, not an authority.
	if c := op[0]; !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') {
		return op
	}
	for j := 1; j < i; j++ {
		c := op[j]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9',
			c == '+', c == '-', c == '.':
		default:
			return op
		}
	}
	rest := op[i+3:]
	slash := strings.IndexByte(rest, '/')
	if slash < 0 {
		return "" // a bare authority names no resource path at all
	}
	return rest[slash:]
}

// isShortCluster reports whether a token is a bundled short-flag cluster
// (-rf, -xfd): a single leading dash followed by letters or digits only.
func isShortCluster(arg string) bool {
	if len(arg) < 2 || arg[0] != '-' || arg[1] == '-' {
		return false
	}
	for i := 1; i < len(arg); i++ {
		c := arg[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		default:
			return false
		}
	}
	return true
}

func containsString(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
