// Package storepath names where a repository's embargo store lives
// (adr-2610120558328615). It stands apart from package embargo so a reader
// that must skip the store, the peers listing, can name it without importing
// the capture code embargo writes its private records through: capture imports
// site, site claim, and claim peers, so peers importing embargo is a cycle.
//
// The store is a linked worktree of the repository at
// worktrees/<key>/embargo in abcd's home, and it is found by that
// membership: whichever worktree of the repository git lists at a path of
// that shape is the store, whatever its key. The key is chosen once, when the
// store is made, from the root commits of the default branch; nothing that
// changes the roots afterwards moves the store or hides it: a merge of an
// unrelated history, origin's HEAD moved to another branch, a shallow clone
// deepened, a replace ref or a graft file. What does hide it is git losing
// the registration: the store's folder moved or deleted and pruned, or the
// checkout cloned afresh; a folder left at the place the store would be made
// is then refused, never taken for "no store".
package storepath

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// Name is the store's folder under its key's level of the worktree store.
const Name = "embargo"

// ErrWorktreesUnread is Locate's answer when git could not list the
// repository's worktrees, so whether a store is registered cannot be told.
var ErrWorktreesUnread = errors.New("the repository's worktrees could not be listed")

// ErrSeveralStores is Locate's answer when more than one worktree of the
// repository is an embargo store: which one holds the records cannot be
// told, so none is chosen.
var ErrSeveralStores = errors.New("more than one embargo store is a worktree of the repository")

// ErrRootsUnread is Locate's answer, with no store registered, when git
// could not list the default branch's root commits to key a new store on.
var ErrRootsUnread = errors.New("the root commits of the default branch could not be read")

// ErrNoRootCommit is Locate's answer, with no store registered, for a
// repository whose default branch has no root commit to key a store on.
var ErrNoRootCommit = errors.New("the default branch has no root commit to key the embargo store on")

// ErrPlaceTaken is Locate's answer, with no store registered, when something
// sits at the place the store would be made that is neither this
// repository's worktree nor shown to be another repository's, or the place
// cannot be examined: it may be a store whose registration git lost, so it
// is never answered as "no store".
var ErrPlaceTaken = errors.New("something that is not a store of this repository sits where its embargo store would be made")

// ErrNoFreePlace is Locate's answer, with no store registered, when the place
// under every root of the default branch holds another repository's store.
// This repository has no store; one cannot be made here without touching
// another's.
var ErrNoFreePlace = errors.New("every place the embargo store could be made holds another repository's store")

// Location is where a repository's embargo store is, or is to be made.
type Location struct {
	// Path is worktrees/<key>/embargo under abcd's home.
	Path string
	// Registered is true when Path is a worktree of the repository, so the
	// store exists as far as git knows; false when Path is where it is to be
	// made, and nothing is there.
	Registered bool
}

// Locate is the one rule for where the store of the repository at repoRoot
// is under home. Every reader and writer of the store asks here.
//
// Finding: every worktree of the repository (gitutil.EachWorktree) whose path
// is a store's (IsStorePath) is a store. Exactly one is the store, whatever
// its key; more than one is ErrSeveralStores; a listing git could not give is
// ErrWorktreesUnread, never "none".
//
// Making, when none is registered: the key is a root commit of the default
// branch (gitutil.DefaultBranchRoots), oldest first, so a repository with one
// root keeps the place it always had. A place holding another repository's
// live store is passed over and never touched; the first place where nothing
// is, is the answer; a place holding anything else, or that cannot be
// examined, is ErrPlaceTaken.
func Locate(home, repoRoot string) (Location, error) {
	var found []string
	err := gitutil.EachWorktree(repoRoot, func(w gitutil.Worktree) {
		if key := filepath.Base(filepath.Dir(w.Path)); !w.Bare && IsStorePath(home, w.Path) && !slices.Contains(found, key) {
			found = append(found, key)
		}
	})
	if err != nil {
		return Location{}, ErrWorktreesUnread
	}
	switch len(found) {
	case 0:
	case 1:
		return Location{Path: place(home, found[0]), Registered: true}, nil
	default:
		return Location{}, ErrSeveralStores
	}
	own, err := gitutil.RevParseAbsPath(repoRoot, "--git-common-dir")
	if err != nil {
		return Location{}, ErrWorktreesUnread
	}
	roots, err := gitutil.DefaultBranchRoots(repoRoot)
	if err != nil {
		return Location{}, ErrRootsUnread
	}
	if len(roots) == 0 {
		return Location{}, ErrNoRootCommit
	}
	for _, sha := range roots {
		p := place(home, sha)
		switch _, err := os.Lstat(p); {
		case errors.Is(err, fs.ErrNotExist):
			return Location{Path: p}, nil
		case err != nil:
			return Location{}, ErrPlaceTaken
		case !anotherRepositorysStore(p, own):
			return Location{}, ErrPlaceTaken
		}
	}
	return Location{}, ErrNoFreePlace
}

