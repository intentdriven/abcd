package credential

// store.go is the credential store proper (itd-2609221017023290,
// adr-2609221017021499): one reader, Store(home).Resolve, and one write, Set,
// over the three homes a person chooses among once per credential.
//
//   - abcd: the owner-only ~/.abcd/credentials.json (credential.go), which
//     holds the value itself.
//   - keychain: the platform's keychain, under abcd's service name
//     (keychain.go); the index holds only that the name lives there.
//   - external: a pointer at a setup outside abcd, an environment variable or
//     a named field of a tool's JSON configuration (external.go); the index
//     holds the pointer, never the value.
//
// The index, ~/.abcd/credential-homes.json, maps a name to its keychain or
// external home. A name the index does not hold resolves from the abcd home,
// so a store written before the index existed reads unchanged. A name held in
// both is ambiguous and refused, never guessed.
//
// The index is the one file the store writes that must never hold a secret,
// so the secret scanner reads its bytes before they are written, and a
// finding refuses the write. No home is written inside a git working tree.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The homes a credential lives in (adr-2609221017021499 ruling 2).
const (
	// HomeExternal is a setup outside abcd; the store holds the pointer.
	HomeExternal = "external"
	// HomeABCD is abcd-only: the owner-only ~/.abcd/credentials.json.
	HomeABCD = "abcd"
	// HomeKeychain is the platform keychain.
	HomeKeychain = "keychain"
)

// Homes returns the homes in the order every walkthrough offers them. None is
// marked: the keychain is recommended in HomesProse, never in the list.
func Homes() []string { return []string{HomeExternal, HomeABCD, HomeKeychain} }

// IndexFileName is the index under ~/.abcd/, and IndexPath its tilde form.
const (
	IndexFileName = "credential-homes.json"
	IndexPath     = "~/.abcd/" + IndexFileName
)

// Walkthrough is the command that explains a credential and stores it, named
// by every refusal of a name that is not set.
func Walkthrough(name string) string { return "abcd ahoy credential " + name }

// Pointer is an external home's pointer: an environment variable, or a field
// of a tool's JSON configuration file under the home directory.
type Pointer struct {
	Env   string `json:"env,omitempty"`
	File  string `json:"file,omitempty"`
	Field string `json:"field,omitempty"`
}

// Choice is one home chosen for one credential: the value for the abcd and
// keychain homes, the pointer for the external one.
type Choice struct {
	Home    string
	Value   string
	Pointer Pointer
}

// indexEntry is one name's line in the index.
type indexEntry struct {
	Home string `json:"home"`
	Pointer
}

// Store is the one reader every adapter resolves a credential through, by
// name, rooted at home (the caller's home directory). An empty home resolves
// every name to ErrNotSet.
func Store(home string) Source { return store{home: home} }

// UserStore is Store at the caller's home directory.
func UserStore() Source {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return Store(home)
}

type store struct{ home string }

// notSetError is ErrNotSet naming the walkthrough.
type notSetError struct{ name, why string }

func (e notSetError) Error() string {
	why := ""
	if e.why != "" {
		why = " (" + e.why + ")"
	}
	return fmt.Sprintf("credential %s is not set on this machine%s; `%s` explains what it unlocks and stores it", e.name, why, Walkthrough(e.name))
}

func (notSetError) Is(target error) bool { return target == ErrNotSet }

func (s store) Resolve(name string) (string, error) {
	if !nameRe.MatchString(name) {
		return "", errors.New("credential: the name is not a plain credential name")
	}
	if s.home == "" {
		return "", notSetError{name: name}
	}
	idx, err := readIndex(s.home)
	if err != nil {
		return "", err
	}
	v, err := Machine(s.home).Resolve(name)
	e, indexed := idx[name]
	switch {
	case err != nil && !errors.Is(err, ErrNotSet):
		return "", err
	case !indexed && err != nil:
		return "", notSetError{name: name}
	case !indexed:
		return v, nil
	case err == nil:
		return "", fmt.Errorf("credential: %s is named in two homes, %s and the %s home in %s, so it is not read; remove one by hand",
			name, StorePath, e.Home, IndexPath)
	}
	switch e.Home {
	case HomeKeychain:
		return keychainLookup(name)
	case HomeExternal:
		return resolvePointer(s.home, name, e.Pointer)
	}
	return "", fmt.Errorf("credential: %s names an unknown home for %s, so it is not read", IndexPath, name)
}

// Where names the home that holds name ("" when none does), reading the index
// and the abcd home and never a value: a surface shows presence and the home,
// and this is the home. Presence is Resolve.
func Where(home, name string) (string, error) {
	if !nameRe.MatchString(name) {
		return "", errors.New("credential: the name is not a plain credential name")
	}
	if home == "" {
		return "", nil
	}
	idx, err := readIndex(home)
	if err != nil {
		return "", err
	}
	if e, ok := idx[name]; ok {
		return e.Home, nil
	}
	if _, err := Machine(home).Resolve(name); err == nil {
		return HomeABCD, nil
	} else if !errors.Is(err, ErrNotSet) {
		return "", err
	}
	return "", nil
}

