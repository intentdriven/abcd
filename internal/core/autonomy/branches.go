package autonomy

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// MaxCreatedBranches bounds a working copy's created-branch list; past it the
// oldest record is dropped. A hundred names at git's 255-byte limit stay well
// within the store's 64 KiB per file (spc-2610180914469489 leaves the bound to
// the builder).
const MaxCreatedBranches = 100

// createdSchemaVersion is the created-branch list's shape version.
const createdSchemaVersion = 1

// createdFile is the store file holding the branches the working copy whose
// key is copyKey created.
func createdFile(copyKey string) string { return "branches-" + copyKey + ".json" }

// CreatedBranch is one branch a session in the working copy created, and when
// it was recorded.
type CreatedBranch struct {
	Name string    `json:"name"`
	At   time.Time `json:"at"`
	// Session is the host session that created it, as the guard's payload
	// named it; "" in a record written before sessions were kept, or by a
	// hook with no session id, which says nothing of whose it is.
	Session string `json:"session,omitempty"`
}

type createdList struct {
	SchemaVersion int             `json:"schema_version"`
	Branches      []CreatedBranch `json:"branches"`
}

// BranchChangeKind is what a recorded command did to a branch.
type BranchChangeKind string

const (
	// BranchCreated is a branch the command creates; RecordCreated keeps it
	// only when no branch of that name exists yet.
	BranchCreated BranchChangeKind = "created"
	// BranchCheckedOut is a branch `git worktree add` checks out in a new
	// worktree; RecordCreated keeps it when it exists as a branch, or when the
	// name resolves to nothing at all, so git creates it, which it records
	// as BranchCreated.
	BranchCheckedOut BranchChangeKind = "checked-out"
	// BranchMoved is a rename: a record of From (the current branch of the
	// command's worktree when "") is carried to Name.
	BranchMoved BranchChangeKind = "moved"
)

// BranchChange is one change the created-branch recorder applies.
type BranchChange struct {
	Kind BranchChangeKind
	Name string
	From string
	// Session is the host session making the change, kept on a name the
	// change adds; "" keeps none. A rename carries the record's own.
	Session string
}

// CreatedBranches is the created-branch list of the working copy whose key is
// copyKey, oldest first; empty when none is kept. A list the store's checks
// refuse, or that does not parse, is ErrUnsafe.
func (s Store) CreatedBranches(copyKey string) ([]CreatedBranch, error) {
	raw, err := s.ReadFile(createdFile(copyKey))
	if errors.Is(err, fs.ErrNotExist) && !errors.Is(err, ErrUnsafe) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var l createdList
	if err := json.Unmarshal(raw, &l); err != nil || l.SchemaVersion != createdSchemaVersion {
		return nil, fmt.Errorf("%w: %s does not parse as a created-branch list", ErrUnsafe, s.Display(createdFile(copyKey)))
	}
	return l.Branches, nil
}

// RecordBranches applies changes, already judged, to the created-branch list
// of the working copy whose key is copyKey, under the store's lock and with
// the list re-read inside it: a created name is recorded anew, with this
// change's session and time, even when the list already holds it (a branch
// deleted and made again is a new branch, and its first creator's record
// would hand it to that session); a checked-out name is added once; a rename
// carries the record whose name is From, whole name to whole name; and the
// list keeps the newest MaxCreatedBranches.
func (s Store) RecordBranches(copyKey string, changes []BranchChange, now time.Time) error {
	if len(changes) == 0 {
		return nil
	}
	now = now.Truncate(time.Second)
	return s.WithLock(func() error {
		cur, err := s.CreatedBranches(copyKey)
		if err != nil {
			return err
		}
		next := applyBranchChanges(cur, changes, now)
		raw, err := json.MarshalIndent(createdList{SchemaVersion: createdSchemaVersion, Branches: next}, "", "  ")
		if err != nil {
			return err
		}
		return s.WriteFile(createdFile(copyKey), append(raw, '\n'))
	})
}

func applyBranchChanges(list []CreatedBranch, changes []BranchChange, now time.Time) []CreatedBranch {
	index := func(name string) int {
		for i, b := range list {
			if b.Name == name {
				return i
			}
		}
		return -1
	}
	for _, c := range changes {
		switch c.Kind {
		case BranchCreated, BranchCheckedOut:
			if c.Name == "" {
				continue
			}
			i := index(c.Name)
			if i >= 0 && c.Kind == BranchCheckedOut {
				continue
			}
			// A name created again is a new branch (RecordCreated has
			// proved its first is gone): its record is this creation's,
			// session and time, and the newest, never the first creator's.
			if i >= 0 {
				list = append(list[:i], list[i+1:]...)
			}
			list = append(list, CreatedBranch{Name: c.Name, At: now, Session: c.Session})
		case BranchMoved:
			i := index(c.From)
			if i < 0 || c.Name == "" {
				continue
			}
			if j := index(c.Name); j >= 0 && j != i {
				list = append(list[:j], list[j+1:]...)
				i = index(c.From)
			}
			list[i].Name = c.Name
		}
	}
	if n := len(list); n > MaxCreatedBranches {
		list = append([]CreatedBranch(nil), list[n-MaxCreatedBranches:]...)
	}
	return list
}

