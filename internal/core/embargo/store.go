// Package embargo keeps private security advisories private until they are
// published (itd-2610112239022766, spc-2610120557432880). Its records live in
// the embargo store, a worktree in the machine-scoped worktree store that is
// created detached and never committed, and every sub-verb verifies the store
// before it reads or writes anything (adr-2610120558328615, decisions 1 and
// 2). This file is the store; state.go is its state file.
package embargo

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/localtier"
)

// StoreName is the store's folder under the repository's level of the
// worktree store.
const StoreName = storepath.Name

// ledgerRel is where a private record sits inside the store: the ledger's own
// layout, so the capture and resolve code read it unchanged.
const ledgerRel = recordid.IssuesRelDir

// localTierRel is the local tier inside the store, where its state lives
// (StateRel). Every abcd repository ignores it, and it holds only files abcd
// writes.
const localTierRel = ".abcd/.work.local"

// localTierSpec is the pathspec naming the local tier under every spelling
// of its case.
const localTierSpec = ":(icase,literal)" + localTierRel

// maxListing caps the store's ls-files answer.
const maxListing = 16 << 20

// The modes of the store's folder and of each private record in it: this
// account's alone. git makes a worktree at the umask (0755 by default), and
// capture writes a record at 0644, either of which lets every account in the
// owner's group, or on the machine, read the advisory's text.
const (
	storeRootMode os.FileMode = 0o700
	recordMode    os.FileMode = 0o600
)

// RefusalError is a refusal with nothing written: the reason, and the one
// thing the person can do about it. The surfaces exit 2 on it. Store is the
// store whose reading refused, when one did; the message never names it, and
// a caller acting for another verb may.
type RefusalError struct {
	Reason string
	Remedy string
	Store  string
}

// atStore is err with Store set to root when it is a refusal; any other
// error is returned as it is.
func atStore(err error, root string) error {
	var r *RefusalError
	if !errors.As(err, &r) {
		return err
	}
	c := *r
	c.Store = root
	return &c
}

func (e *RefusalError) Error() string {
	if e.Remedy == "" {
		return "embargo: " + e.Reason
	}
	return "embargo: " + e.Reason + " (" + e.Remedy + ")"
}

func refuse(reason, remedy string) error { return &RefusalError{Reason: reason, Remedy: remedy} }

// untracked refuses the store file at rel, under the store worktree root, when
// git tracks it, or cannot say it does not: the store is a worktree at the
// default branch's tip, so a file a commit carries there was written by
// whoever made the commit, not by this account's abcd (localtier.Untracked).
func untracked(root, rel, what string) error {
	err := localtier.Untracked(root, rel)
	var r *localtier.Refused
	if errors.As(err, &r) {
		return refuse(what+" is refused: "+r.Reason(), r.Remedy())
	}
	return err
}

// trackedThere reports whether git tracks the store file at rel under the
// store worktree root. Another store is read for the ids it holds back, so
// one git cannot answer for (its checkout deleted, say) is read as before:
// refusing it would hold none of them back. What that accepts is a state a
// commit replaced being read while git cannot answer for its store, which
// nothing in a commit can bring about.
func trackedThere(root, rel string) bool {
	var r *localtier.Refused
	return errors.As(localtier.Untracked(root, rel), &r) && r.Cause == nil
}

// Store is the embargo store of one repository: Root is the store worktree,
// Checkout the working-tree root of the checkout it belongs to.
type Store struct {
	Root     string `json:"root"`
	Checkout string `json:"checkout"`
}

// locate is where the store of the repository at top is, or is to be made,
// under home: storepath.Locate's answer, the one rule every reader and
// writer of the store shares, with each of its refusals worded for the
// person. Its refusals name neither a store nor a key. A caller that only
// reads takes isNoFreePlace for "no store".
func locate(home, top string) (storepath.Location, error) {
	loc, err := storepath.Locate(home, top)
	return loc, worded(err)
}

// reachable is every store the outbound check of the repository at top
// reads under home (storepath.Reachable: its own and every other store
// there, live or not), its refusals worded as locate words them.
func reachable(home, top string) (storepath.Reach, error) {
	r, err := storepath.Reachable(home, top)
	return r, worded(err)
}

