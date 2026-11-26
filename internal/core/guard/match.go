package guard

import (
	"path"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/shellquote"
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

	// Node package runners: `npx <pkg> [args]` runs the program its first
	// operand names, fetching the package when it is not installed
	// (iss-2610112232319827). Not a util-linux shim like those above; its
	// value flags come from npm's documented grammar (wrappersWithoutProbes).
	// Its siblings are two words, `npm exec`, `pnpm dlx`, `pnpm exec`,
	// `yarn dlx` and `bun x`, and are read by runnerSubcommands.
	"npx": true,
	// bun's package runner, `bunx <pkg> [args]`, the one-word spelling of
	// `bun x`: it runs the program its first operand names as npx does.
	"bunx": true,
}

// packageRunners names the one-word package runners among the wrappers. The
// walk steps them as wrappers, and arrives at the word itself as a command as
// well, so an entry naming the runner still matches it (walkToCommand).
var packageRunners = map[string]bool{"npx": true, "bunx": true}

// runnerSubcommands names the package managers whose subcommand makes them a
// runner as npx is, and the wrapper each subcommand is read as: `npm exec`
// (and its alias `npm x`), `pnpm dlx`, `pnpm exec`, `yarn dlx` and `bun x`.
// A subcommand the line does not fix can be any of the runner's. The
// manager's own options before it are read by wrapperValueFlags under the
// manager's name.
var runnerSubcommands = map[string]map[string]string{
	"npm":  {"exec": "npm exec", "x": "npm exec"},
	"pnpm": {"dlx": "pnpm dlx", "exec": "pnpm exec"},
	"yarn": {"dlx": "yarn dlx"},
	"bun":  {"x": "bunx"},
	// `go run <package> [args]` builds and runs the program its package path
	// names (`./cmd/abcd` runs abcd), the way npx runs the one its package
	// names: a source checkout runs its own program this way.
	"go": {"run": "go run"},
}

// runnerStart is one runner a package manager word can start: the wrapper it
// is read as (wrapperValueFlags names its options) and the index of the first
// word after its subcommand, where the runner's own options begin.
type runnerStart struct {
	wrapper string
	at      int
}

// runnersAt reports the runners the word at pos of tokens can start, in name
// order, when name, its lower-cased basename, is a package manager whose
// subcommand is a runner subcommand (one runner), or a word the line does not
// fix (every runner subcommand that manager has). The manager's own options
// before the subcommand are stepped, with the value each of its value flags
// takes (wrapperValueFlags under the manager's name): `pnpm --filter web exec`
// and `npm --prefix sub exec` start the runner `pnpm exec` and `npm exec` do.
// Each word is read every way readWord reads it.
func runnersAt(name string, tokens []string, pos int) []runnerStart {
	subs, ok := runnerSubcommands[name]
	if !ok {
		return nil
	}
	valueFlags := wrapperValueFlags[name]
	got := map[runnerStart]bool{}
	var out []runnerStart
	add := func(rs runnerStart) {
		if !got[rs] {
			got[rs] = true
			out = append(out, rs)
		}
	}
	seen := map[int]bool{}
	stack := []int{pos + 1}
	for len(stack) > 0 {
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if i >= len(tokens) || seen[i] {
			continue
		}
		seen[i] = true
		tally(1)
		tok := tokens[i]
		if w, ok := subs[tok]; ok {
			add(runnerStart{wrapper: w, at: i + 1})
			continue
		}
		r := readWord(tok, valueFlags)
		if r.operand && isUnknown(tok) {
			for _, w := range subs {
				add(runnerStart{wrapper: w, at: i + 1})
			}
		}
		if r.vanish || r.flag {
			stack = append(stack, i+1)
		}
		if r.takes {
			stack = append(stack, i+2)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].wrapper != out[j].wrapper {
			return out[i].wrapper < out[j].wrapper
		}
		return out[i].at < out[j].at
	})
	return out
}

// runnerStartsAt is every runner the word at idx of tokens starts: a one-word
// runner (npx, bunx) at the word after it, or a package manager's runner
// subcommand (runnersAt).
func runnerStartsAt(tokens []string, idx int) []runnerStart {
	w := strings.ToLower(path.Base(tokens[idx]))
	if packageRunners[w] {
		return []runnerStart{{wrapper: w, at: idx + 1}}
	}
	return runnersAt(w, tokens, idx)
}

// packagePrograms names, by the package that ships it, the program a package
// runner starts for a package whose name is not the program's: `npx
// @anthropic-ai/claude-code` starts claude, and `npx opencode-ai` starts
// opencode. A package named for its program (`@openai/codex`, `kill-port`)
// needs no line here; packageName reads it.
var packagePrograms = map[string]string{
	"@anthropic-ai/claude-code": "claude",
	"opencode-ai":               "opencode",
}

// packageProgram is the program packagePrograms names for a package runner's
// operand, its version left aside (`@anthropic-ai/claude-code@latest`), and
// whether it names one. A word the line does not fix names none.
func packageProgram(tok string) (string, bool) {
	if isUnknown(tok) {
		return "", false
	}
	spec := strings.ToLower(tok)
	if at := strings.LastIndexByte(spec, '@'); at > 0 && at > strings.LastIndexByte(spec, '/') {
		spec = spec[:at]
	}
	prog, ok := packagePrograms[spec]
	return prog, ok
}

// packageName is a package runner's operand without its version: npx runs the
// program `kill-port` for `kill-port@latest`, and `@scope/pkg@1` for `pkg`.
// An `@` before a substitution's output is not a version the line fixes, and
// the word is returned as it is.
func packageName(tok string) string {
	base := path.Base(tok)
	at := strings.LastIndexByte(base, '@')
	if at <= 0 || at < strings.LastIndexByte(base, unknownMark) {
		return tok
	}
	return base[:at]
}

// wrapperValueFlags names, per wrapper, that wrapper's OWN flags which consume
// the FOLLOWING token, so the walk to command position steps over the value
// rather than reading it as the command (`sudo -u bob rm -rf *` is an `rm`).
//
// The table is deliberately explicit and small: only wrappers in the set above
// and the package managers whose subcommand starts one (runnerSubcommands),
// only flags those programs actually document, and only the separate-token form
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
	// -P is the BSD env's search path for the program, and -a or --argv0 the
	// GNU env's name for it, each a value-taking flag.
	"env":     {"-u", "--unset", "-C", "--chdir", "-P", "-a", "--argv0"},
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
	// npm's documented exec grammar (`npm help exec`), for npx and `npm
	// exec` alike: the package to install and the workspace to run in. NOT
	// -c/--call, whose value is a command string a shell runs: listing it
	// would step that string over unread.
	"npx":      {"-p", "--package", "-w", "--workspace"},
	"npm exec": {"-p", "--package", "-w", "--workspace"},
	"pnpm dlx": {"--package"},
	"yarn dlx": {"-p", "--package"},
	// bun's documented `bunx` grammar, for `bunx` and `bun x` alike: the
	// package to install. `--bun` takes nothing.
	"bunx": {"-p", "--package"},
	// go's documented build flags that take a value (`go help build`), which
	// `go run` reads before the package: a value-taking flag missing here
	// would have its value read as the package, a miss and never a false
	// block. `-exec` names the program go runs the binary under, and is a
	// value like the rest.
	"go run": {
		"-C", "-p", "-asmflags", "-buildmode", "-compiler", "-gccgoflags", "-gcflags",
		"-installsuffix", "-ldflags", "-mod", "-modfile", "-overlay", "-pgo", "-pkgdir", "-tags",
		"-toolexec", "-exec", "-coverpkg", "-covermode",
	},
	// pnpm's documented exec grammar: the packages to run in and the one to
	// resume from. NOT -c/--shell-mode, which takes nothing and hands the
	// command to a shell.
	"pnpm exec": {"--filter", "-F", "--resume-from"},
	// A package manager's own value flags, read before its runner subcommand
	// (runnersAt): `npm --prefix sub exec`, `pnpm --filter web exec`, `yarn
	// --cwd sub dlx`. npm's are its documented configuration options with a
	// short form or a path or name value; pnpm's are its documented global
	// options (directory, filtering, reporting); yarn's only global value
	// flag is --cwd. A boolean (`pnpm -r`, `npm --yes`) needs no entry.
	"npm": {
		"-C", "--prefix", "-w", "--workspace", "-L", "--location",
		"--loglevel", "--userconfig", "--globalconfig", "--cache", "--registry",
	},
	"pnpm": {
		"-C", "--dir", "--filter", "-F", "--filter-prod", "--test-pattern",
		"--changed-files-ignore-pattern", "--reporter", "--loglevel",
		"--workspace-concurrency",
	},
	"yarn": {"--cwd"},
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

// envSetBefore reports whether a word before the command at site sets one
// of names in its environment: an assignment word, which the walk steps as
// the shell's own prefix or as an operand of env or sudo (envAssigns). A
// word the line does not fix counts where it writes an `=` its name can
// stand before (`env "$(true)"NAME=…`, `env "$N"=…`), or, with no `=`
// written, where it stands among env's or sudo's operands, which a
// substitution can print whole (`env "$PAIR" opencode`); a word that is an
// option's value (`sudo -u "$U"`, `env -C "$D"`, envOptionValueNext,
// sudoOptionValueNext) is no assignment, unless an expansion that can yield
// several words stands in it (segment.fieldSplit: an unquoted one, or a
// quoted `"$@"` or `"${a[@]}"`): a word after the first is an operand of its
// own, which the expansion prints whole whatever text the line writes
// before it (`env -u $V opencode`, `env -u x"$@" opencode`).
func envSetBefore(s segment, site int, names []string) bool {
	operands := ""       // the envAssigns wrapper whose operands the words since it are
	optionValue := false // the word is the value of the wrapper's option before it
	for i := 0; i < site && i < len(s.tokens); i++ {
		tok := s.tokens[i]
		if optionValue {
			optionValue = false
			if !s.fieldSplit[i] {
				continue
			}
			return true
		}
		if setsName(tok, names) || couldSetName(tok, names) && (strings.IndexByte(tok, '=') >= 0 || operands != "") {
			return true
		}
		switch w := strings.ToLower(path.Base(tok)); {
		case !isUnknown(tok) && envAssigns[w]:
			operands = w
		case operands == "env" && strings.HasPrefix(tok, "-"):
			optionValue = envOptionValueNext(tok)
		case operands == "sudo" && !isUnknown(tok) && strings.HasPrefix(tok, "-"):
			optionValue = sudoOptionValueNext(tok)
		case operands != "" && (isUnknown(tok) || isWrapperAssignment(tok)):
		default:
			operands = ""
		}
	}
	return false
}