// Set stores a credential under name in the chosen home: the one write every
// setup goes through (the walkthrough, Walk, is its only caller outside this
// package's tests). It refuses, before writing anything and never echoing a
// value: a home inside a git working tree; a name another home already holds,
// or a different value in the same home, because a stored secret is never
// replaced unasked; a keychain on a platform without one; and a pointer that
// does not resolve. The same value again is no change.
func Set(home, name string, c Choice) (changed bool, err error) {
	if !nameRe.MatchString(name) {
		return false, errors.New("credential: the name is not a plain credential name (lower case letters, digits, '.', '_' and '-')")
	}
	if home == "" {
		return false, errors.New("credential: the home directory is unresolved, so there is nowhere to keep the credential")
	}
	switch c.Home {
	case HomeABCD, HomeKeychain:
		if err := CheckValue(c.Value); err != nil {
			return false, err
		}
		if c.Pointer != (Pointer{}) {
			return false, fmt.Errorf("credential: the %s home keeps a value, and a pointer was given", c.Home)
		}
	case HomeExternal:
		if c.Value != "" {
			return false, errors.New("credential: the external home keeps a pointer, never a value, and a value was given")
		}
		if err := checkPointer(c.Pointer); err != nil {
			return false, err
		}
	default:
		return false, fmt.Errorf("credential: home %q is not one of external, abcd, keychain", boundHome(c.Home))
	}
	dir := filepath.Join(home, ".abcd")
	if tree := workingTreeAbove(dir); tree != "" {
		return false, fmt.Errorf("credential: ~/.abcd lies inside a git working tree, where a commit could carry the credential, so nothing was written")
	}
	held, err := Where(home, name)
	if err != nil {
		return false, err
	}
	if held != "" && held != c.Home {
		return false, fmt.Errorf("credential: %s is already held in the %s home, and abcd never replaces a stored secret; remove it there by hand to choose another home", name, held)
	}
	switch c.Home {
	case HomeABCD:
		return SetMachine(home, name, c.Value)
	case HomeKeychain:
		if held == HomeKeychain {
			return sameOrRefuse(home, name, c.Value)
		}
		if _, err := locateKeychain(); err != nil {
			return false, err
		}
		// The index is judged before the keychain is touched, so a refusal
		// leaves nothing in either.
		if err := setIndex(home, name, indexEntry{Home: HomeKeychain}, true); err != nil {
			return false, err
		}
		// An item a failed earlier write left behind is adopted when it holds
		// this value and refused when it holds another.
		switch stored, err := keychainLookup(name); {
		case err == nil && stored != c.Value:
			return false, fmt.Errorf("credential: the keychain already holds a value for %s, and abcd never replaces a stored secret; remove that item by hand to store a new one", name)
		case err == nil:
		case !errors.Is(err, ErrNotSet):
			return false, err
		default:
			if err := keychainStore(name, c.Value); err != nil {
				return false, err
			}
		}
		if err := setIndex(home, name, indexEntry{Home: HomeKeychain}, false); err != nil {
			return false, fmt.Errorf("%w; the keychain holds the item, and the next write of the same value records it", err)
		}
		return true, nil
	}
	if _, err := resolvePointer(home, name, c.Pointer); err != nil {
		return false, err
	}
	if held == HomeExternal {
		return false, samePointer(home, name, c.Pointer)
	}
	if err := setIndex(home, name, indexEntry{Home: HomeExternal, Pointer: c.Pointer}, false); err != nil {
		return false, err
	}
	return true, nil
}

// samePointer refuses a pointer other than the one the index holds for name.
func samePointer(home, name string, p Pointer) error {
	idx, err := readIndex(home)
	if err != nil {
		return err
	}
	if idx[name].Pointer == p {
		return nil
	}
	return fmt.Errorf("credential: %s already points elsewhere in %s, and abcd never replaces a stored pointer; remove that entry by hand", name, IndexPath)
}

// sameOrRefuse is a second Set of a name already in the keychain: the same
// value is no change, a different one is refused.
func sameOrRefuse(home, name, value string) (bool, error) {
	stored, err := Store(home).Resolve(name)
	if err != nil {
		return false, err
	}
	if stored == value {
		return false, nil
	}
	return false, fmt.Errorf("credential: the keychain already holds a value for %s, and abcd never replaces a stored secret; remove that item by hand to store a new one", name)
}

// workingTreeAbove returns the first directory at or above dir that carries a
// .git entry, or "" when none does.
func workingTreeAbove(dir string) string {
	d := filepath.Clean(dir)
	for {
		if _, err := os.Lstat(filepath.Join(d, ".git")); err == nil {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}
		d = parent
	}
}

func boundHome(h string) string {
	if len(h) > 16 {
		return h[:16] + "…"
	}
	return h
}

