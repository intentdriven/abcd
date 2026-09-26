// Package readingitem locates the reading ledger's records by id and resolves
// an occasion — the record a later write names as what occasioned it.
//
// It is a leaf because the writers that name an occasion live on both sides of
// an import edge: core/capture (the admission, surprise and disposition paths)
// imports core/intent, and core/intent (the condition verb) needs the same
// locator. Neither can import the other's copy, and two copies of a locator is
// how they come to disagree about whether a symlinked run directory is part of
// the ledger. So the walk lives here once, and capture keeps its historical
// names as thin wrappers over it (spc-2609020626046252).
//
// Every walk is shallow and symlink-refusing: a symlinked readings root, run
// directory or item directory is refused rather than followed, because a caller
// that writes back to what this returns would otherwise write outside the tree
// that is supposed to contain it.
package readingitem

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

var (
	// ErrUnknown: no record in the ledger carries the id.
	ErrUnknown = errors.New("unknown reading record")
	// ErrDuplicate: more than one file in the ledger claims the id.
	ErrDuplicate = errors.New("duplicate reading record")
	// ErrPathUnsafe: a directory on the walk is a symlink or not a directory.
	ErrPathUnsafe = errors.New("path unsafe")
)

// Family is a record family an occasion may name.
type Family string

// The families ResolveOccasion knows. A caller names the ones it admits.
const (
	FamilyItem        Family = issueschema.ReadingItemFamily // rdi-N, a reading item
	FamilyDisposition Family = issueschema.DispositionFamily // dsp-N, a disposition
	FamilyAdmission   Family = issueschema.AdmissionFamily   // adm-N, an admission
	FamilySurprise    Family = issueschema.SurpriseFamily    // srp-N, a surprise
	FamilyIntent      Family = "itd"                         // itd-N, a shipped intent
)

// ledgerRelDir is where the issues root sits under a repository, so
// ResolveOccasion reaches the reading ledger from the repository root it is
// handed.
const ledgerRelDir = recordid.IssuesRelDir

