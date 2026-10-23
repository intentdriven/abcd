// Package worktree is the machine-scoped worktree store
// (itd-2609091014076309, spc-2609301811532881): a session's or an agent's
// worktree lives at ~/.abcd.noindex/worktrees/<root-sha>/<name>/, keyed on the
// repository's root commit in the full form the history and transcript stores
// use, and never beside the user's own projects (adr-2609091248200336, whose
// location adr-2610031751065746 spells under the renamed home).
//
// It is the only code that makes a directory in the store. The lane for a
// repository is LaneFor's answer, its levels are made by Lane.Ensure, a
// worktree already there is found by Adopt, and a new one is made by Add; the
// build loop's lane step calls these rather than a copy of them, so there is
// one way a worktree enters the store.
//
// The package never prints. It returns values and a typed *Refusal, whose
// text is home-redacted, and a front door formats them.
package worktree

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// StoreRel is the worktree store, relative to the caller's home.
var StoreRel = abcdhome.Rel("worktrees")

// dirPerm is the mode a store level is made with: the caller's alone.
const dirPerm = abcdhome.DirMode

// Refusal is the store declining an operation for a reason the caller can act
// on. Reason and Remedy are home-redacted. The remedy names the act that
// clears the reason and not how to retry, which is the caller's to say. Err,
// when set, is the failure underneath (git's), which Unwrap hands on.
type Refusal struct {
	Reason string
	Remedy string
	Err    error
}

func (r *Refusal) Error() string { return r.Reason + "; " + r.Remedy }

// Unwrap is the failure underneath the refusal, if any.
func (r *Refusal) Unwrap() error { return r.Err }

func refuse(reason, remedy string) *Refusal {
	return &Refusal{Reason: fsutil.RedactHome(reason), Remedy: fsutil.RedactHome(remedy)}
}

// Lane is one repository's directory in the store.
type Lane struct {
	// Home is the caller's home, the base the store is proved real from.
	Home string
	// RootSHA is the repository's root commit, the lane's key.
	RootSHA string
	// Rel is the lane, relative to Home, slash-separated: StoreRel/<RootSHA>.
	Rel string
	// Dir is the lane's absolute path, Home/Rel.
	Dir string
}

// Path is the worktree called name in the lane. It builds the path only; a
// caller that has not validated name with ValidName must not use it.
func (l Lane) Path(name string) string { return filepath.Join(l.Dir, name) }

// LaneFor derives the lane of the repository at repoRoot from its root commit
// and the caller's home. It makes nothing. A repository with no root commit
// is refused; git failing to answer is an error, since that is no evidence
// the repository has none.
func LaneFor(repoRoot string) (Lane, error) {
	sha, err := gitutil.RootCommit(repoRoot)
	if err != nil {
		return Lane{}, fmt.Errorf("reading the repository's root commit to key the worktree store on: %w", err)
	}
	if !gitutil.IsFullSHA(sha) {
		return Lane{}, refuse("the repository has no root commit to key the worktree store on",
			"commit to the repository first; the store is keyed on its root commit")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return Lane{}, fmt.Errorf("cannot resolve the caller's home directory for the worktree store: %v", err)
	}
	rel := StoreRel + "/" + sha
	return Lane{Home: home, RootSHA: sha, Rel: rel, Dir: filepath.Join(home, filepath.FromSlash(rel))}, nil
}

// Ensure makes the lane's levels under the home one at a time, from the top
// down, and holds each to two tests before the next is made: it is a real
// directory (a symlink or a file anywhere in the chain refuses), and it is the
// caller's alone (fsutil.CallersAlone: owned by this uid, no group or other
// write bit), so no other account can replace a checkout in it. A level made
// here is made 0700; one that already exists keeps its mode and is judged as
// it stands, and nothing is made inside a level that fails. A Lane that is not
// the store's shape for its own root commit makes nothing.
func (l Lane) Ensure() error {
	if err := l.shaped(); err != nil {
		return err
	}
	_, err := levels(l.Home, l.Rel, "worktree store", true)
	return err
}

// shaped refuses a Lane that is not LaneFor's answer for its own root commit.
func (l Lane) shaped() error {
	if !gitutil.IsFullSHA(l.RootSHA) || l.Rel != StoreRel+"/"+l.RootSHA ||
		l.Dir != filepath.Join(l.Home, filepath.FromSlash(l.Rel)) {
		return refuse("the lane is not the worktree store's for its root commit",
			"derive the lane with LaneFor")
	}
	return nil
}

// present reports whether the lane's levels all exist, holding each that does
// to Ensure's two tests, from the top down, and making none: a reader of the
// store refuses what a maker would refuse, and an absent level is a store that
// holds nothing yet.
func (l Lane) present() (bool, error) {
	if err := l.shaped(); err != nil {
		return false, err
	}
	return levels(l.Home, l.Rel, "worktree store", false)
}

