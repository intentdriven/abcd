package interview

// treewatch.go holds an AI-written interview's role to the paths its contract
// lets it change. The role runs with the person's own route in the checkout,
// and with no host session nobody watches its edits between the questions
// abcd draws, so abcd does: before each dispatch it reads the state of every
// place the role could leave something behind, after the dispatch it reads it
// again, and a path whose state moved that the interview did not grant stops
// the interview, naming each. abcd cannot tell who wrote a path in that
// window, so a path anything else on the machine changes while the role runs
// stops it too (a second interview in the same checkout among them: one
// checkout is one session).
//
// Five places are read:
//
//   - What differs from HEAD, as git lists it (gitutil.Status, the one reader
//     of `git status --porcelain=v1 -z --untracked-files=all`): each path
//     paired with its content's mode and hash, so a second edit to a file
//     already changed before the dispatch moves its state too.
//   - What git ignores (the same listing's --ignored=matching entries, an
//     ignored directory walked whole), and an untracked nested repository
//     walked the same way: each file by its mode, size and modification time,
//     never its content, since an ignored tree (a build's output, the local
//     tier's scratch) can be large. The local tier is watched like any other
//     path: a push receipt under preflight-receipts/ or the handover changed
//     by the role stops the interview.
//   - git's own directory, which no status lists: every entry of hooks/ and
//     info/, the configuration (config, and config.worktree), HEAD, the refs
//     (refs/ and packed-refs), each submodule's hooks and configuration under
//     modules/, and a linked worktree's .git file, by mode, size and content
//     hash. A hook runs on the person's next git command, the configuration
//     can name a hooks directory, an alias or a credential helper, and HEAD
//     and the refs decide what the next commit or push carries. In a linked
//     worktree these live in the repository's common directory and are read
//     there, besides the worktree's own HEAD and config.worktree. Every
//     worktree's entry under the common directory's worktrees/ is read too:
//     the entry itself, so one made or removed is noticed, and its HEAD,
//     commondir, gitdir, config.worktree and locked, which decide where git
//     places that worktree, which configuration and hooks it runs there, and
//     where git lists it. git follows a link there, so a link (a
//     dotfiles-managed hooks directory, a hook or a configuration file) is
//     read where it leads too.
//   - The directory each core.hooksPath value names, in any scope the
//     person's git reads, when it is outside the working tree (inside, the
//     listing above already covers it), read as git's own directory is. The
//     value is resolved through every link first, so a hooks directory
//     reached through a link (a dotfiles-managed one, or an in-tree link to a
//     directory elsewhere) is read where git runs its hooks from.
//   - The push receipts (.abcd/.work.local/preflight-receipts/) of every
//     worktree git lists, by mode, size and content hash: the pre-push gate
//     takes a commit's receipt from any of them, so a receipt written into
//     another worktree's local tier during a turn stops the interview too.
//
// Two places are left out because abcd itself writes them while a role runs:
// the run's own turn directory, where abcd writes each brief and the role its
// receipt, and the local transcript store a checkout can pull its runners'
// transcripts into. Every other run's turn directory is watched. Two paths are
// left out because something else on the machine writes them while a role
// runs and neither executes anything: a file named .DS_Store anywhere
// (Finder's metadata), and the host scheduler's lock at the repository's root,
// .claude/scheduled_tasks.lock, that one path alone. Every other path under
// .claude/ is watched: the host's settings there can name hooks.
//
// Paths are relative to the repository's root; a path outside it (a linked
// worktree's common directory, a hooks directory elsewhere) is named in full.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// maxStatusBytes bounds the status listing; a tree whose listing exceeds it
// is refused rather than read in part.
const maxStatusBytes = 8 << 20

// maxWatched bounds the paths one reading holds. A tree past it is refused
// rather than read in part, before the first dispatch: a guard that stopped
// counting would let a role write past the bound unseen. It is several times
// what a large checkout's local tier and build output hold, so a tree
// reaching it has something in it worth clearing.
var maxWatched = 500_000

