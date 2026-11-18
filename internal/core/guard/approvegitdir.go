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
//     directory";
//   - git init --separate-git-dir <git-dir>: the repository is made there,
//     or, on a reinitialization, moved there.
//
// The directory git runs in is the shell's, as each cd the line fixes leaves
// it, then as each env -C or --chdir (env(1)) and sudo -D or --chdir
// (sudo(8)) before git changes it, resolved the way -C is.
//
// A submodule's git file points into the working tree's own git directory,
// and is left alone; so is a git directory outside the working tree, which
// the approver's other rules judge. A directory the line does not fix is
// refused, and so is one git would find its repository from when the line
// leaves it unknown. After an earlier command of the line writes, which may
// make a repository where none stood, a git command is refused that runs in
// a directory inside the working tree other than its root or its own git
// directories, or names a git directory inside the working tree other than
// its own.

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
// earlier command of the line writes or removes a path.
func (f approveFacts) gitDirRefusal(rc *readCtx, s segment, st *shellState, valueFlags []string, carried map[string]string, wrote bool) (Approval, bool) {
	if len(f.gitTrees) == 0 {
		return Approval{}, false
	}
	for _, site := range sitesNamed(s, "git") {
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
	return Approval{}, false
}

// wrapperChdirFlags names, per wrapper, the options of its
// wrapperValueFlags that change the directory the command after it runs in:
// env's -C and --chdir (env(1)) and sudo's -D and --chdir (sudo(8)).
var wrapperChdirFlags = map[string][]string{
	"env":  {"-C", "--chdir"},
	"sudo": {"-D", "--chdir"},
}

// wrapperChdir is one change of directory a wrapper makes before the command
// it runs: the option that makes it, the index of the word holding its value
// and the value, with whether the shell expands a leading ~ in it.
type wrapperChdir struct {
	flag  string
	idx   int
	word  string
	tilde bool
}

// wrapperDir is the shell state st as the wrappers before the git site at
// index end of s leave it: each change of directory wrapperChdirs reads,
// resolved in order the way git's -C is. One whose value the line does not
// fix is refused; so, where wrapperChdirs cannot place the site, is any word
// before it that may be a change of directory (mayChangeDir).
func wrapperDir(rc *readCtx, s segment, end int, st *shellState) (*shellState, Approval, bool) {
	at := *st
	at.dirOK = st.dirOK && st.dir != ""
	changes, placed := wrapperChdirs(s, end)
	if !placed {
		for i := 0; i < end; i++ {
			if mayChangeDir(s.tokens[i]) {
				return nil, gitDirUnfixedRefusal("a wrapper's change of directory"), true
			}
		}
		return &at, Approval{}, false
	}
	for _, c := range changes {
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
// directory wrapperChdirFlags names, written whole, abbreviated, glued or
// last in a short-option cluster, and whether the reading arrives at end. A
// word the line does not fix, a program that is not a wrapper, or a word
// left over leaves the site unplaced.
func wrapperChdirs(s segment, end int) ([]wrapperChdir, bool) {
	toks := s.tokens
	var out []wrapperChdir
	value := func(flag string, i int) (wrapperChdir, bool) {
		if i >= end {
			return wrapperChdir{}, false
		}
		return wrapperChdir{flag, i, toks[i], strings.HasPrefix(toks[i], "~")}, true
	}
	i, assigns := 0, false
	for i < end {
		tok := toks[i]
		if isAssignment(tok) || reserved[tok] || assigns && isWrapperAssignment(tok) {
			i++
			continue
		}
		w := strings.ToLower(path.Base(tok))
		if isUnknown(tok) || !wrappers[w] {
			return nil, false
		}
		assigns = envAssigns[w]
		vf, cf := wrapperValueFlags[w], wrapperChdirFlags[w]
		long := containsString(cf, "--chdir")
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
				return nil, false
			}
			if strings.HasPrefix(o, "--") {
				name, v, glued := strings.Cut(o, "=")
				switch {
				case long && longFlagIs(o, "--chdir") && glued:
					out = append(out, wrapperChdir{name, i, v, false})
				case long && longFlagIs(o, "--chdir"):
					i++
					c, ok := value(name, i)
					if !ok {
						return nil, false
					}
					out = append(out, c)
				case !glued && containsString(vf, o):
					i++
				}
				continue
			}
			// A short option, or a cluster of them: the first that takes a
			// value takes the rest of the word, or the next word.
			for k := 1; k < len(o); k++ {
				flag := "-" + o[k:k+1]
				if !containsString(vf, flag) {
					continue
				}
				c := wrapperChdir{flag, i, o[k+1:], false}
				if c.word == "" {
					i++
					var ok bool
					if c, ok = value(flag, i); !ok {
						return nil, false
					}
				}
				if containsString(cf, flag) {
					out = append(out, c)
				}
				break
			}
		}
		for left := wrapperOperands[w]; left > 0; left-- {
			if i >= end {
				return nil, false
			}
			i++
		}
	}
	return out, i == end
}

// mayChangeDir reports whether a word may be one of wrapperChdirFlags, written
// whole, abbreviated or glued, or a short-option cluster carrying one, read
// as flagCouldBe and longFlagIs read a word the line does not fix.
func mayChangeDir(tok string) bool {
	lead := knownLead(tok)
	for _, flags := range wrapperChdirFlags {
		for _, flag := range flags {
			if strings.HasPrefix(flag, "--") {
				if strings.HasPrefix(lead, "--") && longFlagIs(tok, flag) || isUnknown(tok) && flagCouldBe(tok, flag) {
					return true
				}
				continue
			}
			if isUnknown(tok) && (flagCouldBe(tok, flag) || clusterCouldCarry(tok, flag[1])) {
				return true
			}
			if strings.HasPrefix(lead, "-") && !strings.HasPrefix(lead, "--") && strings.IndexByte(lead[1:], flag[1]) >= 0 {
				return true
			}
		}
	}
	return false
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
	bare := false
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
			bare = true
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
	case dir == nil && bare:
		if !at.dirOK {
			return gitDirUnfixedRefusal("--bare"), true
		}
		if name, ok := f.strayGitDir(at.dir); ok {
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
	return f.separateGitDir(rc, s, from, &at, valueFlags, wrote)
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

// strayGitDir is strayPath for a git directory git reads: a git file
// (a .git that is a regular file) is judged by the directory its "gitdir: "
// line names (gitutil.GitfileTarget), and one that names none, or a regular
// file by another name, is refused.
func (f approveFacts) strayGitDir(p string) (string, bool) {
	name, ok := f.strayPath(p)
	if !ok {
		return "", false
	}
	real := fsutil.RealExistingPath(p)
	if fi, err := os.Stat(real); err == nil && fi.Mode().IsRegular() {
		if !strings.EqualFold(filepath.Base(real), ".git") {
			return name, true
		}
		target, ok := gitutil.GitfileTarget(filepath.Dir(real))
		if !ok {
			return name, true
		}
		return f.strayPath(target)
	}
	return name, true
}

// discoveredStray judges the git directory git finds from dir by its walk
// upward: the first directory at or above dir that is itself a repository
// (gitDirShaped), below the nearest one holding a .git entry
// (gitutil.RepoShapedRoot), else that entry. A dir outside the working
// tree, or inside its own git directories, is left alone.
func (f approveFacts) discoveredStray(dir string) (string, bool) {
	real := fsutil.RealExistingPath(dir)
	if !f.within(real, f.gitTrees) || f.within(real, f.ownGitDirs) {
		return "", false
	}
	marker := gitutil.RepoShapedRoot(real)
	for d := real; d != marker && f.within(d, f.gitTrees); d = filepath.Dir(d) {
		if gitDirShaped(d) {
			return d, true
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
// repository's or a worktree's: a HEAD file, with an objects or refs
// directory or a commondir file beside it. It is looser than git's own
// check, so it refuses more, never less.
func gitDirShaped(d string) bool {
	if fi, err := os.Stat(filepath.Join(d, "HEAD")); err != nil || !fi.Mode().IsRegular() {
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
		"this git command runs in, or reads its git directory from, a place the line does not fix ("+termsafe.Sanitize(what)+"): a word the line does not spell out, a directory or relative path after a change of directory the line leaves unknown, or a directory inside the working tree after an earlier command of the line writes, which may have made a repository there, so the guard cannot say which repository's configuration git reads.",
		"the same git command with the directory spelled out in full, run from the working tree's root or in a line of its own.")
}
