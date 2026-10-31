package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// autoPlanDraft is a draft every auto-plan row passes, its one scope
// condition not yet stamped with an identity.
const autoPlanDraft = "---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: standalone\norigin: researcher-authored\nproduction_mode: hand-written\n---\n# alpha\n\n" +
	"## Mechanism\n\nWe expect it to work because it is small; shown wrong if it is not.\n\n" +
	"## Scope Conditions\n\n- Holds for one repository with one person in it.\n\n" +
	"## Acceptance Criteria\n\n- Given x, when y, then z.\n\n" +
	"## Open Questions\n\n_None open._\n\n" +
	"## Grounds\n\n- pursued: we expect the loop to plan the draft unattended; shown wrong if a review needs a human\n"

const autoPlanDraftRel = ".abcd/development/intents/drafts/itd-10-alpha.md"

// TestAutoPlanChecksPassAnUnstampedDraftAndNameEachRow: a draft whose
// decisions are recorded passes every row, a scope condition plan has yet to
// stamp included, in the order the substitute lists them, with the digest of
// the bytes judged; the rows read and write nothing.
func TestAutoPlanChecksPassAnUnstampedDraftAndNameEachRow(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, autoPlanDraftRel, autoPlanDraft)
	res, err := AutoPlanChecks(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, r := range res.Rows {
		names = append(names, r.Name)
		if !r.OK {
			t.Errorf("row %s fails on a recorded draft: %s", r.Name, r.Detail)
		}
	}
	want := []string{AutoPlanCheckDraft, AutoPlanCheckCriteria, AutoPlanCheckOpenQuestions, AutoPlanCheckClaimSections,
		AutoPlanCheckGrounds, AutoPlanCheckProvenance, AutoPlanCheckHold, AutoPlanCheckBlocked}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("rows = %v, want %v", names, want)
	}
	if !res.OK() || res.Path != autoPlanDraftRel || res.Digest != DraftDigest([]byte(autoPlanDraft)) {
		t.Fatalf("result = %+v", res)
	}
	got, err := os.ReadFile(filepath.Join(root, autoPlanDraftRel))
	if err != nil || string(got) != autoPlanDraft {
		t.Fatalf("the checks write nothing: %v", err)
	}
	if PlannedPathOf(autoPlanDraftRel) != ".abcd/development/intents/planned/itd-10-alpha.md" {
		t.Fatalf("planned path = %s", PlannedPathOf(autoPlanDraftRel))
	}
}

// TestAutoPlanChecksRefuseAPlannedRecord: a planned record has nothing to plan.
func TestAutoPlanChecksRefuseAPlannedRecord(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/development/intents/planned/itd-10-alpha.md",
		strings.Replace(autoPlanDraft, "spec_id: null", "spec_id: spc-1", 1))
	res, err := AutoPlanChecks(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	if res.OK() || res.Rows[0].Name != AutoPlanCheckDraft || res.Rows[0].OK || !strings.Contains(res.Rows[0].Remedy, "without --auto-plan") {
		t.Fatalf("a planned record is refused at the draft row: %+v", res.Rows[0])
	}
}
