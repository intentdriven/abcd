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
	"xargs":   {"-a", "--arg-file", "-d", "--delimiter", "-E", "-I", "-L", "-n", "--max-args", "-P", "--max-procs", "-s", "--max-chars", "--process-slot-var"},
	"timeout": {"-k", "--kill-after", "-s", "--signal"},
	"exec":    {"-a"},
	// `command` and `nohup` take no value flags at all.

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

// precededByCD reports whether an earlier command in the SAME chain is a `cd`.
// A cd on a previous logical line does not chain: a new line is a new shell
// command, and its failure cannot redirect this one.
func precededByCD(before []segment, chain int) bool {
	for _, s := range before {
		if s.chain != chain {
			continue
		}
		if cmd, _ := commandOf(s); cmd == "cd" {
			return true
		}
	}
	return false
}

// commandOf returns the segment's command name (basename, wrappers and
// environment-assignment prefixes stepped over) and the arguments that follow
// it. An empty name means the segment holds no command (assignments only).
func commandOf(s segment) (string, []string) {
	i, _ := commandIndex(s)
	if i < 0 {
		return "", nil
	}
	return path.Base(s.tokens[i]), s.tokens[i+1:]
}

// commandIndex locates command position: the index of the token commandOf
// names, or -1 when the segment holds no command. noglob reports that the
// wrapper chain stepped on the way carried zsh's `noglob`, which turns
// filename expansion off for the command it precedes — so the segment's glob
// record describes words bash would NOT rewrite, and the matcher compares them
// literally.
func commandIndex(s segment) (idx int, noglob bool) {
	i := 0
	for i < len(s.tokens) {
		tok := s.tokens[i]
		if isAssignment(tok) || reserved[tok] {
			i++
			continue
		}
		if tok == "coproc" {
			i = skipCoproc(s.tokens, i+1)
			continue
		}
		// The wrapper name is folded to lower case before lookup: on a
		// case-insensitive filesystem (macOS's default) `SUDO`/`ENV`/`NICE`
		// resolve to and run the real binary, so a case-varied wrapper must be
		// stepped over exactly as its lowercase spelling is, or the command it
		// launches never reaches command position (gh-315).
		if w := strings.ToLower(path.Base(tok)); wrappers[w] {
			if w == "noglob" {
				noglob = true
			}
			i = skipWrapperArgs(s.tokens, i+1, w)
			continue
		}
		break
	}
	if i >= len(s.tokens) {
		return -1, noglob
	}
	return i, noglob
}

// skipWrapperArgs advances past one wrapper's own arguments, from pos (the token
// just after the wrapper name), and returns the index where the command it
// launches begins. Without it a known wrapper turned an entry the registry does
// describe into an allow with one extra token — `sudo <hazard>` was seen,
// `sudo -u bob <hazard>` was not, because `-u` was read as the command name
// (iss-148).
func skipWrapperArgs(tokens []string, pos int, wrapper string) int {
	valueFlags := wrapperValueFlags[wrapper]
	for pos < len(tokens) {
		tok := tokens[pos]
		if tok == "--" {
			// End of the wrapper's options: everything after it is the command.
			pos++
			break
		}
		if tok == "-" || !strings.HasPrefix(tok, "-") {
			break
		}
		pos++
		if !strings.Contains(tok, "=") && containsString(valueFlags, tok) {
			pos++ // its value belongs to the wrapper, not to command position
		}
	}
	for n := wrapperOperands[wrapper]; n > 0 && pos < len(tokens); n-- {
		pos++
	}
	return pos
}

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
	ci, noglob := commandIndex(s)
	if ci < 0 {
		return false
	}
	// The command NAME is folded: a case-insensitive filesystem resolves `GIT`,
	// `RM`, `GH` to the real binaries and executes the hazard, so a byte-exact
	// compare here was a silent allow on macOS (gh-315). Only the name is folded —
	// subcommands, flags and values stay case-sensitive below, because git/gh/rm
	// parse THOSE case-sensitively, so a case-varied subcommand does not run the
	// hazard and must not be blocked.
	cmd := path.Base(s.tokens[ci])
	if !strings.EqualFold(cmd, p.Command) &&
		!(!noglob && s.globAt(ci) && globMatches(strings.ToLower(cmd), strings.ToLower(p.Command))) {
		return false
	}
	args := s.tokens[ci+1:]
	// glob reports, per ARGUMENT index, whether bash would expand that token.
	glob := func(i int) bool { return !noglob && s.globAt(ci+1+i) }
	opIdx := operandIndexes(args, p.ValueFlags)
	ops := make([]string, len(opIdx))
	for n, i := range opIdx {
		ops[n] = args[i]
	}
	if p.Subcommand != "" && !operandMatches(args, opIdx, 0, p.Subcommand, glob) {
		return false
	}
	if p.Subcommand2 != "" && !operandMatches(args, opIdx, 1, p.Subcommand2, glob) {
		return false
	}
	for _, group := range p.Flags {
		if !flagGroupMatches(group, args, glob) {
			return false
		}
	}
	for _, fv := range p.FlagValues {
		if !flagValueMatches(fv, args, glob) {
			return false
		}
	}
	for _, prefix := range p.ArgPrefixes {
		if !argPrefixMatches(prefix, ops) {
			return false
		}
	}
	for _, pa := range p.ArgPaths {
		if !pathArgMatches(pa, ops) {
			return false
		}
	}
	return true
}