// maxIndexBytes bounds the index read.
const maxIndexBytes = 64 << 10

// readIndex reads the index under the same refusals as the abcd home: a
// regular file, owned by the caller, owner-only, naming each credential once,
// every entry a known home. An absent index is empty.
func readIndex(home string) (map[string]indexEntry, error) {
	p := filepath.Join(home, ".abcd", IndexFileName)
	fi, err := os.Lstat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]indexEntry{}, nil
		}
		return nil, fmt.Errorf("credential: %s could not be examined, so it is not read", IndexPath)
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("credential: %s is not a regular file (a symlink is never followed), so it is not read", IndexPath)
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("credential: %s can be read or written by group or other (mode %04o), so it is not read; `chmod 0600 %s`", IndexPath, fi.Mode().Perm(), IndexPath)
	}
	raw, refusal, err := fsutil.ReadDeclaration(p, maxIndexBytes)
	switch {
	case refusal == fsutil.DeclarationAbsent && errors.Is(err, os.ErrNotExist):
		return map[string]indexEntry{}, nil
	case refusal == fsutil.DeclarationForeignOwner:
		return nil, fmt.Errorf("credential: %s is not owned by you, so it is not read", IndexPath)
	case err != nil:
		return nil, fmt.Errorf("credential: %s could not be read safely (mode 0600, owned by you, a regular file), so it is not read", IndexPath)
	}
	var idx map[string]indexEntry
	if err := jsonstrict.Decode(raw, &idx); err != nil || idx == nil {
		// The decoder's message can quote the file's bytes; it is dropped.
		return nil, fmt.Errorf("credential: %s is not a JSON object naming each credential once with its home, so it is not read", IndexPath)
	}
	for name, e := range idx {
		ok := nameRe.MatchString(name)
		switch e.Home {
		case HomeKeychain:
			ok = ok && e.Pointer == (Pointer{})
		case HomeExternal:
			ok = ok && checkPointer(e.Pointer) == nil
		default:
			ok = false
		}
		if !ok {
			return nil, fmt.Errorf("credential: %s holds an entry that is not a plain name with a keychain home or a well-formed external pointer, so it is not read", IndexPath)
		}
	}
	return idx, nil
}

// indexLockFileName is the lock every writer of the index takes, beside it.
const indexLockFileName = "." + IndexFileName + ".lock"

// indexLockTimeout bounds the wait for another writer of the index.
var indexLockTimeout = 5 * time.Second

// setIndex adds e under name to the index, or, with dryRun, judges the write
// (the scanner included) without making it. The read, the scan and the write
// hold the index's lock.
func setIndex(home, name string, e indexEntry, dryRun bool) error {
	dir := filepath.Join(home, ".abcd")
	if !dryRun {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return errors.New("credential: ~/.abcd could not be created, so nothing was written")
		}
	}
	write := func() error {
		idx, err := readIndex(home)
		if err != nil {
			return err
		}
		if old, ok := idx[name]; ok && old != e {
			return fmt.Errorf("credential: %s already names a home for %s, and abcd never replaces it; remove that entry by hand", IndexPath, name)
		}
		idx[name] = e
		body, err := json.MarshalIndent(idx, "", "  ")
		if err != nil {
			return errors.New("credential: the index could not be encoded")
		}
		body = append(body, '\n')
		if err := scanIndex(body); err != nil {
			return err
		}
		if dryRun {
			return nil
		}
		if err := fsutil.WriteFileAtomic(filepath.Join(dir, IndexFileName), body, 0o600); err != nil {
			return fmt.Errorf("credential: %s could not be written, so the credential's home was not recorded", IndexPath)
		}
		return nil
	}
	if dryRun {
		return write()
	}
	err := fsutil.WithFileLock(filepath.Join(dir, indexLockFileName), indexLockTimeout, write)
	switch {
	case errors.Is(err, fsutil.ErrLockContention):
		return fmt.Errorf("credential: %s is being written by another abcd, so nothing was written; retry", IndexPath)
	case errors.Is(err, fsutil.ErrLockPathUnsafe):
		return fmt.Errorf("credential: the lock ~/.abcd/%s is not a regular file (a symlink, or something else), so it is refused and nothing was written; remove it, and the next write creates it afresh", indexLockFileName)
	}
	return err
}

// scanIndex runs the secret scanner over the index's bytes before they are
// written (adr-2609221017021499 ruling 4). The index holds names, homes and
// pointers, never a value, so any finding refuses the write, and the refusal
// names the kind and never the match.
func scanIndex(body []byte) error {
	findings := scanner.ScanText(string(body), scanner.Identity{}, scanner.DefaultPatterns(), nil, IndexFileName)
	if len(findings) == 0 {
		return nil
	}
	return fmt.Errorf("credential: the secret scanner found a %s in what %s would hold, which holds names and pointers only, so nothing was written",
		findings[0].Kind, IndexPath)
}

// envNameRe is an environment variable's name as a pointer names it.
var envNameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,127}$`)
