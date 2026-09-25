package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/spec"
)

// plan_nullspec_test.go — a planned intent with no spec is given one by
// `intent plan` (iss-2609211738504433). Intents planned before the spec seam
// existed sit in planned/ with spec_id null, and every route to a spec was
// closed: plan on a planned record did the identity step alone, link writes a
// spec_id only for a spec that exists, and the spec store mints nothing of its
// own. Plan now mints and links the spec for such a record as it does for a
// draft, on the same Acceptance Criteria bar, without moving a bucket.

const nullSpecPlanned = "---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: standalone\n---\n# alpha\n\n" +
	"## Acceptance Criteria\n\n- the spec is minted in place\n"

func TestPlanMintsAndLinksASpecForAPlannedRecordWithNone(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", nullSpecPlanned)

	res, err := Plan(root, "itd-10", PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.StampOnly || !res.LinkedInPlace {
		t.Fatalf("the run must report a spec linked in place, not a stamp or a move: %+v", res)
	}
	if res.Intent.Bucket != BucketPlanned || res.Intent.Path != plannedDir+"/itd-10-alpha.md" {
		t.Fatalf("the record moved: %+v", res.Intent)
	}
	if res.Spec.ID == "" || res.Intent.SpecID != res.Spec.ID {
		t.Fatalf("spec %q not linked from the intent (spec_id %q)", res.Spec.ID, res.Intent.SpecID)
	}
	store, err := spec.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	sp, ok := store.Lookup(res.Spec.ID)
	if !ok || sp.Intent != "itd-10" {
		t.Fatalf("the minted spec does not name the intent back: %+v (found %v)", sp, ok)
	}
	body, err := os.ReadFile(filepath.Join(root, res.Intent.Path))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "spec_id: "+res.Spec.ID+"\n") {
		t.Fatalf("the record does not carry the spec_id:\n%s", body)
	}

	// The gate now finds the link it reported missing.
	r, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range r.Checks {
		if c.Name == CheckSpecLink && !c.OK {
			t.Fatalf("the spec link still fails after the mint: %+v", c)
		}
	}

	// A second run finds the spec linked and has nothing to stamp.
	if _, err := Plan(root, "itd-10", PlanOptions{}); err == nil || !strings.Contains(err.Error(), "nothing to stamp") {
		t.Fatalf("a re-run must refuse as the stamp step, got %v", err)
	}
}

func TestPlanRefusesAPlannedRecordWithNoSpecAndNoCriteria(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: standalone\n---\n# alpha\n\n## Acceptance Criteria\n\n")
	_, err := Plan(root, "itd-10", PlanOptions{})
	if err == nil || !strings.Contains(err.Error(), "Acceptance Criteria") {
		t.Fatalf("a planned record with no criteria must be refused, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, specsOpen)); !os.IsNotExist(err) {
		t.Fatal("a refused run must mint no spec")
	}
}

// The readiness gate's remedy names the command that now runs.
func TestReadyRemedyForAPlannedRecordWithNoSpecNamesPlan(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", nullSpecPlanned)
	r, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range r.Checks {
		if c.Name == CheckSpecLink {
			if !strings.Contains(c.Remedy, "abcd intent plan itd-10") {
				t.Fatalf("remedy = %q, want it to name `abcd intent plan itd-10`", c.Remedy)
			}
			return
		}
	}
	t.Fatal("no spec-link check reported")
}
