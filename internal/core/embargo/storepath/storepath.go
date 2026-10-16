// Package storepath names where a repository's embargo store lives
// (adr-2610120558328615). It stands apart from package embargo so a reader
// that must skip the store, the peers listing, can name it without importing
// the capture code embargo writes its private records through: capture imports
// site, site claim, and claim peers, so peers importing embargo is a cycle.
package storepath

import (
	"errors"
	"io/fs"
	"os"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// Name is the store's folder under the repository's level of the worktree
// store.
const Name = "embargo"

// ErrNoRootCommit is Locate's answer for a repository whose default branch
// has no root commit to key the store on.
var ErrNoRootCommit = errors.New("the default branch has no root commit to key the embargo store on")

// ErrRootsUnread is Locate's answer when git could not list the default
// branch's root commits.
var ErrRootsUnread = errors.New("the root commits of the default branch could not be read")

// ErrSeveralStores is Locate's answer when an embargo store sits under more
// than one root commit of the default branch: which of them holds the records
// cannot be told, so none is chosen.
var ErrSeveralStores = errors.New("an embargo store sits under more than one root commit of the default branch")

// Location is where a repository's embargo store is, or is to be made.
type Location struct {
	// Path is worktrees/<root-sha>/embargo under abcd's home.
	Path string
	// Found is every store Locate found, one per root that has one: empty
	// when there is none yet, Path alone when there is one, and with
	// ErrSeveralStores every one of them.
	Found []string
}

// Locate is the one rule for where the store of the repository at repoRoot
// lives under home: worktrees/<root-sha>/embargo in abcd's home, keyed on a
// root commit so every worktree of the repository finds the same store
// (adr-2610031751065746). Every reader and writer of the store, and every
// reader that must skip it, asks here.
//
// The roots are those of origin's default branch, read as stored
// (gitutil.DefaultBranchRoots): the branch every checkout shares, so HEAD on
// an orphan branch or a replace ref or graft file cannot move the store. A
// branch can hold several roots, and which one `rev-list` meets first changes
// when a merge brings in an unrelated history, so the store is looked for
// under each:
//
//   - under exactly one: that is the store, whichever root it is keyed on;
//   - under none: the store is to be made under the oldest root (the first
//     DefaultBranchRoots lists), so a repository with one root keeps the
//     place it always had;
//   - under more than one: ErrSeveralStores, with every one in Found.
//
// A store's place that cannot be examined counts as found: an answer of
// "no store" is what lets a check pass, so it is given only when no place
// exists.
func Locate(home, repoRoot string) (Location, error) {
	roots, err := gitutil.DefaultBranchRoots(repoRoot)
	if err != nil {
		return Location{}, ErrRootsUnread
	}
	if len(roots) == 0 {
		return Location{}, ErrNoRootCommit
	}
	var loc Location
	for _, sha := range roots {
		p := abcdhome.Path(home, "worktrees", sha, Name)
		if _, err := os.Lstat(p); errors.Is(err, fs.ErrNotExist) {
			continue
		}
		loc.Found = append(loc.Found, p)
		loc.Path = p
	}
	switch len(loc.Found) {
	case 0:
		return Location{Path: abcdhome.Path(home, "worktrees", roots[0], Name)}, nil
	case 1:
		return loc, nil
	default:
		return Location{Found: loc.Found}, ErrSeveralStores
	}
}

// Path is Locate's place alone: where the store is or is to be made, and ok
// false on any refusal Locate gives.
func Path(home, repoRoot string) (p string, ok bool) {
	loc, err := Locate(home, repoRoot)
	if err != nil {
		return "", false
	}
	return loc.Path, true
}
