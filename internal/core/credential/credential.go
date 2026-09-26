// Package credential is the one reader every adapter resolves an external
// credential through, by NAME.
//
// This is the interim source itd-2609061543533170 ruled for `abcd site setup`
// (its `## Decisions`, 2026-09-25): the credential store proper, with its three
// homes and its walkthrough, is itd-2609221017023290, which is planned and not
// built. Until it lands, a credential is read from one machine-scoped file,
// ~/.abcd/credentials.json, a JSON object mapping a credential name to its
// value. The successor replaces the source behind Source; no reader changes.
//
// The file is refused, loudly and never treated as absent, unless it is a
// regular file (not a symlink), owned by the caller, and readable and writable
// by the owner alone (mode 0600 or tighter): a secret that group or other can
// read is not kept, and one that somebody else wrote is not the caller's.
//
// The value never leaves Resolve except as its return: no error formats it,
// nothing logs it, and nothing here writes to the repository. The one write is
// SetMachine, into this same file, for the setup of the OpenAI-compatible API
// adapter (itd-2609081951381895). A malformed file is refused without echoing
// a byte of it.
package credential

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// StoreFileName is the interim store's file under ~/.abcd/.
const StoreFileName = "credentials.json"

// maxStoreBytes bounds the store read.
const maxStoreBytes = 64 << 10

// ErrNotSet is a credential that resolves to nothing: no store, no entry, or an
// empty value. It is the one error a caller may treat as "carry on without it".
var ErrNotSet = errors.New("credential not set")

// Source resolves a credential by name.
type Source interface {
	Resolve(name string) (string, error)
}

// nameRe is the credential-name charset. A name is looked up, never used as a
// path, but it is printed in refusals, so it is held to plain characters.
var nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// ValidName reports whether name is a plain credential name, the shape every
// source resolves, so a configuration that names a credential is checked when
// it is read rather than at the first call.
func ValidName(name string) bool { return nameRe.MatchString(name) }

// Machine is the interim machine-scoped source rooted at home (the caller's
// home directory). An empty home resolves every name to ErrNotSet.
func Machine(home string) Source { return machine{home: home} }

// UserMachine is Machine at the caller's home directory.
func UserMachine() Source {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return Machine(home)
}

type machine struct{ home string }

// StorePath is where the interim store lives, displayed with ~ so no
// developer-identity path reaches output.
const StorePath = "~/.abcd/" + StoreFileName

func (m machine) Resolve(name string) (string, error) {
	if !nameRe.MatchString(name) {
		return "", errors.New("credential: the name is not a plain credential name")
	}
	if m.home == "" {
		return "", ErrNotSet
	}
	store, err := readStore(m.home)
	if err != nil {
		return "", err
	}
	v := store[name]
	if v == "" {
		return "", ErrNotSet
	}
	return v, nil
}

// readStore reads the store at home under every refusal the package doc
// names. An absent store is an empty map and no error.
func readStore(home string) (map[string]string, error) {
	p := filepath.Join(home, ".abcd", StoreFileName)
	fi, err := os.Lstat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("credential: %s could not be examined, so it is not read", StorePath)
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("credential: %s is not a regular file (a symlink is never followed), so it is not read", StorePath)
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("credential: %s can be read or written by group or other (mode %04o), so it is not read; `chmod 0600 %s`", StorePath, fi.Mode().Perm(), StorePath)
	}
	// ReadDeclaration re-checks the leaf on its own descriptor and refuses a
	// file this uid does not own.
	raw, refusal, err := fsutil.ReadDeclaration(p, maxStoreBytes)
	switch {
	case refusal == fsutil.DeclarationAbsent && errors.Is(err, os.ErrNotExist):
		return map[string]string{}, nil
	case refusal == fsutil.DeclarationForeignOwner:
		return nil, fmt.Errorf("credential: %s is not owned by you, so it is not read", StorePath)
	case err != nil:
		return nil, fmt.Errorf("credential: %s could not be read safely (mode 0600, owned by you, a regular file), so it is not read", StorePath)
	}
	var store map[string]string
	if err := json.Unmarshal(raw, &store); err != nil || store == nil {
		// The decoder's message can quote the file's bytes; it is dropped.
		return nil, fmt.Errorf("credential: %s is not a JSON object of names to strings", StorePath)
	}
	return store, nil
}

