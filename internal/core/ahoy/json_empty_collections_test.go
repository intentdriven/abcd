package ahoy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// assertNoNullCollections marshals v and fails on every named field, at the
// dotted path given, that renders as JSON null. A consumer iterating a list
// field (jq '.gaps[]', an agent following the command doc) errors on null, and
// the empty case is the healthy one most scripts hit (iss-2609120447487070).
func assertNoNullCollections(t *testing.T, what string, v any, paths ...string) {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("%s: marshal: %v", what, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("%s: not a JSON object: %v", what, err)
	}
	for _, p := range paths {
		cur := any(doc)
		parts := strings.Split(p, ".")
		for i, k := range parts {
			m, ok := cur.(map[string]any)
			if !ok {
				t.Errorf("%s: %s is not an object at %q", what, p, strings.Join(parts[:i], "."))
				cur = nil
				break
			}
			val, present := m[k]
			if !present {
				t.Errorf("%s: field %s absent", what, p)
				cur = nil
				break
			}
			cur = val
			if i == len(parts)-1 && val == nil {
				t.Errorf("%s: field %s rendered as null, want [] (a collection is never null)", what, p)
			}
		}
	}
}

// TestAhoyJSONCollectionsAreEmptyArraysNotNull pins, for every result the ahoy
// verbs render as JSON, that a list field with nothing in it is [] and never
// null — the invariant the other verbs' collections already hold.
func TestAhoyJSONCollectionsAreEmptyArraysNotNull(t *testing.T) {
	setupHermetic(t)
	harnessFixture(t, "")
	folder := t.TempDir() // no .git: an unmanaged folder, which raises no gaps

	det, err := Detect(folder)
	if err != nil {
		t.Fatal(err)
	}
	if len(det.Gaps) != 0 {
		t.Fatalf("fixture raised gaps, so it does not exercise the empty case: %+v", det.Gaps)
	}
	assertNoNullCollections(t, "ahoy (detect)", det, "gaps")

	doc, err := Doctor(folder)
	if err != nil {
		t.Fatal(err)
	}
	assertNoNullCollections(t, "ahoy doctor", doc, "detection.gaps", "audit_gaps")

	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	refuse := false
	res, err := Install(repo, InstallOptions{Adopt: &refuse}, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "aborted" {
		t.Fatalf("status = %q, want aborted", res.Status)
	}
	assertNoNullCollections(t, "ahoy install (aborted)", res, "writes", "remaining", "declined_categories", "summary")

	rec, err := Uninstall(folder, "")
	if err != nil {
		t.Fatal(err)
	}
	assertNoNullCollections(t, "ahoy uninstall", rec, "marker.removed", "marker.skipped")
}
