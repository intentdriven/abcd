package guard

// approvegitdir.go is the approver's reading of which git directory a git
// command reads (approve.go). git takes its configuration, and every program
// that configuration names, from the git directory it runs against, and the
// approver's other readers judge the working tree's own (approvepaths.go). A
// command aimed at a second git directory standing inside the working tree
// reads a configuration the guard never read, so it is refused, naming the
// directory. The forms read are git's own, as git(1) and git-init(1)
// document them:
//   - -C <path>: git runs "as if git was started in <path>", a later
//     relative -C read from the one before and an empty one leaving the
//     directory as it is, and finds its repository from there by the walk
//     upward it always makes (a directory's .git, or the directory itself
//     when it is a repository); a change of directory the line fixes is read
//     the same way;
//   - --git-dir=<path> (or the next word) and GIT_DIR, on the command or
//     carried from an earlier command of the line (carryEnvironment): "the
//     path to the repository", read from where -C leaves git, which may be a
//     git file whose "gitdir: " line names it;
//   - GIT_COMMON_DIR: where the files a worktree shares, its config among
//     them, are taken from;
//   - --bare with no git directory named: "it is set to the current working
//     directory", the one current when git reads --bare, so the first
//     --bare counts and a later -C does not move it;
//   - git init --separate-git-dir <git-dir>: the repository is made there,
//     or, on a reinitialization, moved there;
//   - a path among the operands of a subcommand that reaches another
//     repository (push, fetch, pull, ls-remote, clone, submodule, and
//     archive's --remote): git runs against the repository it names
//     (gitRemoteSubcommands).
//
// A git directory that is a git file, or that holds a commondir file, is
// judged by where that file points, wherever the file itself stands, so a
// git file outside the working tree that points inside it is refused.
//
// The directory git runs in is the shell's, as each cd the line fixes leaves
// it, then as each env -C or --chdir (env(1)), sudo -D or --chdir (sudo(8))
// and unshare -w or --wd (unshare(1)) before git changes it, resolved the way
// -C is. sudo -i or --login runs git from the target user's home directory,
// which the line does not fix. A wrapper is read only when it is on the short
// list of wrappers the approver knows (wrapperDirFlags), and only through the
// options that list classifies as leaving the directory alone or changing
// it; behind any other wrapper the walk to a command steps through, or any
// other option, where git runs is not fixed, and so it is behind a wrapper
// given two changes of directory, which its parser may not apply in order as
// git's -C is. Behind a program the walk does not step through, git is no
// command site at all, so a word naming git there, followed by a word
// starting with -, a word the line does not fix or a subcommand git 2.52
// documents, is refused as unfixed too (gitBehindProgram); the words after a
// git site the walk reached are that site's own, judged with it, and are not
// read this way, except the command git submodule foreach or git bisect run
// is given to run. A word git does
// not document after it (an alias, a git-<name> command, a subcommand a later
// release adds) and git a program starts by a name the line does not spell
// are not read.
//
// A submodule's git file points into the working tree's own git directory,
// and is left alone; so is a git directory outside the working tree that
// points nowhere inside it, which the approver's other rules judge. A
// directory the line does not fix is refused, and so is one git would find
// its repository from when the line leaves it unknown. After an earlier
// command of the line writes, which may make a repository where none stood,
// a git command is refused that runs in a directory inside the working tree
// other than its root or its own git directories, or names a git directory
// inside the working tree other than its own; a git command not known to
// leave the working tree alone (gitLeavesTreeAlone) counts as such a write.
// A line that runs its commands out of text order (runsOutOfTextOrder, or a
// command run in the background, backgrounded) and writes or changes
// directory anywhere has every git command in it refused, since a later
// command's write or cd may come first.

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// gitDirEnv are the variables that name the git directory a git command
// reads, or the one it takes its shared files from (git(1)).
var gitDirEnv = []string{"GIT_DIR", "GIT_COMMON_DIR"}

// gitDirRefusal reports a git command in s, run in the shell state st with
// carried the environment the line's earlier commands leave it, whose git
// directory stands inside the working tree and is not the working tree's
// own, or is named by a directory the line does not fix. wrote says an
// earlier command of the line writes or removes a path, or is a git command
// not known to leave the working tree alone; unordered that the line runs
// its commands out of text order and writes or changes directory somewhere,
// which refuses every git command in it. A word naming git that the walk to
// a command did not reach, standing behind a program, is refused as unfixed
// (gitBehindProgram).
func (f approveFacts) gitDirRefusal(rc *readCtx, s segment, st *shellState, valueFlags []string, carried map[string]string, wrote, unordered bool) (Approval, bool) {
	if len(f.gitTrees) == 0 {
		return Approval{}, false
	}
	for _, site := range sitesNamed(s, "git") {
		// A program name the line does not fix is refused as unnamed
		// (unnamedProgramEntryID), whose reason says more.
		if _, named := fixedToken(s, site.idx); unordered && named {
			return gitDirUnfixedRefusal("a line that runs its commands out of text order, and writes or changes directory"), true
		}
		from := site.idx + 1
		env := readGitConfigIn(carried, s.tokens[:site.idx], s.tokens[from:], valueFlags).env
		at, a, refused := wrapperDir(rc, s, site.idx, st)
		if refused {
			return a, true
		}
		if a, ok := f.gitDirOf(rc, s, from, at, valueFlags, env, wrote); ok {
			return a, true
		}
	}
	if program, ok := gitBehindProgram(s); ok {
		return gitDirUnfixedRefusal("git, standing behind " + program + ", a program the approver does not step through"), true
	}
	return Approval{}, false
}

