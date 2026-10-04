package credential

// store.go is the credential store proper (itd-2609221017023290,
// adr-2609221017021499): one reader, Store(home).Resolve, and one write, Set,
// over the three homes a person chooses among once per credential.
//
//   - abcd: the owner-only ~/.abcd.noindex/credentials.json (credential.go), which
//     holds the value itself.
//   - keychain: the platform's keychain, under abcd's service name
//     (keychain.go); the index holds only that the name lives there.
//   - external: a pointer at a setup outside abcd, an environment variable or
//     a named field of a tool's JSON configuration (external.go); the index
//     holds the pointer, never the value.
//
// The index, ~/.abcd.noindex/credential-homes.json, maps a name to its keychain or
// external home. A name the index does not hold resolves from the abcd home,
// so a store written before the index existed reads unchanged. A name held in
// both is ambiguous and refused, never guessed.
//
// The index is the one file the store writes that must never hold a secret,
// so the secret scanner reads its bytes before they are written, and a
// finding refuses the write. The abcd home, the one that keeps a value under
// ~/.abcd.noindex, is never written inside a git working tree.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The homes a credential lives in (adr-2609221017021499 ruling 2).
const (
	// HomeExternal is a setup outside abcd; the store holds the pointer.
	HomeExternal = "external"
	// HomeABCD is abcd-only: the owner-only ~/.abcd.noindex/credentials.json.
	HomeABCD = "abcd"
	// HomeKeychain is the platform keychain.
	HomeKeychain = "keychain"
)

// Homes returns the homes in the order every walkthrough offers them. None is
// marked: the keychain is recommended in HomesProse, never in the list.
func Homes() []string { return []string{HomeExternal, HomeABCD, HomeKeychain} }

// IndexFileName is the index under ~/.abcd.noindex/, and IndexPath its tilde form.
const IndexFileName = "credential-homes.json"

var IndexPath = abcdhome.Display(IndexFileName)

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
// package's tests). It refuses, before writing a value or an index entry and
// never echoing a value: a ~/.abcd.noindex that is a symlink, in every home, before
// anything is created; the abcd home inside a git working tree; a name
// another home already holds, or a different value in the same home, because a
// stored secret is never replaced unasked; a keychain on a platform without
// one; and a pointer that does not resolve. The same value again is no change.
// The whole write, from reading where the name is held to the last write,
// holds the index's lock, so concurrent Sets of one name cannot land in two
// homes.
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
	// A ~/.abcd.noindex that is a symlink (into a dotfiles repository, say) is
	// refused first, in every home, before anything is created: the value,
	// the index and both locks would land wherever the link points, and a
	// working-tree check of the lexical path below cannot see a repository
	// the link leads into. It is the rule every other reader and writer of
	// ~/.abcd.noindex applies (fsutil.HomeScopeLink); the walk below holds it against
	// a race.
	if err := fsutil.HomeScopeLink(home, indexRel); err != nil {
		return false, fmt.Errorf("credential: nothing was written: %v", err)
	}
	// The abcd home is the one home that writes a value under ~/.abcd.noindex, so it
	// alone is refused inside a git working tree. The keychain keeps its value
	// outside the home, and the index holds names and pointers only, scanned
	// before every write, so a home directory that is itself a working tree (a
	// dotfiles repository) keeps those homes.
	if c.Home == HomeABCD && workingTreeAbove(home, abcdhome.Rel()) != "" {
		return false, errors.New("credential: " + abcdhome.Display() + " lies inside a git working tree, where a commit could carry the credential, so the abcd home is refused and nothing was written; choose the keychain or an external home")
	}
	// ~/.abcd.noindex is created, judged and opened in one walk relative to the
	// descriptor of home (fsutil.EnsureHomeScope), and the index's lock and
	// its write are reached through that descriptor, so a link swapped in
	// after the judgement is refused rather than written through
	// (iss-2609281310017733).
	dir, err := fsutil.EnsureHomeScope(home, abcdhome.Rel(), 0o700)
	if errors.Is(err, fsutil.ErrHomeScopeSymlinked) {
		return false, fmt.Errorf("credential: nothing was written: %v", err)
	}
	if err != nil {
		return false, errors.New("credential: " + abcdhome.Display() + " could not be created, so nothing was written")
	}
	defer dir.Close()
	// One lock, the index's, is held across the whole write: where the name
	// is held, the value's write and the index's. Two Sets of one name to two
	// homes therefore cannot both land. The abcd home's own lock (SetMachine)
	// is taken inside this one, always in that order; no writer takes the two
	// the other way round.
	err = fsutil.WithFileLockIn(dir, indexLockFileName, indexLockTimeout, func() error {
		var werr error
		changed, werr = setUnderLock(home, dir, name, c)
		return werr
	})
	switch {
	case errors.Is(err, fsutil.ErrLockContention):
		return false, fmt.Errorf("credential: %s is being written by another abcd, so nothing was written; retry", IndexPath)
	case errors.Is(err, fsutil.ErrLockPathUnsafe):
		return false, fmt.Errorf("credential: the lock %s is not a regular file (a symlink, or something else), so it is refused and nothing was written; remove it, and the next write creates it afresh", abcdhome.Display(indexLockFileName))
	}
	return changed, err
}

// setUnderLock is Set's read of where name is held and its write, run under
// the index's lock. dir is ~/.abcd.noindex as Set's walk opened it; the index is
// written through it.
func setUnderLock(home string, dir *os.Root, name string, c Choice) (bool, error) {
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
		if err := setIndex(home, dir, name, indexEntry{Home: HomeKeychain}, true); err != nil {
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
		if err := setIndex(home, dir, name, indexEntry{Home: HomeKeychain}, false); err != nil {
			return false, fmt.Errorf("%w; the keychain holds the item, and the next write of the same value records it", err)
		}
		return true, nil
	}
	if _, err := resolvePointer(home, name, c.Pointer); err != nil {
		return false, err
	}
	if held == HomeExternal {
		return false, samePointer(home, dir, name, c.Pointer)
	}
	if err := setIndex(home, dir, name, indexEntry{Home: HomeExternal, Pointer: c.Pointer}, false); err != nil {
		return false, err
	}
	return true, nil
}

