package identity

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// A person's commit identity is THEIR OWN, kept on their own machine, and
// nothing in a repository names who may commit to it (itd-2610240918547728,
// adr-2610241022541063 reverses that part of itd-131). The record is read from
// two places under the user-level home, the more specific first:
//
//   - identity/<root-sha>.json — the identity this person commits to ONE
//     repository under, keyed on the full object name of that repository's
//     root commit (as the worktree and transcript stores are), so every clone
//     and worktree of it agrees; it is how a person with a work and a personal
//     identity keeps each where it belongs;
//   - identity.json — the identity this person commits under everywhere else
//     on this machine: the machine-wide default.
//
// Both hold {"name": "...", "email": "..."} and nothing else. The committed
// .githooks/pre-commit guard reads the same two files with the same order
// and refuses on the same terms, so the two cannot disagree about who the
// person is.
const (
	machineRecordLeaf = "identity.json"
	repoRecordDir     = "identity"
	// recordLimit caps the bytes read from a record: two short strings fit in
	// a fraction of it, and the shell guard refuses a larger file too.
	recordLimit = 4096
)

// Record is one person's own commit identity.
type Record struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// RecordScope says which of the two records a RecordRef names.
type RecordScope string

const (
	// ScopeMachine is the machine-wide default record.
	ScopeMachine RecordScope = "machine"
	// ScopeRepository is the record for one repository, by its root commit.
	ScopeRepository RecordScope = "repository"
)

// RecordRef names one record file: its scope, its slash path relative to the
// person's home directory (the form fsutil's home-scope primitives take), and
// the tilde form a message shows.
type RecordRef struct {
	Scope   RecordScope `json:"scope"`
	Rel     string      `json:"rel"`
	Display string      `json:"display"`
}

// MachineRecordRef is the machine-wide default record.
func MachineRecordRef() RecordRef {
	return RecordRef{Scope: ScopeMachine, Rel: abcdhome.Rel(machineRecordLeaf), Display: abcdhome.Display(machineRecordLeaf)}
}

// RepoRecordRef is the record for the repository whose root commit is rootSHA.
// The name must be a full object name (40 or 64 lowercase hex digits): it
// becomes a file name, so nothing else is accepted.
func RepoRecordRef(rootSHA string) (RecordRef, error) {
	if !validRootSHA(rootSHA) {
		return RecordRef{}, fmt.Errorf("%q is not a full root-commit object name", rootSHA)
	}
	leaf := rootSHA + ".json"
	return RecordRef{Scope: ScopeRepository, Rel: abcdhome.Rel(repoRecordDir, leaf), Display: abcdhome.Display(repoRecordDir, leaf)}, nil
}

// RecordRefs is the lookup order for a repository: its own record, when the
// repository has a root commit, then the machine-wide default.
func RecordRefs(rootSHA string) []RecordRef {
	var refs []RecordRef
	if ref, err := RepoRecordRef(rootSHA); err == nil {
		refs = append(refs, ref)
	}
	return append(refs, MachineRecordRef())
}