// gitBehindProgram reports a word of s naming git (git, or a fixed path whose
// last element is git) that the walk to a command did not reach as a git
// site, followed by a word starting with -, a word the line does not fix or a
// subcommand git documents (gitSubcommands), with the program it stands
// behind: the nearest word before it the walk reached. The walk steps only
// through the wrappers it knows, so behind any other program git is no site
// and its directory is never read; such a word is read as git that program
// runs. A word naming git with nothing after it, or an ordinary word after
// it, is left alone (grep git file). The words after a git site the walk
// reached are git's own arguments, which gitDirOf judges at that site, so
// they are not read here (git -C vendor/git status, git log --grep git),
// except the command git submodule foreach or git bisect run is given to run
// (gitRunsCommandFrom), which is read as standing behind that subcommand.
func gitBehindProgram(s segment) (string, bool) {
	sites := map[int]bool{}
	for _, site := range sitesNamed(s, "git") {
		sites[site.idx] = true
	}
	arrived := map[int]bool{}
	for _, a := range arrivalsOf(s) {
		arrived[a.idx] = true
	}
	const unspelled = "a program the line does not spell out"
	program, own := unspelled, -1
	for i := 0; i+1 < len(s.tokens); i++ {
		if sites[i] {
			own = len(s.tokens)
			if from, runner, ok := gitRunsCommandFrom(s, i); ok {
				own, program = from-1, runner
			}
			continue
		}
		if i <= own {
			continue
		}
		tok, fixed := fixedToken(s, i)
		if fixed && !strings.HasSuffix(tok, "/") && strings.EqualFold(path.Base(tok), "git") {
			next, fixed := fixedToken(s, i+1)
			if !fixed || strings.HasPrefix(next, "-") || gitSubcommands[next] {
				return program, true
			}
		}
		if arrived[i] {
			program = unspelled
			if w, ok := fixedToken(s, i); ok {
				program = w
			}
		}
	}
	return "", false
}

// gitRunsCommandFrom reports where, among the words after the git site at
// site, the command begins that git submodule foreach (git-submodule(1):
// "Evaluates an arbitrary shell command") or git bisect run (git-bisect(1))
// is given to run, and names that subcommand. Every word submodule, then the
// first word after it not starting with -, is read as the subcommand, and so
// is every word bisect, so a value of git's own spelled the same way only
// reads more. A word the line does not spell out ends nothing here; the
// readers in approvegitrun.go refuse a subcommand word the line does not fix
// that may be foreach or run.
func gitRunsCommandFrom(s segment, site int) (int, string, bool) {
	for j := site + 1; j < len(s.tokens); j++ {
		w, fixed := fixedToken(s, j)
		if !fixed {
			continue
		}
		switch w {
		case "submodule":
			k := j + 1
			for ; k < len(s.tokens); k++ {
				if v, ok := fixedToken(s, k); !ok || !strings.HasPrefix(v, "-") {
					break
				}
			}
			if v, ok := fixedToken(s, k); ok && v == "foreach" {
				return k + 1, "git submodule foreach", true
			}
		case "bisect":
			if v, ok := fixedToken(s, j+1); ok && v == "run" {
				return j + 2, "git bisect run", true
			}
		}
	}
	return 0, "", false
}

// backgrounded reports whether a command of the line is run in the
// background, ended by a lone & (segment.background): the shell starts it
// and goes on, so a command after it in the text may write or change
// directory before it looks for its repository.
func backgrounded(placed []placedSeg) bool {
	for _, p := range placed {
		if p.seg.background {
			return true
		}
	}
	return false
}

// wrapperDirOptions is one wrapper's options in wrapperDirFlags, by what each
// does to the directory the command after the wrapper runs in: chdir changes
// it, keep leaves it alone.
type wrapperDirOptions struct{ chdir, keep []string }