// treeState is each watched path mapped to its status and its state.
type treeState map[string]string

// schedulerLock is the one path under .claude/ left out of the reading: the
// host scheduler's lock, which executes nothing.
const schedulerLock = ".claude/scheduled_tasks.lock"

// finderMetadata is the base name of Finder's metadata file, left out of the
// reading wherever it stands.
const finderMetadata = ".DS_Store"

// UnexpectedChangesError is an interview stopped because paths the interview
// does not let its role change changed while the role ran. abcd cannot tell
// who changed them: the role, or anything else writing in that window.
type UnexpectedChangesError struct {
	Role string
	// Paths are the paths changed, relative to the repository's root.
	Paths []string
}

func (e *UnexpectedChangesError) Error() string {
	return fmt.Sprintf("%d path(s) changed while the %s ran that this interview does not let it change, so the interview stopped: %s; "+
		"read each (git status, git diff) and restore what you did not ask for",
		len(e.Paths), e.Role, strings.Join(e.Paths, ", "))
}

// treeReader is one reading of the places a role could change.
type treeReader struct {
	repo string
	// realRepo is repo with every link resolved, so a path read through its
	// real spelling is still keyed by its place in the tree.
	realRepo string
	// skip are repository-relative slash paths left out, with everything
	// under them.
	skip []string
	st   treeState
}

// readTree reads the state of every watched place under repo, leaving out
// turnDir, the run's own turn directory (relative to repo, slash-separated).
func readTree(repo, turnDir string) (treeState, error) {
	r := &treeReader{repo: repo, realRepo: fsutil.RealExistingPath(repo), skip: []string{history.LocalStoreRelPath}, st: treeState{}}
	if turnDir != "" {
		r.skip = append(r.skip, turnDir)
	}
	if err := r.read(); err != nil {
		return nil, fmt.Errorf("interview: the working tree's state cannot be read, so the role's changes cannot be held to its contract: %w", err)
	}
	return r.st, nil
}

func (r *treeReader) read() error {
	entries, err := gitutil.Status(r.repo, maxStatusBytes, gitutil.StatusOptions{Ignored: true})
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(r.repo)
	if err != nil {
		return err
	}
	defer root.Close()
	for _, e := range entries {
		switch {
		case strings.HasSuffix(e.Path, "/"):
			// An ignored directory, or an untracked nested repository: git
			// lists the directory alone, so it is walked.
			err = r.walkStat(e.XY, strings.TrimSuffix(e.Path, "/"))
		case e.XY == "!!":
			err = r.statOne(e.XY, e.Path)
		default:
			err = r.hashOne(root, e.XY, e.Path)
			if err == nil && e.Orig != "" {
				// A rename's or copy's source is watched with it.
				err = r.hashOne(root, e.XY+"<", e.Orig)
			}
		}
		if err != nil {
			return err
		}
	}
	return r.readGitDirs()
}

// skipped reports whether rel is left out of the reading.
func (r *treeReader) skipped(rel string) bool {
	for _, s := range r.skip {
		if rel == s || strings.HasPrefix(rel, s+"/") {
			return true
		}
	}
	return false
}

// exempt reports whether key is one of the two paths something else on the
// machine writes while a role runs that execute nothing: a file named
// .DS_Store anywhere, and the scheduler's lock at the root.
func exempt(key string) bool {
	return key == schedulerLock || path.Base(key) == finderMetadata
}

// put records one path's state, refusing past maxWatched. An exempt path is
// not recorded.
func (r *treeReader) put(key, state string) error {
	if exempt(key) {
		return nil
	}
	if _, ok := r.st[key]; !ok && len(r.st) >= maxWatched {
		return fmt.Errorf("the tree holds more than %d paths outside what git tracks unchanged (untracked, ignored, or in git's own directory), "+
			"too many to read around each turn; remove what it no longer needs (a scratch directory, a build's output) and run the interview again", maxWatched)
	}
	r.st[key] = state
	return nil
}