// sudoOptionValueNext reports whether the sudo option tok takes the next word
// as its value: one of sudo's value flags (wrapperValueFlags) standing alone,
// or a cluster whose first value flag stands last (`-nu`), read as the
// wrapper walk reads it (abbreviatesValueFlag): in `-uroot` the `-u` takes
// the rest of the word, so the next word is not its value.
func sudoOptionValueNext(tok string) bool {
	flags := wrapperValueFlags["sudo"]
	return containsString(flags, tok) || abbreviatesValueFlag(tok, flags)
}

// envOptionValueNext reports whether the env option tok takes the next word
// as its value: `-u`, `-C`, `-P` and `-a` standing last in a cluster of
// short options (`-iu`), and `--unset`, `--chdir` and `--argv0` without an
// `=`, abbreviated as getopt allows. `-S` is not among them: env reads the
// string it splits as its own operands, assignments included.
func envOptionValueNext(tok string) bool {
	if long, ok := strings.CutPrefix(tok, "--"); ok {
		if long == "" || strings.ContainsRune(long, '=') {
			return false
		}
		for _, name := range []string{"unset", "chdir", "argv0"} {
			if strings.HasPrefix(name, long) {
				return true
			}
		}
		return false
	}
	for k := 1; k < len(tok); k++ {
		switch c := tok[k]; {
		case strings.IndexByte("uCPa", c) >= 0:
			return k == len(tok)-1
		case c == 'S' || !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'):
			return false
		}
	}
	return false
}

// setsName reports whether the word tok sets one of names: `NAME=…`,
// `NAME+=…`, which appends to it, or `NAME[SUB]=…` (splitAssignment),
// whatever its value, a value the line does not fix among them.
func setsName(tok string, names []string) bool {
	name, _, _, _, ok := splitAssignment(tok)
	return ok && containsString(names, name)
}

// couldSetName is setsName for an operand a program reads as an assignment
// after the shell has expanded it: env's and sudo's operands before their
// command, and export's and the declaration builtins'. A word whose name a
// substitution prints can set any name its known lead allows
// (assignmentCouldName), so `env "$(true)"NAME=…`, `env "$X"=…` and `export
// "$PAIR"` can each set NAME.
func couldSetName(tok string, names []string) bool {
	if setsName(tok, names) {
		return true
	}
	for _, n := range names {
		if assignmentCouldName(tok, n) {
			return true
		}
	}
	return false
}

// assignmentCouldName reports whether a word the line does not fix can print
// an assignment to name: the part of it before its first written `=` (the
// whole word where it has none, the `=` then being one a substitution
// prints) holds a substitution, and the text before that substitution is
// the start of name, or name followed by the `+` or `[` that can stand
// between a name and its `=`.
func assignmentCouldName(tok, name string) bool {
	head := tok
	if eq := strings.IndexByte(tok, '='); eq >= 0 {
		head = tok[:eq]
	}
	if !isUnknown(head) {
		return false
	}
	lead := knownLead(head)
	if strings.HasPrefix(name, lead) {
		return true
	}
	return strings.HasPrefix(lead, name) && (lead[len(name)] == '+' || lead[len(name)] == '[')
}

// exportState is what the export walk (exportsBefore) carries from one
// command to the next: whether allexport holds, whether bash's keyword
// option holds (`set -k`: an assignment word anywhere among a command's
// words is in that command's environment), and the names the walk reads an
// export of, the entry's own, each nameref the line points at one of them
// (namerefsTo) and each array it ties to one of them (tiedArraysTo). shellopts is the SHELLOPTS the shell hands the
// programs it starts, "" where it hands none: bash starts with every option
// an inherited SHELLOPTS names (shellStartsWith). optsAssigned is a value the
// line assigned SHELLOPTS without exporting it, which an `export SHELLOPTS`
// after it hands on.
type exportState struct {
	allexport, keyword      bool
	names                   []string
	shellopts, optsAssigned string
}

// shelloptsName is the variable whose value bash reads its options from when
// it starts.
const shelloptsName = "SHELLOPTS"

// shelloptsValue reads tok as an assignment to SHELLOPTS: its value, which
// is unknownText where the word's name is not fixed and expanded is set, the
// word being an operand a program reads after expansion (couldSetName).
func shelloptsValue(tok string, expanded bool) (string, bool) {
	if name, eq, _, _, ok := splitAssignment(tok); ok {
		return tok[eq+1:], name == shelloptsName
	}
	if expanded && assignmentCouldName(tok, shelloptsName) {
		return unknownText, true
	}
	return "", false
}

// shelloptsNames reports whether a SHELLOPTS value turns opt on: it names
// it among its colon-separated options, or the line does not fix it.
func shelloptsNames(v string, opt shellOption) bool {
	return v != "" && (isUnknown(v) || containsString(strings.Split(v, ":"), opt.name))
}

// ownShellopts is the SHELLOPTS bash holds for its own options in st, which
// `export SHELLOPTS` hands on where the line assigned it no value: the
// options the walk follows that hold, and one that always does.
func ownShellopts(st exportState) string {
	opts := []string{"braceexpand"}
	if st.allexport {
		opts = append(opts, optAllexport.name)
	}
	if st.keyword {
		opts = append(opts, optKeyword.name)
	}
	return strings.Join(opts, ":")
}

// The options the export walk follows, by the letter `set` takes and the
// name `set -o` takes.
var (
	optAllexport = shellOption{letter: 'a', name: "allexport"}
	optKeyword   = shellOption{letter: 'k', name: "keyword"}
)

// shellOption is one shell option as `set` spells it: its letter and its
// long name.
type shellOption struct {
	letter byte
	name   string
}

// exportStep reads the command s for the export walk (exportsBefore): whether
// it exports one of st.names, and the state after it. It exports one with
// `export NAME` or `export NAME=…`, and with `declare -x`, `typeset -x` or
// `local -x` and the same operands (exportOperands). Where allexport holds,
// a bare assignment (`NAME=…` with no command after it), an assignment a
// declaration builtin makes, and a name a builtin assigns by its operands
// (assignsOperand) or a `for` or `select` loop takes (loopNames) export too,
// as does a name a declaration builtin declares without a value.
// A file s sources runs in the same shell, so a variable it exports counts.
// A name a substitution prints can be any of them (nameCouldBe). The same
// reading follows SHELLOPTS: assigned, and exported (exportState.shellopts).
func exportStep(s segment, st exportState) (exports bool, after exportState) {
	names := st.names
	shellopts, assigned := st.shellopts, st.optsAssigned
	sites := commandSites(s)
	if len(sites) == 0 {
		for _, tok := range s.tokens {
			if st.allexport && setsName(tok, names) {
				exports = true
			}
			if v, ok := shelloptsValue(tok, false); ok {
				assigned = v
				if st.allexport {
					shellopts = v
				}
			}
		}
	}
	for _, a := range sites {
		if st.allexport && (assignsOperand(s.tokens, a.idx, names) || loopNames(s.tokens, a.idx, names)) {
			exports = true
		}
		first, explicit, unexports, ok := exportOperands(s.tokens, a.idx)
		if !ok || !(explicit || st.allexport && !unexports) {
			continue
		}
		// A builtin the line names reads each operand after expansion, so
		// one whose name a substitution prints can be any of the names
		// (couldSetName); one whose own name the line does not fix is read
		// by the operands it spells, or every command it ran would export.
		fixed := !isUnknown(s.tokens[a.idx])
		named := func(tok, n string) bool { return tok == n || fixed && wordCouldBe(tok, n) }
		for _, tok := range s.tokens[first:] {
			if setsName(tok, names) || fixed && couldSetName(tok, names) {
				exports = true
			}
			// A name declared with no value is exported under allexport as
			// with -x: the scalar `typeset -T` ties to an array among them,
			// which an assignment to the array then sets.
			for _, n := range names {
				if named(tok, n) {
					exports = true
				}
			}
			if explicit && named(tok, shelloptsName) {
				shellopts = assigned
				if shellopts == "" {
					shellopts = ownShellopts(st)
				}
			}
			if v, ok := shelloptsValue(tok, fixed); ok {
				shellopts = v
			}
		}
	}
	after = exportState{
		allexport:    optionAfter(s, st.allexport, optAllexport, st.shellopts),
		keyword:      optionAfter(s, st.keyword, optKeyword, st.shellopts),
		names:        tiedArraysTo(s, namerefsTo(s, names)),
		shellopts:    shellopts,
		optsAssigned: assigned,
	}
	// Each file s sources is read as its own walk, from the state the source
	// runs under; where s sources several files (one per way the line can be
	// read), an option holds after it where any of them leaves it on.
	if len(s.sourced) > 0 {
		in, out := after, after
		for _, file := range s.sourced {
			_, e, end := exportsBefore(file, in)
			exports = exports || e
			out.allexport = out.allexport || end.allexport
			out.keyword = out.keyword || end.keyword
			if end.shellopts != in.shellopts {
				out.shellopts = end.shellopts
			}
		}
		after = out
	}
	return exports, after
}