// wrapperDirFlags is the short list of wrappers the approver can read the
// directory of, each with its documented options classified. A git command
// behind a wrapper the list does not name, or behind an option of a listed
// one that it does not classify, runs in a directory the line does not fix
// and is refused (wrapperDir): a wrapper the walk to a command steps through
// and this list does not name fails closed, since naming the wrappers that
// change the directory one by one never closes the set. git behind a program
// the walk does not step through is gitBehindProgram's.
// Which options take the next word is the wrapper's wrapperValueFlags.
//
// A chdir option changes the directory to its value, or, when it takes none,
// to one the line does not fix: env's -C and --chdir (env(1)), sudo's -D and
// --chdir (sudo(8)) and unshare's -w and --wd (unshare(1)) take a value;
// sudo's -i and --login run the command from the target user's home
// directory (sudo(8)). Left out on purpose: sudo's -R and --chroot and
// unshare's -R and --root, which change the root the directory is read
// against; sudo's -s and --shell, which hand the command to the target
// user's shell; and xargs, which adds git's arguments, -C and --git-dir
// among them, from its input, which the line does not fix.
var wrapperDirFlags = map[string]wrapperDirOptions{
	// env(1), GNU and BSD: environment and signal options, the name the
	// program is given (GNU -a and --argv0) and the path searched for it
	// (BSD -P).
	"env": {
		chdir: []string{"-C", "--chdir"},
		keep: []string{
			"-i", "--ignore-environment", "-0", "--null", "-u", "--unset", "-v", "--debug",
			"--default-signal", "--ignore-signal", "--block-signal", "--list-signal-handling",
			"-a", "--argv0", "-P",
		},
	},
	// sudo(8): who the command runs as, how sudo asks, and what it keeps.
	"sudo": {
		chdir: []string{"-D", "--chdir", "-i", "--login"},
		keep: []string{
			"-A", "--askpass", "-B", "--bell", "-b", "--background", "-C", "--close-from",
			"-E", "--preserve-env", "-e", "--edit", "-g", "--group", "-H", "--set-home",
			"-h", "--host", "-K", "--remove-timestamp", "-k", "--reset-timestamp",
			"-l", "--list", "-N", "--no-update", "-n", "--non-interactive",
			"-P", "--preserve-groups", "-p", "--prompt", "-r", "--role", "-S", "--stdin",
			"-T", "--command-timeout", "-t", "--type", "-U", "--other-user", "-u", "--user",
			"-V", "--version", "-v", "--validate",
		},
	},
	// unshare(1): the namespaces made, and the ids and clocks set in them.
	"unshare": {
		chdir: []string{"-w", "--wd"},
		keep: []string{
			"-i", "--ipc", "-m", "--mount", "-n", "--net", "-p", "--pid", "-u", "--uts",
			"-U", "--user", "-C", "--cgroup", "-T", "--time", "-f", "--fork", "--kill-child",
			"--mount-proc", "-r", "--map-root-user", "-c", "--map-current-user", "--map-auto",
			"--map-user", "--map-group", "--map-users", "--map-groups", "-S", "--setuid",
			"-G", "--setgid", "--propagation", "--setgroups", "--monotonic", "--boottime",
		},
	},
	// nice(1): the adjustment.
	"nice": {keep: []string{"-n", "--adjustment"}},
	// nohup(1) takes no options.
	"nohup": {},
	// timeout(1), GNU and BSD: the signal, when to send it, and the exit.
	"timeout": {keep: []string{
		"-k", "--kill-after", "-s", "--signal", "--preserve-status", "-p",
		"--foreground", "-f", "-v", "--verbose",
	}},
	// time(1), GNU and BSD, and the shell's time -p: what is reported, and where.
	"time": {keep: []string{
		"-p", "--portability", "-v", "--verbose", "-q", "--quiet", "-a", "--append",
		"-f", "--format", "-o", "--output", "-l", "-h",
	}},
	// The shell's command and exec: the default path, a name printed, the
	// name the program is given, its environment cleared, a login dash.
	"command": {keep: []string{"-p", "-v", "-V"}},
	"exec":    {keep: []string{"-a", "-c", "-l"}},
	// stdbuf(1): each stream's buffering.
	"stdbuf": {keep: []string{"-i", "-o", "-e", "--input", "--output", "--error"}},
	// ionice(1): the class and its data, and whether a failure is ignored.
	"ionice": {keep: []string{"-c", "--class", "-n", "--classdata", "-t", "--ignore"}},
}

// wrapperChdir is one change of directory a wrapper makes before the command
// it runs: the option that makes it, the index of the word holding its value
// and the value, with whether the shell expands a leading ~ in it. unfixed
// says the option names no directory, so the line does not fix where it
// leads.
type wrapperChdir struct {
	flag    string
	idx     int
	word    string
	tilde   bool
	unfixed bool
}

// wrapperDir is the shell state st as the wrappers before the git site at
// index end of s leave it: each change of directory wrapperChdirs reads,
// resolved in order the way git's -C is. A change whose value the line does
// not fix is refused, and so is a site behind a wrapper or an option
// wrapperChdirs cannot read.
func wrapperDir(rc *readCtx, s segment, end int, st *shellState) (*shellState, Approval, bool) {
	at := *st
	at.dirOK = st.dirOK && st.dir != ""
	changes, stop := wrapperChdirs(s, end)
	if stop != "" {
		return nil, gitDirUnfixedRefusal(stop), true
	}
	for _, c := range changes {
		if c.unfixed {
			return nil, gitDirUnfixedRefusal(c.flag), true
		}
		if _, ok := fixedToken(s, c.idx); !ok {
			return nil, gitDirUnfixedRefusal(c.flag), true
		}
		p, ok := at.resolve(rc, c.word, c.tilde)
		if !ok {
			return nil, gitDirUnfixedRefusal(c.flag), true
		}
		at.dir, at.dirOK = p, true
	}
	return &at, Approval{}, false
}

