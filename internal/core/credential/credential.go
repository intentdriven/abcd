// Package credential is the credential store (itd-2609221017023290,
// adr-2609221017021499): the one reader every adapter resolves an external
// credential through, by NAME (Store, store.go), the one write (Set), and the
// walkthrough that chooses a credential's home (Walk, walk.go). This file is
// the abcd home: one machine-scoped file, ~/.abcd/credentials.json, a JSON
// object mapping a credential name to its value. The keychain and external
// homes are keychain.go and external.go.
//
// The file is refused, loudly and never treated as absent, unless it is a
// regular file (not a symlink), owned by the caller, and readable and writable
// by the owner alone (mode 0600 or tighter), in a ~/.abcd that is not itself a
// symlink: a secret that group or other can read is not kept, one that
// somebody else wrote is not the caller's, and one behind a symlinked ~/.abcd
// lives in whatever the link points at.
//
// The value never leaves Resolve except as its return: no error formats it,
// nothing logs it, and nothing here writes to the repository. The one write is
// SetMachine, into this same file, which Set calls for the abcd home. A
// malformed file is refused without echoing a byte of it.
package credential

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// StoreFileName is the abcd home's file under ~/.abcd/.
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

// Machine is the abcd home alone, rooted at home (the caller's home
// directory). An empty home resolves every name to ErrNotSet. Every reader
// outside this package resolves through Store, which reads this home among
// the three; a test refuses any other.
func Machine(home string) Source { return machine{home: home} }

type machine struct{ home string }

// StorePath is where the abcd home lives, displayed with ~ so no
// developer-identity path reaches output.
const StorePath = "~/.abcd/" + StoreFileName

// storeRel is the store's place in the home, in the slash form the
// home-scoped primitives take.
const storeRel = ".abcd/" + StoreFileName

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
//
// Every guard is judged by fsutil.ReadHomeDeclarationDenying on the store it
// reads, never on a path first: absence on the Lstat that decides it (a
// symlinked ~/.abcd holding no store is no store), a store behind a symlinked
// ~/.abcd — which sits wherever the link points, a dotfiles checkout
// typically, and is refused as the rules loader refuses a rules.json there —
// on the descriptor walk of ~/.abcd, and the leaf's type, owner and mode on
// the opened file's own fstat. A mode judged by path would vouch for a file
// other than the one read: a store swapped for a group-readable file after
// that check would be read once (iss-2609281310017733).
func readStore(home string) (map[string]string, error) {
	raw, refusal, err := fsutil.ReadHomeDeclarationDenying(home, storeRel, maxStoreBytes, 0o077)
	return decodeStore(raw, refusal, err)
}

// readStoreIn is readStore through dir, ~/.abcd as SetMachine's walk opened
// it. The writer under the store's lock reads the store here, through the
// directory it writes through, never by walking ~/.abcd again: a same-uid swap
// of ~/.abcd between the two walks would otherwise read one directory's
// entries and write them, with the new value, into the other
// (iss-2609290300313698).
func readStoreIn(home string, dir *os.Root) (map[string]string, error) {
	raw, refusal, err := fsutil.ReadHomeDeclarationDenyingIn(dir, home, storeRel, maxStoreBytes, 0o077)
	return decodeStore(raw, refusal, err)
}