// worded is a storepath refusal worded for the person, naming neither a
// store nor a key; nil stays nil.
func worded(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, storepath.ErrWorktreesUnread):
		return refuse("git could not list this repository's worktrees, so whether it has an embargo store cannot be told",
			"settle what `git worktree list --porcelain` reports, then run it again")
	case errors.Is(err, storepath.ErrStoreUnderAnotherHome):
		return refuse("the embargo store this repository registered lives under another home than the one abcd runs with, so this home would read as holding nothing under embargo",
			"run abcd with that home, the HOME it had when the store was made; if that home no longer exists, `git worktree prune` removes the stale registration")
	case errors.Is(err, storepath.ErrSeveralStores):
		return refuse("more than one embargo store is a worktree of this repository, so which one holds the records cannot be told",
			"inspect each with `git worktree list`, keep the one holding the records, remove the other with `git worktree remove <path>`, then run it again")
	case errors.Is(err, storepath.ErrRootsUnread):
		return refuse("git could not list the root commits of this repository's default branch, so where its embargo stores are, or one would be made, cannot be told", "settle what git reports, then run it again")
	case errors.Is(err, storepath.ErrNoRootCommit):
		return refuse("this repository has no root commit to key the embargo store on", "commit once, then run it again")
	case errors.Is(err, storepath.ErrPlaceTaken):
		return refuse("the place this repository's embargo store would be made could not be examined, or holds a folder that is no worktree of this repository nor of another repository on this machine, perhaps a store git has lost track of",
			"look under abcd's worktree store, keep any private records the folder holds, move it aside, then run `abcd embargo intake`")
	case errors.Is(err, storepath.ErrPlaceUnread):
		return refuse("abcd's worktree store could not be listed, or a place in it where an embargo store is kept could not be examined, so what is under embargo cannot be told",
			"check the permissions under abcd's worktree store, then run it again")
	case errors.Is(err, storepath.ErrPlaceHeldByAClone):
		return errPlaceHeldByAClone
	case errors.Is(err, storepath.ErrNoFreePlace):
		return errNoFreePlace
	default:
		return refuse("the embargo store could not be located", "run it again")
	}
}

// errNoFreePlace is locate's refusal for storepath.ErrNoFreePlace: this
// repository has no store, and one cannot be made without touching
// another's. It names no path: the person finds the store in the checkout
// that owns it. A store moved anywhere but a worktrees/<root>/embargo place
// is no longer found, so the move it advises keeps that shape.
var errNoFreePlace = refuse("every place this repository's embargo store could be made already holds another repository's store, which is left alone",
	"run the embargo sub-verbs from the checkout that owns the store sitting there, or, in that checkout, run `git worktree move <store> <a place under that checkout's own root>` to free the place, then run it again here; the store must stay at a `worktrees/<root>/embargo` place under abcd's home, where it is still found")

// errPlaceHeldByAClone is locate's refusal for
// storepath.ErrPlaceHeldByAClone: the store sitting there is another
// clone's of this project, under the root they share, so no move frees the
// place, and that clone's checkout is where the sub-verbs run.
var errPlaceHeldByAClone = refuse("every place this repository's embargo store could be made already holds the store of another clone of this project, under the root they share, which is left alone",
	"run the embargo sub-verbs from that clone's checkout, which owns the store")

// isNoFreePlace reports whether err is either of locate's refusals for no
// free place, which a caller that only reads takes for "no store".
func isNoFreePlace(err error) bool { return err == errNoFreePlace || err == errPlaceHeldByAClone }

// registered returns the store at a registered location, refusing a
// registration whose folder is gone or cannot be examined.
func registered(loc storepath.Location, top string) (Store, error) {
	switch _, err := os.Lstat(loc.Path); {
	case errors.Is(err, fs.ErrNotExist):
		return Store{}, refuse("the embargo store is registered as a worktree of this repository, but its folder is gone",
			"if it was removed on purpose, run `git worktree prune`, then run it again")
	case err != nil:
		return Store{}, refuse("the embargo store's folder could not be examined", "check the permissions under abcd's home")
	}
	return Store{Root: loc.Path, Checkout: top}, nil
}