// wrapperChdirs reads the words of s before index end as the walk to command
// position reads known words (commandArrivals): assignments and reserved
// words stepped, and each wrapper with its options, the value of each of its
// wrapperValueFlags, and its wrapperOperands. It returns each change of
// directory a chdir option of wrapperDirFlags makes, written whole,
// abbreviated, glued or in a short-option cluster. A keep option is read
// written whole only: what an abbreviation stands for depends on every
// option the wrapper has, and the list names only those it classifies. When
// the reading meets a word the line does not fix, a program wrapperDirFlags
// does not list, an option it does not classify, or does not arrive at end,
// it stops and returns what stopped it, which wrapperDir refuses. It stops,
// too, at one wrapper given two changes of directory: env, sudo and unshare
// keep one value and change to it from where they start, not each in turn as
// git's -C does, and the reading does not model which one each keeps.
func wrapperChdirs(s segment, end int) ([]wrapperChdir, string) {
	toks := s.tokens
	var out []wrapperChdir
	const short = "a value the line does not reach"
	value := func(flag string, i int) (wrapperChdir, bool) {
		if i >= end {
			return wrapperChdir{}, false
		}
		return wrapperChdir{flag: flag, idx: i, word: toks[i], tilde: strings.HasPrefix(toks[i], "~")}, true
	}
	i, assigns := 0, false
	for i < end {
		tok := toks[i]
		if isAssignment(tok) || reserved[tok] || assigns && isWrapperAssignment(tok) {
			i++
			continue
		}
		if isUnknown(tok) {
			return nil, "a wrapper the line does not spell out"
		}
		name := path.Base(tok)
		w := strings.ToLower(name)
		opts, listed := wrapperDirFlags[w]
		if !listed || !wrappers[w] {
			return nil, "the wrapper " + name + ", which the approver does not read as leaving the directory alone"
		}
		unread := func(o string) string {
			return "the option " + o + " of " + name + ", which the approver does not read as leaving the directory alone"
		}
		assigns = envAssigns[w]
		vf := wrapperValueFlags[w]
		first := len(out)
		for i++; i < end; i++ {
			o := toks[i]
			if o == "--" {
				i++
				break
			}
			if o == "-" || !strings.HasPrefix(o, "-") {
				break
			}
			if isUnknown(o) {
				return nil, "an option of " + name + " the line does not spell out"
			}
			if strings.HasPrefix(o, "--") {
				long, v, glued := strings.Cut(o, "=")
				if containsString(opts.keep, long) {
					if !glued && containsString(vf, long) {
						if i++; i >= end {
							return nil, short
						}
					}
					continue
				}
				cd, isCd := longChdirFlag(o, opts.chdir)
				switch {
				case !isCd:
					return nil, unread(long)
				case !containsString(vf, cd):
					out = append(out, wrapperChdir{flag: cd, idx: i, unfixed: true})
				case glued:
					out = append(out, wrapperChdir{flag: long, idx: i, word: v})
				default:
					i++
					c, ok := value(long, i)
					if !ok {
						return nil, short
					}
					out = append(out, c)
				}
				continue
			}
			// A short option, or a cluster of them: the first that takes a
			// value takes the rest of the word, or the next word.
			for k := 1; k < len(o); k++ {
				flag := "-" + o[k:k+1]
				isCd := containsString(opts.chdir, flag)
				if !isCd && !containsString(opts.keep, flag) {
					return nil, unread(flag)
				}
				if !containsString(vf, flag) {
					if isCd {
						out = append(out, wrapperChdir{flag: flag, idx: i, unfixed: true})
					}
					continue
				}
				c := wrapperChdir{flag: flag, idx: i, word: o[k+1:]}
				if c.word == "" {
					i++
					var ok bool
					if c, ok = value(flag, i); !ok {
						return nil, short
					}
				}
				if isCd {
					out = append(out, c)
				}
				break
			}
		}
		if len(out)-first > 1 {
			return nil, "a second change of directory given to " + name
		}
		for left := wrapperOperands[w]; left > 0; left-- {
			if i >= end {
				return nil, short
			}
			i++
		}
	}
	if i != end {
		return nil, "a wrapper's words the approver cannot place"
	}
	return out, ""
}

// longChdirFlag is the first long option of a wrapper's chdir options in
// wrapperDirFlags, cf, that the known word o can be, written whole or
// abbreviated, with or without its `=value` (longFlagIs).
func longChdirFlag(o string, cf []string) (string, bool) {
	for _, f := range cf {
		if strings.HasPrefix(f, "--") && longFlagIs(o, f) {
			return f, true
		}
	}
	return "", false
}

// gitDirName is one git directory a command names, with what names it.
type gitDirName struct {
	what, word string
	// tilde says the shell expands a leading ~ in word: a word of its own or
	// an assignment's value, not the value glued to --git-dir=.
	tilde bool
}

// gitDirOf judges the one git command whose arguments start at index from of
// s, run in the directory start leaves it: git's own options up to its
// subcommand, read in order as git reads them, then the git directory they
// leave it with. wrote says an earlier command of the line writes or removes
// a path (writtenStray).
func (f approveFacts) gitDirOf(rc *readCtx, s segment, from int, start *shellState, valueFlags []string, env map[string]string, wrote bool) (Approval, bool) {
	args := s.tokens[from:]
	at := *start
	var named []gitDirName
	var dir *gitDirName
	if v, ok := env["GIT_DIR"]; ok {
		dir = &gitDirName{"GIT_DIR", v, strings.HasPrefix(v, "~")}
	}
	if v, ok := env["GIT_COMMON_DIR"]; ok {
		named = append(named, gitDirName{"GIT_COMMON_DIR", v, strings.HasPrefix(v, "~")})
	}
	// bareAt is where git stands when it reads the first --bare, which sets
	// the git directory to that directory unless one is already named
	// (git(1)): a later -C does not move it, and a later --git-dir names
	// another.
	var bareAt *shellState
	limit := firstOperandLimit(args, valueFlags)
	for i := 0; i < limit && i < len(args); i++ {
		w := args[i]
		switch {
		case w == "-C" || w == "--git-dir":
			if i+1 >= len(args) {
				continue
			}
			i++
			tok, ok := fixedToken(s, from+i)
			if !ok {
				return gitDirUnfixedRefusal(w), true
			}
			if w == "--git-dir" {
				dir = &gitDirName{w, tok, strings.HasPrefix(tok, "~")}
				continue
			}
			if tok == "" {
				continue
			}
			p, ok := at.resolve(rc, tok, strings.HasPrefix(tok, "~"))
			if !ok {
				return gitDirUnfixedRefusal(w), true
			}
			at.dir, at.dirOK = p, true
		case strings.HasPrefix(w, "--git-dir="):
			if _, ok := fixedToken(s, from+i); !ok {
				return gitDirUnfixedRefusal("--git-dir"), true
			}
			dir = &gitDirName{"--git-dir", strings.TrimPrefix(w, "--git-dir="), false}
		case w == "--bare":
			if bareAt == nil {
				b := at
				bareAt = &b
			}
		case isUnknown(w):
			if flagCouldBe(w, "-C") || flagCouldBe(w, "--git-dir") || flagCouldBe(w, "--bare") ||
				strings.HasPrefix(knownLead(w), "--git-dir=") || unknownFlagCouldBe(w, "--git-dir=") {
				return gitDirUnfixedRefusal("-C or --git-dir"), true
			}
		case containsString(valueFlags, w):
			i++
		}
	}
	if dir != nil {
		named = append(named, *dir)
	}
	for _, n := range named {
		if a, ok := f.namedGitDir(rc, &at, n, true, wrote); ok {
			return a, true
		}
	}
	if wrote && at.dirOK && f.writtenStray(at.dir, true) {
		return gitDirUnfixedRefusal(writtenWhat("the directory git runs in")), true
	}
	switch {
	case dir == nil && bareAt != nil:
		if !bareAt.dirOK {
			return gitDirUnfixedRefusal("--bare"), true
		}
		if wrote && f.writtenStray(bareAt.dir, false) {
			return gitDirUnfixedRefusal(writtenWhat("--bare")), true
		}
		if name, ok := f.strayGitDir(bareAt.dir); ok {
			return f.gitDirStrayRefusal(name), true
		}
	case dir == nil && at.dirOK:
		if name, ok := f.discoveredStray(at.dir); ok {
			return f.gitDirStrayRefusal(name), true
		}
	case dir == nil:
		// git finds its repository by its walk upward from a directory the
		// line leaves unknown. A program name the line does not fix is
		// refused as unnamed (unnamedProgramEntryID), whose reason says more.
		if _, ok := fixedToken(s, from-1); ok {
			return gitDirUnfixedRefusal("the directory git runs in"), true
		}
	}
	if a, ok := f.remoteRepoRefusal(rc, s, from, &at, valueFlags); ok {
		return a, true
	}
	return f.separateGitDir(rc, s, from, &at, valueFlags, wrote)
}

