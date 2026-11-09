package gitutil

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Held is every local branch a worktree holds that git's worktree listing does
// not name: git lists a worktree whose rebase or bisect stopped as detached,
// yet counts the branch the rebase or bisect started from, and each branch a
// rebase's --update-refs will rewrite, as in use there. It is the one rule
// every caller asks before it deletes a branch, moves one, or counts one as
// checked out, alongside the branches ListWorktrees names.
type Held struct {
	// Branches maps each branch a stopped rebase or bisect started from, by
	// short name, to the git dir whose state names it: the common dir for the
	// main checkout, or <common>/worktrees/<id> for a linked worktree. These
	// are the branches `git switch` and `git worktree add` also refuse.
	Branches map[string]string
	// Rewrites maps each branch a stopped rebase's --update-refs list will
	// rewrite to that git dir in the same way: `git branch -d` and `git
	// branch -f` refuse these too, though `git switch` does not.
	Rewrites map[string]string
	// Unreadable is the first state file, or the worktrees directory, that is
	// there and could not be read through fsutil.ReadGuarded (a symlink, not a
	// regular file, past its cap) or that git would not have written; "" when
	// every one was read. The branch it names is unknown, so a caller about to
	// act counts every branch as held.
	Unreadable string
	// Common is the common dir the branches were read under, the git dir
	// Branches and Rewrites name for the main checkout.
	Common string
}

// Checkout is the checkout a git dir h names belongs to, for a message to
// point the person at: the main checkout (wts' first entry, which git lists
// first, unless it is bare) for the common dir itself, and for a linked worktree's
// <common>/worktrees/<id> the checkout its gitdir file names (that checkout's
// .git, relative to the admin dir when worktree.useRelativePaths wrote it),
// since git picks the id when the worktree is added and it can differ from the
// checkout's folder (a second "look" gets look1). A relative gitdir is joined
// to dir's real path, as git writes and reads it, so a common dir reached
// through a symlink still names the checkout. The path is given as wts lists
// it when an entry is that checkout. dir itself when it is neither, when the
// gitdir file cannot be read, or when its text holds a rune termsafe.Sanitize
// would mask (a control character, a line break, a bidi or invisible rune):
// that text names no folder a person could open, and the answer is printed.
// Every answer is passed through termsafe.Sanitize, which leaves a clean path
// as it is: dir and the common dir end in names read from disk unchecked (an
// admin dir is any folder under <common>/worktrees), so a line break or an
// escape in one is masked rather than printed.
func (h Held) Checkout(dir string, wts []Worktree) string {
	return termsafe.Sanitize(h.checkout(dir, wts))
}

func (h Held) checkout(dir string, wts []Worktree) string {
	if dir == h.Common {
		if len(wts) > 0 && !wts[0].Bare {
			return wts[0].Path
		}
		return dir
	}
	b, err := fsutil.ReadGuarded(filepath.Join(dir, "gitdir"), heldStateCap)
	if err != nil {
		return dir
	}
	dotGit := strings.TrimRight(string(b), "\r\n")
	if dotGit == "" || termsafe.Sanitize(dotGit) != dotGit {
		return dir
	}
	if !filepath.IsAbs(dotGit) {
		dotGit = filepath.Join(fsutil.RealExistingPath(dir), dotGit)
	}
	checkout := filepath.Dir(filepath.Clean(dotGit))
	real := fsutil.RealExistingPath(checkout)
	for _, wt := range wts {
		if !wt.Bare && fsutil.RealExistingPath(wt.Path) == real {
			return wt.Path
		}
	}
	return checkout
}

// heldStateCap bounds one rebase or bisect state file that names one branch;
// heldUpdateRefsCap bounds a rebase's update-refs list, three lines a branch.
const (
	heldStateCap      = 4 << 10
	heldUpdateRefsCap = 256 << 10
)

