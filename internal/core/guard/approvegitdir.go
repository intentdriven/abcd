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
// A submodule's git file points into the working tree's own git directory,
// and is left alone; so is a git directory outside the working tree, which
// the approver's other rules judge. A directory the line does not fix is
// refused.

import (
	"os"
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
// own, or is named by a directory the line does not fix.
func (f approveFacts) gitDirRefusal(rc *readCtx, s segment, st *shellState, valueFlags []string, carried map[string]string) (Approval, bool) {
	if len(f.gitTrees) == 0 {
		return Approval{}, false
	}
	for _, site := range sitesNamed(s, "git") {
		from := site.idx + 1
		env := readGitConfigIn(carried, s.tokens[:site.idx], s.tokens[from:], valueFlags).env
		if a, ok := f.gitDirOf(rc, s, from, st, valueFlags, env); ok {
			return a, true
		}
	}
	return Approval{}, false
}

// gitDirName is one git directory a command names, with what names it.
type gitDirName struct {
	what, word string
	// tilde says the shell expands a leading ~ in word: a word of its own or
	// an assignment's value, not the value glued to --git-dir=.
	tilde bool
}

// gitDirOf judges the one git command whose arguments start at index from of
// s: git's own options up to its subcommand, read in order as git reads
// them, then the git directory they leave it with.
func (f approveFacts) gitDirOf(rc *readCtx, s segment, from int, st *shellState, valueFlags []string, env map[string]string) (Approval, bool) {
	args := s.tokens[from:]
	at := *st
	at.dirOK = st.dirOK && st.dir != ""
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
		if a, ok := f.namedGitDir(rc, &at, n, true); ok {
			return a, true
		}
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
	}
	return f.separateGitDir(rc, s, from, &at, valueFlags)
}

// namedGitDir judges a git directory n names, read from the directory at
// leaves git in; follow says a git file there is read for the directory it
// names.
func (f approveFacts) namedGitDir(rc *readCtx, at *shellState, n gitDirName, follow bool) (Approval, bool) {
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
func (f approveFacts) separateGitDir(rc *readCtx, s segment, from int, at *shellState, valueFlags []string) (Approval, bool) {
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
			if a, ok := f.namedGitDir(rc, at, n, false); ok {
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
// git directory, what names by a word the line does not fix.
func gitDirUnfixedRefusal(what string) Approval {
	return refusal(gitDirEntryID,
		"this git command names the directory it runs in or the git directory it reads ("+termsafe.Sanitize(what)+") by a word the line does not spell out, or by a relative path after a change of directory the line leaves unknown, so the guard cannot say which repository's configuration git reads.",
		"the same git command with the directory spelled out in full.")
}