// gitRemoteSubcommands are git's subcommands that run against a second
// repository named among their operands, by a path or a URL: push, fetch,
// pull and ls-remote take "the remote repository" (git-push(1),
// git-fetch(1), git-pull(1), git-ls-remote(1)), clone "the (possibly
// remote) repository to clone from" (git-clone(1)), and submodule add "the
// URL of the new submodule's origin repository" (git-submodule(1)); a local
// path there runs git against that repository, its config read and its
// hooks run. archive reaches one only through --remote (git-archive(1)),
// marked true, so its other operands, a tree and paths, are not read.
var gitRemoteSubcommands = map[string]bool{
	"push": false, "fetch": false, "pull": false, "ls-remote": false,
	"clone": false, "submodule": false, "archive": true,
}

// remoteRepoRefusal reads a git command whose subcommand reaches another
// repository (gitRemoteSubcommands), its arguments starting at index from of
// s, run in the directory at leaves git in: each operand, and the value of a
// --repo or --remote option (glued, or for archive the next word), that is
// an existing path holding a git directory other than the working tree's own
// is refused (strayRepoAt). An unknown subcommand is read as one of these,
// and an operand the line does not fix is refused, since it may name such a
// path. A URL with a scheme or an scp-like host: names no local path, and
// file:// is read as the path after it.
func (f approveFacts) remoteRepoRefusal(rc *readCtx, s segment, from int, at *shellState, valueFlags []string) (Approval, bool) {
	args := s.tokens[from:]
	for _, op := range firstOperands(args, valueFlags) {
		sub := args[op]
		remoteOnly, known := gitRemoteSubcommands[sub]
		if !known && !isUnknown(sub) {
			continue
		}
		what := "an operand of git " + sub
		if isUnknown(sub) {
			what = "an operand of a git subcommand the line does not spell out"
		}
		judge := func(j int, word string, glued bool) (Approval, bool) {
			if _, ok := fixedToken(s, from+j); !ok {
				return gitDirUnfixedRefusal(what), true
			}
			p, tilde := word, !glued && strings.HasPrefix(word, "~")
			if rest, ok := strings.CutPrefix(p, "file://"); ok {
				p, tilde = rest, false
			} else if strings.Contains(p, "://") {
				return Approval{}, false
			}
			if c := strings.IndexByte(p, ':'); c >= 0 && !strings.Contains(p[:c], "/") {
				return Approval{}, false // host:path, which git reads as a remote host
			}
			if p == "" {
				return Approval{}, false
			}
			path, ok := at.resolve(rc, p, tilde)
			if !ok {
				return gitDirUnfixedRefusal(what), true
			}
			if name, ok := f.strayRepoAt(path); ok {
				return f.gitDirStrayRefusal(name), true
			}
			return Approval{}, false
		}
		for j := op + 1; j < len(args); j++ {
			w := args[j]
			switch {
			case w == "--":
				continue
			case isUnknown(w):
				// A word the line does not spell out may be such an operand,
				// or archive's --remote.
				if lead := knownLead(w); !remoteOnly || lead == "" || strings.HasPrefix("--remote", lead) || longFlagIs(w, "--remote") {
					return gitDirUnfixedRefusal(what), true
				}
			case strings.HasPrefix(w, "--"):
				name, v, glued := strings.Cut(w, "=")
				switch {
				case glued && (longFlagIs(name, "--repo") || longFlagIs(name, "--remote")):
					if a, ok := judge(j, v, true); ok {
						return a, true
					}
				case remoteOnly && !glued && longFlagIs(name, "--remote") && j+1 < len(args):
					j++
					if a, ok := judge(j, args[j], false); ok {
						return a, true
					}
				}
			case strings.HasPrefix(w, "-") || remoteOnly:
				continue
			default:
				if a, ok := judge(j, w, false); ok {
					return a, true
				}
			}
		}
	}
	return Approval{}, false
}

