package memnote

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// write.go — the two writes the machine layer takes, add and remove. Nothing
// here writes the repository layer.
//
// A write creates and opens ~/.abcd.noindex in one walk relative to home's
// descriptor (fsutil.EnsureHomeScope, which refuses a symlinked scope), takes
// an exclusive flock on the directory at that path (fsutil.WithDirLock, which
// opens it O_NOFOLLOW and leaves no lock file beside the notes), re-reads the
// file under the lock through the guarded read relative to the descriptor the
// walk opened (readMachineIn), applies the change, and writes through that
// same descriptor with fsutil.WriteFileAtomicInRoot at 0600 — so two agents
// writing at once both land, a file the read refuses is never overwritten, and
// the file re-read is the file replaced even if the directory is renamed after
// the walk.

// AddRequest is one note to write. Session "" records the session as unknown.
type AddRequest struct {
	// Home is the account's home directory; RepoRoot is the repository the
	// note is written from (the rules loader's root), whose root commit ties
	// the note to it and whose scanner and private names judge it.
	Home, RepoRoot string
	Title, Body    string
	Recall         []string
	// By is the writer, <Vendor>:<model-version>; Session the host's id for
	// the session, or "".
	By, Session string
	// Everywhere recalls the note in every repository (root null).
	Everywhere bool
	// MovedFromHost stamps the origin host-memory-move.
	MovedFromHost bool
}

// ErrNotFound is Remove's answer for an id the machine layer does not hold.
var ErrNotFound = errors.New("no personal note has that id")

// ErrNoRoot refuses a note tied to its repository where the repository has no
// root commit to tie it to.
var ErrNoRoot = errors.New("this repository has no root commit to tie the note to; pass --everywhere for a lesson that holds in every project")

// ErrNoHome refuses a write with no absolute home to hold the machine layer.
var ErrNoHome = errors.New("the home directory is unresolved, so there is no machine layer to write")

// Seams the tests replace.
var (
	now        = time.Now
	rootCommit = gitutil.RootCommit
	minter     = recordid.Minter{}
)

// machineScopeOpened runs once the write has opened ~/.abcd.noindex and before
// it takes the lock. It does nothing in production; it is a var so a detector
// can rename the directory inside that window, which a real race cannot be
// relied on to hit.
var machineScopeOpened = func() {}

// lockTimeout bounds the wait for another writer of the machine layer.
var lockTimeout = 10 * time.Second

// machineLeaf is the machine layer's name inside the abcd home.
const machineLeaf = "memory.json"

// Add judges and writes one note to the machine layer and returns it. Every
// refusal comes before anything is created: a malformed field (*FieldError),
// a tied note with no root commit (ErrNoRoot), the gate (*GateError, or an
// error when the gate cannot judge), or a machine file the guarded read
// refuses.
func Add(req AddRequest) (Note, error) {
	n := Note{Title: req.Title, Body: req.Body, Recall: append([]string(nil), req.Recall...), WrittenBy: req.By, Origin: OriginAgent}
	if req.MovedFromHost {
		n.Origin = OriginHostMemoryMove
	}
	for _, err := range []error{CheckTitle(n.Title), CheckBody(n.Body), CheckRecall(n.Recall), CheckWrittenBy(n.WrittenBy)} {
		if err != nil {
			return Note{}, err
		}
	}
	if req.Session != "" {
		if err := CheckSession(req.Session); err != nil {
			return Note{}, err
		}
		s := req.Session
		n.Session = &s
	}
	if !req.Everywhere {
		root := rootCommit(req.RepoRoot)
		if !gitutil.IsFullSHA(root) {
			return Note{}, ErrNoRoot
		}
		n.Root = &root
	}
	if err := judge(req.RepoRoot, n); err != nil {
		return Note{}, err
	}
	err := withMachineLayer(req.Home, func(f *File) error {
		taken := map[string]bool{}
		for _, o := range f.Notes {
			taken[o.ID] = true
		}
		at := now().UTC()
		m := recordid.Minter{Now: func() time.Time { return at }, Entropy: minter.Entropy}
		for {
			id, err := m.Mint(Family)
			if err != nil {
				return err
			}
			if !taken[id] {
				n.ID = id
				break
			}
		}
		n.WrittenAt = at.Format(time.RFC3339)
		f.Notes = append(f.Notes, n)
		return nil
	})
	if err != nil {
		return Note{}, err
	}
	return n, nil
}

// Remove removes the personal note id from the machine layer, whoever wrote
// it, and returns it. ErrNotFound when the machine layer holds no such note.
func Remove(home, id string) (Note, error) {
	if !ValidID(id) {
		return Note{}, &FieldError{"id", "is not a mem-<16 digits> note id"}
	}
	if _, present, err := ReadMachine(home); err != nil {
		return Note{}, err
	} else if !present {
		return Note{}, ErrNotFound
	}
	var removed Note
	err := withMachineLayer(home, func(f *File) error {
		for i, o := range f.Notes {
			if o.ID == id {
				removed = o
				f.Notes = append(f.Notes[:i:i], f.Notes[i+1:]...)
				return nil
			}
		}
		return ErrNotFound
	})
	return removed, err
}

// withMachineLayer runs change over the machine layer under its lock and
// writes the result. change's error writes nothing and is returned as is.
func withMachineLayer(home string, change func(*File) error) error {
	if home == "" || !filepath.IsAbs(home) {
		return ErrNoHome
	}
	// The file is judged before anything is created, so a refused file never
	// gains a sibling lock or a fresh scope.
	if _, _, err := ReadMachine(home); err != nil {
		return err
	}
	dir, err := fsutil.EnsureHomeScope(home, abcdhome.Rel(), abcdhome.DirMode)
	if errors.Is(err, fsutil.ErrHomeScopeSymlinked) {
		return fmt.Errorf("memory: nothing was written to %s: %v", MachineDisplayPath, err)
	}
	if err != nil {
		return fmt.Errorf("memory: %s could not be created, so nothing was written", abcdhome.Display())
	}
	defer dir.Close()
	machineScopeOpened()
	err = fsutil.WithDirLock(abcdhome.Path(home), lockTimeout, func() error {
		f, present, err := readMachineIn(dir, home)
		if err != nil {
			return err
		}
		if !present {
			f = File{SchemaVersion: SchemaVersion}
		}
		if err := change(&f); err != nil {
			return err
		}
		data, err := encodeMachine(f)
		if err != nil {
			return fmt.Errorf("memory: %s could not be encoded: %v", MachineDisplayPath, err)
		}
		if len(data) > maxFileBytes {
			return fmt.Errorf("memory: %s would exceed its %d-byte cap; remove a note first (nothing was written)", MachineDisplayPath, maxFileBytes)
		}
		if err := fsutil.WriteFileAtomicInRoot(dir, machineLeaf, data, abcdhome.FileMode); err != nil {
			return fmt.Errorf("memory: %s could not be written, so nothing was written", MachineDisplayPath)
		}
		return nil
	})
	switch {
	case errors.Is(err, fsutil.ErrLockContention):
		return fmt.Errorf("memory: %s is being written by another abcd, so nothing was written; retry", MachineDisplayPath)
	case errors.Is(err, fsutil.ErrLockPathUnsafe):
		return fmt.Errorf("memory: %s is not a real directory, so nothing was written", abcdhome.Display())
	case errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("memory: %s vanished before the write, so nothing was written; retry", abcdhome.Display())
	}
	return err
}