// Paths returns every file in the ledger that claims item, across all run
// directories. Zero matches means the id is free; one is the ordinary case;
// more is a ledger fault Locate names. An absent readings tree is no matches,
// not an error: a repository that has commissioned no reading is in a state.
func Paths(issuesRoot, item string) ([]string, error) {
	if !recordid.ValidReadingItemID(item) {
		return nil, fmt.Errorf("invalid %s-N identifier: %q", issueschema.ReadingItemFamily, item)
	}
	readingsRoot := filepath.Join(issuesRoot, issueschema.ReadingsDir)
	if err := RefuseSymlinkedDir(readingsRoot); err != nil {
		return nil, err
	}
	runs, err := os.ReadDir(readingsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var matches []string
	for _, run := range runs {
		if !recordid.ValidReadingRunID(run.Name()) {
			continue
		}
		// Every run directory is checked, not only the ones a walk would descend
		// into: a symlink IS a directory to ReadDir.
		runDir := filepath.Join(readingsRoot, run.Name())
		if err := RefuseSymlinkedDir(runDir); err != nil {
			return nil, err
		}
		if !run.IsDir() {
			continue
		}
		cand := filepath.Join(runDir, item+".md")
		if fi, err := os.Lstat(cand); err == nil {
			if !fi.Mode().IsRegular() {
				// Present and not a regular file is a path to refuse, worded as
				// the outstanding board words it, never an id the ledger lacks
				// (iss-2608300848049813).
				return nil, fmt.Errorf("%w: not a regular file (a symlink, a directory, or a device): %s", ErrPathUnsafe, cand)
			}
			matches = append(matches, cand)
		}
	}
	return matches, nil
}

// Locate finds the one reading record carrying item and the run that holds it.
// An id is unique to the LEDGER, not to its run, so no run argument is needed.
func Locate(issuesRoot, item string) (run, path string, err error) {
	matches, err := Paths(issuesRoot, item)
	if err != nil {
		return "", "", err
	}
	switch len(matches) {
	case 0:
		return "", "", fmt.Errorf("%w: %s is not a reading item this ledger holds", ErrUnknown, item)
	case 1:
		return filepath.Base(filepath.Dir(matches[0])), matches[0], nil
	default:
		return "", "", fmt.Errorf("%w: %s is present in more than one run directory", ErrDuplicate, item)
	}
}

// LocateDisposition finds the one disposition record carrying id across every
// item directory under dispositions/, and the item it answers.
func LocateDisposition(issuesRoot, id string) (item, path string, err error) {
	if _, ok := issueschema.DispositionFileID(id + ".md"); !ok {
		return "", "", fmt.Errorf("invalid %s-N identifier: %q", issueschema.DispositionFamily, id)
	}
	root := filepath.Join(issuesRoot, issueschema.DispositionsDir)
	if err := RefuseSymlinkedDir(root); err != nil {
		return "", "", err
	}
	items, err := os.ReadDir(root)
	if err != nil && !os.IsNotExist(err) {
		return "", "", err
	}
	var found []string
	for _, e := range items {
		if !recordid.ValidReadingItemID(e.Name()) {
			continue
		}
		dir := filepath.Join(root, e.Name())
		if err := RefuseSymlinkedDir(dir); err != nil {
			return "", "", err
		}
		cand := filepath.Join(dir, id+".md")
		if fi, err := os.Lstat(cand); err == nil {
			if !fi.Mode().IsRegular() {
				return "", "", fmt.Errorf("%w: not a regular file (a symlink, a directory, or a device): %s", ErrPathUnsafe, cand)
			}
			found = append(found, cand)
		}
	}
	switch len(found) {
	case 0:
		return "", "", fmt.Errorf("%w: %s is not a disposition this ledger holds", ErrUnknown, id)
	case 1:
		return filepath.Base(filepath.Dir(found[0])), found[0], nil
	default:
		return "", "", fmt.Errorf("%w: %s is present under more than one item", ErrDuplicate, id)
	}
}

// LocateAdmission finds the one admission record carrying id across every run
// bucket under admissions/, and the run it is filed under. An admission is
// bucketed by RUN, so the walk is Paths' walk over the admission store: every
// run bucket is checked for a symlink, and the id is unique to the ledger.
func LocateAdmission(issuesRoot, id string) (run, path string, err error) {
	if !recordid.ValidAdmissionID(id) {
		return "", "", fmt.Errorf("invalid %s-N identifier: %q", issueschema.AdmissionFamily, id)
	}
	root := filepath.Join(issuesRoot, issueschema.AdmissionsDir)
	if err := RefuseSymlinkedDir(root); err != nil {
		return "", "", err
	}
	runs, err := os.ReadDir(root)
	if err != nil && !os.IsNotExist(err) {
		return "", "", err
	}
	var found []string
	for _, e := range runs {
		if !recordid.ValidReadingRunID(e.Name()) {
			continue
		}
		dir := filepath.Join(root, e.Name())
		if err := RefuseSymlinkedDir(dir); err != nil {
			return "", "", err
		}
		cand := filepath.Join(dir, id+".md")
		if fi, err := os.Lstat(cand); err == nil && fi.Mode().IsRegular() {
			found = append(found, cand)
		}
	}
	switch len(found) {
	case 0:
		return "", "", fmt.Errorf("%w: %s is not an admission this ledger holds", ErrUnknown, id)
	case 1:
		return filepath.Base(filepath.Dir(found[0])), found[0], nil
	default:
		return "", "", fmt.Errorf("%w: %s is present under more than one run", ErrDuplicate, id)
	}
}

// LocateSurprise finds the surprise record carrying id. The surprise store is
// FLAT (surprises/srp-N.md), so the walk is one leaf: the store is refused if it
// is a symlink, and the leaf is admitted only as a regular file by Lstat, so a
// symlinked record is never followed. Resolution is by presence in the store,
// so a surprise written by hand and one the surprise verb wrote resolve alike
// (spc-2609020626048705).
func LocateSurprise(issuesRoot, id string) (string, error) {
	if !recordid.ValidSurpriseID(id) {
		return "", fmt.Errorf("invalid %s-N identifier: %q", issueschema.SurpriseFamily, id)
	}
	root := filepath.Join(issuesRoot, issueschema.SurprisesDir)
	if err := RefuseSymlinkedDir(root); err != nil {
		return "", err
	}
	cand := filepath.Join(root, id+".md")
	fi, err := os.Lstat(cand)
	switch {
	case err == nil && fi.Mode().IsRegular():
		return cand, nil
	case err == nil:
		return "", fmt.Errorf("%w: %s is not a regular file: %s", ErrPathUnsafe, id, cand)
	case os.IsNotExist(err):
		return "", fmt.Errorf("%w: %s is not a surprise this ledger holds", ErrUnknown, id)
	default:
		return "", err
	}
}

// ResolveOccasion resolves id in one of the families the caller admits and
// returns the path of the record it names. An id outside those families is
// refused by shape before any path is built. A reading item, a disposition or an
// admission resolves through the ledger walk above, and a surprise through its
// flat store, under repoRoot's issue ledger; an
// intent resolves only in repoRoot's intent store's shipped/ bucket, and a
// record in any other bucket is refused naming the bucket.
func ResolveOccasion(repoRoot, id string, families ...Family) (string, error) {
	issuesRoot := filepath.Join(repoRoot, filepath.FromSlash(ledgerRelDir))
	fam := Family(id[:max(0, strings.Index(id, "-"))])
	admitted := false
	names := make([]string, 0, len(families))
	for _, f := range families {
		names = append(names, string(f)+"-N")
		if f == fam {
			admitted = true
		}
	}
	if !admitted {
		return "", fmt.Errorf("occasion %q is not one of %s", id, strings.Join(names, ", "))
	}
	switch fam {
	case FamilyItem:
		_, path, err := Locate(issuesRoot, id)
		return path, err
	case FamilyDisposition:
		_, path, err := LocateDisposition(issuesRoot, id)
		return path, err
	case FamilyAdmission:
		_, path, err := LocateAdmission(issuesRoot, id)
		return path, err
	case FamilySurprise:
		return LocateSurprise(issuesRoot, id)
	case FamilyIntent:
		return resolveShippedIntent(repoRoot, id)
	}
	return "", fmt.Errorf("occasion %q: the %s family has no resolver", id, fam)
}

// resolveShippedIntent resolves an itd-N occasion to a record in shipped/,
// reading the intent store alone.
func resolveShippedIntent(repoRoot, id string) (string, error) {
	if !recordid.ValidIntentID(id) {
		return "", fmt.Errorf("invalid itd-N identifier: %q", id)
	}
	rel, ok, err := recordid.LookupOne(repoRoot, recordid.CanonCitedID(id))
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("%w: occasion %s names no intent this repository holds", ErrUnknown, id)
	}
	// rel is .abcd/development/intents/<bucket>/<file>, slash-separated.
	parts := strings.Split(rel, "/")
	if len(parts) < 2 || parts[len(parts)-2] != "shipped" {
		bucket := "the store's root"
		if len(parts) >= 2 {
			bucket = parts[len(parts)-2] + "/"
		}
		return "", fmt.Errorf("occasion %s is in %s, not shipped/; only a delivered intent occasions a disposition", id, bucket)
	}
	return filepath.Join(repoRoot, filepath.FromSlash(rel)), nil
}

// RefuseSymlinkedDir refuses a path that exists and is not a real directory,
// under ErrPathUnsafe. An absent path is not a fault: an unpopulated tree is a
// state. It is the one guard every reading-ledger walk meets, this leaf's and
// capture's alike, so no two walks can disagree about what the ledger contains.
func RefuseSymlinkedDir(dir string) error {
	fi, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%w: lstat failed for %s: %v", ErrPathUnsafe, dir, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
		return fmt.Errorf("%w: not a real directory: %s", ErrPathUnsafe, dir)
	}
	return nil
}