// exportsBefore walks the commands of segs in the order they run (runOrder)
// and reports, for each, whether one of st.names is in its environment from
// a command that runs before it (exportStep), or from an assignment word
// among its own words after its name while the keyword option holds; whether
// any command of segs exports one; and the state after the last of them. st
// is the state before the first. A string an eval runs is placed where the
// eval stands, so an export in it holds for the commands after the eval.
//
// A `${NAME:=…}` or `${NAME=…}` assigns where the expansion stands, and so
// does an arithmetic assignment or a redirection that names its descriptor,
// which the text records only for the whole text (segment.assignsInText,
// segment.numbersInText), so while allexport holds such an assignment of one
// of the names counts as an export from the first command allexport holds at.
func exportsBefore(segs []segment, st exportState) (before []bool, exported bool, end exportState) {
	before, _, exported, end = exportWalk(segs, st)
	return before, exported, end
}

// exportWalk is exportsBefore, and the state before each command of segs.
func exportWalk(segs []segment, st exportState) (before []bool, states []exportState, exported bool, end exportState) {
	expands := false
	for _, s := range segs {
		for _, n := range s.assignsInText {
			expands = expands || containsString(st.names, n)
		}
		for _, n := range s.numbersInText {
			expands = expands || n == anyNameAssigned || containsString(st.names, n)
		}
	}
	before = make([]bool, len(segs))
	states = make([]exportState, len(segs))
	for _, i := range runSequence(segs) {
		states[i] = st
		before[i] = exported || st.keyword && assignsAfterCommand(segs[i], st.names)
		e, next := exportStep(segs[i], st)
		if expands && (st.allexport || next.allexport) {
			e = true
		}
		exported, st = exported || e, next
	}
	return before, states, exported, st
}

// assignsAfterCommand reports whether an assignment word to one of names
// stands after the name of a command of s, which bash's keyword option puts
// in that command's environment.
func assignsAfterCommand(s segment, names []string) bool {
	for _, a := range commandSites(s) {
		for _, tok := range s.tokens[a.idx+1:] {
			if setsName(tok, names) {
				return true
			}
		}
	}
	return false
}

// operandSetters are the builtins, besides export and the declaration
// builtins, that assign a variable their operands name: `read NAME`,
// `printf -v NAME`, `getopts OPTS NAME`, `mapfile NAME`, `let NAME=…`,
// bash's `wait -p NAME` and `compgen -V NAME`, and zsh's `vared`,
// `zparseopts`, `print -v`, `strftime -s NAME`, `zstyle -s … NAME`,
// `zformat -f NAME`, `getln NAME` and `sysread NAME`.
var operandSetters = []string{"read", "mapfile", "readarray", "getopts", "printf", "print", "let", "vared", "zparseopts",
	"wait", "compgen", "strftime", "zstyle", "zformat", "getln", "sysread"}

// assignsOperand reports whether the command at site is one of
// operandSetters handed one of names to assign: as a word of its own, as
// `NAME=…`, as printf's `-vNAME`, or as the `NAME?prompt` zsh's read takes.
// Any operand that spells the name counts, a prompt or a format among them,
// which reaches past what the builtin assigns.
func assignsOperand(tokens []string, site int, names []string) bool {
	if !nameCouldBeAny(tokens[site], operandSetters) {
		return false
	}
	for _, tok := range tokens[site+1:] {
		if setsName(tok, names) {
			return true
		}
		for _, n := range names {
			if tok == n || tok == "-v"+n || strings.HasPrefix(tok, n+"?") {
				return true
			}
		}
	}
	return false
}

// loopNames reports whether the command at site is a `for` or `select`
// loop that assigns one of names: the words after the keyword up to `in`,
// `do` or a `(`, of which zsh takes several (`for a b in …`).
func loopNames(tokens []string, site int, names []string) bool {
	if tok := tokens[site]; tok != "for" && tok != "select" {
		return false
	}
	for _, tok := range tokens[site+1:] {
		if tok == "in" || tok == "do" || strings.HasPrefix(tok, "(") || !isName(tok) {
			return false
		}
		if containsString(names, tok) {
			return true
		}
	}
	return false
}

// namerefsTo is names and every name s makes a reference to one of them:
// `declare -n REF=NAME`, with typeset or local too, after which an export
// or an assignment through REF is one of NAME.
func namerefsTo(s segment, names []string) []string {
	out := names
	for _, a := range commandSites(s) {
		if !nameCouldBeAny(s.tokens[a.idx], []string{"declare", "typeset", "local"}) {
			continue
		}
		ref := false
		i := a.idx + 1
		for ; i < len(s.tokens); i++ {
			w := s.tokens[i]
			if w == "--" {
				i++
				break
			}
			if isUnknown(w) {
				ref = ref || clusterCouldCarry(w, 'n')
				continue
			}
			if len(w) < 2 || (w[0] != '-' && w[0] != '+') {
				break
			}
			if strings.ContainsRune(w[1:], 'n') {
				ref = w[0] == '-'
			}
		}
		if !ref {
			continue
		}
		for _, tok := range s.tokens[i:] {
			name, eq, _, _, ok := splitAssignment(tok)
			if !ok || isUnknown(tok) || !containsString(out, tok[eq+1:]) || containsString(out, name) {
				continue
			}
			if len(out) == len(names) {
				out = append([]string(nil), names...)
			}
			out = append(out, name)
		}
	}
	return out
}

// tiedArraysTo is names and every array s ties to one of them as zsh's
// `typeset -T NAME arr` does (with declare, local or export, `-T` in any
// cluster), the array given a value or not (`arr=(…)`): the array sets the
// scalar whenever it is assigned, so under allexport an assignment to it,
// however long after the tie, is an export of NAME. An option word the line
// does not fix that can carry T counts as one, and a scalar the line does
// not fix (`typeset -T "$S" arr`) can be any name, so it ties the array to
// one of names; such a word read as an option is also read as the scalar.
func tiedArraysTo(s segment, names []string) []string {
	out := names
	add := func(tok string) {
		array := declaredName(tok)
		if isUnknown(array) || !isName(array) || containsString(out, array) {
			return
		}
		if len(out) == len(names) {
			out = append([]string(nil), names...)
		}
		out = append(out, array)
	}
	for _, a := range commandSites(s) {
		if !nameCouldBeAny(s.tokens[a.idx], append([]string{"export"}, declarationBuiltins...)) {
			continue
		}
		tie := false
		i := a.idx + 1
		for ; i < len(s.tokens); i++ {
			w := s.tokens[i]
			if w == "--" {
				i++
				break
			}
			if isUnknown(w) {
				if tie && i+1 < len(s.tokens) {
					add(s.tokens[i+1])
				}
				tie = tie || clusterCouldCarry(w, 'T')
				continue
			}
			if len(w) < 2 || (w[0] != '-' && w[0] != '+') {
				break
			}
			if w[0] == '-' && strings.ContainsRune(w[1:], 'T') {
				tie = true
			}
		}
		if !tie || i+1 >= len(s.tokens) {
			continue
		}
		if scalar := s.tokens[i]; isUnknown(scalar) || containsString(out, declaredName(scalar)) {
			add(s.tokens[i+1])
		}
	}
	return out
}

// declaredName is the name a declaration builtin's operand tok declares:
// the text before its `=` or `+=` where it assigns a value (`arr=(…)`), tok
// itself where it does not.
func declaredName(tok string) string {
	if name, _, _, _, ok := splitAssignment(tok); ok {
		return name
	}
	return tok
}

// runOrder is, for each command of segs, where it runs on the line: its own
// index, after the place of the command that carries its string
// (segment.carrier), so earlier compares where two commands run.
func runOrder(segs []segment) [][]int {
	order := make([][]int, len(segs))
	for i, s := range segs {
		if s.carrier > 0 && s.carrier-1 < i {
			order[i] = append(append([]int(nil), order[s.carrier-1]...), i)
		} else {
			order[i] = []int{i}
		}
	}
	return order
}

// runSequence is the indices of segs in the order the commands run
// (runOrder): a string's commands where the command that carries them stands,
// rather than after the line's own.
func runSequence(segs []segment) []int {
	seq := make([]int, len(segs))
	carried := false
	for i := range segs {
		seq[i] = i
		carried = carried || segs[i].carrier > 0
	}
	if !carried {
		return seq
	}
	order := runOrder(segs)
	sort.SliceStable(seq, func(a, b int) bool { return earlier(order[seq[a]], order[seq[b]]) })
	return seq
}

// optionAfter reports whether the shell option opt holds after the command
// s, given whether it held before it. Each way a line sets an option is
// read: `set` with the option's letter, alone or in a cluster, or its name
// after `-o` (setOption); zsh's `setopt` and `unsetopt`, by letter, by name
// in any of its spellings, or by a `-m` pattern, which is read as setting
// it (setoptOption); zsh's `emulate`, with set's options after the mode
// (emulateOption); bash's `shopt -s -o` and `shopt -u -o` (shoptOption); an
// assignment to zsh's options array (optionsArrayOption); and a shell started
// with the option (shellStartsWith). A shell's string runs
// where the shell stands (runOrder), so an option the shell starts with
// holds for the commands after it on the line too, a reading that counts
// more exports, never fewer. A word a substitution prints can be an option
// that sets it. shellopts is the SHELLOPTS the shell hands the shells it
// starts (exportState.shellopts).
func optionAfter(s segment, on bool, opt shellOption, shellopts string) bool {
	if s.arrayOf == zshOptionsArray {
		return true
	}
	sites := commandSites(s)
	if len(sites) == 0 {
		return optionsArrayOption(s.tokens, on, opt)
	}
	// Each place the command can sit, and each builtin a name the line does
	// not fix can be, is one way the command runs: opt holds after it where
	// any of them leaves it on (`$S -a` is set's -a and also unsetopt's).
	after := false
	for _, a := range sites {
		tok := s.tokens[a.idx]
		args := s.tokens[a.idx+1:]
		var ways []bool
		if nameCouldBe(tok, "set") {
			ways = append(ways, setOption(args, on, opt, !isUnknown(tok)))
		}
		if nameCouldBe(tok, "setopt") {
			ways = append(ways, setoptOption(args, on, opt, true))
		}
		if nameCouldBe(tok, "unsetopt") {
			ways = append(ways, setoptOption(args, on, opt, false))
		}
		if nameCouldBe(tok, "emulate") {
			ways = append(ways, emulateOption(args, on, opt, !isUnknown(tok)))
		}
		if nameCouldBe(tok, "shopt") {
			ways = append(ways, shoptOption(args, on, opt))
		}
		if len(ways) == 0 {
			ways = append(ways, on)
		}
		if shellStartsWith(s, a.idx, opt, shellopts) {
			ways = append(ways, true)
		}
		for _, w := range ways {
			after = after || w
		}
	}
	return after
}

