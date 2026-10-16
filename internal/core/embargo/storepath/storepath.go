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
//
// A worktree of the repository at that shape under some other home (abcd
// once run under sudo, a sandbox or an aliased HOME) is the store this
// repository registered, so it is refused (ErrStoreUnderAnotherHome), never
// read as "no store" and never answered with a second store here.
//
// Which stores the outbound check reads is a wider question (Reachable):
// every place a store sits under this home, whatever repository it belongs
// to and whether or not git still lists it, found by listing the worktree
// store rather than by any root commit, so a second clone whose roots miss
// the store's key (a shallow clone, rewritten history) still reads it, and a
// store whose checkout was deleted still has its records read. Those that
// are not this checkout's own are read, and never written.
package storepath

import (
	"errors"
	"fmt"
	"io"
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

// ErrStoreUnderAnotherHome is the answer of Locate and Reachable when git
// lists a worktree of the repository at the shape of a store under some
// other home: the store this repository registered lives there, so this home
// would read as holding nothing, and a store made here would be a second one.
var ErrStoreUnderAnotherHome = errors.New("the embargo store this repository registered lives under another home")

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

// ErrPlaceUnread is Reachable's answer when the worktree store under the
// home cannot be listed, or a place worktrees/<key>/embargo in it cannot be
// examined: it may be a store whose records the check would otherwise miss,
// so it is never skipped.
var ErrPlaceUnread = errors.New("a place where an embargo store under this home would be cannot be examined")

// ErrNoFreePlace is Locate's answer, with no store registered, when the place
// under every root of the default branch holds another repository's store.
// This repository has no store; one cannot be made here without touching
// another's.
var ErrNoFreePlace = errors.New("every place the embargo store could be made holds another repository's store")

// ErrPlaceHeldByAClone is ErrNoFreePlace when each store sitting at those
// places is a clone's of the same project, under a root of that clone's own
// default branch: moving it under its own root could not free the place.
var ErrPlaceHeldByAClone = fmt.Errorf("%w, each a clone's of the same project", ErrNoFreePlace)

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
// its key; more than one is ErrSeveralStores; one under another home is
// ErrStoreUnderAnotherHome; a listing git could not give is
// ErrWorktreesUnread, never "none".
//
// Making, when none is registered: the place under each root of the default
// branch (placesUnderRoots), oldest first, so a repository with one root
// keeps the place it always had. A place holding another repository's live
// store is passed over and never touched; the first free place is the
// answer; a place holding anything else, or that cannot be examined, is
// ErrPlaceTaken; with every place another's, ErrNoFreePlace, or
// ErrPlaceHeldByAClone when each of those stores sits under a root of its
// own repository's default branch (keyedOnOwnRoot).
func Locate(home, repoRoot string) (Location, error) {
	if p, err := registeredStore(home, repoRoot); err != nil || p != "" {
		return Location{Path: p, Registered: p != ""}, err
	}
	loc, found, clones := Location{}, ErrNoFreePlace, true
	err := placesUnderRoots(home, repoRoot, func(p string, k placeKind) bool {
		switch k {
		case placeAnothers:
			clones = clones && keyedOnOwnRoot(p)
			return true
		case placeFree:
			loc, found = Location{Path: p}, nil
		default:
			found = ErrPlaceTaken
		}
		return false
	})
	switch {
	case err != nil:
		return Location{}, err
	case found == ErrNoFreePlace && clones:
		return loc, ErrPlaceHeldByAClone
	}
	return loc, found
}

// keyedOnOwnRoot reports whether the store at p sits under a root of its
// own repository's default branch, as a clone of the same project's store
// does under the root they share; false when those roots cannot be read.
func keyedOnOwnRoot(p string) bool {
	roots, err := gitutil.DefaultBranchRoots(p)
	return err == nil && slices.Contains(roots, filepath.Base(filepath.Dir(p)))
}

// registeredStore is the path of the one worktree of the repository at
// repoRoot that is a store under home, "" when none is. A worktree of the
// store's shape under any other home counts as a store too: one alone there
// is ErrStoreUnderAnotherHome, and with another it is one of several.
func registeredStore(home, repoRoot string) (string, error) {
	var found []string
	here, elsewhere := "", false
	err := gitutil.EachWorktree(repoRoot, func(w gitutil.Worktree) {
		mine := IsStorePath(home, w.Path)
		if w.Bare || (!mine && !HasStoreShape(w.Path)) {
			return
		}
		if r := Resolve(w.Path); !slices.Contains(found, r) {
			found = append(found, r)
			if mine {
				here = place(home, filepath.Base(filepath.Dir(filepath.Clean(w.Path))))
			} else {
				elsewhere = true
			}
		}
	})
	switch {
	case err != nil:
		return "", ErrWorktreesUnread
	case len(found) > 1:
		return "", ErrSeveralStores
	case elsewhere:
		return "", ErrStoreUnderAnotherHome
	}
	return here, nil
}

// placeKind is what sits at a place the store could be made.
type placeKind int

const (
	placeFree     placeKind = iota // nothing
	placeAnothers                  // a live worktree of another repository
	placeTaken                     // anything else, or the place cannot be examined
)

// placesUnderRoots hands fn the place under each root of the default branch
// (gitutil.DefaultBranchRoots, read as stored), oldest first, with what sits
// there, until fn returns false. A root listing git could not give is
// ErrRootsUnread, and no root at all ErrNoRootCommit.
func placesUnderRoots(home, repoRoot string, fn func(p string, k placeKind) bool) error {
	own, err := gitutil.RevParseAbsPath(repoRoot, "--git-common-dir")
	if err != nil {
		return ErrWorktreesUnread
	}
	roots, err := gitutil.DefaultBranchRoots(repoRoot)
	if err != nil {
		return ErrRootsUnread
	}
	if len(roots) == 0 {
		return ErrNoRootCommit
	}
	for _, sha := range roots {
		if p := place(home, sha); !fn(p, classify(p, own)) {
			break
		}
	}
	return nil
}

// classify is what sits at p, for the repository whose common dir is own:
// nothing; a live worktree (liveWorktree) of another repository; or
// anything else, which is taken: a plain folder, a worktree whose
// repository is gone, a place that cannot be examined, and a live worktree
// of this repository that registeredStore did not find there.
func classify(p, own string) placeKind {
	switch _, err := os.Lstat(p); {
	case errors.Is(err, fs.ErrNotExist):
		return placeFree
	case err != nil, !liveWorktree(p):
		return placeTaken
	}
	common, err := gitutil.RevParseAbsPath(p, "--git-common-dir")
	if err != nil || Resolve(common) == Resolve(own) {
		return placeTaken
	}
	return placeAnothers
}

// Reach is every store the outbound check of a repository reads.
type Reach struct {
	// Own is the repository's registered store (Locate), "" when none is.
	Own string
	// Others is every other place a store sits under the home, in path
	// order: another clone's of the same project, or another project's,
	// live or not.
	Others []string
	// NotLive is how many of Others are no live worktree (liveWorktree):
	// a store whose repository or checkout is gone, a copy git does not
	// list, or a plain folder.
	NotLive int
}

// Reachable is the one rule for which stores the outbound check of the
// repository at repoRoot reads, and it makes and changes nothing.
//
// Own is found as Locate finds the store, with Locate's refusals for the
// worktree listing, several stores and a store under another home. Others
// are read from the worktree store under home (storePlaces): every place
// worktrees/<key>/embargo, the key a full object name, where anything sits,
// whoever's it is and whether or not it is a live worktree, but Own. Whether
// a store is live decides nothing of what is read, since what is held comes
// from its state file alone; NotLive only counts those that are not. A place
// that cannot be examined, and a worktree store that cannot be listed, are
// ErrPlaceUnread, never skipped. No root commit is read: the roots decide
// only where a new store is made.
//
// Its one known limit: a store under another home that this repository
// never registered (another clone's, made under sudo or a sandbox) is not
// read.
func Reachable(home, repoRoot string) (Reach, error) {
	own, err := registeredStore(home, repoRoot)
	if err != nil {
		return Reach{}, err
	}
	places, err := storePlaces(home)
	if err != nil {
		return Reach{}, err
	}
	r := Reach{Own: own}
	for _, p := range places {
		if own != "" && Resolve(p) == Resolve(own) {
			continue
		}
		r.Others = append(r.Others, p)
		if !liveWorktree(p) {
			r.NotLive++
		}
	}
	return r, nil
}

// maxKeyName bounds an entry name of the worktree store that storePlaces
// looks under: a key is a full object name, at most sixty-four characters,
// so a longer name is passed over unread.
const maxKeyName = 64

// storePlaces is every place under home where a store sits, in path order:
// the worktree store's entries are read a batch at a time, and each named by
// a full object name is a key whose place worktrees/<key>/embargo is
// examined. Nothing there is passed over; anything there is a store; a place
// that cannot be examined and a worktree store that cannot be listed are
// ErrPlaceUnread. No worktree store is no store.
func storePlaces(home string) ([]string, error) {
	dir, err := os.Open(abcdhome.Path(home, "worktrees"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, ErrPlaceUnread
	}
	defer dir.Close()
	var stores []string
	for {
		names, err := dir.Readdirnames(256)
		for _, name := range names {
			if len(name) > maxKeyName || !gitutil.IsFullSHA(name) {
				continue
			}
			p := place(home, name)
			switch _, err := os.Lstat(p); {
			case errors.Is(err, fs.ErrNotExist):
				continue
			case err != nil:
				return nil, ErrPlaceUnread
			}
			stores = append(stores, p)
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, ErrPlaceUnread
		}
	}
	slices.Sort(stores)
	return stores, nil
}

// place is the store's path under key.
func place(home, key string) string { return abcdhome.Path(home, "worktrees", key, Name) }

// liveWorktree reports whether p is a live worktree of some repository: git
// in p names p as its working tree, and that repository lists p among its
// worktrees.
func liveWorktree(p string) bool {
	if top, err := gitutil.Toplevel(p); err != nil || Resolve(top) != Resolve(p) {
		return false
	}
	listed := false
	err := gitutil.EachWorktree(p, func(w gitutil.Worktree) {
		if Resolve(w.Path) == Resolve(p) {
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
	return dir == base || Resolve(dir) == Resolve(base)
}

// HasStoreShape reports whether p, in its spelling or with its links
// resolved, ends in .abcd.noindex/worktrees/<key>/embargo under any home: a
// reader that must never open a store whoever's home it is in, or when no
// home can be named, skips every path of that shape.
func HasStoreShape(p string) bool {
	for _, q := range []string{filepath.Clean(p), Resolve(p)} {
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

// Resolve is filepath.EvalSymlinks over the longest existing prefix of p, with
// the rest joined on: a path not made yet, or a store's folder deleted while
// git still lists it, still resolves through the links above it (macOS's
// /var is one). It is the one spelling every comparison of store and
// checkout paths makes.
func Resolve(p string) string {
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

// PathForTest is Locate's place alone, for tests in other packages that make
// or read the store: where it is or is to be made, and ok false on any
// refusal Locate gives.
func PathForTest(home, repoRoot string) (p string, ok bool) {
	loc, err := Locate(home, repoRoot)
	if err != nil {
		return "", false
	}
	return loc.Path, true
}