// readOtherStore reads the state of a store the check reads that is not
// this checkout's own: another clone's of the same project, or another
// project's under the same home. It is read and never written, and it is
// not held to Verify's "a worktree of this repository": its records are
// private to the same person, so they are under embargo here too. The state
// is read inside the store's folder (os.Root), so no link in it can lead the
// read out of the store, and held to loadState's checks: a regular file,
// this account's alone, within its cap, in shape.
func readOtherStore(root string) (State, error) {
	bad := func(why string) error {
		return refuse("the state file of another embargo store under this home is refused: "+why,
			"settle it from the checkout that store belongs to, then run it again")
	}
	if !fsutil.IsRealDir(root) {
		return State{}, bad("its store is not a real directory")
	}
	// A live worktree is held to its own store's rule: its index tracks
	// nothing in its local tier. A folder git does not answer for (a store
	// whose checkout is gone) has no index to ask.
	if top, err := gitutil.Toplevel(root); err == nil && samePath(top, root) {
		tracked, err := tracksLocalTier(root)
		if err != nil {
			return State{}, bad("git could not list what it tracks")
		}
		if tracked {
			return State{}, bad("its index tracks a file in its local tier, which holds only what abcd writes there")
		}
	}
	if err := spelledExactly(root, "the state file of another embargo store under this home is refused: "); err != nil {
		return State{}, err
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return State{}, bad("its store could not be opened")
	}
	defer r.Close()
	fi, err := r.Lstat(StateRel)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return emptyState(), nil
	case err != nil:
		return State{}, bad("it could not be examined")
	case trackedThere(root, StateRel):
		return State{}, bad("git tracks it, so a commit carried it there")
	case !fi.Mode().IsRegular():
		return State{}, bad("it is not a regular file")
	case fsutil.CallersAlone(filepath.Join(root, filepath.FromSlash(StateRel)), fi) != nil:
		return State{}, bad("it is not this account's alone (another owner, or a group or other write bit)")
	}
	data, err := fsutil.ReadGuardedInRoot(r, StateRel, maxStateBytes)
	if err != nil {
		return State{}, bad("it could not be read within its size limit")
	}
	st, why := decodeState(data)
	if why != "" {
		return State{}, bad(why)
	}
	return st, nil
}

// openStore returns the store of the repository at repoRoot, making it on first
// use as a detached worktree at the default branch's tip, and verifies it
// either way. created reports that this call made it. A store that would sit
// inside the checkout is refused before anything is made.
func openStore(home, repoRoot string) (s Store, created bool, err error) {
	top, err := gitutil.Toplevel(repoRoot)
	if err != nil {
		return Store{}, false, refuse("this is not inside a git checkout", "run it from the repository's checkout")
	}
	loc, err := locate(home, top)
	if err != nil {
		return Store{}, false, err
	}
	if loc.Registered {
		if s, err = registered(loc, top); err != nil {
			return Store{}, false, err
		}
	} else {
		s = Store{Root: loc.Path, Checkout: top}
		if inside(s.Root, top) {
			return Store{}, false, refuse("the embargo store would sit inside this checkout, where its records are one `git add` from the public history",
				"run abcd with a home directory outside the repository")
		}
		if err := s.create(home); err != nil {
			return Store{}, false, err
		}
		created = true
	}
	if _, err := s.Verify(); err != nil {
		return Store{}, created, err
	}
	return s, created, nil
}

// openExisting returns the store of the repository at repoRoot and its
// verified state, making nothing: the sub-verbs after intake act on a store
// intake made, and with none there is nothing under embargo to act on.
func openExisting(home, repoRoot string) (Store, State, error) {
	top, err := gitutil.Toplevel(repoRoot)
	if err != nil {
		return Store{}, State{}, refuse("this is not inside a git checkout", "run it from the repository's checkout")
	}
	loc, err := locate(home, top)
	if isNoFreePlace(err) || (err == nil && !loc.Registered) {
		return Store{}, State{}, refuse("there is no embargo store for this repository on this machine", "run `abcd embargo intake` first")
	}
	if err != nil {
		return Store{}, State{}, err
	}
	s, err := registered(loc, top)
	if err != nil {
		return Store{}, State{}, err
	}
	st, err := s.Verify()
	if err != nil {
		return Store{}, State{}, err
	}
	return s, st, nil
}

