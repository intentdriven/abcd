package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/dryrun"
)

// plan_test.go pins the check/write split of the intent store's filled-in
// inputs (spc-2610301431131636, approach 1): each Plan runs every check its
// write runs and writes nothing, and the write is that plan applied.

func planTreeUnchanged(t *testing.T, root string, before map[string]string) {
	t.Helper()
	after := prepassTree(t, root)
	for p, h := range after {
		if before[p] != h {
			t.Errorf("the plan wrote %s", p)
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			t.Errorf("the plan removed %s", p)
		}
	}
}

// TestPlanPrepassBriefWritesNothingAndApplyLaysThePlannedBrief: the plan
// names the one brief, with its bytes, and writes nothing; Apply lays exactly
// those bytes; and a refusal is the write's own.
func TestPlanPrepassBriefWritesNothingAndApplyLaysThePlannedBrief(t *testing.T) {
	root := prepassRepo(t)
	in := prepassAssemble(t, root)
	raw := prepassFindings(t, in, map[string]any{"unanchored": []map[string]any{{"question": "Who reads it first?"}}})
	before := prepassTree(t, root)

	plan, err := PlanPrepassBrief(root, "itd-500", raw)
	if err != nil {
		t.Fatalf("PlanPrepassBrief: %v", err)
	}
	planTreeUnchanged(t, root, before)
	if len(plan.Plan.Writes) != 1 {
		t.Fatalf("the plan names %d writes, want the brief: %+v", len(plan.Plan.Writes), plan.Plan.Writes)
	}
	w := plan.Plan.Writes[0]
	if w.Path != PlanningBriefsRelDir+"/itd-500.md" || w.Action != dryrun.Create || w.Bytes != len(w.Content) || w.Bytes == 0 {
		t.Fatalf("the planned write is %+v", w)
	}
	res, err := plan.Apply()
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(res.BriefPath)))
	if err != nil || string(got) != string(w.Content) {
		t.Fatalf("Apply laid %q (%v), not the planned brief", got, err)
	}
	// A second plan over the brief the pre-pass wrote would replace it.
	again, err := PlanPrepassBrief(root, "itd-500", raw)
	if err != nil || again.Plan.Writes[0].Action != dryrun.Replace {
		t.Fatalf("a re-plan is %+v (%v), want a replace", again.Plan.Writes, err)
	}

	// Refusal parity: findings over a moved input are refused by the plan
	// in the words the write refuses them in.
	inv := filepath.Join(root, filepath.FromSlash(invariantsRelPath))
	if err := os.WriteFile(inv, []byte(prepassInvariants+"\n4. **A fourth** — added since.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, perr := PlanPrepassBrief(root, "itd-500", raw)
	_, werr := WritePrepassBrief(root, "itd-500", raw)
	if perr == nil || werr == nil || perr.Error() != werr.Error() {
		t.Fatalf("plan refuses as %v, the write as %v", perr, werr)
	}
}

// TestPlanVerdictIngestWritesNothing: a valid verdict's plan replaces the
// shipped intent's record with the block the ingest writes, and a verdict the
// ingest would quarantine plans both quarantine writes; neither plan writes,
// and the ingest then lays what was planned.
func TestPlanVerdictIngestWritesNothing(t *testing.T) {
	for _, tc := range []struct {
		name, status string
		payload      func(rcp string) string
		writes       int
	}{
		{"ingested", "ingested", validVerdict, 1},
		{"dead_letter", "dead_letter", func(rcp string) string {
			return strings.Replace(validVerdict(rcp), `"criterion_id": "ac-1"`, `"criterion_id": "ac-9"`, 1)
		}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			rcp := shipOne(t, root)
			raw, err := os.ReadFile(writeVerdict(t, root, tc.payload(rcp)))
			if err != nil {
				t.Fatal(err)
			}
			before := prepassTree(t, root)
			plan, err := PlanVerdictIngest(root, raw)
			if err != nil {
				t.Fatalf("PlanVerdictIngest: %v", err)
			}
			planTreeUnchanged(t, root, before)
			if plan.Result.Status != tc.status || len(plan.Plan.Writes) != tc.writes {
				t.Fatalf("plan = %s with writes %+v; want %s with %d", plan.Result.Status, plan.Plan.Writes, tc.status, tc.writes)
			}
			res, err := IngestVerdictBytes(root, raw)
			if err != nil || res.Status != tc.status {
				t.Fatalf("the ingest = %+v (%v)", res, err)
			}
			for _, w := range plan.Plan.Writes {
				got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(w.Path)))
				if err != nil || string(got) != string(w.Content) {
					t.Errorf("the ingest laid %s differently from the plan (%v)", w.Path, err)
				}
			}
		})
	}
}

// TestPlanVerdictIngestRefusesAsTheIngestDoes: an unsolicited receipt is
// refused by the plan and the ingest in the same words, writing nothing.
func TestPlanVerdictIngestRefusesAsTheIngestDoes(t *testing.T) {
	root := t.TempDir()
	shipOne(t, root)
	raw := []byte(validVerdict("rcp-000000000000"))
	before := prepassTree(t, root)
	_, perr := PlanVerdictIngest(root, raw)
	planTreeUnchanged(t, root, before)
	_, ierr := IngestVerdictBytes(root, raw)
	if perr == nil || ierr == nil || perr.Error() != ierr.Error() {
		t.Fatalf("plan refuses as %v, the ingest as %v", perr, ierr)
	}
}