// zshOptionsArray is zsh's associative array of its options, whose elements
// set them: `options[allexport]=on` turns allexport on.
const zshOptionsArray = "options"

// optionsArrayOption is opt after the assignments toks, a command of
// assignments alone, make to zsh's options array (zshOptionsArray), given on
// before them: an element named for opt in any of zsh's spellings
// (zshOption) set to exactly `off`, or its `no` form set to exactly `on`,
// turns it off, and any other value is read as turning it on: zsh accepts
// only those two values in lower case and keeps the option as it was for
// any other, so `OFF` or `0` after `set -a` leaves allexport on. An element
// whose name or value the line does not fix, and the whole array assigned,
// can turn it on.
func optionsArrayOption(toks []string, on bool, opt shellOption) bool {
	for _, tok := range toks {
		name, eq, _, subscript, ok := splitAssignment(tok)
		if !ok || name != zshOptionsArray {
			continue
		}
		if !subscript {
			on = true
			continue
		}
		key, value := tok[len(name)+1:strings.IndexByte(tok, ']')], tok[eq+1:]
		if isUnknown(key) || strings.ContainsRune(key, varMark) || isUnknown(value) || strings.ContainsRune(value, varMark) {
			on = true
			continue
		}
		switch zshOption(key) {
		case opt.name:
			on = value != "off"
		case "no" + opt.name:
			on = value != "on"
		}
	}
	return on
}

// setOption is opt after set's arguments args, given on before them: `-X`
// sets the option whose letter is X and `+X` unsets it, alone or in a
// cluster, and `-o NAME` and `+o NAME` name it, zsh also reading the name
// from the rest of the word (`-oallexport`). Options end at `--`, `-`, or the
// first word that is not one, which starts the positional parameters. A word
// the line does not fix whose known lead is empty or a dash can be any
// option, opt among them, so it sets opt where named records that the line
// spells set's own name: `set "$X"` and `set -e$X` turn allexport on, while
// `set x"$X"` starts the positional parameters. A command whose name a
// substitution prints is read by the option words it can be
// (clusterCouldCarry), or every such command with an argument it does not
// fix would turn the option on.
func setOption(args []string, on bool, opt shellOption, named bool) bool {
	for i := 0; i < len(args); i++ {
		w := args[i]
		if w == "--" || w == "-" {
			break
		}
		if isUnknown(w) {
			lead := knownLead(w)
			if named && lead != "" && lead[0] != '-' && lead[0] != '+' {
				break
			}
			on = on || named && (lead == "" || lead[0] == '-') || clusterCouldCarry(w, opt.letter) || clusterCouldCarry(w, 'o')
			continue
		}
		if len(w) < 2 || (w[0] != '-' && w[0] != '+') {
			break
		}
		var j int
		on, j = optionCluster(args, i, on, opt, w[0] == '-')
		i = j
	}
	return on
}

// optionCluster reads the option word args[i] as set reads it, given on
// before it and whether its options are set (turnsOn) or unset: opt's
// letter sets or unsets it, and an `o` names an option in the rest of the
// word or, where the word ends there, in the next. It returns opt after the
// word and the index of the last word it read.
func optionCluster(args []string, i int, on bool, opt shellOption, turnsOn bool) (bool, int) {
	w := args[i]
	for j := 1; j < len(w); j++ {
		switch w[j] {
		case opt.letter:
			on = turnsOn
		case 'o':
			name := w[j+1:]
			if name == "" {
				if i+1 >= len(args) {
					return on, i
				}
				i++
				name = args[i]
			}
			return optionNamed(name, on, turnsOn, opt), i
		}
	}
	return on, i
}

// optionNamed is opt after a word naming an option is set (sets) or unset,
// given on before it: opt by its name in any of zsh's spellings (zshOption)
// takes the change, its `no` form the opposite one, and a word a
// substitution prints can set it.
func optionNamed(name string, on, sets bool, opt shellOption) bool {
	switch {
	case isUnknown(name):
		return true
	case zshOption(name) == opt.name:
		return sets
	case zshOption(name) == "no"+opt.name:
		return !sets
	}
	return on
}

// setoptOption is opt after the arguments of zsh's setopt (sets) or
// unsetopt, given on before them: an option named, set or unset; a letter
// cluster as set reads it, `-` setting and `+` unsetting under setopt and
// the other way under unsetopt; and, after `-m`, names that are patterns,
// which the guard does not match, so each is read as one that sets opt.
func setoptOption(args []string, on bool, opt shellOption, sets bool) bool {
	patterns := false
	for i := 0; i < len(args); i++ {
		w := args[i]
		switch {
		case isUnknown(w):
			on = true
		case len(w) >= 2 && (w[0] == '-' || w[0] == '+'):
			if strings.ContainsRune(w[1:], 'm') && !strings.ContainsRune(w[1:], 'o') {
				patterns = true
			}
			on, i = optionCluster(args, i, on, opt, (w[0] == '-') == sets)
		case patterns:
			on = true
		default:
			on = optionNamed(w, on, sets, opt)
		}
	}
	return on
}

// emulateOption is opt after the arguments of zsh's `emulate [-lLR] MODE
// [OPTIONS]`, given on before them: the options after the mode are set's.
func emulateOption(args []string, on bool, opt shellOption, named bool) bool {
	for i, w := range args {
		if !isUnknown(w) && strings.HasPrefix(w, "-") {
			continue
		}
		return setOption(args[i+1:], on, opt, named)
	}
	return on
}

// shoptOption is opt after the arguments of bash's shopt, given on before
// them: with -o its operands are set's own long names, which -s sets and -u
// unsets (`shopt -so allexport`); without -o they are shopt's options, none
// of which the walk follows.
func shoptOption(args []string, on bool, opt shellOption) bool {
	sets, unsets, named := false, false, false
	i := 0
	for ; i < len(args); i++ {
		w := args[i]
		if w == "--" {
			i++
			break
		}
		if isUnknown(w) {
			sets = sets || clusterCouldCarry(w, 's')
			named = named || clusterCouldCarry(w, 'o')
			continue
		}
		if len(w) < 2 || w[0] != '-' {
			break
		}
		sets = sets || strings.ContainsRune(w[1:], 's')
		unsets = unsets || strings.ContainsRune(w[1:], 'u')
		named = named || strings.ContainsRune(w[1:], 'o')
	}
	if !named || sets == unsets {
		return on
	}
	for _, w := range args[i:] {
		on = optionNamed(w, on, sets, opt)
	}
	return on
}

// shellStartsWith reports whether the command at site starts a shell
// (isShellFamily) with opt on: its letter in a cluster of options, its name
// after `-o`, in the rest of an `-o` word or as zsh's `--NAME`, among the
// options before the shell's first operand; or a SHELLOPTS among the
// assignments before the shell, through env too, or in shellopts, the
// SHELLOPTS the line exported before it, that names it. A word the line
// does not fix among the options, or as SHELLOPTS's value or name, can set
// it.
func shellStartsWith(s segment, site int, opt shellOption, shellopts string) bool {
	tok := s.tokens[site]
	if isUnknown(tok) || strings.ContainsRune(tok, varMark) || !isShellFamily(path.Base(tok)) {
		return false
	}
	if shelloptsNames(shellopts, opt) {
		return true
	}
	for _, w := range s.tokens[:site] {
		if v, ok := shelloptsValue(w, true); ok && shelloptsNames(v, opt) {
			return true
		}
	}
	args := s.tokens[site+1:]
	for i := 0; i < len(args); i++ {
		w := args[i]
		switch {
		case isUnknown(w):
			if clusterCouldCarry(w, opt.letter) || clusterCouldCarry(w, 'o') {
				return true
			}
			continue
		case w == "--" || w == "-" || len(w) < 2 || w[0] != '-' && w[0] != '+':
			return false
		case strings.HasPrefix(w, "--"):
			if zshOption(w[2:]) == opt.name {
				return true
			}
			if w == "--rcfile" || w == "--init-file" {
				i++
			}
			continue
		}
		for j := 1; j < len(w); j++ {
			switch c := w[j]; {
			case c == opt.letter && w[0] == '-':
				return true
			case c == 'o' && j+1 < len(w):
				return w[0] == '-' && zshOption(w[j+1:]) == opt.name
			case c == 'o' || c == 'O':
				// The option's name is the next word.
				if i+1 < len(args) {
					i++
					if c == 'o' && w[0] == '-' && zshOption(args[i]) == opt.name {
						return true
					}
				}
			}
		}
	}
	return false
}

// zshOption is an option name as zsh compares it: case and underscores
// ignored (`ALL_EXPORT` is `allexport`).
func zshOption(w string) string {
	return strings.ToLower(strings.ReplaceAll(w, "_", ""))
}

// declarationBuiltins set the variables their operands name and take
// typeset's options: zsh's `integer` and `float` among them, which are
// typeset with a number's type.
var declarationBuiltins = []string{"declare", "typeset", "local", "readonly", "integer", "float"}