func validRootSHA(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// ErrRecordRefused wraps every record that is there and cannot be trusted or
// read: the gate fails closed on it, exactly as the shell guard does.
var ErrRecordRefused = errors.New("identity record refused")

// LoadRecord returns the record that applies to the repository whose root
// commit is rootSHA ("" for a repository with no commit yet), for the person
// whose home directory is home: the repository's own record, else the
// machine-wide default. found is false when neither exists.
//
// Each file is read through fsutil.ReadHomeDeclaration, the guard every
// home-scoped declaration takes (the rules loader's own): a regular file this
// account owns that no other account can write, within the size cap, with no
// symlinked directory between home and it. A record that is there and fails a
// guard, or does not parse, or lacks a field, is an error wrapping
// ErrRecordRefused, never a fall-through to the next file: a record the gate
// cannot read must not read as no record.
func LoadRecord(home, rootSHA string) (rec Record, ref RecordRef, found bool, err error) {
	if home == "" {
		return Record{}, RecordRef{}, false, nil
	}
	for _, ref := range RecordRefs(rootSHA) {
		rec, ok, err := readRecord(home, ref)
		if err != nil {
			return Record{}, ref, false, err
		}
		if ok {
			return rec, ref, true, nil
		}
	}
	return Record{}, RecordRef{}, false, nil
}

// ReadRecordAt reads the one record ref names, with LoadRecord's guards.
func ReadRecordAt(home string, ref RecordRef) (Record, bool, error) {
	return readRecord(home, ref)
}

func readRecord(home string, ref RecordRef) (Record, bool, error) {
	data, refusal, err := fsutil.ReadHomeDeclaration(home, ref.Rel, recordLimit)
	switch refusal {
	case fsutil.DeclarationOK:
	case fsutil.DeclarationAbsent:
		return Record{}, false, nil
	default:
		return Record{}, false, fmt.Errorf("%w: %s cannot be trusted or read (%s); it must be a regular file you own that no other account can write, with no symlinked folder above it", ErrRecordRefused, ref.Display, errText(err))
	}
	var r Record
	if err := json.Unmarshal(data, &r); err != nil {
		return Record{}, false, fmt.Errorf("%w: %s is not valid JSON: %v", ErrRecordRefused, ref.Display, err)
	}
	r.Name = strings.TrimSpace(r.Name)
	r.Email = strings.TrimSpace(r.Email)
	if r.Name == "" || r.Email == "" {
		return Record{}, false, fmt.Errorf("%w: %s must set both name and email", ErrRecordRefused, ref.Display)
	}
	return r, true, nil
}

// WriteRecord writes r as the record ref names, for the person whose home
// directory is home, creating the user-level home (and the per-repository
// folder) at abcdhome.DirMode and the file at abcdhome.FileMode. It refuses a
// record the shell guard could not read back (a double-quote, backslash or
// control character), an incomplete one, and a machine identity: the record
// names a person, and a machine is the one thing it must never name.
//
// The write goes through the home-scope primitives, so a symlinked level is
// refused rather than written through, and it is atomic, so a crash never
// leaves a half-written record the gate would then refuse.
func WriteRecord(home string, ref RecordRef, r Record) error {
	if home == "" {
		return fmt.Errorf("the home directory could not be resolved, so %s has nowhere to go", ref.Display)
	}
	r.Name = strings.TrimSpace(r.Name)
	r.Email = strings.TrimSpace(r.Email)
	if r.Name == "" || r.Email == "" {
		return fmt.Errorf("an identity record needs both a name and an email")
	}
	if unpinnable(r.Name) || unpinnable(r.Email) {
		return fmt.Errorf("an identity record must not hold a double-quote, backslash or control character (the pre-commit guard could not read it back); adjust git config user.name/user.email")
	}
	if IsToolIdentity(RoleAuthor, r.Name, r.Email) {
		return fmt.Errorf("%s <%s> is a machine identity, not a person's; an identity record names the person who commits", r.Name, r.Email)
	}
	if !fsutil.ValidRelPath(ref.Rel) {
		return fmt.Errorf("%q is not a place in the home to write", ref.Rel)
	}
	body, err := marshalRecord(r)
	if err != nil {
		return err
	}
	dir, err := fsutil.EnsureHomeScope(home, path.Dir(ref.Rel), abcdhome.DirMode)
	if err != nil {
		return err
	}
	defer dir.Close()
	return fsutil.WriteFileAtomicInRoot(dir, path.Base(ref.Rel), body, abcdhome.FileMode)
}

// marshalRecord renders r pretty-printed with a trailing newline and WITHOUT
// HTML escaping, so &, <, > (legal in a git user.name) are stored literally and
// the shell guard's sed reads back exactly what git config holds (iss-63).
func marshalRecord(r Record) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UserHome is the person's home directory, or "" when it cannot be resolved.
func UserHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func errText(err error) string {
	if err == nil {
		return "unknown error"
	}
	return err.Error()
}