// RecordCreated is the created-branch recorder the guard's pre-tool hook
// runs, before the command does, whether or not the person is away: it judges
// each change against the repository as it stands (a created branch only when
// no branch of that name exists yet; a checked-out one when it is a branch no
// worktree has checked out, or names nothing git can resolve) and records
// what passes under the copy's key (claim.CopyKey, minted as a reservation
// verb mints it) in the away store, each added name with the session o names
// (Options.SessionID) as its creator, so the stop check can tell one
// session's branches from another's in a copy two share. copyRoot is the
// session's working copy; runDir is where the command runs, whose current
// branch a rename with no old name renames. A checkout abcd does not manage (Managed), or with no root
// commit, records nothing and is no error. The error is a store the recorder
// could not write; it never refuses the command.
func RecordCreated(copyRoot, runDir string, changes []BranchChange, o Options) error {
	if len(changes) == 0 {
		return nil
	}
	if !Managed(copyRoot) {
		return nil
	}
	home, err := o.home()
	if err != nil {
		return err
	}
	st, err := openStore(home, copyRoot)
	if errors.Is(err, ErrNoRootCommit) {
		return nil
	}
	if err != nil {
		return err
	}
	var keep []BranchChange
	var checkedOut map[string]bool
	for _, c := range changes {
		switch c.Kind {
		case BranchCreated:
			if !branchExists(copyRoot, c.Name) {
				keep = append(keep, c)
			}
		case BranchCheckedOut:
			if checkedOut == nil {
				checkedOut = worktreeBranches(copyRoot)
			}
			// git refuses to check out a branch another worktree has
			// checked out, and the recorder runs before the command, so
			// such a name is a peer's branch, not this session's.
			if checkedOut[c.Name] {
				continue
			}
			switch {
			case branchExists(copyRoot, c.Name):
				keep = append(keep, c)
			case !resolves(copyRoot, c.Name):
				// git makes the branch it checks out: a creation.
				c.Kind = BranchCreated
				keep = append(keep, c)
			}
		case BranchMoved:
			if c.From == "" {
				cur, err := gitutil.Run(runDir, "symbolic-ref", "--quiet", "--short", "HEAD")
				if err != nil || cur == "" {
					continue
				}
				c.From = cur
			}
			keep = append(keep, c)
		}
	}
	if len(keep) == 0 {
		return nil
	}
	if ValidSessionID(o.SessionID) {
		for i := range keep {
			keep[i].Session = o.SessionID
		}
	}
	key, err := claim.CopyKey(copyRoot)
	if err != nil {
		return err
	}
	return st.RecordBranches(key, keep, o.now())
}

// worktreeBranches are the branches the repository's worktrees have checked
// out, by short name, as git's checkout counts them: each listed worktree's
// branch, and each branch a rebase or bisect stopped in a worktree started
// from (gitutil.HeldBranches' Branches), which git lists detached but still
// refuses to check out elsewhere. A rebase's --update-refs list is left out,
// since git checks those branches out. Empty when they cannot be listed, and
// a state file that cannot be read adds nothing, as the listing's failure
// does: the recorder then records the checkout, as it did before.
func worktreeBranches(root string) map[string]bool {
	out := map[string]bool{}
	wts, err := gitutil.ListWorktrees(root, gitutil.MaxWorktreeListing)
	if err != nil {
		return out
	}
	for _, wt := range wts {
		if name, ok := strings.CutPrefix(wt.Branch, "refs/heads/"); ok {
			out[name] = true
		}
	}
	if common, err := gitutil.CommonDir(root); err == nil {
		for name := range gitutil.HeldBranches(common).Branches {
			out[name] = true
		}
	}
	return out
}

// branchExists reports whether refs/heads/name names a commit in the
// repository at root.
func branchExists(root, name string) bool {
	if name == "" {
		return false
	}
	_, err := gitutil.Run(root, "rev-parse", "--verify", "--quiet", "refs/heads/"+name+"^{commit}")
	return err == nil
}

// resolves reports whether name resolves to a commit by any of git's rules
// (a tag, a remote-tracking ref, an object name).
func resolves(root, name string) bool {
	_, err := gitutil.Run(root, "rev-parse", "--verify", "--quiet", "--end-of-options", name+"^{commit}")
	return err == nil
}