// exportOperands reads the command at site as one that sets the variables
// its operands name: the index of its first operand, whether it exports them
// (explicit), whether it takes the export away (unexports), and ok where the
// command is export or a declaration builtin at all. export always exports.
// declare, typeset, local, readonly, integer and float do when an option
// among the words before their first operand carries x (`-x`, `-gx`, `-rx`),
// as bash and zsh read it, and take the export away when the last such
// option is `+x`; `--` ends the options. A word a substitution prints can be an option, -x among
// them.
func exportOperands(tokens []string, site int) (first int, explicit, unexports, ok bool) {
	tok := tokens[site]
	if nameCouldBe(tok, "export") {
		return site + 1, true, false, true
	}
	if !nameCouldBeAny(tok, declarationBuiltins) {
		return 0, false, false, false
	}
	i := site + 1
	for ; i < len(tokens); i++ {
		w := tokens[i]
		if w == "--" {
			i++
			break
		}
		if isUnknown(w) {
			// It can be an option, so the words after it can be too.
			explicit = explicit || clusterCouldCarry(w, 'x')
			continue
		}
		if len(w) < 2 || (w[0] != '-' && w[0] != '+') {
			break
		}
		if strings.ContainsRune(w[1:], 'x') {
			explicit, unexports = w[0] == '-', w[0] == '+'
		}
	}
	return i, explicit, unexports, true
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

// skipFunction advances past the `function` keyword's NAME, from pos (the
// token just after `function`), and returns the index where the function's
// body begins. `function NAME { …; }` defines a function as `NAME() { …; }`
// does, and the body runs when the line calls it (iss-2610090821313095): the
// NAME is stepped so the body's first command reaches command position, and
// a `{` or a reserved word opening the body is stepped as reserved. The
// tokenizer splits `function NAME() {` at the parentheses, which leaves
// `function NAME` a segment of its own and the body a segment after it, so
// stepping the NAME leaves that segment no command, as `NAME()` leaves none.
// The NAME may be a word no identifier rule covers: bash accepts
// `function a-b {`.
//
// The guard reads the keyword in its ordinary form only
// (dcn-2611260046000744): one plain NAME (plainFunctionName), then the end of
// the segment, an unquoted `{`, or a compound command bash reads as the body
// with no `{` after it in the segment. ok is false for every other form, the
// several NAMEs and options zsh takes among them, and the caller holds the
// command (functionFormSignal) rather than reading the shell's wider grammar.
// lastBrace is the index of the segment's last `{` token, or -1, so a line of
// keywords costs one pass.
func skipFunction(tokens []string, pos, lastBrace int) (body int, ok bool) {
	if pos >= len(tokens) || !plainFunctionName(tokens[pos]) {
		return pos, false
	}
	next := pos + 1
	if next == len(tokens) || tokens[next] == "{" || functionBodyWords[tokens[next]] && lastBrace < next {
		return next, true
	}
	return pos, false
}

// functionBodyWords are the words that open a compound command bash takes as
// a function's body without a `{`.
var functionBodyWords = map[string]bool{"if": true, "while": true, "until": true, "for": true, "case": true, "select": true, "[[": true}

// functionShellWords are the words a shell gives a meaning of its own, which
// a plain function NAME is never.
var functionShellWords = map[string]bool{
	"fi": true, "done": true, "esac": true, "in": true, "for": true, "case": true, "select": true,
	"function": true, "time": true, "coproc": true, "repeat": true, "[[": true, "]]": true,
}

// plainFunctionName reports whether name is the one plain NAME of the
// keyword's ordinary form: a word the line fixes, no reserved or shell word,
// no option, and no brace in it.
func plainFunctionName(name string) bool {
	return name != "" && !reserved[name] && !functionShellWords[name] && !isUnknown(name) &&
		name[0] != '-' && !strings.ContainsAny(name, "{}")
}

// lastBraceIndex is the index of the last `{` among tokens, or -1.
func lastBraceIndex(tokens []string) int {
	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i] == "{" {
			return i
		}
	}
	return -1
}

// functionQuoted reports whether a `function` keyword the line leaves
// unquoted, at command position, takes a NAME the line quotes or escapes, or
// has a quoted or escaped `{` after its NAME, where the tokens alone read an
// opener (`function f '{' g { …; }`). bare holds the indices of the words the
// tokenizer saw wholly unquoted among the keyword, its NAME and `{` (its
// keywords record). The command is held (functionFormSignal). defined is the
// NAME of each such keyword at command position.
func functionQuoted(toks []string, bare map[int]bool) (quoted bool, defined []string) {
	for i := 0; i < len(toks); {
		switch t := toks[i]; {
		case isAssignment(t) || reserved[t] || t == "time":
			i++
		case t == "coproc":
			i = skipCoproc(toks, i+1)
		case t == "repeat":
			i = skipWord(toks, i+1)
		case t == "function" && bare[i]:
			if i+1 < len(toks) {
				defined = append(defined, toks[i+1])
			}
			if i+1 < len(toks) && !bare[i+1] || i+2 < len(toks) && toks[i+2] == "{" && !bare[i+2] {
				return true, defined
			}
			i += 2
		default:
			return false, defined
		}
	}
	return false, defined
}

// argumentName reports whether name, a parameter's name as a `$` or `${`
// writes it (its flags and a `#` stripped), is one of a function's
// arguments: `1`…`9` and on, `@`, `*`, or zsh's `argv`.
func argumentName(name string) bool {
	name = strings.TrimLeft(name, "#!^~=")
	// zsh's flags in parentheses: `${(@)argv}`, `${(z)1}`.
	if strings.HasPrefix(name, "(") {
		if k := strings.IndexByte(name, ')'); k > 0 {
			name = strings.TrimLeft(name[k+1:], "#!^~=")
		}
	}
	if name == "" {
		return false
	}
	if c := name[0]; c >= '1' && c <= '9' || c == '@' || c == '*' {
		return true
	}
	if rest, ok := strings.CutPrefix(name, "argv"); ok {
		return rest == "" || !isShellNameByte(rest[0])
	}
	return false
}

