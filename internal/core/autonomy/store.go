package autonomy

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

const (
	// StoreDir is the away store's folder below abcd's home; each repository
	// has StoreDir/<root-sha>/.
	StoreDir = "away"
	// RecordFile is the away record's file name in a repository's folder.
	RecordFile = "record.json"
	// LockFile is the lock file's name in a repository's folder.
	LockFile = "lock"
	// MaxFileBytes is the most any file in the store may hold.
	MaxFileBytes = 64 << 10
	// LockTimeout is how long a writer waits for the lock.
	LockTimeout = 2 * time.Second
)

var (
	// ErrUnsafe is a store file that is not a regular file this account owns
	// that no one else can write, within MaxFileBytes, or that does not parse.
	ErrUnsafe = errors.New("autonomy: away store file refused")
	// ErrLockBusy is the store's lock not had within LockTimeout.
	ErrLockBusy = errors.New("autonomy: away store lock busy")
	// ErrNoRootCommit is a repository with no root commit to key the store on.
	ErrNoRootCommit = errors.New("autonomy: the repository has no root commit to key the away store on")
	// ErrNoHome is a home directory that is unset or not absolute.
	ErrNoHome = errors.New("autonomy: no home directory for the away store")
	// ErrRecordFull is a record that would pass MaxFileBytes if written.
	ErrRecordFull = errors.New("autonomy: the away record would pass 64 KiB")
)

// Store is one repository's folder in the away store.
type Store struct {
	home    string
	rootSHA string
}

// OpenStore is the away store of the repository whose working copy is at
// repoRoot, below home: keyed on the repository's root commit, so every
// working copy of it opens the same store. Nothing is created.
func OpenStore(home, repoRoot string) (Store, error) {
	sha := gitutil.RootCommit(repoRoot)
	if !gitutil.IsFullSHA(sha) {
		return Store{}, ErrNoRootCommit
	}
	return StoreAt(home, sha)
}

// StoreAt is the away store keyed on rootSHA, a full object name, below home.
func StoreAt(home, rootSHA string) (Store, error) {
	if home == "" || !filepath.IsAbs(home) {
		return Store{}, ErrNoHome
	}
	if !gitutil.IsFullSHA(rootSHA) {
		return Store{}, ErrNoRootCommit
	}
	return Store{home: home, rootSHA: rootSHA}, nil
}

// RootSHA is the root commit the store is keyed on.
func (s Store) RootSHA() string { return s.rootSHA }

// Rel is the store's folder as a slash path relative to the home directory.
func (s Store) Rel() string { return abcdhome.Rel(StoreDir, s.rootSHA) }

// Dir is the store's folder.
func (s Store) Dir() string { return abcdhome.Path(s.home, StoreDir, s.rootSHA) }

// RecordPath is the away record's path.
func (s Store) RecordPath() string { return filepath.Join(s.Dir(), RecordFile) }

// LockPath is the lock file's path.
func (s Store) LockPath() string { return filepath.Join(s.Dir(), LockFile) }

// Display is a file of the store in the tilde form a message names it by.
func (s Store) Display(name string) string { return abcdhome.Display(StoreDir, s.rootSHA, name) }

// siblingRe is a sibling file's name: one path segment, no leading dot.
var siblingRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// ReadFile reads the store file name under the store's checks: a regular
// file this account owns that no one else can write, within MaxFileBytes,
// reached through folders no one else can change. An absent file is an error
// for which errors.Is(err, fs.ErrNotExist) holds; a refused one is ErrUnsafe,
// naming it. A caller reading to modify holds WithLock around it.
func (s Store) ReadFile(name string) ([]byte, error) {
	if !siblingRe.MatchString(name) {
		return nil, fmt.Errorf("%w: %q is not a store file name", ErrUnsafe, name)
	}
	raw, refusal, err := fsutil.ReadHomeDeclaration(s.home, s.Rel()+"/"+name, MaxFileBytes)
	switch refusal {
	case fsutil.DeclarationOK:
		return raw, nil
	case fsutil.DeclarationAbsent:
		if errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("%w: %s %s", ErrUnsafe, s.Display(name), refusalWords(refusal, err))
}

// WriteFile replaces the store file name through fsutil's atomic writer,
// creating the store's folders (owner-only) on the way, and refuses data past
// MaxFileBytes. The caller holds WithLock.
func (s Store) WriteFile(name string, data []byte) error {
	if !siblingRe.MatchString(name) || name == LockFile {
		return fmt.Errorf("%w: %q is not a store file name", ErrUnsafe, name)
	}
	if len(data) > MaxFileBytes {
		return fmt.Errorf("%w: %s would hold %d bytes, at most %d", ErrRecordFull, s.Display(name), len(data), MaxFileBytes)
	}
	dir, err := fsutil.EnsureHomeScope(s.home, s.Rel(), abcdhome.DirMode)
	if err != nil {
		return fmt.Errorf("opening %s: %w", abcdhome.Display(StoreDir, s.rootSHA), err)
	}
	defer dir.Close()
	if err := fsutil.WriteFileAtomicInRoot(dir, name, data, abcdhome.FileMode); err != nil {
		return fmt.Errorf("writing %s: %w", s.Display(name), err)
	}
	return nil
}

func refusalWords(r fsutil.DeclarationRefusal, err error) string {
	switch r {
	case fsutil.DeclarationNotRegular:
		return "is not a regular file (a symlink or something else stands there)"
	case fsutil.DeclarationWritableByOthers:
		return "is writable by another account"
	case fsutil.DeclarationForeignOwner:
		return "is not this account's"
	case fsutil.DeclarationBehindSymlink:
		return "is reached through a symlink"
	case fsutil.DeclarationDirectoryExposed:
		return "sits in a folder another account can change"
	}
	if errors.Is(err, fsutil.ErrTooBig) {
		return fmt.Sprintf("is over %d KiB", MaxFileBytes>>10)
	}
	return "could not be read safely"
}

// vetLock checks the lock file, when there is one, on the store's terms: a
// regular file this account owns that no one else can write, within
// MaxFileBytes.
func (s Store) vetLock() error {
	p := s.LockPath()
	fi, err := os.Lstat(p)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%w: %s could not be checked", ErrUnsafe, s.Display(LockFile))
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("%w: %s %s", ErrUnsafe, s.Display(LockFile), refusalWords(fsutil.DeclarationNotRegular, nil))
	}
	if err := fsutil.CallersAlone(p, fi); err != nil {
		r := fsutil.DeclarationForeignOwner
		if errors.Is(err, fsutil.ErrDeclarationWritable) {
			r = fsutil.DeclarationWritableByOthers
		}
		return fmt.Errorf("%w: %s %s", ErrUnsafe, s.Display(LockFile), refusalWords(r, nil))
	}
	if fi.Size() > MaxFileBytes {
		return fmt.Errorf("%w: %s %s", ErrUnsafe, s.Display(LockFile), refusalWords(fsutil.DeclarationUnreadable, fsutil.ErrTooBig))
	}
	return nil
}