// strayRepoAt reports whether the local path p, an operand naming a
// repository, holds a git directory other than the working tree's own, as
// git looks for one there: p/.git, p itself as a git directory or a git
// file, then p.git/.git and p.git. Each found is judged as strayGitDir
// judges it; a plain directory, or nothing at all, holds none.
func (f approveFacts) strayRepoAt(p string) (string, bool) {
	real := fsutil.RealExistingPath(p)
	for _, base := range []string{real, real + ".git"} {
		if _, err := os.Lstat(filepath.Join(base, ".git")); err == nil {
			return f.strayGitDir(filepath.Join(base, ".git"))
		}
		fi, err := os.Stat(base)
		if err != nil {
			continue
		}
		if fi.IsDir() && gitDirShaped(base) {
			return f.strayGitDir(base)
		}
		if fi.Mode().IsRegular() {
			if _, ok := pointedTo(base, "gitdir: "); ok {
				return f.strayGitDir(base)
			}
		}
	}
	return "", false
}

// gitReadsOnly are git's subcommands known to leave the working tree alone,
// each by its page's own account of what it does: status "displays paths
// that have differences" (git-status(1)); log "shows the commit logs"
// (git-log(1)); diff "shows changes" (git-diff(1)); show "shows one or more
// objects" (git-show(1)); rev-parse picks out and massages parameters
// (git-rev-parse(1)); rev-list lists commit objects (git-rev-list(1));
// ls-files "shows information about files in the index and the working
// tree" (git-ls-files(1)); ls-tree lists the contents of a tree object
// (git-ls-tree(1)); cat-file provides the contents of repository objects
// (git-cat-file(1)); show-ref and for-each-ref list references
// (git-show-ref(1), git-for-each-ref(1)); describe gives an object a name
// (git-describe(1)); merge-base finds common ancestors (git-merge-base(1));
// blame shows who last changed each line (git-blame(1)); shortlog
// summarizes the log (git-shortlog(1)); version prints git's version. With
// true, branch and remote are on the list only with no arguments, where they
// list (git-branch(1), git-remote(1)). Any other subcommand may change the
// working tree, and so counts as a write for git-dir-not-own.
var gitReadsOnly = map[string]bool{
	"status": false, "log": false, "diff": false, "show": false, "rev-parse": false,
	"rev-list": false, "ls-files": false, "ls-tree": false, "cat-file": false,
	"show-ref": false, "for-each-ref": false, "describe": false, "merge-base": false,
	"blame": false, "shortlog": false, "version": false,
	"branch": true, "remote": true,
}

// gitSubcommands are the subcommands git(1) documents, in its lists of
// porcelain, ancillary, interacting, low-level and helper commands, as git
// 2.52 gives them (git help -a), which gitReadsOnly and gitRemoteSubcommands
// are drawn from. A word naming git that the walk to a command did not
// reach, followed by one of these, is git run by a program the walk does not
// step through (gitBehindProgram). A subcommand a git release after 2.52 adds
// is not on the list, so git behind a program followed by it is a case this
// rule does not read.
var gitSubcommands = map[string]bool{
	// Main porcelain commands.
	"add": true, "am": true, "archive": true, "backfill": true, "bisect": true,
	"branch": true, "bundle": true, "checkout": true, "cherry-pick": true,
	"citool": true, "clean": true, "clone": true, "commit": true, "describe": true,
	"diff": true, "fetch": true, "format-patch": true, "gc": true, "grep": true, "gui": true, "init": true, "log": true, "maintenance": true,
	"merge": true, "mv": true, "notes": true, "pull": true, "push": true,
	"range-diff": true, "rebase": true, "reset": true, "restore": true,
	"revert": true, "rm": true, "shortlog": true, "show": true,
	"sparse-checkout": true, "stash": true, "status": true, "submodule": true,
	"switch": true, "tag": true, "worktree": true,
	// Ancillary manipulators and interrogators.
	"config": true, "fast-export": true, "fast-import": true, "filter-branch": true,
	"mergetool": true, "pack-refs": true, "prune": true, "reflog": true,
	"refs": true, "remote": true, "repack": true, "replace": true,
	"annotate": true, "blame": true, "bugreport": true, "count-objects": true,
	"diagnose": true, "difftool": true, "fsck": true, "help": true,
	"instaweb": true, "merge-tree": true, "rerere": true, "show-branch": true,
	"verify-commit": true, "verify-tag": true, "version": true, "whatchanged": true,
	// Interacting with others.
	"archimport": true, "cvsexportcommit": true, "cvsimport": true,
	"cvsserver": true, "imap-send": true, "p4": true, "quiltimport": true,
	"request-pull": true, "send-email": true, "svn": true,
	// Low-level manipulators and interrogators.
	"apply": true, "checkout-index": true, "commit-graph": true, "commit-tree": true,
	"hash-object": true, "index-pack": true, "merge-file": true, "merge-index": true,
	"mktag": true, "mktree": true, "multi-pack-index": true, "pack-objects": true,
	"prune-packed": true, "read-tree": true, "replay": true, "symbolic-ref": true,
	"unpack-objects": true, "update-index": true, "update-ref": true, "write-tree": true,
	"cat-file": true, "cherry": true, "diff-files": true, "diff-index": true,
	"diff-pairs": true, "diff-tree": true, "for-each-ref": true, "for-each-repo": true,
	"get-tar-commit-id": true, "last-modified": true, "ls-files": true, "ls-remote": true,
	"ls-tree": true, "merge-base": true, "name-rev": true, "pack-redundant": true,
	"repo": true, "rev-list": true, "rev-parse": true, "show-index": true,
	"show-ref": true, "unpack-file": true, "var": true, "verify-pack": true,
	// Syncing repositories, and the helpers pulling and pushing run.
	"daemon": true, "fetch-pack": true, "http-backend": true, "send-pack": true,
	"update-server-info": true, "http-fetch": true, "http-push": true,
	"receive-pack": true, "shell": true, "upload-archive": true, "upload-pack": true,
	// Internal helper commands.
	"check-attr": true, "check-ignore": true, "check-mailmap": true,
	"check-ref-format": true, "column": true, "credential": true,
	"credential-cache": true, "credential-store": true, "fmt-merge-msg": true,
	"hook": true, "interpret-trailers": true, "mailinfo": true, "mailsplit": true,
	"merge-one-file": true, "patch-id": true, "sh-i18n": true, "sh-setup": true,
	"stripspace": true,
}