// isShellNameByte reports whether c can continue a parameter's name.
func isShellNameByte(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// plainArgument reports whether name, a `$` parameter's, is an argument
// whose value is the call's word as it is: `1`…`9`, `@` or `*`.
func plainArgument(name string) bool {
	return len(name) == 1 && (name[0] >= '1' && name[0] <= '9' || name[0] == '@' || name[0] == '*')
}

// argumentBody reports whether body, a `${…}` expansion's, prints an argument
// as the call's word or a default in its place: the argument alone (`${1}`,
// `${12}`, `${@}`), or one with `-`, `=` or `?` and their `:` forms.
func argumentBody(body string) bool {
	rest := body
	switch {
	case rest == "":
		return false
	case rest[0] == '@' || rest[0] == '*':
		rest = rest[1:]
	default:
		rest = strings.TrimLeft(rest, "0123456789")
		if len(rest) == len(body) {
			return false
		}
	}
	rest = strings.TrimPrefix(rest, ":")
	return rest == "" || rest[0] == '-' || rest[0] == '=' || rest[0] == '?'
}

// maxFunctionWords bounds the words functionCallReadings scans for one line,
// the commands it scans for the bodies' spans and the words each reading puts
// included, and maxFunctionLineBytes the text it hands the guard again, all
// the readings together. Past either the line is held
// (functionArgumentsSignal).
const (
	maxFunctionWords     = 4096
	maxFunctionLineBytes = 1 << 16
)

// functionLaunchers are the programs a function's one command may not run for
// its call to be read (simpleFunction), and the names it may not have: a
// shell, a string or a file run as commands, a wrapper that runs the command
// after it, and the builtins that change the shell's own state. The walk's own
// wrappers (walkToCommand) are refused besides.
var functionLaunchers = map[string]bool{
	"sh": true, "bash": true, "zsh": true, "dash": true, "ksh": true, "mksh": true, "yash": true,
	"posh": true, "fish": true, "csh": true, "tcsh": true, "busybox": true,
	"eval": true, "source": true, ".": true, "exec": true, "env": true, "nohup": true,
	"command": true, "builtin": true, "xargs": true, "time": true, "repeat": true, "coproc": true,
	"sudo": true, "doas": true, "su": true, "nice": true, "ionice": true, "timeout": true,
	"stdbuf": true, "setsid": true, "chroot": true, "flock": true, "watch": true, "script": true,
	"nocorrect": true, "noglob": true, "function": true, "declare": true, "local": true,
	"typeset": true, "export": true, "readonly": true, "integer": true, "float": true, "set": true,
	"shift": true, "unset": true, "alias": true, "unalias": true, "shopt": true, "setopt": true,
	"unsetopt": true, "emulate": true, "trap": true, "read": true, "mapfile": true,
	"readarray": true, "getopts": true, "let": true, "enable": true, "hash": true, "autoload": true,
	"zmodload": true, "disable": true, "return": true, "caller": true, "fc": true, "[[": true,
}

// bodyWord is one word after the program of a function's one command
// (simpleFunction): a word the line fixes (ref 0), or one argument as it
// stands alone, `$@` (ref -1) or `$N` (ref N), quoted or not.
type bodyWord struct {
	text   string
	glob   bool
	ref    int
	quoted bool
}

// simpleBody is a function whose calls are read with their words: its one
// command's program and the words after it.
type simpleBody struct {
	prog  string
	words []bodyWord
	// unquoted records an argument the command reads unquoted, which a line
	// that names IFS splits as the guard does not.
	unquoted bool
}

// functionCallReadings reads each call, with words, of a function the line
// defines with those words in its arguments' place, in the one case it reads
// (simpleFunction): a function of one command on its own line, called at a
// plain command position of the line itself (spliceCall). Each reading is the
// line again with the call's command replaced by the command its words make,
// which Check judges as a line of its own. hold reports any other call with
// words of a function that reads its arguments (mayReadArgument), one at a
// wrapper's place or after one, a line that defines an alias beside such a
// function, zsh's anonymous function on a line that reads an argument, and a
// line past maxFunctionWords or maxFunctionLineBytes.
func functionCallReadings(command string, segs []segment) (lines []string, hold bool) {
	defs := map[string][]int{}
	anonymous, aliases, ifs := false, false, false
	for i, s := range segs {
		for _, n := range s.definesFunction {
			defs[n] = append(defs[n], i)
		}
		anonymous = anonymous || s.anonymousFunction
		for _, a := range arrivalsOf(s) {
			if t := s.tokens[a.idx]; t == "alias" || t == "shopt" || t == "setopt" {
				aliases = true
			}
		}
		for _, t := range s.tokens {
			ifs = ifs || strings.Contains(t, "IFS")
		}
	}
	if anonymous {
		for _, s := range segs {
			if mayReadArgument(s) {
				return nil, true
			}
		}
	}
	if len(defs) == 0 {
		return nil, false
	}
	budget, bytes := maxFunctionWords, maxFunctionLineBytes
	charge := func(n int) bool {
		budget -= n
		return budget >= 0
	}
	type info struct {
		reads bool
		body  *simpleBody
	}
	infos := map[string]*info{}
	infoOf := func(name string) *info {
		if in, ok := infos[name]; ok {
			return in
		}
		in := &info{}
		infos[name] = in
		for _, at := range defs[name] {
			lo, hi, _ := functionBody(segs, at, charge)
			for j := lo; j <= hi && !in.reads; j++ {
				// A span past the budget is not read: it is taken to read
				// its arguments, and its call is held.
				in.reads = !charge(len(segs[j].tokens)+1) || mayReadArgument(segs[j])
			}
		}
		if in.reads && len(defs[name]) == 1 {
			in.body = simpleFunction(command, segs, defs[name][0], defs)
		}
		return in
	}
	// A function whose name a wrapper or a keyword spells is not seen where
	// it is called (`nohup NAME`, `repeat NAME`), and one the line names
	// through an alias is called by another name: either is held once it
	// reads its arguments.
	for name := range defs {
		if (aliases || !plainCallName(name)) && infoOf(name).reads {
			return nil, true
		}
	}
	for ci, s := range segs {
		wrapped := false
		for _, a := range arrivalsOf(s) {
			name := s.tokens[a.idx]
			_, defined := defs[name]
			if !defined || a.idx+1 >= len(s.tokens) || containsString(s.definesFunction, name) {
				wrapped = wrapped || a.wrapper
				continue
			}
			in := infoOf(name)
			if budget < 0 {
				return nil, true
			}
			if !in.reads {
				wrapped = wrapped || a.wrapper
				continue
			}
			if in.body == nil || a.wrapper || wrapped || ifs && in.body.unquoted {
				return nil, true
			}
			line, ok := spliceCall(command, segs, ci, a, in.body)
			if !ok {
				return nil, true
			}
			if bytes -= len(line); bytes < 0 {
				return nil, true
			}
			lines = append(lines, line)
			wrapped = wrapped || a.wrapper
		}
	}
	return lines, false
}

// simpleFunction is the body of the function segs[at] defines when it is the
// one shape whose calls are read: `NAME() { CMD; }` or `function NAME { CMD;
// }` on one line, where NAME is a plain name no wrapper, keyword or launcher
// spells, defined once and alone, and CMD is one command with no assignment
// before it and no redirection, whose program is a word the line fixes that is
// no launcher (functionLaunchers), no function the line defines and no wrapper
// the walk steps, and whose every other word is fixed or is exactly one of
// `"$@"`, `$@`, `"$N"`, `$N`, `"${N}"` and `${N}` (N from 1 to 9). It is nil
// for any other function.
func simpleFunction(command string, segs []segment, at int, defs map[string][]int) *simpleBody {
	def := segs[at]
	if len(def.definesFunction) != 1 || def.functionQuoted || def.home != segs[0].home || def.nest != 0 {
		return nil
	}
	name := def.definesFunction[0]
	if !plainCallName(name) {
		return nil
	}
	var body, closing segment
	off := 0
	switch {
	case len(def.tokens) == 1 && def.tokens[0] == name || len(def.tokens) == 2 && def.tokens[0] == "function" && def.tokens[1] == name:
		if at+2 >= len(segs) || len(def.redirects) > 0 {
			return nil
		}
		body, closing, off = segs[at+1], segs[at+2], 1
		if len(body.tokens) == 0 || body.tokens[0] != "{" {
			return nil
		}
	case len(def.tokens) >= 3 && def.tokens[0] == "function" && def.tokens[1] == name && def.tokens[2] == "{":
		if at+1 >= len(segs) {
			return nil
		}
		body, closing, off = def, segs[at+1], 3
	default:
		return nil
	}
	if len(closing.tokens) != 1 || closing.tokens[0] != "}" || len(closing.redirects) > 0 || len(body.redirects) > 0 {
		return nil
	}
	for _, s := range []segment{body, closing} {
		if s.home != def.home || s.nest != def.nest || s.chain != def.chain || s.braceOdd ||
			s.argumentsUnread || s.functionQuoted || s.heredocUnterminated || s.substitutionUnread {
			return nil
		}
	}
	if def.begin < 0 || closing.end > len(command) || def.begin > closing.end || strings.ContainsAny(command[def.begin:closing.end], "\n\r") {
		return nil
	}
	if len(body.tokens) <= off {
		return nil
	}
	arr := arrivalsOf(body)
	if len(arr) != 1 || arr[0].idx != off || arr[0].wrapper {
		return nil
	}
	prog := body.tokens[off]
	if _, isVar := body.variable[off]; isVar || isUnknown(prog) || body.globAt(off) || isAssignment(prog) ||
		reserved[prog] || functionLaunchers[prog] || functionLaunchers[path.Base(prog)] {
		return nil
	}
	if _, ok := defs[prog]; ok {
		return nil
	}
	sb := &simpleBody{prog: prog}
	for k := off + 1; k < len(body.tokens); k++ {
		tok := body.tokens[k]
		if v, isVar := body.variable[k]; isVar {
			sp := body.spelled[k]
			if v != string(varMark) || len(sp) == 0 {
				return nil
			}
			// A default, an alternative or any other text the word can
			// print besides the argument (`${1:-x}`) is no plain argument.
			for _, alt := range sp[1:] {
				if alt != "" {
					return nil
				}
			}
			w := bodyWord{}
			switch t := sp[0]; {
			case t == "$@":
				w.ref, w.quoted = -1, !body.ifsSplit[k]
			case len(t) == 2 && t[0] == '$' && t[1] >= '1' && t[1] <= '9':
				w.ref, w.quoted = int(t[1]-'0'), !body.fieldSplit[k]
			case len(t) == 4 && t[:2] == "${" && t[2] >= '1' && t[2] <= '9' && t[3] == '}':
				w.ref, w.quoted = int(t[2]-'0'), !body.fieldSplit[k]
			default:
				return nil
			}
			sb.unquoted = sb.unquoted || !w.quoted
			sb.words = append(sb.words, w)
			continue
		}
		if isUnknown(tok) {
			return nil
		}
		sb.words = append(sb.words, bodyWord{text: tok, glob: body.globAt(k)})
	}
	return sb
}

// plainCallName reports whether name is one a call reaches as the function:
// a shell name the walk takes as a command, not a keyword, a launcher
// (functionLaunchers) or a wrapper it steps through.
func plainCallName(name string) bool {
	if !isShellName(name) || reserved[name] || functionShellWords[name] || functionLaunchers[name] {
		return false
	}
	out, _, _ := walkToCommand([]string{name, "x"})
	return len(out) == 1 && out[0].idx == 0 && !out[0].wrapper
}

// spliceCall is command with the call at arrival a of segs[ci] replaced by
// the command body makes of the call's words: the call must stand at a plain
// command position of the line's own text (only reserved words before it, no
// redirection, not inside a substitution), and every word it hands must be
// one the line fixes. ok is false otherwise, and where a word cannot be
// written back as the shell would read it (renderWord).
func spliceCall(command string, segs []segment, ci int, a arrival, body *simpleBody) (string, bool) {
	s := segs[ci]
	if s.home != segs[0].home || s.nest != 0 || len(s.redirects) > 0 || s.heredocUnterminated ||
		s.begin < 0 || s.end > len(command) || s.begin > s.end || !strings.HasPrefix(command[s.begin:], s.tokens[0]) {
		return "", false
	}
	for _, t := range s.tokens[:a.idx] {
		if !reserved[t] {
			return "", false
		}
	}
	var args []bodyWord
	for k := a.idx + 1; k < len(s.tokens); k++ {
		if _, isVar := s.variable[k]; isVar || isUnknown(s.tokens[k]) {
			return "", false
		}
		args = append(args, bodyWord{text: s.tokens[k], glob: s.globAt(k)})
	}
	parts := append([]string(nil), s.tokens[:a.idx]...)
	prog, ok := renderWord(body.prog, false, true)
	if !ok {
		return "", false
	}
	parts = append(parts, prog)
	put := func(w bodyWord, quoted bool) bool {
		text, ok := renderWord(w.text, w.glob, quoted)
		if ok && text != "" {
			parts = append(parts, text)
		}
		return ok
	}
	for _, w := range body.words {
		switch {
		case w.ref == 0:
			if !put(w, true) {
				return "", false
			}
		case w.ref == -1:
			for _, arg := range args {
				if !put(arg, w.quoted) {
					return "", false
				}
			}
		case w.ref <= len(args):
			if !put(args[w.ref-1], w.quoted) {
				return "", false
			}
		case w.quoted:
			parts = append(parts, "''")
		}
	}
	return command[:s.begin] + strings.Join(parts, " ") + command[s.end:], true
}

// renderWord writes a word back as shell text that reads as the word does,
// or more widely: as it stands where every byte is one the shell takes as it
// is (a glob's pattern bytes included, where the word was a pattern), and
// single-quoted otherwise. A word read unquoted, which the shell would split
// and match as a pattern, is written only as it stands, and an empty one is
// no word; a pattern or a home spelling that needs quoting is not written.
func renderWord(text string, glob, quoted bool) (string, bool) {
	if text == "" {
		if quoted {
			return "''", true
		}
		return "", true
	}
	plain := true
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case strings.IndexByte("_./:,+=@%~-", c) >= 0:
		case glob && strings.IndexByte("*?[]", c) >= 0:
		default:
			plain = false
		}
	}
	switch {
	case plain:
		return text, true
	case !quoted || glob || strings.HasPrefix(text, "~"):
		return "", false
	}
	return shellquote.Single(text), true
}

