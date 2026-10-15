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
)

// StoreName is the store's folder under the repository's level of the
// worktree store.
const StoreName = storepath.Name

// ledgerRel is where a private record sits inside the store: the ledger's own
// layout, so the capture and resolve code read it unchanged.
const ledgerRel = recordid.IssuesRelDir

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
// thing the person can do about it. The surfaces exit 2 on it.
type RefusalError struct {
	Reason string
	Remedy string
}

func (e *RefusalError) Error() string {
	if e.Remedy == "" {
		return "embargo: " + e.Reason
	}
	return "embargo: " + e.Reason + " (" + e.Remedy + ")"
}

func refuse(reason, remedy string) error { return &RefusalError{Reason: reason, Remedy: remedy} }

// Store is the embargo store of one repository: Root is the store worktree,
// Checkout the working-tree root of the checkout it belongs to.
type Store struct {
	Root     string `json:"root"`
	Checkout string `json:"checkout"`
}

// StorePath is where the store of the repository at repoRoot lives under
// home: worktrees/<root-sha>/embargo in abcd's home, keyed on the root commit
// so every worktree of the repository finds the same store
// (adr-2610031751065746).
func StorePath(home, repoRoot string) (string, error) {
	p, ok := storepath.Path(home, repoRoot)
	if !ok {
		return "", refuse("this repository has no root commit to key the embargo store on", "commit once, then run it again")
	}
	return p, nil
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
	p, err := StorePath(home, top)
	if err != nil {
		return Store{}, false, err
	}
	s = Store{Root: p, Checkout: top}
	if inside(p, top) {
		return Store{}, false, refuse("the embargo store would sit inside this checkout, where its records are one `git add` from the public history",
			"run abcd with a home directory outside the repository")
	}
	switch _, err := os.Lstat(p); {
	case errors.Is(err, fs.ErrNotExist):
		if err := s.create(home); err != nil {
			return Store{}, false, err
		}
		created = true
	case err != nil:
		return Store{}, false, refuse("the embargo store's folder could not be examined", "check the permissions under abcd's home")
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
	p, err := StorePath(home, top)
	if err != nil {
		return Store{}, State{}, err
	}
	s := Store{Root: p, Checkout: top}
	switch _, err := os.Lstat(p); {
	case errors.Is(err, fs.ErrNotExist):
		return Store{}, State{}, refuse("there is no embargo store for this repository on this machine", "run `abcd embargo intake` first")
	case err != nil:
		return Store{}, State{}, refuse("the embargo store's folder could not be examined", "check the permissions under abcd's home")
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
	return nil
}

// Verify is the check every sub-verb runs before it reads or writes the
// store, and returns the store's state. It refuses, writing nothing, when the
// store is not a real directory that is this repository's own worktree; when
// it resolves inside the checkout; when its HEAD is on a branch; when its
// state file is refused (state.go); or when its index tracks a record the
// state names; or when another account could read its folder or a record
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

// inside reports whether p is dir or lies beneath it, after resolving the
// symlinks of the longest part of each that exists.
func inside(p, dir string) bool {
	rel, err := filepath.Rel(resolve(dir), resolve(p))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func samePath(a, b string) bool { return resolve(a) == resolve(b) }

// resolve is filepath.EvalSymlinks over the longest existing prefix of p, with
// the rest joined on: a path not made yet still resolves through the links
// above it (macOS's /var is one).
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