// levels walks rel's levels under home one at a time, from the top down,
// making each that is absent when create is set and holding each to the two
// tests Ensure names before the next is reached. Without create, it answers
// false at the first absent level and reads nothing beneath it. It is the
// one maker the worktree store and the notes archive share.
func levels(home, rel, what string, create bool) (bool, error) {
	if !fsutil.IsRealDir(home) || !fsutil.ValidRelPath(rel) {
		return false, refuse("the home directory is not a real directory to make the "+what+" under",
			"make the home a real directory, or work from an account whose home is one")
	}
	dir, shown := home, "~"
	for _, seg := range strings.Split(rel, "/") {
		dir, shown = filepath.Join(dir, seg), shown+"/"+seg
		if create {
			if err := fsutil.EnsureRealDir(dir, dirPerm); err != nil {
				return false, refuse(fmt.Sprintf("the %s ~/%s cannot be made as real directories: %v", what, rel, err),
					"remove what stands in for the store (a symlink or a file)")
			}
		}
		fi, err := os.Lstat(dir)
		switch {
		case !create && errors.Is(err, fs.ErrNotExist):
			return false, nil
		case err != nil:
			return false, fmt.Errorf("checking the %s level %s: %w", what, shown, err)
		case !fi.IsDir():
			return false, refuse(fmt.Sprintf("the %s ~/%s is not real directories: %s is a symlink or a file", what, rel, shown),
				"remove what stands in for the store (a symlink or a file)")
		}
		switch err := fsutil.CallersAlone(dir, fi); {
		case errors.Is(err, fsutil.ErrDeclarationWritable):
			return false, refuse(fmt.Sprintf("the %s level %s is writable by its group or by every user, so another account could replace what it holds", what, shown),
				"make it yours alone (`chmod go-w` it)")
		case err != nil:
			return false, refuse(fmt.Sprintf("the %s level %s is owned by another account, or its owner could not be read, so that account could replace what it holds", what, shown),
				"move it aside and let the store make itself")
		}
	}
	return true, nil
}

// ValidName reports whether name is one plain path segment that cannot escape
// the lane it is joined onto and cannot be read as an option: letters, digits,
// '.', '_' and '-', not led by '-' or '.', and holding no "..". A name a
// sibling store holds in the lane is not a worktree's (reserved).
func ValidName(name string) bool {
	if name == "" || name[0] == '-' || name[0] == '.' || strings.Contains(name, "..") || reserved(name) {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '.', c == '_', c == '-':
		default:
			return false
		}
	}
	return true
}

// reserved reports whether name is one abcd's sibling stores hold in the lane:
// the embargo store (iss-2610230014564359), compared without case, since a
// case-folding filesystem reaches the same directory through either spelling.
func reserved(name string) bool { return strings.EqualFold(name, storepath.Name) }

func invalidName(name string) *Refusal {
	if reserved(name) {
		return refuse(fmt.Sprintf("%q is the name of abcd's embargo store, which lives in the same lane", name),
			"name the worktree something else")
	}
	return refuse(fmt.Sprintf("%q is not one plain path segment (letters, digits, '.', '_' and '-', not led by '-' or '.', and no '..')", name),
		"name the worktree with one plain path segment")
}

// Adopt reports whether git already lists a worktree at name's path in the
// lane on branch (a short branch name): one the store made earlier, whose
// maker was stopped before it recorded it. One at that path on any other
// branch is refused, and so is a listed worktree with no real directory where
// git says it is; neither is touched.
func Adopt(repoRoot string, l Lane, name, branch string) (bool, error) {
	if !ValidName(name) {
		return false, invalidName(name)
	}
	path := l.Path(name)
	wts, err := gitutil.ListWorktrees(repoRoot, gitutil.MaxWorktreeListing)
	if err != nil {
		return false, fmt.Errorf("listing the repository's worktrees: %w", err)
	}
	want := fsutil.RealExistingPath(path)
	for _, wt := range wts {
		if fsutil.RealExistingPath(wt.Path) != want {
			continue
		}
		if wt.Branch != "refs/heads/"+branch {
			return false, refuse(fmt.Sprintf("git lists a worktree at %s on %q, not on the branch %s", path, wt.Branch, branch),
				"the store never adopts what it cannot prove it made; remove that worktree (`git worktree remove`)")
		}
		fi, err := os.Lstat(path)
		if err != nil || !fi.IsDir() {
			return false, refuse(fmt.Sprintf("git lists the worktree at %s, but no real directory stands there", path),
				"run `git worktree prune`")
		}
		return true, nil
	}
	return false, nil
}