// place is the store's path under key.
func place(home, key string) string { return abcdhome.Path(home, "worktrees", key, Name) }

// anotherRepositorysStore reports whether p is a live worktree of a
// repository other than the one whose common dir is own: git in p names p
// as its working tree and another common dir, and that repository lists p
// among its worktrees. Anything less (a plain folder, a worktree whose
// repository is gone) is not shown to be another's.
func anotherRepositorysStore(p, own string) bool {
	if top, err := gitutil.Toplevel(p); err != nil || resolve(top) != resolve(p) {
		return false
	}
	common, err := gitutil.RevParseAbsPath(p, "--git-common-dir")
	if err != nil || resolve(common) == resolve(own) {
		return false
	}
	listed := false
	err = gitutil.EachWorktree(p, func(w gitutil.Worktree) {
		if resolve(w.Path) == resolve(p) {
			listed = true
		}
	})
	return err == nil && listed
}

// IsStorePath reports whether p is the path of an embargo store under home:
// worktrees/<key>/embargo in abcd's home, the key a full object name (forty
// or sixty-four lower-case hex digits), in p's spelling or with its links
// resolved, and the same for home.
func IsStorePath(home, p string) bool {
	if home == "" || !shaped(p) {
		return false
	}
	base := abcdhome.Path(home, "worktrees")
	dir := filepath.Dir(filepath.Dir(filepath.Clean(p)))
	return dir == base || resolve(dir) == resolve(base)
}

// HasStoreShape reports whether p, in its spelling or with its links
// resolved, ends in .abcd.noindex/worktrees/<key>/embargo under any home: a
// reader that must never open a store whoever's home it is in, or when no
// home can be named, skips every path of that shape.
func HasStoreShape(p string) bool {
	for _, q := range []string{filepath.Clean(p), resolve(p)} {
		if !shaped(q) {
			continue
		}
		worktrees := filepath.Dir(filepath.Dir(q))
		if filepath.Base(worktrees) == "worktrees" && filepath.Base(filepath.Dir(worktrees)) == abcdhome.Rel() {
			return true
		}
	}
	return false
}

// shaped reports whether p ends in <key>/embargo with key a full object name.
func shaped(p string) bool {
	p = filepath.Clean(p)
	return filepath.Base(p) == Name && gitutil.IsFullSHA(filepath.Base(filepath.Dir(p)))
}

// resolve is filepath.EvalSymlinks over the longest existing prefix of p, with
// the rest joined on: a path not made yet, or a store's folder deleted while
// git still lists it, still resolves through the links above it (macOS's
// /var is one).
func resolve(p string) string {
	p = filepath.Clean(p)
	var rest []string
	for {
		if r, err := filepath.EvalSymlinks(p); err == nil {
			return filepath.Join(append([]string{r}, rest...)...)
		}
		parent := filepath.Dir(p)
		if parent == p {
			return filepath.Join(append([]string{p}, rest...)...)
		}
		rest = append([]string{filepath.Base(p)}, rest...)
		p = parent
	}
}