// mayReadArgument reports whether a command can read a function's
// arguments: a word that reads one (readsArgument), one in a form no word
// stands for (segment.argumentsUnread), or a spelling that names one in any
// form (spellsArgument) or that a line naming IFS capped.
func mayReadArgument(s segment) bool {
	if s.argumentsUnread || readsArgument(s) {
		return true
	}
	for _, texts := range s.spelled {
		for _, t := range texts {
			// A spelling a line that names IFS capped (capIFSSplits) may
			// have been one.
			if t == spellCapped || spellsArgument(t) {
				return true
			}
		}
	}
	return false
}

// spellsArgument reports whether text holds a `$` that can read a
// function's arguments in any form: `$1`, `$@`, `$*`, `${!…}`, `${#1}`, a
// flag in parentheses, zsh's `$=1`, `$~1`, `$^1` and `$argv`.
func spellsArgument(text string) bool {
	for i := 0; i < len(text); i++ {
		if text[i] != '$' {
			continue
		}
		j := i + 1
		if j < len(text) && text[j] == '{' {
			j++
			if j < len(text) && text[j] == '(' {
				return true
			}
		}
		for j < len(text) && strings.IndexByte("=~^", text[j]) >= 0 {
			j++
		}
		if j >= len(text) {
			continue
		}
		switch c := text[j]; {
		case c >= '1' && c <= '9', c == '@', c == '*', c == '!':
			return true
		case c == '#' && j+1 < len(text) && (text[j+1] >= '1' && text[j+1] <= '9' || text[j+1] == '@' || text[j+1] == '*'):
			return true
		case strings.HasPrefix(text[j:], "argv"):
			return true
		}
	}
	return false
}

// functionBody is the span of segs, lo to hi, that holds the body of the
// function segs[at] defines: from the definition to the `}` that closes the
// `{` it opens, counting the groups opened and closed at the definition's own
// level between (segment.nest; a substitution's commands and a string's are
// their own). certain is false where the span cannot be placed so: a body
// that opens no `{`, a `{` or `}` word anywhere but where it opens or closes
// a group (segment.braceOdd), a group the line leaves open, or more words
// scanned than charge allows. The span then runs to the end of segs.
func functionBody(segs []segment, at int, charge func(int) bool) (lo, hi int, certain bool) {
	def := segs[at]
	lo, hi = at, len(segs)-1
	depth, opened := 0, false
	for j := at; j < len(segs); j++ {
		s := segs[j]
		if !charge(len(s.tokens) + 1) {
			return lo, hi, false
		}
		if s.home != def.home || s.nest > def.nest {
			continue
		}
		if s.nest < def.nest || s.braceOdd {
			return lo, hi, false
		}
		toks := s.tokens
		if len(toks) > 0 && toks[0] == "}" {
			depth--
			if depth < 0 || !opened {
				return lo, hi, false
			}
			if depth == 0 {
				return lo, j, true
			}
			continue
		}
		i := 0
		if len(toks) >= 2 && toks[0] == "function" {
			i = 2
		}
		for ; i < len(toks) && reserved[toks[i]]; i++ {
			if toks[i] == "{" {
				depth++
				opened = true
			}
		}
		if j > at && !opened {
			return lo, hi, false
		}
	}
	return lo, hi, false
}

// argumentRef is one parameter a word reads, as the line wrote it, from the
// word's written spelling (segment.spelled) at the place its value goes
// (segment.variable).
type argumentRef struct {
	literal string // the text before it
	name    string // its name: `1`, `@`, `HOME`; "" after the last one
}

// argumentRefs splits the word at idx of s into its parameters, in order. ok
// is false where the word's spelling does not line up with the places its
// values go and any text it can print names an argument; one that names none
// is read as holding no argument.
func argumentRefs(s segment, idx int) (refs []argumentRef, ok bool) {
	tmpl, has := s.variable[idx]
	spelled := s.spelled[idx]
	if !has || tmpl == "" || len(spelled) == 0 {
		return nil, true
	}
	refs, ok = alignedRefs(tmpl, spelled[0])
	if ok {
		return refs, true
	}
	for _, text := range spelled {
		if namesArgument(text) {
			return nil, false
		}
	}
	return nil, true
}

// namesArgument reports whether text, a word's spelling, holds a `$` that
// reads a function's argument (`$1`, `${1`, `$@`, `$*`).
func namesArgument(text string) bool {
	for i := 0; i+1 < len(text); i++ {
		if text[i] != '$' {
			continue
		}
		c := text[i+1]
		if c == '{' && i+2 < len(text) {
			c = text[i+2]
		}
		if c >= '1' && c <= '9' || c == '@' || c == '*' {
			return true
		}
	}
	return false
}

// alignedRefs is argumentRefs on the word's template (segment.variable) and
// its written spelling.
func alignedRefs(tmpl, w string) (refs []argumentRef, ok bool) {
	var lit strings.Builder
	j := 0
	for i := 0; i < len(tmpl); i++ {
		if tmpl[i] != varMark {
			if j >= len(w) || w[j] != tmpl[i] {
				return nil, false
			}
			lit.WriteByte(tmpl[i])
			j++
			continue
		}
		if j >= len(w) || w[j] != '$' {
			return nil, false
		}
		j++
		start := j
		switch {
		case j < len(w) && w[j] == '{':
			end := strings.IndexByte(w[j:], '}')
			if end < 0 {
				return nil, false
			}
			refs = append(refs, argumentRef{literal: lit.String(), name: w[j+1 : j+end]})
			j += end + 1
		case j < len(w) && (w[j] >= '0' && w[j] <= '9' || strings.IndexByte("@*#?$!-", w[j]) >= 0):
			refs = append(refs, argumentRef{literal: lit.String(), name: w[j : j+1]})
			j++
		default:
			for j < len(w) && isShellNameByte(w[j]) {
				j++
			}
			refs = append(refs, argumentRef{literal: lit.String(), name: w[start:j]})
		}
		lit.Reset()
	}
	if j != len(w) {
		return nil, false
	}
	return append(refs, argumentRef{literal: lit.String()}), true
}

// readsArgument reports whether a word of s reads a function's argument, or
// spells one that does not line up (which withArguments refuses).
func readsArgument(s segment) bool {
	for idx := range s.variable {
		refs, ok := argumentRefs(s, idx)
		if !ok || readsArgumentRefs(refs) {
			return true
		}
	}
	return false
}

// argumentIndex is name's place among a call's words: 1… for `1`…, -1 for
// `@` and `*`, and 0 for a parameter that is no argument.
func argumentIndex(name string) int {
	if name == "@" || name == "*" {
		return -1
	}
	if name == "" || strings.Trim(name, "0123456789") != "" || name[0] == '0' {
		return 0
	}
	n := 0
	for _, c := range name {
		n = n*10 + int(c-'0')
		if n > 1<<16 {
			return 1 << 16
		}
	}
	return n
}

// readsArgumentRefs reports whether any of refs is a function's argument.
func readsArgumentRefs(refs []argumentRef) bool {
	for _, r := range refs {
		if argumentIndex(r.name) != 0 {
			return true
		}
	}
	return false
}