// create makes the store's parent levels, each a real directory this account
// alone can write, and adds the store detached at the default branch's tip:
// with no branch, nothing in it has a ref to push.
func (s Store) create(home string) error {
	rel := filepath.ToSlash(strings.TrimPrefix(filepath.Dir(s.Root), filepath.Clean(home)+string(filepath.Separator)))
	if err := fsutil.EnsureRealDirAll(home, rel, abcdhome.DirMode); err != nil {
		return refuse("the worktree store could not be made as a real directory under abcd's home", "move aside whatever occupies it")
	}
	parent := filepath.Dir(s.Root)
	fi, err := os.Lstat(parent)
	if err != nil || fsutil.CallersAlone(parent, fi) != nil {
		return refuse("the worktree store's folder for this repository is not this account's alone", "fix its owner or mode, then run it again")
	}
	defRef := gitutil.DefaultRef(s.Checkout)
	if defRef == "" {
		return refuse("the repository has no default branch to start the embargo store from", "fetch the remote, then run it again")
	}
	base, err := gitutil.Run(s.Checkout, "rev-parse", "--verify", "--quiet", defRef+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(base) {
		return refuse("the default branch's tip could not be resolved", "fetch the remote, then run it again")
	}
	if _, err := gitutil.Run(s.Checkout, "worktree", "add", "--detach", "--", s.Root, base); err != nil {
		return refuse("git could not add the embargo store: "+fsutil.RedactHome(err.Error()), "settle what git reports, then run it again")
	}
	if err := os.Chmod(s.Root, storeRootMode); err != nil {
		return refuse("the embargo store was made, but its folder could not be closed to other accounts", "run `chmod 700` on the store, then run it again")
	}
	// The store's local tier holds only files abcd writes, so its index
	// tracks nothing there, under any spelling of its case.
	if _, err := gitutil.Run(s.Root, "rm", "-r", "-q", "--ignore-unmatch", "--", localTierSpec); err != nil {
		return refuse("the embargo store was made, but git could not clear what its index tracks in its local tier: "+fsutil.RedactHome(err.Error()),
			"move the store aside, then run `abcd embargo intake` again")
	}
	return nil
}

// Verify is the check every sub-verb runs before it reads or writes the
// store, and returns the store's state. It refuses, writing nothing, when the
// store is not a real directory that is this repository's own worktree; when
// it resolves inside the checkout; when its HEAD is on a branch; when its
// state file is refused (state.go); or when its index tracks anything in its
// local tier, or a record the state names; or when another account could read its folder or a record
// the state names. A refusal never names a record's file or an advisory.
func (s Store) Verify() (State, error) {
	if !fsutil.IsRealDir(s.Root) {
		return State{}, refuse("the embargo store is not a real directory", "move aside whatever occupies its path, then run `abcd embargo intake`")
	}
	if fi, err := os.Stat(s.Root); err != nil || fi.Mode().Perm()&0o077 != 0 {
		return State{}, refuse("the embargo store's folder is open to other accounts on this machine (a group or other permission bit), so they can read its private records",
			"run `chmod 700` on the store, then run it again")
	}
	if inside(s.Root, s.Checkout) {
		return State{}, refuse("the embargo store resolves inside this checkout, where its records are one `git add` from the public history",
			"move the store out of the checkout")
	}
	top, err := gitutil.Toplevel(s.Root)
	if err != nil || !samePath(top, s.Root) {
		return State{}, refuse("the embargo store's folder is not a git worktree of its own", "move it aside, then run `abcd embargo intake` to make the store")
	}
	storeCommon, err1 := gitutil.RevParseAbsPath(s.Root, "--git-common-dir")
	ownCommon, err2 := gitutil.RevParseAbsPath(s.Checkout, "--git-common-dir")
	if err1 != nil || err2 != nil || !samePath(storeCommon, ownCommon) {
		return State{}, refuse("the embargo store is not a worktree of this repository", "move it aside, then run `abcd embargo intake` to make the store")
	}
	if _, exitOne, err := gitutil.RunCappedExitOne(s.Root, 4096, "symbolic-ref", "--quiet", "HEAD"); err != nil {
		return State{}, refuse("git could not say whether the embargo store's HEAD is detached", "settle what git reports, then run it again")
	} else if !exitOne {
		return State{}, refuse("the embargo store's HEAD is on a branch, which gives its records a ref that can be pushed",
			"detach it with `git -C <store> switch --detach`, then run it again")
	}
	if err := s.refuseTrackedLocalTier(); err != nil {
		return State{}, err
	}
	st, err := s.loadState()
	if err != nil {
		return State{}, err
	}
	if err := s.refuseTracked(st); err != nil {
		return State{}, err
	}
	if err := s.refuseRecordsOpen(st); err != nil {
		return State{}, err
	}
	return st, nil
}

// refuseRecordsOpen refuses when a record the state links to an advisory, in
// any status folder of the store's ledger, carries a group or other
// permission bit. The store's other records are the public ledger's own,
// checked out with the worktree, and are not judged.
func (s Store) refuseRecordsOpen(st State) error {
	if len(st.Advisories) == 0 {
		return nil
	}
	ledger := filepath.Join(s.Root, filepath.FromSlash(ledgerRel))
	statuses, err := os.ReadDir(ledger)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return refuse("the embargo store's ledger could not be listed", "check the permissions of the store, then run it again")
	}
	for _, d := range statuses {
		if !d.IsDir() {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(ledger, d.Name()))
		if err != nil {
			return refuse("the embargo store's ledger could not be listed", "check the permissions of the store, then run it again")
		}
		for _, e := range entries {
			if !isPrivateRecord(e.Name(), st) {
				continue
			}
			fi, err := os.Lstat(filepath.Join(ledger, d.Name(), e.Name()))
			if err != nil || (fi.Mode().IsRegular() && fi.Mode().Perm()&0o077 != 0) {
				return refuse("a private record in the embargo store is open to other accounts on this machine (a group or other permission bit)",
					"run `chmod -R go-rwx` on the store, then run it again")
			}
		}
	}
	return nil
}

