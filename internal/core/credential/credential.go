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
// nothing logs it, and nothing here writes anywhere, least of all the
// repository. A malformed file is refused without echoing a byte of it.
package credential

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/fsutil"
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
	p := filepath.Join(m.home, ".abcd", StoreFileName)
	fi, err := os.Lstat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotSet
		}
		return "", fmt.Errorf("credential: %s could not be examined, so it is not read", StorePath)
	}
	if !fi.Mode().IsRegular() {
		return "", fmt.Errorf("credential: %s is not a regular file (a symlink is never followed), so it is not read", StorePath)
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("credential: %s can be read or written by group or other (mode %04o), so it is not read; `chmod 0600 %s`", StorePath, fi.Mode().Perm(), StorePath)
	}
	// ReadDeclaration re-checks the leaf on its own descriptor and refuses a
	// file this uid does not own.
	raw, refusal, err := fsutil.ReadDeclaration(p, maxStoreBytes)
	switch {
	case refusal == fsutil.DeclarationAbsent && errors.Is(err, os.ErrNotExist):
		return "", ErrNotSet
	case refusal == fsutil.DeclarationForeignOwner:
		return "", fmt.Errorf("credential: %s is not owned by you, so it is not read", StorePath)
	case err != nil:
		return "", fmt.Errorf("credential: %s could not be read safely (mode 0600, owned by you, a regular file), so it is not read", StorePath)
	}
	// A repeated key, or a case twin encoding/json binds to the same entry, is
	// read last-wins by the decoder, so a store naming one credential twice would
	// resolve silently to whichever spelling came last. It is refused, and the
	// refusal names neither spelling: a key is file content too
	// (iss-2609260120380520).
	if err := jsonstrict.NoDuplicateKeys(raw); err != nil {
		return "", fmt.Errorf("credential: %s names one credential twice (a repeated key, or two spellings of one), so it is not read", StorePath)
	}
	var store map[string]string
	if err := json.Unmarshal(raw, &store); err != nil {
		// The decoder's message can quote the file's bytes; it is dropped.
		return "", fmt.Errorf("credential: %s is not a JSON object of names to strings", StorePath)
	}
	v := store[name]
	if v == "" {
		return "", ErrNotSet
	}
	return v, nil
}