// hashOne records a path git lists by its content's mode and hash.
func (r *treeReader) hashOne(root *os.Root, xy, rel string) error {
	if r.skipped(rel) {
		return nil
	}
	h, err := contentState(root, rel)
	if err != nil {
		return err
	}
	return r.put(rel, xy+" "+h)
}

// statOne records an ignored path by its mode, size and modification time.
func (r *treeReader) statOne(xy, rel string) error {
	if r.skipped(rel) {
		return nil
	}
	fi, err := os.Lstat(filepath.Join(r.repo, filepath.FromSlash(rel)))
	if errors.Is(err, fs.ErrNotExist) {
		return r.put(rel, xy+" absent")
	}
	if err != nil {
		return err
	}
	return r.put(rel, xy+" "+statState(fi))
}

// walkStat records every file under the directory rel by its mode, size and
// modification time. A directory itself is not recorded: what it holds is,
// and git keeps no empty directory either. A link is recorded, never
// followed.
func (r *treeReader) walkStat(xy, rel string) error {
	if r.skipped(rel) {
		return nil
	}
	base := filepath.Join(r.repo, filepath.FromSlash(rel))
	return filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		sub, err := filepath.Rel(base, p)
		if err != nil {
			return err
		}
		key := path.Join(rel, filepath.ToSlash(sub))
		if r.skipped(key) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		fi, err := d.Info()
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		return r.put(key, xy+" "+statState(fi))
	})
}

// statState is a file's mode, size and modification time.
func statState(fi fs.FileInfo) string {
	return fmt.Sprintf("%v %d %d", fi.Mode(), fi.Size(), fi.ModTime().UnixNano())
}

// readGitDirs records git's own directory and every hooks directory
// core.hooksPath names outside the working tree, each entry by its mode,
// size and content hash.
func (r *treeReader) readGitDirs() error {
	out, err := gitutil.Run(r.repo, "rev-parse", "--git-dir", "--git-common-dir")
	if err != nil {
		return err
	}
	lines := strings.Split(out, "\n")
	if len(lines) != 2 || lines[0] == "" || lines[1] == "" {
		return fmt.Errorf("git named no git directory and common directory (%q)", out)
	}
	abs := func(p string) string {
		if !filepath.IsAbs(p) {
			p = filepath.Join(r.repo, p)
		}
		return filepath.Clean(p)
	}
	gitDir, common := abs(lines[0]), abs(lines[1])
	// A linked worktree's .git is a file naming its git directory.
	// The git directory it names is read below, so a link there is not
	// followed.
	if fi, err := os.Lstat(filepath.Join(r.repo, ".git")); err == nil && !fi.IsDir() {
		if err := r.hashAt(r.repo, ".git", false); err != nil {
			return err
		}
	}
	for _, rel := range []string{"hooks", "info", "config", "config.worktree", "HEAD", "packed-refs", "refs"} {
		if err := r.hashIn(common, rel); err != nil {
			return err
		}
	}
	if err := r.readModules(common); err != nil {
		return err
	}
	if err := r.readWorktrees(common); err != nil {
		return err
	}
	if gitDir != common {
		for _, rel := range []string{"config.worktree", "HEAD"} {
			if err := r.hashIn(gitDir, rel); err != nil {
				return err
			}
		}
	}
	hooksPaths, err := gitutil.HooksPaths(r.repo)
	if err != nil {
		return err
	}
	fold := fsutil.CaseFoldingFS()
	realGitDir, realCommon := fsutil.RealExistingPath(gitDir), fsutil.RealExistingPath(common)
	for _, hp := range hooksPaths {
		// git runs a hook from where the value leads, so a link on the way
		// (the directory itself, or one above it) is resolved before the
		// directory is placed and read.
		dir := fsutil.RealExistingPath(abs(filepath.FromSlash(hp)))
		// Inside the working tree, and outside git's own directory, the
		// status listing already reads it.
		if fsutil.PathWithin(dir, r.realRepo, fold) && !fsutil.PathWithin(dir, realGitDir, fold) && !fsutil.PathWithin(dir, realCommon, fold) {
			continue
		}
		if err := r.hashIn(filepath.Dir(dir), filepath.Base(dir)); err != nil {
			return err
		}
	}
	return r.readReceipts()
}