// samePointer refuses a pointer other than the one the index holds for name.
// Under the index's lock dir is the ~/.abcd.noindex Set holds and the index is read
// through it; a caller holding no directory (Walk's check before any write)
// passes nil and the index is read by walking ~/.abcd.noindex.
func samePointer(home string, dir *os.Root, name string, p Pointer) error {
	read := func() (map[string]indexEntry, error) { return readIndex(home) }
	if dir != nil {
		read = func() (map[string]indexEntry, error) { return readIndexIn(home, dir) }
	}
	idx, err := read()
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

// workingTreeAbove returns the first directory at or above home/rel that
// carries a .git entry, or "" when none does. The place is judged twice: by
// its lexical path, and with home replaced by where home resolves
// (filepath.EvalSymlinks), so a home directory that is itself a symlink into
// a checkout (~ -> <repo>/home) is seen as lying inside it
// (iss-2609290259108077). Home is never refused for being a link; only the
// working tree it leads into is judged. A home that does not resolve is
// judged by its lexical path alone: nothing can then be written under it.
func workingTreeAbove(home, rel string) string {
	if tree := gitEntryAbove(filepath.Join(home, rel)); tree != "" {
		return tree
	}
	real, err := filepath.EvalSymlinks(home)
	if err != nil || real == filepath.Clean(home) {
		return ""
	}
	return gitEntryAbove(filepath.Join(real, rel))
}

// gitEntryAbove returns the first directory at or above dir, climbed
// lexically, that carries a .git entry, or "" when none does.
func gitEntryAbove(dir string) string {
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

// indexRel is the index's place in the home, in the slash form the
// home-scoped primitives take.
var indexRel = abcdhome.Rel(IndexFileName)

// readIndex reads the index under the same refusals as the abcd home: a
// regular file, owned by the caller, owner-only, naming each credential once,
// every entry a known home. An absent index is empty.
//
// Every guard is judged by fsutil.ReadHomeDeclarationDenying, as readStore's
// are: an index behind a symlinked ~/.abcd.noindex is refused on the descriptor walk
// of ~/.abcd.noindex, and the leaf's type, owner and mode on the opened file's own
// fstat, never on a path first.
func readIndex(home string) (map[string]indexEntry, error) {
	raw, refusal, err := fsutil.ReadHomeDeclarationDenying(home, indexRel, maxIndexBytes, 0o077)
	return decodeIndex(raw, refusal, err)
}

// readIndexIn is readIndex through dir, ~/.abcd.noindex as Set's walk opened it. A
// writer that holds the index's lock reads the index here, through the
// directory it writes through, never by walking ~/.abcd.noindex again: a same-uid swap
// of ~/.abcd.noindex between the two walks would otherwise read one directory's index
// and write it, with the new name, into the other (iss-2609290300313698).
func readIndexIn(home string, dir *os.Root) (map[string]indexEntry, error) {
	raw, refusal, err := fsutil.ReadHomeDeclarationDenyingIn(dir, home, indexRel, maxIndexBytes, 0o077)
	return decodeIndex(raw, refusal, err)
}

// decodeIndex judges one read of the index and decodes what it read.
func decodeIndex(raw []byte, refusal fsutil.DeclarationRefusal, err error) (map[string]indexEntry, error) {
	var mode *fsutil.DeclarationModeError
	switch {
	case refusal == fsutil.DeclarationOK:
	case refusal == fsutil.DeclarationAbsent && errors.Is(err, os.ErrNotExist):
		return map[string]indexEntry{}, nil
	case refusal == fsutil.DeclarationAbsent:
		return nil, fmt.Errorf("credential: %s could not be examined, so it is not read", IndexPath)
	case refusal == fsutil.DeclarationBehindSymlink, refusal == fsutil.DeclarationDirectoryExposed:
		return nil, fmt.Errorf("credential: %s is not read: %v", IndexPath, err)
	case refusal == fsutil.DeclarationNotRegular:
		return nil, fmt.Errorf("credential: %s is not a regular file (a symlink is never followed), so it is not read", IndexPath)
	case refusal == fsutil.DeclarationExposed && errors.As(err, &mode):
		return nil, fmt.Errorf("credential: %s can be read or written by group or other (mode %04o), so it is not read; `chmod 0600 %s`", IndexPath, uint32(mode.Perm), IndexPath)
	case refusal == fsutil.DeclarationWritableByOthers:
		return nil, fmt.Errorf("credential: %s can be written by group or other, so it is not read; `chmod 0600 %s`", IndexPath, IndexPath)
	case refusal == fsutil.DeclarationForeignOwner:
		return nil, fmt.Errorf("credential: %s is not owned by you, so it is not read", IndexPath)
	default:
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
// (the scanner included) without making it. The caller, Set, holds the index's
// lock across the read, the scan and the write, and passes dir, ~/.abcd.noindex as its
// walk created, judged and opened it; the read and the write both go through
// that descriptor.
func setIndex(home string, dir *os.Root, name string, e indexEntry, dryRun bool) error {
	idx, err := readIndexIn(home, dir)
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
	if err := fsutil.WriteFileAtomicInRoot(dir, IndexFileName, body, 0o600); err != nil {
		return fmt.Errorf("credential: %s could not be written, so the credential's home was not recorded", IndexPath)
	}
	return nil
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