// heldStateFiles are the files git reads, in a worktree's own git dir, for the
// branch a stopped rebase or bisect started from: wt-status.c's
// wt_status_check_rebase and wt_status_check_bisect, which worktree.c's
// is_worktree_being_rebased and is_worktree_being_bisected ask, so that
// `git branch -d`, `git switch` and `git worktree add` refuse that branch as
// in use there while the worktree's HEAD is detached.
var heldStateFiles = []string{"rebase-apply/head-name", "rebase-merge/head-name", "BISECT_START"}

// heldUpdateRefs is the list a rebase run with --update-refs keeps of the
// branches it will rewrite (sequencer.c's rebase_path_update_refs), which
// branch.c's prepare_checked_out_branches reads so that `git branch -d` and
// `git branch -f` refuse each as used by that worktree.
const heldUpdateRefs = "rebase-merge/update-refs"

// HeldBranches reads every branch a rebase or a bisect stopped in some
// worktree of the repository whose common dir is common holds: the main
// checkout's git dir (common itself) and each linked worktree's
// (common/worktrees/<id>, the entries `git worktree list` lists).
func HeldBranches(common string) Held {
	h := Held{Branches: map[string]string{}, Rewrites: map[string]string{}, Common: common}
	unreadable := func(p string) {
		if h.Unreadable == "" {
			h.Unreadable = p
		}
	}
	dirs := []string{common}
	wtDir := filepath.Join(common, "worktrees")
	entries, err := os.ReadDir(wtDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		unreadable(wtDir)
	}
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(wtDir, e.Name()))
		}
	}
	hold := func(in map[string]string, branch, dir string) {
		if _, ok := in[branch]; branch != "" && !ok {
			in[branch] = dir
		}
	}
	for _, dir := range dirs {
		for _, name := range heldStateFiles {
			p := filepath.Join(dir, filepath.FromSlash(name))
			b, err := fsutil.ReadGuarded(p, heldStateCap)
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if err != nil {
				unreadable(p)
				continue
			}
			hold(h.Branches, stateBranch(string(b)), dir)
		}
		p := filepath.Join(dir, filepath.FromSlash(heldUpdateRefs))
		b, err := fsutil.ReadGuarded(p, heldUpdateRefsCap)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		refs, ok := updateRefs(string(b))
		if err != nil || !ok {
			unreadable(p)
			continue
		}
		for _, ref := range refs {
			if branch, ok := strings.CutPrefix(ref, "refs/heads/"); ok {
				hold(h.Rewrites, branch, dir)
			}
		}
	}
	return h
}

// stateBranch is the local branch a state file names, read as git's
// wt-status.c get_branch reads it: trailing newlines dropped, a refs/heads/
// name shortened, and no branch for another ref, an object name (a bisect
// started detached) or rebase's "detached HEAD"; any other text is a bisect's
// short branch name.
func stateBranch(s string) string {
	s = strings.TrimRight(s, "\n")
	if b, ok := strings.CutPrefix(s, "refs/heads/"); ok {
		return b
	}
	if s == "" || s == "detached HEAD" || strings.HasPrefix(s, "refs/") || objectName(s) {
		return ""
	}
	return s
}

// updateRefs is the refs an update-refs file lists, read as sequencer.c's
// sequencer_get_update_refs_state reads it: records of three lines, the ref
// then its object names before and after, each line ended by a newline (a
// carriage return before it dropped, as strbuf_getline drops it). ok is false
// for a file git would call invalid: a record cut short, or a line where an
// object name belongs that does not open with one.
func updateRefs(s string) (refs []string, ok bool) {
	if s == "" {
		return nil, true
	}
	lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	if len(lines)%3 != 0 {
		return nil, false
	}
	for i := 0; i < len(lines); i += 3 {
		if !objectName(lines[i+1]) || !objectName(lines[i+2]) {
			return nil, false
		}
		refs = append(refs, strings.TrimSuffix(lines[i], "\r"))
	}
	return refs, true
}

// objectName reports whether s opens with a full hex object name, as git's
// get_oid_hex reads one (SHA-1's 40 digits; a SHA-256 name opens with 40 too).
func objectName(s string) bool {
	if len(s) < 40 {
		return false
	}
	for _, c := range s[:40] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return true
}