// worktreeFiles are the files of a linked worktree's entry under the common
// directory's worktrees/ that git reads to place the worktree and run there:
// its HEAD, the common directory it uses (and so the configuration, the
// hooks and the refs), the working tree it belongs to (and so the place git
// lists it, whose local tier the pre-push gate reads), its own configuration,
// and its lock.
var worktreeFiles = []string{"HEAD", "commondir", "gitdir", "config.worktree", "locked"}

// readWorktrees records every entry of common's worktrees/ by its kind, so
// one made or removed is noticed, and each entry's worktreeFiles by mode,
// size and content hash. A link, there or as worktrees/ itself, is recorded
// by its target's text and read where it leads, as git reads it.
func (r *treeReader) readWorktrees(common string) error {
	dir := filepath.Join(common, "worktrees")
	fi, err := os.Lstat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if fi.Mode()&fs.ModeSymlink != 0 {
		if err := r.hashAt(common, "worktrees", false); err != nil {
			return err
		}
		dir = fsutil.RealExistingPath(dir)
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		at := filepath.Join(dir, e.Name())
		if e.Type()&fs.ModeSymlink != 0 {
			if err := r.hashAt(dir, e.Name(), false); err != nil {
				return err
			}
			at = fsutil.RealExistingPath(at)
		} else if err := r.put(r.keyOf(dir, e.Name()), "worktree "+e.Type().String()); err != nil {
			return err
		}
		if fi, err := os.Stat(at); err != nil || !fi.IsDir() {
			continue
		}
		for _, f := range worktreeFiles {
			if err := r.hashIn(at, f); err != nil {
				return err
			}
		}
	}
	return nil
}

// receiptsRel is where a checkout's push receipts stand, under each worktree
// (scripts/preflight-receipt.sh).
const receiptsRel = ".abcd/.work.local/preflight-receipts"

// readReceipts records the push receipts of every worktree git lists by
// mode, size and content hash: the pre-push gate takes a receipt for the
// commit being pushed from any of them, so one written into another
// worktree's local tier, or into a worktree the role made git list, would
// pass a push the gates never read.
func (r *treeReader) readReceipts() error {
	wts, err := gitutil.ListWorktrees(r.repo, maxStatusBytes)
	if err != nil {
		return err
	}
	for _, wt := range wts {
		if wt.Path == "" {
			continue
		}
		dir := filepath.Join(wt.Path, filepath.FromSlash(receiptsRel))
		if err := r.hashIn(filepath.Dir(dir), filepath.Base(dir)); err != nil {
			return err
		}
	}
	return nil
}

// readModules records the hooks and the configuration of every submodule's
// git directory under common's modules/. A directory there holding a HEAD, a
// config or a hooks entry is a submodule's git directory, and its own
// modules/ is read the same way; any other directory is the leading part of
// a submodule's name, which may hold a slash, and is read through. A link is
// never followed.
func (r *treeReader) readModules(common string) error {
	root, err := os.OpenRoot(common)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer root.Close()
	return r.readModulesIn(root.FS(), common, "modules")
}

func (r *treeReader) readModulesIn(fsys fs.FS, common, dir string) error {
	entries, err := fs.ReadDir(fsys, dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sub := path.Join(dir, e.Name())
		if !isGitDir(fsys, sub) {
			if err := r.readModulesIn(fsys, common, sub); err != nil {
				return err
			}
			continue
		}
		for _, rel := range []string{"hooks", "config"} {
			if err := r.hashIn(common, path.Join(sub, rel)); err != nil {
				return err
			}
		}
		if err := r.readModulesIn(fsys, common, path.Join(sub, "modules")); err != nil {
			return err
		}
	}
	return nil
}