// gitLeavesTreeAlone reports whether every git command in s is one of
// gitReadsOnly, read from its subcommand on: a subcommand the line does not
// spell out, or one off the list, may change the working tree, and so may a
// listed one given a word the line does not spell out or --output, written
// whole or abbreviated, which writes the diff to a file (git-diff(1)).
func gitLeavesTreeAlone(s segment, valueFlags []string) bool {
	for _, site := range sitesNamed(s, "git") {
		args := s.tokens[site.idx+1:]
		for _, op := range firstOperands(args, valueFlags) {
			bare, listed := gitReadsOnly[args[op]]
			rest := args[op+1:]
			if isUnknown(args[op]) || !listed || bare && len(rest) > 0 {
				return false
			}
			for _, w := range rest {
				if isUnknown(w) || longFlagIs(w, "--output") {
					return false
				}
			}
		}
	}
	return true
}

// writtenWhat names what a refusal after an earlier write judged.
func writtenWhat(what string) string {
	return what + ", after an earlier command of the line writes"
}

// writtenStray reports whether p, a clean absolute path git runs in (dir) or
// reads as its git directory, may stand where an earlier command of the
// line made a repository: inside the working tree, judged as written and
// with its existing directories resolved through symlinks, and neither in
// the working tree's own git directories nor, for a directory git runs in,
// the working tree's root, from which git finds the tree's own .git.
func (f approveFacts) writtenStray(p string, dir bool) bool {
	for _, c := range []string{p, fsutil.RealExistingPath(p)} {
		if !f.within(c, f.gitTrees) || f.within(c, f.ownGitDirs) {
			continue
		}
		if dir && f.isGitTree(c) {
			continue
		}
		return true
	}
	return false
}

// isGitTree reports whether p is one of gitTrees itself.
func (f approveFacts) isGitTree(p string) bool {
	for _, t := range f.gitTrees {
		if fsutil.PathWithin(p, t, f.fold) && fsutil.PathWithin(t, p, f.fold) {
			return true
		}
	}
	return false
}

// namedGitDir judges a git directory n names, read from the directory at
// leaves git in; follow says a git file there is read for the directory it
// names, and wrote that an earlier command of the line writes (writtenStray).
func (f approveFacts) namedGitDir(rc *readCtx, at *shellState, n gitDirName, follow, wrote bool) (Approval, bool) {
	if isUnknown(n.word) || strings.ContainsRune(n.word, varMark) {
		return gitDirUnfixedRefusal(n.what), true
	}
	if n.word == "" {
		return Approval{}, false
	}
	p, ok := at.resolve(rc, n.word, n.tilde)
	if !ok {
		return gitDirUnfixedRefusal(n.what), true
	}
	if wrote && f.writtenStray(p, false) {
		return gitDirUnfixedRefusal(writtenWhat(n.what)), true
	}
	stray := f.strayGitDir
	if !follow {
		stray = f.strayPath
	}
	if name, ok := stray(p); ok {
		return f.gitDirStrayRefusal(name), true
	}
	return Approval{}, false
}

// separateGitDir reads `git init --separate-git-dir <git-dir>`
// (git-init(1)), the option written whole or abbreviated, its value glued
// after `=` or the next word, read from the directory at leaves git in.
func (f approveFacts) separateGitDir(rc *readCtx, s segment, from int, at *shellState, valueFlags []string, wrote bool) (Approval, bool) {
	args := s.tokens[from:]
	for _, op := range firstOperands(args, valueFlags) {
		if args[op] != "init" && !isUnknown(args[op]) {
			continue
		}
		for j := op + 1; j < len(args); j++ {
			w := args[j]
			if w == "--" {
				break
			}
			if !strings.HasPrefix(w, "-") && !isUnknown(w) || !longFlagIs(w, "--separate-git-dir") {
				continue
			}
			n := gitDirName{what: "--separate-git-dir"}
			switch name, v, glued := strings.Cut(w, "="); {
			case isUnknown(name):
				return gitDirUnfixedRefusal(n.what), true
			case glued:
				if _, ok := fixedToken(s, from+j); !ok {
					return gitDirUnfixedRefusal(n.what), true
				}
				n.word = v
			case j+1 < len(args):
				j++
				tok, ok := fixedToken(s, from+j)
				if !ok {
					return gitDirUnfixedRefusal(n.what), true
				}
				n.word, n.tilde = tok, strings.HasPrefix(tok, "~")
			default:
				continue
			}
			if a, ok := f.namedGitDir(rc, at, n, false, wrote); ok {
				return a, true
			}
		}
	}
	return Approval{}, false
}

// strayPath reports whether p, a clean absolute path, stands inside the
// working tree and outside the working tree's own git directories, judged
// with its existing directories resolved through symlinks.
func (f approveFacts) strayPath(p string) (string, bool) {
	real := fsutil.RealExistingPath(p)
	if !f.within(real, f.gitTrees) || f.within(real, f.ownGitDirs) {
		return "", false
	}
	return p, true
}