// operandIndexes returns the INDEXES into args of the segment's non-flag
// arguments, in order, stepping over the value of any flag listed in valueFlags
// (`git -C /repo push` is a push). An unknown value-taking flag is not stepped
// over — the miss is a non-match, never a false block. The subcommand is operand
// 0. Indexes rather than the words themselves is what lets every caller pair an
// operand with its token's glob record, which decides whether the compare is
// literal or a pattern match.
func operandIndexes(args []string, valueFlags []string) []int {
	var idx []int
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") {
			idx = append(idx, i)
			continue
		}
		if a == "--" {
			continue
		}
		if !strings.Contains(a, "=") && containsString(valueFlags, a) {
			i++ // its value is an option argument, not an operand
		}
	}
	return idx
}

// operandMatches reports whether the n-th operand is want — literally, or as a
// word its glob pattern can produce.
func operandMatches(args []string, opIdx []int, n int, want string, glob func(int) bool) bool {
	if n < 0 || n >= len(opIdx) {
		return false
	}
	i := opIdx[n]
	return args[i] == want || (glob(i) && globMatches(args[i], want))
}

// globMatches reports whether the shell pattern can produce the literal. A
// pattern path.Match cannot parse (an unmatched `[`) is one bash leaves
// unexpanded too, so it produces nothing but itself and the literal compare
// has already had its say.
func globMatches(pattern, literal string) bool {
	ok, err := path.Match(bashGlobPattern(pattern), literal)
	return err == nil && ok
}

// bashGlobPattern rewrites a bash pattern into the dialect path.Match reads.
// One difference changes an answer: bash spells a negated character class
// `[!…]`, path.Match spells it `[^…]`, and path.Match reads bash's `!` as an
// ordinary member of the set. So `git clea[!x] -fd` — which bash expands to
// `git clean -fd` whenever a file called `clean` is there — compared as "clea
// followed by `!` or `x`", matched nothing, and allowed.
//
// A `]` first in the set is a literal member in both dialects (`[!]a]`), and a
// `[` inside an open set is literal in both, so the scan walks each class to
// its close rather than translating every `[!` it sees. An unterminated class
// is left as it stands: bash leaves such a word unexpanded too, and the literal
// compare beside this one has already had its say.
func bashGlobPattern(pattern string) string {
	if !strings.Contains(pattern, "[!") {
		return pattern
	}
	b := []byte(pattern)
	for i := 0; i < len(b); i++ {
		switch b[i] {
		case '\\':
			i++ // an escaped byte never opens a class
		case '[':
			j := i + 1
			if j < len(b) && (b[j] == '!' || b[j] == '^') {
				b[j] = '^'
				j++
			}
			if j < len(b) && b[j] == ']' {
				j++ // a `]` first in the set is a member, not the close
			}
			for j < len(b) && b[j] != ']' {
				if b[j] == '\\' {
					j++
				}
				j++
			}
			if j >= len(b) {
				return string(b)
			}
			i = j
		}
	}
	return string(b)
}

// argPrefixMatches reports whether some operand carries the prefix. Only
// operands are considered, so a prefix like "+" can never be satisfied by an
// option token: the constraint describes an argument (`git push origin
// +main:main`), not a flag.
func argPrefixMatches(prefix string, ops []string) bool {
	for _, op := range ops {
		if strings.HasPrefix(op, prefix) {
			return true
		}
	}
	return false
}

// flagGroupMatches reports whether any alternative in one "a|b" group is present
// among the argument tokens. glob reports, per argument index, whether bash
// would expand that token. The scan stops at `--`: after the terminator every
// word is an operand, so `git push -- --force origin main` pushes a refspec
// called `--force` and is not a force push.
func flagGroupMatches(group string, args []string, glob func(int) bool) bool {
	for _, alt := range strings.Split(group, "|") {
		if alt == "" {
			continue
		}
		for i, arg := range args {
			if arg == "--" {
				break
			}
			if flagMatches(alt, arg, glob(i)) {
				return true
			}
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
		if !glob || len(arg) < 2 || arg[0] != '-' || arg[1] == '-' {
			return false
		}
		return strings.ContainsRune(arg[1:], rune(alt[1])) || strings.ContainsAny(arg[1:], "*?[")
	}
	return false
}

// flagValueMatches reports whether some argument SETS one of the flag
// alternatives to one of the accepted values. All three spellings a shell user
// reaches for are read — `-X DELETE`, `-XDELETE`, `--method=DELETE` — because a
// constraint another spelling of the same call steps past is not one. A
// globbed flag or value token is compared as a pattern in the separate-token
// form; the attached forms stay literal (the floor flagMatches names). The
// FLAG half carries the same two narrowings flagGroupMatches does — the token
// must be flag-shaped, and the scan stops at `--` — because this is a flag
// position too, and the rule cannot hold at two of its three sites. The VALUE
// half is a different position: a globbed value is an ordinary word, and
// `-X DELET?` is compared as the pattern it is.
func flagValueMatches(fv FlagValue, args []string, glob func(int) bool) bool {
	for _, alt := range strings.Split(fv.Flag, "|") {
		if alt == "" {
			continue
		}
		for i, arg := range args {
			if arg == "--" {
				break
			}
			switch {
			case arg == alt || (glob(i) && flagShaped(arg) && globMatches(arg, alt)):
				// The separate-token form: the value is the next argument.
				if i+1 < len(args) && acceptsValue(fv.Values, args[i+1], glob(i+1)) {
					return true
				}
			case strings.HasPrefix(arg, alt+"="):
				if acceptsValue(fv.Values, arg[len(alt)+1:], false) {
					return true
				}
			case isShortFlag(alt) && len(arg) > len(alt) && strings.HasPrefix(arg, alt):
				// A short flag's value may be attached with no separator at all.
				if acceptsValue(fv.Values, arg[len(alt):], false) {
					return true
				}
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
func pathArgMatches(pa PathArg, ops []string) bool {
	for _, op := range ops {
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