// decodeStore judges one read of the store and decodes what it read.
func decodeStore(raw []byte, refusal fsutil.DeclarationRefusal, err error) (map[string]string, error) {
	var mode *fsutil.DeclarationModeError
	switch {
	case refusal == fsutil.DeclarationOK:
	case refusal == fsutil.DeclarationAbsent && errors.Is(err, os.ErrNotExist):
		return map[string]string{}, nil
	case refusal == fsutil.DeclarationAbsent:
		return nil, fmt.Errorf("credential: %s could not be examined, so it is not read", StorePath)
	case refusal == fsutil.DeclarationBehindSymlink:
		return nil, fmt.Errorf("credential: %s is not read: %v", StorePath, err)
	case refusal == fsutil.DeclarationNotRegular:
		return nil, fmt.Errorf("credential: %s is not a regular file (a symlink is never followed), so it is not read", StorePath)
	case refusal == fsutil.DeclarationExposed && errors.As(err, &mode):
		return nil, fmt.Errorf("credential: %s can be read or written by group or other (mode %04o), so it is not read; `chmod 0600 %s`", StorePath, uint32(mode.Perm), StorePath)
	case refusal == fsutil.DeclarationWritableByOthers:
		return nil, fmt.Errorf("credential: %s can be written by group or other, so it is not read; `chmod 0600 %s`", StorePath, StorePath)
	case refusal == fsutil.DeclarationForeignOwner:
		return nil, fmt.Errorf("credential: %s is not owned by you, so it is not read", StorePath)
	default:
		return nil, fmt.Errorf("credential: %s could not be read safely (mode 0600, owned by you, a regular file), so it is not read", StorePath)
	}
	// A repeated key, or a case twin encoding/json binds to the same entry, is
	// read last-wins by the decoder, so a store naming one credential twice would
	// resolve silently to whichever spelling came last. It is refused, and the
	// refusal names neither spelling: a key is file content too
	// (iss-2609260120380520).
	if err := jsonstrict.NoDuplicateKeys(raw); err != nil {
		return nil, fmt.Errorf("credential: %s names one credential twice (a repeated key, or two spellings of one), so it is not read", StorePath)
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

// SetMachine writes value under name in the abcd home at home
// (~/.abcd/credentials.json): the abcd home's write, which Set makes for it
// and no reader outside this package calls.
//
// It refuses, before writing anything and without echoing either value: a
// name that is not plain; a value that is empty, longer than MaxValueBytes,
// padded with white space or carrying a control, bidirectional or zero-width
// character; a store Resolve would refuse (a symlink, group- or other-
// readable, not owned by the caller, malformed), so a write never launders an
// unsafe file; a ~/.abcd that is a symlink, because the secret would land
// wherever the link points (fsutil.EnsureHomeScope); and a name already holding a
// different value, because a stored secret is never replaced by a second one
// unasked. The same value already
// stored is no change (changed is false). The file is written atomically at
// mode 0600, and ~/.abcd is created owner-only when it is absent. The read,
// the change and the write hold the store's lock (fsutil.WithFileLockIn, beside
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
	// A ~/.abcd symlinked into a dotfiles checkout would carry the secret into
	// that repository, and the store's own read refuses a file behind the link
	// (iss-2609260958587561). Refused before anything is created, the lock
	// included.
	// ~/.abcd is created, judged and opened in one walk relative to the
	// descriptor of home (fsutil.EnsureHomeScope), and the lock and the store
	// are reached through that descriptor, so a link swapped in after the
	// judgement is refused rather than written through (iss-2609281310017733).
	dir, err := fsutil.EnsureHomeScope(home, ".abcd", 0o700)
	if errors.Is(err, fsutil.ErrHomeScopeSymlinked) {
		return false, fmt.Errorf("credential: nothing was written to %s: %v", StorePath, err)
	}
	if err != nil {
		return false, fmt.Errorf("credential: ~/.abcd could not be created, so nothing was written")
	}
	defer dir.Close()
	// The store is read, changed and renamed into place, so a second writer
	// between the read and the rename would lose this entry or its own; the
	// write holds the store's lock across all three.
	err = fsutil.WithFileLockIn(dir, storeLockFileName, storeLockTimeout, func() error {
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
// lock, the read and the write both through dir.
func setLocked(home string, dir *os.Root, name, value string) (bool, error) {
	store, err := readStoreIn(home, dir)
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
	if err := fsutil.WriteFileAtomicInRoot(dir, StoreFileName, append(body, '\n'), 0o600); err != nil {
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