// isGitDir reports whether dir holds a HEAD, a config or a hooks entry, as
// a submodule's git directory does.
func isGitDir(fsys fs.FS, dir string) bool {
	for _, n := range []string{"HEAD", "config", "hooks"} {
		if _, err := fs.Lstat(fsys, path.Join(dir, n)); err == nil {
			return true
		}
	}
	return false
}

// hashIn records rel under base, and every entry under it when it is a
// directory, by mode, size and content hash, keyed by its path relative to
// the repository when it is inside it and in full when not. A link is
// recorded by its target's text, and what it leads to is read too, once: git
// follows a link in its own directory or in a hooks directory, so a hooks
// directory, a configuration file or a hook that is a link (a
// dotfiles-managed one) runs or is read where it leads.
func (r *treeReader) hashIn(base, rel string) error {
	return r.hashAt(base, rel, true)
}

// hashAt is hashIn, following each link it meets only when follow is set,
// and never further than the link's own target.
func (r *treeReader) hashAt(base, rel string, follow bool) error {
	root, err := os.OpenRoot(base)
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		// Nothing can stand under a base that is absent or is not a
		// directory.
		return nil
	}
	if err != nil {
		return err
	}
	defer root.Close()
	fi, err := root.Lstat(rel)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		// A file, or a link standing where a directory is expected: recorded
		// as itself, never followed.
		h, err := contentState(root, rel)
		if err != nil {
			return err
		}
		if err := r.put(r.keyOf(base, rel), "git "+h); err != nil {
			return err
		}
		if follow && fi.Mode()&fs.ModeSymlink != 0 {
			return r.followLink(base, rel)
		}
		return nil
	}
	return fs.WalkDir(root.FS(), rel, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		h, err := contentState(root, p)
		if err != nil {
			return err
		}
		if err := r.put(r.keyOf(base, p), "git "+h); err != nil {
			return err
		}
		if follow && d.Type()&fs.ModeSymlink != 0 {
			return r.followLink(base, p)
		}
		return nil
	})
}

// followLink reads what the link rel under base leads to, every link on the
// way resolved, without following any link it finds there. A link leading
// nowhere reads as nothing.
func (r *treeReader) followLink(base, rel string) error {
	target := fsutil.RealExistingPath(filepath.Join(base, filepath.FromSlash(rel)))
	return r.hashAt(filepath.Dir(target), filepath.Base(target), false)
}

// keyOf is the key of rel under base: its path relative to the repository
// when it is inside it, spelled through the repository's path or its real
// one, and in full when not.
func (r *treeReader) keyOf(base, rel string) string {
	full := filepath.Join(base, filepath.FromSlash(rel))
	for _, repo := range []string{r.repo, r.realRepo} {
		if k, err := filepath.Rel(repo, full); err == nil && k != ".." && !strings.HasPrefix(k, ".."+string(filepath.Separator)) {
			return filepath.ToSlash(k)
		}
	}
	return filepath.ToSlash(full)
}

// contentState is rel's kind, mode, size and content hash: a link's target is
// hashed, never followed.
func contentState(root *os.Root, rel string) (string, error) {
	fi, err := root.Lstat(rel)
	if errors.Is(err, os.ErrNotExist) {
		return "absent", nil
	}
	if err != nil {
		return "", err
	}
	h := sha256.New()
	switch {
	case fi.Mode()&os.ModeSymlink != 0:
		target, err := root.Readlink(rel)
		if err != nil {
			return "", err
		}
		h.Write([]byte(target))
	case fi.Mode().IsRegular():
		f, err := root.Open(rel)
		if err != nil {
			return "", err
		}
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%v %d %s", fi.Mode(), fi.Size(), hex.EncodeToString(h.Sum(nil))), nil
}

// changedSince is every path whose state differs between before and after,
// sorted.
func changedSince(before, after treeState) []string {
	var out []string
	for p, s := range after {
		if before[p] != s {
			out = append(out, p)
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}