// GitRunner runs one git command in dir and returns its output, or its
// failure. Add takes it from the caller, because how a checkout runs (the
// person's filters and hooks, its time bound) is the caller's to decide.
type GitRunner func(dir string, args ...string) (string, error)

// AddOptions is one worktree for Add to make.
type AddOptions struct {
	// Name is the worktree's name in the lane, one plain path segment.
	Name string
	// Branch is the short branch name; "" means Name.
	Branch string
	// Existing checks out Branch, which must already exist, and makes no
	// branch. Otherwise Branch is made, from Base, and must not exist yet.
	Existing bool
	// Base is the commit a new branch is cut from; unused with Existing.
	Base string
	// Git runs `git worktree add`. It is required.
	Git GitRunner
}

// Added is the worktree Add made.
type Added struct {
	Lane   Lane
	Path   string
	Branch string
}

// Add makes a worktree in the lane of the repository at repoRoot: the name
// and branch are judged, the lane is derived and read (LaneFor), and nothing
// may stand at the path; only then are the lane's levels made (Ensure), git
// adds the worktree and its directory is narrowed to 0700. Every refusal
// comes before anything is written, so a refused Add writes nothing; a git
// that fails is a refusal whose cause is git's error (Refusal.Err).
//
// Once git has made the worktree, Add records where its branch starts
// (start.go). That failing is a refusal returned with the Added it made,
// since the worktree stands and git's own removal is the way back.
func Add(repoRoot string, opts AddOptions) (Added, error) {
	if opts.Git == nil {
		return Added{}, errors.New("worktree: Add needs a git runner")
	}
	if !ValidName(opts.Name) {
		return Added{}, invalidName(opts.Name)
	}
	branch := opts.Branch
	if branch == "" {
		branch = opts.Name
	}
	if strings.HasPrefix(branch, "-") {
		return Added{}, refuse(fmt.Sprintf("%q is not a valid branch name", branch), "name a branch git accepts")
	}
	if _, err := gitutil.Run(repoRoot, "check-ref-format", "--branch", branch); err != nil {
		return Added{}, refuse(fmt.Sprintf("%q is not a valid branch name", branch), "name a branch git accepts")
	}
	lane, err := LaneFor(repoRoot)
	if err != nil {
		return Added{}, err
	}
	if !opts.Existing && !gitutil.IsFullSHA(opts.Base) {
		return Added{}, refuse(fmt.Sprintf("there is no base commit to cut the new branch %s from", branch),
			"check an existing branch out instead, or give the repository a default branch")
	}
	// Read the lane as Ensure would judge it, making nothing, so the refusals
	// below leave the home as it was.
	if _, err := lane.present(); err != nil {
		return Added{}, err
	}
	path := lane.Path(opts.Name)
	if _, err := os.Lstat(path); err == nil {
		return Added{}, refuse(fmt.Sprintf("%s is occupied by something git does not list as this worktree", path),
			"the store never adopts what it cannot prove it made; move it aside")
	} else if !errors.Is(err, fs.ErrNotExist) {
		return Added{}, fmt.Errorf("checking %s: %w", fsutil.RedactHome(path), err)
	}
	ref := "refs/heads/" + branch
	_, verr := gitutil.Run(repoRoot, "rev-parse", "--verify", "--quiet", ref, "--")
	exists := verr == nil
	var args []string
	switch {
	case opts.Existing && !exists:
		return Added{}, refuse(fmt.Sprintf("there is no local branch %s to check out", branch),
			"name a local branch that exists, or let the worktree make its own")
	case opts.Existing:
		args = []string{"worktree", "add", "--", path, branch}
	case exists:
		return Added{}, refuse(fmt.Sprintf("a branch named %s already exists", branch),
			"check the existing branch out instead of making it")
	default:
		args = []string{"worktree", "add", "-b", branch, "--", path, opts.Base}
	}
	if err := lane.Ensure(); err != nil {
		return Added{}, err
	}
	if _, err := opts.Git(repoRoot, args...); err != nil {
		r := refuse("git could not add the worktree: "+err.Error(), "settle what git reports")
		r.Err = err
		return Added{}, r
	}
	// git makes the worktree at the process umask, 0755 under the usual 022;
	// a worktree in the store is the caller's alone, like the levels above it.
	// A filesystem that cannot change modes (exFAT, a network share) keeps it
	// as git made it: the home above it is narrowed at every command, so other
	// accounts cannot reach it.
	_, _ = fsutil.NarrowToOwner(path)
	added := Added{Lane: lane, Path: path, Branch: branch}
	if err := recordStart(repoRoot, lane, opts.Name, ref); err != nil {
		return added, refuse(fmt.Sprintf("git made the worktree at %s, but where its branch starts could not be recorded: %v", path, err),
			"settle what stops the write, then remove the worktree (`git worktree remove`) and add it again")
	}
	return added, nil
}