// isPrivateRecord reports whether a ledger file name is the record of an
// advisory the state holds.
func isPrivateRecord(base string, st State) bool {
	for _, a := range st.Advisories {
		if base == a.IssueID+".md" || strings.HasPrefix(base, a.IssueID+"-") {
			return true
		}
	}
	return false
}

// refuseTracked refuses when the store's index tracks a record the state
// links to an advisory: one commit, and it is in the public history.
func (s Store) refuseTracked(st State) error {
	if len(st.Advisories) == 0 {
		return nil
	}
	out, err := gitutil.RunCappedBytes(s.Root, maxListing, "ls-files", "-z", "--", ledgerRel)
	if err != nil {
		return refuse("git could not list what the embargo store tracks", "settle what git reports, then run it again")
	}
	for _, p := range bytes.Split(out, []byte{0}) {
		if len(p) > 0 && isPrivateRecord(filepath.Base(string(p)), st) {
			return refuse("the embargo store's index tracks a private record",
				"unstage it with `git -C <store> rm --cached` on that record, keeping the file, then run it again")
		}
	}
	return nil
}

// refuseTrackedLocalTier refuses when the store's index tracks anything in
// its local tier, under any spelling of its case: the store never commits,
// and its local tier holds only what abcd writes there.
func (s Store) refuseTrackedLocalTier() error {
	tracked, err := tracksLocalTier(s.Root)
	if err != nil {
		return refuse("git could not list what the embargo store tracks", "settle what git reports, then run it again")
	}
	if tracked {
		return refuse("the embargo store's index tracks a file in its local tier, which holds only what abcd writes there",
			"take it out of the store's index with `git -C <store> rm -r --cached -- ':(icase).abcd/.work.local'`, which leaves every file in place, then run it again")
	}
	return nil
}

// tracksLocalTier reports whether the index of the worktree at root tracks
// anything in its local tier, under any spelling of its case.
func tracksLocalTier(root string) (bool, error) {
	out, err := gitutil.RunCappedBytes(root, maxListing, "ls-files", "-z", "--", localTierSpec)
	if err != nil {
		return false, err
	}
	return len(out) > 0, nil
}

// inside reports whether p is dir or lies beneath it, after resolving the
// symlinks of the longest part of each that exists (storepath.Resolve).
func inside(p, dir string) bool {
	rel, err := filepath.Rel(storepath.Resolve(dir), storepath.Resolve(p))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func samePath(a, b string) bool { return storepath.Resolve(a) == storepath.Resolve(b) }