// strayGitDir is strayPath for a git directory git reads, wherever it
// stands: a git file (a regular file, as git reads one named by --git-dir
// or GIT_DIR, or found as a .git) is judged by the directory its "gitdir: "
// line names, and one that names none is refused; a directory, and the
// directory a git file names, is judged with the one its commondir file
// names (strayRepoDir).
func (f approveFacts) strayGitDir(p string) (string, bool) {
	real := fsutil.RealExistingPath(p)
	if fi, err := os.Stat(real); err == nil && fi.Mode().IsRegular() {
		target, ok := pointedTo(real, "gitdir: ")
		if !ok {
			return p, true
		}
		return f.strayRepoDir(target)
	}
	return f.strayRepoDir(p)
}

// strayRepoDir judges the git directory p and, where it holds a commondir
// file, the directory that names, from which git takes the config a
// worktree shares (gitrepository-layout(5)). A commondir that is there and
// cannot be read is refused.
func (f approveFacts) strayRepoDir(p string) (string, bool) {
	if name, ok := f.strayPath(p); ok {
		return name, true
	}
	cd := filepath.Join(fsutil.RealExistingPath(p), "commondir")
	if _, err := os.Lstat(cd); err != nil {
		return "", false
	}
	common, ok := pointedTo(cd, "")
	if !ok {
		return cd, true
	}
	return f.strayPath(common)
}

// pointerCap bounds a git file or commondir file pointedTo reads.
const pointerCap = 4 << 10

// pointedTo is the path the first line of the file p holds after prefix
// ("gitdir: " for a git file, nothing for commondir), joined to p's
// directory when relative and cleaned. It reads through fsutil.ReadGuarded,
// so a symlinked, non-regular or oversized file, or a first line without
// prefix or with nothing after it, answers false.
func pointedTo(p, prefix string) (string, bool) {
	b, err := fsutil.ReadGuarded(p, pointerCap)
	if err != nil {
		return "", false
	}
	line, _, _ := strings.Cut(string(b), "\n")
	target, ok := strings.CutPrefix(strings.TrimRight(line, "\r"), prefix)
	if !ok || target == "" {
		return "", false
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(p), target)
	}
	return filepath.Clean(target), true
}

// discoveredStray judges the git directory git finds from dir by its walk
// upward: the first directory at or above dir that is itself a repository
// (gitDirShaped), below the nearest one holding a .git entry
// (gitutil.RepoShapedRoot), else that entry, each judged as strayGitDir
// judges it, so a walk from outside the working tree that reaches a git
// file or commondir pointing inside it is refused. A dir inside the working
// tree's own git directories is left alone.
func (f approveFacts) discoveredStray(dir string) (string, bool) {
	real := fsutil.RealExistingPath(dir)
	if f.within(real, f.ownGitDirs) {
		return "", false
	}
	marker := gitutil.RepoShapedRoot(real)
	for d := real; d != marker; d = filepath.Dir(d) {
		if gitDirShaped(d) {
			return f.strayGitDir(d)
		}
		if filepath.Dir(d) == d {
			break
		}
	}
	if marker == "" {
		return "", false
	}
	return f.strayGitDir(filepath.Join(marker, ".git"))
}

// gitDirShaped reports whether d has the shape of a git directory, a bare
// repository's or a worktree's: a HEAD entry, a file or a symlink whether
// it leads anywhere or not (git takes a symlink HEAD by its text), with an
// objects or refs directory or a commondir file beside it. It is looser
// than git's own check, so it refuses more, never less.
func gitDirShaped(d string) bool {
	if fi, err := os.Lstat(filepath.Join(d, "HEAD")); err != nil || !fi.Mode().IsRegular() && fi.Mode()&os.ModeSymlink == 0 {
		return false
	}
	for _, n := range []string{"objects", "refs", "commondir"} {
		if _, err := os.Stat(filepath.Join(d, n)); err == nil {
			return true
		}
	}
	return false
}

// inTree is p as a refusal names it: relative to the working tree it stands
// in, as written or with its existing directories resolved, else as
// fsutil.DisplayPath shows it.
func (f approveFacts) inTree(p string) string {
	for _, c := range []string{p, fsutil.RealExistingPath(p)} {
		for _, t := range f.gitTrees {
			if rel, ok := fsutil.RelWithin(t, c, f.fold); ok {
				return rel
			}
		}
	}
	return fsutil.DisplayPath(p)
}

// gitDirStrayRefusal is the refusal of a git command that reads the git
// directory p, inside the working tree and not its own.
func (f approveFacts) gitDirStrayRefusal(p string) Approval {
	name := termsafe.Sanitize(f.inTree(p))
	return refusal(gitDirEntryID,
		"this git command reads the git directory "+name+", which stands inside the working tree and is not the working tree's own, so git takes its configuration, which may name a program git starts, from a file the guard has not read.",
		"run git against the working tree's own repository, or against a copy outside the working tree, or say in the receipt that the task needs "+name+".")
}

// gitDirUnfixedRefusal is the refusal of a git command whose directory, or
// git directory, what names in a way the line does not fix.
func gitDirUnfixedRefusal(what string) Approval {
	return refusal(gitDirEntryID,
		"this git command runs in, or reads its git directory from, a place the line does not fix ("+termsafe.Sanitize(what)+"): a word the line does not spell out, a wrapper or a wrapper's option the approver does not read as leaving the directory alone, a directory or relative path after a change of directory the line leaves unknown, or a directory inside the working tree after an earlier command of the line writes, which may have made a repository there, so the guard cannot say which repository's configuration git reads.",
		"the same git command with the directory spelled out in full, run from the working tree's root or in a line of its own.")
}