// MaxValueBytes bounds one credential's value.
const MaxValueBytes = 4096

// SetMachine writes value under name in the interim store at home
// (~/.abcd/credentials.json): the one write this package makes, for the one
// home it reads (itd-2609081951381895's setup; the credential store,
// itd-2609221017023290, brings the other homes and replaces this backing).
//
// It refuses, before writing anything and without echoing either value: a
// name that is not plain; a value that is empty, longer than MaxValueBytes,
// padded with white space or carrying a control, bidirectional or zero-width
// character; a store Resolve would refuse (a symlink, group- or other-
// readable, not owned by the caller, malformed), so a write never launders an
// unsafe file; and a name already holding a different value, because a stored
// secret is never replaced by a second one unasked. The same value already
// stored is no change (changed is false). The file is written atomically at
// mode 0600, and ~/.abcd is created owner-only when it is absent. The read,
// the change and the write hold the store's lock (fsutil.WithFileLock, beside
// the store), so concurrent writers never lose each other's entries.
func SetMachine(home, name, value string) (changed bool, err error) {
	if !nameRe.MatchString(name) {
		return false, errors.New("credential: the name is not a plain credential name (lower case letters, digits, '.', '_' and '-')")
	}
	if home == "" {
		return false, errors.New("credential: the home directory is unresolved, so there is nowhere to keep the credential")
	}
	if err := CheckValue(value); err != nil {
		return false, err
	}
	dir := filepath.Join(home, ".abcd")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return false, fmt.Errorf("credential: ~/.abcd could not be created, so nothing was written")
	}
	// The store is read, changed and renamed into place, so a second writer
	// between the read and the rename would lose this entry or its own; the
	// write holds the store's lock across all three.
	err = fsutil.WithFileLock(filepath.Join(dir, storeLockFileName), storeLockTimeout, func() error {
		var werr error
		changed, werr = setLocked(home, dir, name, value)
		return werr
	})
	switch {
	case errors.Is(err, fsutil.ErrLockContention):
		return false, fmt.Errorf("credential: %s is being written by another abcd, so nothing was written; retry", StorePath)
	case errors.Is(err, fsutil.ErrLockPathUnsafe):
		// A retry cannot cure a symlinked or non-regular lock, so the
		// refusal names it rather than reading as contention.
		return false, fmt.Errorf("credential: the lock ~/.abcd/%s is not a regular file (a symlink, or something else), so it is refused and nothing was written; remove it, and the next write creates it afresh", storeLockFileName)
	}
	return changed, err
}

// storeLockFileName is the lock every writer of the store takes, beside it.
const storeLockFileName = "." + StoreFileName + ".lock"

// storeLockTimeout bounds the wait for another writer of the store.
var storeLockTimeout = 5 * time.Second

// setLocked is SetMachine's read, change and write, run under the store's
// lock.
func setLocked(home, dir, name, value string) (bool, error) {
	store, err := readStore(home)
	if err != nil {
		return false, err
	}
	switch store[name] {
	case value:
		return false, nil
	case "":
	default:
		return false, fmt.Errorf("credential: %s already holds a value for %s, and abcd never replaces a stored secret; "+
			"remove that entry by hand to store a new one", StorePath, name)
	}
	store[name] = value
	body, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return false, errors.New("credential: the store could not be encoded")
	}
	if err := fsutil.WriteFileAtomic(filepath.Join(dir, StoreFileName), append(body, '\n'), 0o600); err != nil {
		return false, fmt.Errorf("credential: %s could not be written, so the credential was not stored", StorePath)
	}
	return true, nil
}

// CheckValue refuses a value SetMachine would refuse, without echoing it, so a
// caller can refuse before any other work.
func CheckValue(v string) error {
	switch {
	case v == "":
		return errors.New("credential: the value is empty")
	case len(v) > MaxValueBytes:
		return fmt.Errorf("credential: the value is longer than %d bytes", MaxValueBytes)
	case strings.TrimSpace(v) != v:
		return errors.New("credential: the value begins or ends with white space, which a key never does")
	case termsafe.Sanitize(v) != v || !utf8.ValidString(v):
		return errors.New("credential: the value carries a control, bidirectional or zero-width character")
	}
	return nil
}