// WithLock holds the store's lock across fn: the store's folders are made
// (owner-only) if absent, the lock file is checked as every store file is,
// and the lock not had within LockTimeout is ErrLockBusy. fn's error is
// returned as it is.
func (s Store) WithLock(fn func() error) error {
	dir, err := fsutil.EnsureHomeScope(s.home, s.Rel(), abcdhome.DirMode)
	if err != nil {
		return fmt.Errorf("opening %s: %w", abcdhome.Display(StoreDir, s.rootSHA), err)
	}
	dir.Close()
	if err := s.vetLock(); err != nil {
		return err
	}
	err = fsutil.WithFileLock(s.LockPath(), LockTimeout, fn)
	switch {
	case errors.Is(err, fsutil.ErrLockContention):
		return fmt.Errorf("%w: %s was not had within %s", ErrLockBusy, s.Display(LockFile), LockTimeout)
	case errors.Is(err, fsutil.ErrLockPathUnsafe):
		return fmt.Errorf("%w: %s %s", ErrUnsafe, s.Display(LockFile), refusalWords(fsutil.DeclarationNotRegular, nil))
	}
	return err
}

// Read returns the stored record, or nil when there is none. A record or
// lock refused by the store's checks, or a record that does not parse, is
// ErrUnsafe naming the file.
func (s Store) Read() (*Record, error) {
	raw, err := s.ReadFile(RecordFile)
	if errors.Is(err, fs.ErrNotExist) && !errors.Is(err, ErrUnsafe) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := s.vetLock(); err != nil {
		return nil, err
	}
	var r Record
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("%w: %s does not parse", ErrUnsafe, s.Display(RecordFile))
	}
	if r.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("%w: %s has schema_version %d, this abcd reads %d", ErrUnsafe, s.Display(RecordFile), r.SchemaVersion, SchemaVersion)
	}
	return &r, nil
}

// Away is the hooks' read, which fails open: the stored record and whether it
// holds at now. A record that cannot be read safely reads as not away (nil,
// false) with the reason returned, for one line on stderr.
func (s Store) Away(now time.Time) (*Record, bool, error) {
	r, err := s.Read()
	if err != nil {
		return nil, false, err
	}
	return r, r.Holds(now), nil
}

// Update is the store's read-modify-write: under the lock it re-reads the
// record (nil when there is none) and hands it to fn, and writes what fn
// returns. fn returning nil writes nothing; its error is returned and nothing
// is written. Update returns the record written, or the one read when nothing
// was.
func (s Store) Update(fn func(cur *Record) (*Record, error)) (*Record, error) {
	return s.update(s.Read, fn)
}

// update is Update with the read under the lock named.
func (s Store) update(read func() (*Record, error), fn func(cur *Record) (*Record, error)) (*Record, error) {
	var out *Record
	err := s.WithLock(func() error {
		cur, err := read()
		if err != nil {
			return err
		}
		next, err := fn(cur)
		if err != nil {
			return err
		}
		if next == nil {
			out = cur
			return nil
		}
		raw, err := json.MarshalIndent(next, "", "  ")
		if err != nil {
			return err
		}
		if err := s.WriteFile(RecordFile, append(raw, '\n')); err != nil {
			return err
		}
		out = next
		return nil
	})
	return out, err
}

// RecordStop records a stop at now, only if the record re-read inside the
// lock still holds; it reports whether it wrote.
func (s Store) RecordStop(st Stop, now time.Time) (bool, error) {
	wrote := false
	_, err := s.Update(func(cur *Record) (*Record, error) {
		if !cur.Holds(now) {
			return nil, nil
		}
		cur.AddStop(st, now)
		wrote = true
		return cur, nil
	})
	return wrote && err == nil, err
}

// RecordHandBack records an item handed back (its branch and head) at now,
// once per branch and head, only if the record re-read inside the lock still
// holds; it reports whether it wrote.
func (s Store) RecordHandBack(branch, head string, now time.Time) (bool, error) {
	wrote := false
	_, err := s.Update(func(cur *Record) (*Record, error) {
		if !cur.Holds(now) || !cur.AddHandBack(branch, head, now) {
			return nil, nil
		}
		wrote = true
		return cur, nil
	})
	return wrote && err == nil, err
}