// skipWord advances past the one word at pos: zsh's `repeat COUNT command`, a
// keyword and one word of its own before the command it runs. Like `coproc`,
// it is a keyword of one shell only; elsewhere it names a program that is not
// found, so the command after it never runs there.
func skipWord(tokens []string, pos int) int {
	if pos < len(tokens) {
		return pos + 1
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

// isAssignment reports whether a token is an assignment word, which in
// assignment position precedes the command rather than being one: NAME=VALUE,
// and the forms bash and zsh read the same way (splitAssignment).
func isAssignment(tok string) bool {
	_, _, _, _, ok := splitAssignment(tok)
	return ok
}

// splitAssignment splits an assignment word as bash and zsh read one: its
// variable's name, the index of the `=` its value follows, and whether it
// appends (NAME+=VALUE) or names an element (NAME[SUB]=VALUE,
// NAME[SUB]+=VALUE). Both shells take each form before a command as an
// assignment and run the command after it — bash 5 reports a subscripted
// one as not a valid identifier and still runs the command
// (iss-2610121214249290) — so every reading that asks whether a word is an
// assignment asks this one rule. ok is false for any other word.
func splitAssignment(tok string) (name string, eq int, appends, subscript, ok bool) {
	i := 0
	for i < len(tok) && (tok[i] == '_' || tok[i] >= 'a' && tok[i] <= 'z' || tok[i] >= 'A' && tok[i] <= 'Z' ||
		i > 0 && tok[i] >= '0' && tok[i] <= '9') {
		i++
	}
	if i == 0 || i == len(tok) {
		return "", 0, false, false, false
	}
	name = tok[:i]
	if tok[i] == '[' {
		end := strings.IndexByte(tok[i:], ']')
		if end < 0 {
			return "", 0, false, false, false
		}
		i += end + 1
		subscript = true
	}
	if i < len(tok) && tok[i] == '+' {
		i++
		appends = true
	}
	if i >= len(tok) || tok[i] != '=' {
		return "", 0, false, false, false
	}
	return name, i, appends, subscript, true
}

// isName reports whether w is a shell variable name.
func isName(w string) bool {
	name, eq, _, _, ok := splitAssignment(w + "=")
	return ok && eq == len(w) && name == w
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
	return matchSegmentEnv(p, s, false)
}

// matchSegmentEnv is matchSegmentNamed, where exported reports whether an
// earlier command in the line exports a variable p.Env names. An entry that
// names variables fires only where the line sets one for the command: by
// that export, or by an assignment among the words before it (envSetBefore).
func matchSegmentEnv(p Pattern, s segment, exported bool) (hit, named bool) {
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
	if len(p.Env) > 0 && !exported {
		kept := sites[:0:0]
		for _, a := range sites {
			if envSetBefore(s, a.idx, p.Env) {
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
		m := newEntryMatcher(p, s.tokens, s.spelled, s.varLead, glob)
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
		if s.piped.hits(p.ArgsFrom) || anyHits(s.stdinIn, p.ArgsFrom) || anyHits(s.carries, p.ArgsFrom) {
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
		r.base = append(r.base, s.carries...)
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
// a command string it runs does (segList.payloads), however deep they nest, or
// any command whose output reaches it otherwise does (segment.carries).
func segmentHits(s segment, srcs []Pattern) bool {
	for _, src := range srcs {
		if matchSegment(src, s) {
			return true
		}
	}
	if anyHits(s.carries, srcs) {
		return true
	}
	if s.processSearch {
		for _, src := range srcs {
			if matchSegment(src, processSearchReading) {
				return true
			}
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
		// Filed before it is counted: a command's carries (segmentHits) name
		// only commands before it in its own list, which are counted by the
		// time it is read, so the question it asks of them is answered from
		// the counts already made rather than counted again without end.
		if f.list.hits == nil {
			f.list.hits = map[*Pattern][]int{}
		}
		f.list.hits[key] = counts
		for i, s := range f.list.segs {
			counts[i+1] = counts[i]
			if segmentHits(s, srcs) {
				counts[i+1]++
			}
		}
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
	// tokens, valueFlags, notAlts and opts are what standsDown reads; notAlts is
	// empty for an entry that declares no not_flags.
	tokens, valueFlags, notAlts, opts []string
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
// segment.spelled, which only the arg_values clause reads (writtenMatches),
// and varLead the segment's segment.varLead, which only the arg_shapes
// clause reads (ShapeUnguardedVariablePath).
func newEntryMatcher(p Pattern, tokens []string, spelled map[int][]string, varLead map[int]bool, glob func(int) bool) entryMatcher {
	n := len(tokens)
	want := operandWant{
		sub: p.Subcommand, sub2: p.Subcommand2, min: p.MinOperands,
		subNamed: p.SubcommandNamed != nil && *p.SubcommandNamed,
		prefixes: p.ArgPrefixes, paths: p.ArgPaths, values: p.ArgValues, shapes: p.ArgShapes,
		prefixLiteral: p.ArgPrefixesLiteral != nil && *p.ArgPrefixesLiteral,
	}
	m := entryMatcher{accept: operandAcceptance(tokens, spelled, varLead, p.ValueFlags, want, glob), nextStop: make([]int, n+1)}
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
	for _, group := range p.NotFlags {
		m.notAlts = append(m.notAlts, strings.Split(group, "|")...)
	}
	m.tokens, m.valueFlags, m.opts = tokens, p.ValueFlags, opts
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
	if len(m.notAlts) > 0 && m.standsDown(start, stop) {
		return false
	}
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

// standsDown reports whether tokens[start:stop], a command's arguments, carry a
// not_flags alternative for the subcommand. Only words after the subcommand are
// its flags: before it `-n` is a global option's value (`git --namespace -n
// prune`). The reading is the one line that can be sure of it, so a word a
// substitution helps to write, anywhere in the arguments, stands nothing down,
// nor does a line that also undoes the flag.
func (m entryMatcher) standsDown(start, stop int) bool {
	i := start
	for i < stop {
		r := readWord(m.tokens[i], m.valueFlags)
		if isUnknown(m.tokens[i]) {
			return false
		}
		if r.operand {
			break
		}
		i++
		if r.takes {
			i++
		}
	}
	found := false
	for i++; i < stop; i++ {
		tok := m.tokens[i]
		if isUnknown(tok) || notFlagUndone(m.notAlts, tok, m.opts) {
			return false
		}
		found = found || notFlagHit(m.notAlts, tok, m.opts)
	}
	return found
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
// separators taken out (cleanSeparators), as the kernel reads the path, and
// with a leading run of `..` segments written once (parentRun), since a chain
// of parents holds the one parent.
func argValueMatches(values []string, written string) bool {
	written = strings.ReplaceAll(written, quotedFieldText, " ")
	for _, field := range strings.Split(written, fieldText) {
		if field == "" || isUnknown(field) {
			continue
		}
		clean, glob := cleanSeparators(field), globReading(field)
		chain, globChain := parentRun(clean), parentRun(glob)
		for _, v := range values {
			if field == v || clean == v || glob == v || chain == v || globChain == v {
				return true
			}
		}
		if lead := strings.TrimLeft(field, "/"); lead != field && absoluteName(lead) && argValueMatches(values, lead) {
			return true
		}
	}
	return false
}

// parentRun is a relative path whose leading run of two or more `..`
// segments is written as one `..` (iss-2610021542183618): `../..` is `..`,
// `../../` is `../` and `../../*` is `../*`. A chain climbs above the
// working directory's parent and holds it, so it deletes at least what the
// one `..` does, and argValueMatches compares this reading beside the others,
// so it can only add a hit. It is "" where p does not begin with such a run,
// which no value is. A segment that only begins with `..` (`..x`) is a name.
func parentRun(p string) string {
	rest := p
	for strings.HasPrefix(rest, "../..") && (len(rest) == len("../..") || rest[len("../..")] == '/') {
		rest = rest[len("../"):]
	}
	if rest == p {
		return ""
	}
	return rest
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

// notFlagHit reports whether a typed word is one of an entry's not_flags
// alternatives, its abbreviation included.
func notFlagHit(alts []string, tok string, opts []string) bool {
	for _, alt := range alts {
		if alt != "" && (flagMatches(alt, tok, false) || (opts != nil && abbreviatesAlternative(tok, alt, alts, opts))) {
			return true
		}
	}
	return false
}

// notFlagUndone reports whether a typed word is the `--no-` of a long not_flags
// alternative, its abbreviation included.
func notFlagUndone(alts []string, tok string, opts []string) bool {
	for _, alt := range alts {
		name, ok := strings.CutPrefix(alt, "--")
		if !ok {
			continue
		}
		no := "--no-" + name
		if tok == no || (opts != nil && abbreviatesAlternative(tok, no, []string{no}, opts)) {
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
// alternatives to one of the accepted values. Every spelling a shell user
// reaches for is read — `-X DELETE`, `-XDELETE`, `--method=DELETE`, `-iX DELETE` — because a
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
	accepts := func(got string, glob bool) bool {
		switch {
		case fv.Shape != "":
			return flagValueShapeHolds(fv.Shape, got, glob)
		}
		return acceptsValue(fv.Values, got, glob)
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
			// A shape is about the setting's text, and an unknown dash-word is
			// on every line that has one, so the setting is still read. A field's
			// setting can be attached to the dash-word itself, whatever it holds.
			if fv.Shape == "" || fv.Shape == ShapeRefForceField {
				return true
			}
			if accepts(arg, false) || (i+1 < len(tokens) && accepts(tokens[i+1], glob(i+1))) {
				return true
			}
		}
		k := knownText(arg)
		at := -1 // where a short flag sits in a cluster of single-letter flags
		if isShortFlag(alt) {
			at = strings.IndexByte(k, alt[1])
		}
		switch {
		case k == alt || (glob(i) && flagShaped(k) && globMatches(k, alt)):
			// The separate-token form: the value is the next argument.
			if i+1 < len(tokens) && accepts(tokens[i+1], glob(i+1)) {
				return true
			}
		case strings.HasPrefix(arg, alt+"="):
			if accepts(arg[len(alt)+1:], false) {
				return true
			}
		case strings.HasPrefix(k, alt+"="):
			if accepts(k[len(alt)+1:], false) {
				return true
			}
		case isShortFlag(alt) && len(arg) > len(alt) && strings.HasPrefix(arg, alt):
			// A short flag's value may be attached with no separator at all.
			if accepts(arg[len(alt):], false) {
				return true
			}
		case isShortFlag(alt) && len(k) > len(alt) && strings.HasPrefix(k, alt):
			if accepts(k[len(alt):], false) {
				return true
			}
		case at > 1 && isShortCluster(k[:at]):
			// A short flag at the end of a cluster takes its value as it does alone:
			// `-iF x`, `-iFx` or `-iF=x`.
			if at+1 == len(k) {
				if i+1 < len(tokens) && accepts(tokens[i+1], glob(i+1)) {
					return true
				}
			} else if accepts(strings.TrimPrefix(k[at+1:], "="), false) {
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

// positionalsSet reports whether the text command, read as segs, can set
// the positional parameters for a command of its own: a `set` with an
// operand or `--`, or a function, whose body takes its call's arguments
// (a `()` or the word `function` anywhere in the text). A string a shell
// is handed takes the shell's arguments, which the caller reads by the
// segment's carrier. Only a `set` the line names is read: a program name
// the line does not fix is not taken as one, so a positional parameter
// standing as a command does not count.
func positionalsSet(command string, segs []segment) bool {
	if strings.Contains(command, "()") || strings.Contains(command, "function") {
		return true
	}
	for _, s := range segs {
		for _, a := range commandSites(s) {
			if tok := s.tokens[a.idx]; isUnknown(tok) || !nameCouldBe(tok, "set") {
				continue
			}
			for _, w := range s.tokens[a.idx+1:] {
				if isUnknown(w) || w == "--" || w == "-" || len(w) < 2 || w[0] != '-' && w[0] != '+' {
					return true
				}
				if strings.ContainsAny(w[1:], "oO") {
					// `set -o NAME`: the next word is the option's name.
					break
				}
			}
		}
	}
	return false
}
