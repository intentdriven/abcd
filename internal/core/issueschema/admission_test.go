package issueschema_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// The admission record's required set is ONE list, held where every gate that
// asks the question reads it. The gate that judges the committed tree
// (core/lint's record_schema) declares the adm store's requiredFields from this
// value rather than from a hand-copied literal, for the reason the issue store
// already does: a second copy drifts the moment one side gains a field, and the
// drift shows up as a record the gate passes and no reader can read.
func TestAdmissionRequiredIsTheOneList(t *testing.T) {
	want := []string{"schema_version", "id", "run", "proposal", "grounds"}
	if !slices.Equal(issueschema.AdmissionRequired, want) {
		t.Fatalf("AdmissionRequired = %v, want %v", issueschema.AdmissionRequired, want)
	}
	// `grounds` is the whole point of the record: declining a proposal costs
	// nothing epistemically, while admitting one is where the frame is engaged,
	// so an admission that records no grounds records nothing.
	if !slices.Contains(issueschema.AdmissionRequired, "grounds") {
		t.Error("an admission record without grounds is the record this family exists to prevent")
	}
	assertKnownCoversRequired(t, "AdmissionKnown", issueschema.AdmissionKnown, issueschema.AdmissionRequired)
	assertNoSecondSpelling(t, `"run", "proposal", "grounds"`)
}

// The surprise entry is its OWN record: a separate family, a separate store, and
// a join key (`occasioned_by`) rather than the disposition's key. The reading's
// output, the researcher's answer and the surprise that occasions abduction are
// three acts, so a surprise can never be read as a disposition or overwrite one.
func TestSurpriseRequiredIsTheOneList(t *testing.T) {
	want := []string{"schema_version", "id", "occasioned_by"}
	if !slices.Equal(issueschema.SurpriseRequired, want) {
		t.Fatalf("SurpriseRequired = %v, want %v", issueschema.SurpriseRequired, want)
	}
	assertKnownCoversRequired(t, "SurpriseKnown", issueschema.SurpriseKnown, issueschema.SurpriseRequired)
	assertNoSecondSpelling(t, `"schema_version", "id", "occasioned_by"`)

	// Separateness, asserted rather than described: the surprise shares neither
	// family prefix nor store with the disposition, so nothing can file one where
	// the other is read.
	if issueschema.SurpriseFamily == issueschema.DispositionFamily {
		t.Error("the surprise entry must not share the disposition's family prefix")
	}
	if issueschema.SurprisesDir == issueschema.DispositionsDir {
		t.Error("the surprise entry must not share the disposition store")
	}
	// It is keyed to whatever occasioned it, never to the item a disposition is
	// keyed to: sharing the key is how the two records would collide.
	if slices.Contains(issueschema.SurpriseRequired, "item") {
		t.Error("a surprise is keyed by occasioned_by, never by the disposition's item key")
	}
}

// assertKnownCoversRequired pins the allow-list against the required set: a
// required property outside the allow-list is a record the schema demands and
// refuses in one breath.
func assertKnownCoversRequired(t *testing.T, name string, known map[string]bool, required []string) {
	t.Helper()
	for _, f := range required {
		if !known[f] {
			t.Errorf("%s omits required property %q", name, f)
		}
	}
}

// assertNoSecondSpelling refuses a copy of a schema list anywhere else in the Go
// tree, exactly as the status-directory list is refused a second spelling. A copy
// is not a style complaint: it is the drift the one-definition rule exists to
// prevent, waiting for one side to be edited.
func assertNoSecondSpelling(t *testing.T, literal string) {
	t.Helper()
	root := filepath.Join("..", "..", "..")
	canonical := filepath.FromSlash("internal/core/issueschema/admission.go")
	var offenders []string
	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Test files are skipped: a fixture legitimately writes the properties it
		// is building on disk, and that is data, not a second declaration.
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if rel == canonical {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(b), literal) {
			offenders = append(offenders, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/: %v", err)
	}
	if len(offenders) > 0 {
		t.Errorf("the list %s is spelled again in %s;\n"+
			"there is one canonical declaration (core/issueschema) and every Go consumer reads it",
			literal, strings.Join(offenders, ", "))
	}
}
