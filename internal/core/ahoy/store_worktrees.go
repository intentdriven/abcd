package ahoy

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// StoreWorktreeUnlinkedGapID names a worktree in abcd's worktree store whose
// repository does not link back to it (iss-2610050728100598). git records a
// worktree's place in absolute form at both ends: the worktree's .git file
// names its admin folder in the repository, and that folder's gitdir file
// names the worktree. Renaming abcd's home moves every worktree without
// telling the repository, so until `git worktree repair` runs in it, the
// repository lists the worktree as prunable and a `git worktree prune` there
// drops its entry. The gap is one per such worktree, report-only: abcd never
// runs the repair, which writes into the repository's .git, outside its home
// (the technical facilitator's ruling of 2026-10-04 on iss-2610040147016103).
const StoreWorktreeUnlinkedGapID = "store.worktree_unlinked"

// The walk's bounds. A worktree ends the walk below it, so the folders visited
// are the store's own levels (<root-sha>, a branch prefix such as docs/) and
// the worktrees' tops, never a worktree's files. The bounds keep a store that
// holds something else entirely from turning a read-only status pass into a
// walk of the disk.
const (
	// storeWorktreeMaxDepth is how many levels below worktrees/ a worktree is
	// looked for: <root-sha>/<name> is two, and a branch name may add a few.
	storeWorktreeMaxDepth = 8
	// storeWorktreeMaxDirs caps the folders read in one pass.
	storeWorktreeMaxDirs = 4096
	// storeWorktreeMaxRead caps a .git or gitdir file read; each holds one path.
	storeWorktreeMaxRead = 4096
)

// detectStoreWorktrees reports each worktree in ~/.abcd.noindex/worktrees,
// found by its .git file at any depth within the bounds, whose repository's
// back-link does not name it. It reads only. A store behind a symlinked level
// is not walked, as every other reader of the home refuses one; a .git file
// that names no admin folder of a linked worktree (a submodule, a file left
// in an emptied folder) is not abcd's to judge and is passed over.
func detectStoreWorktrees() []Gap {
	home, err := os.UserHomeDir()
	if err != nil || !filepath.IsAbs(home) {
		return nil
	}
	rel := abcdhome.Rel("worktrees")
	if fsutil.HomeScopeLink(home, rel) != nil {
		return nil
	}
	store := filepath.Join(home, filepath.FromSlash(rel))
	if fi, err := os.Lstat(store); err != nil || !fi.IsDir() {
		return nil
	}
	var gaps []Gap
	type level struct {
		dir   string
		depth int
	}
	queue := []level{{store, 0}}
	visited := 0
	for len(queue) > 0 && visited < storeWorktreeMaxDirs {
		cur := queue[0]
		queue = queue[1:]
		visited++
		entries, err := os.ReadDir(cur.dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || e.Name() == ".git" {
				continue // a file, a link (never followed) or a repository's own folder
			}
			dir := filepath.Join(cur.dir, e.Name())
			gitFile, err := os.Lstat(filepath.Join(dir, ".git"))
			switch {
			case err == nil && gitFile.Mode().IsRegular():
				if g, ok := unlinkedWorktreeGap(home, dir); ok {
					gaps = append(gaps, g)
				}
			case err == nil:
				// A .git folder or link: a clone, not a linked worktree.
			case cur.depth+1 < storeWorktreeMaxDepth:
				queue = append(queue, level{dir, cur.depth + 1})
			}
		}
	}
	return gaps
}

// unlinkedWorktreeGap judges the worktree at wt, whose .git is a regular file:
// it names the admin folder, and the admin folder's gitdir file must name this
// worktree's .git back. ok is false when the link holds or when the .git file
// names no linked worktree's admin folder.
func unlinkedWorktreeGap(home, wt string) (Gap, bool) {
	admin, ok := readPathFile(filepath.Join(wt, ".git"), "gitdir:", wt)
	if !ok {
		return Gap{}, false
	}
	// A linked worktree's admin folder carries commondir; a submodule's does not.
	if fi, err := os.Stat(filepath.Join(admin, "commondir")); err != nil || !fi.Mode().IsRegular() {
		return Gap{}, false
	}
	own := filepath.Join(wt, ".git")
	back, ok := readPathFile(filepath.Join(admin, "gitdir"), "", admin)
	if ok && samePath(back, own) {
		return Gap{}, false
	}
	shown := displayPath(wt)
	why := "its repository holds no record of where it is"
	if ok {
		if filepath.Base(back) == ".git" {
			back = filepath.Dir(back)
		}
		why = "its repository records it at " + displayPath(back)
	}
	relWT, err := filepath.Rel(home, wt)
	if err != nil {
		return Gap{}, false
	}
	return Gap{
		ID: StoreWorktreeUnlinkedGapID, Category: UserState, Scope: "machine",
		Title: "worktree not linked back from its repository",
		Detail: termsafe.Sanitize(shown + ": " + why + ", so git lists it as prunable and a `git worktree prune` there would drop its entry."),
		FixHint: termsafe.Sanitize("Run `" + abcdhome.WorktreeRepairCommand(filepath.ToSlash(relWT)) + "`; abcd only reports it."),
	}, true
}

// readPathFile reads the one path a git link file holds, after prefix,
// resolved against base when git wrote it relative. ok is false for a file
// that cannot be read, is larger than a path, or holds no path.
func readPathFile(p, prefix, base string) (string, bool) {
	f, err := os.Open(p)
	if err != nil {
		return "", false
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, storeWorktreeMaxRead+1))
	if err != nil || len(b) > storeWorktreeMaxRead {
		return "", false
	}
	line, _, _ := bytes.Cut(b, []byte("\n"))
	s := string(line)
	if prefix != "" {
		if !strings.HasPrefix(s, prefix) {
			return "", false
		}
		s = s[len(prefix):]
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	if !filepath.IsAbs(s) {
		s = filepath.Join(base, s)
	}
	return s, true
}

// samePath reports whether a and b name one file, so a link written through
// /tmp and read through /private/tmp still holds.
func samePath(a, b string) bool {
	fa, err := os.Stat(a)
	if err != nil {
		return false
	}
	fb, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(fa, fb)
}
