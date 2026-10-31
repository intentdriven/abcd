package intent

import (
	"errors"
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
	want := []string{AutoPlanCheckDraft, AutoPlanCheckPlannable, AutoPlanCheckCriteria, AutoPlanCheckOpenQuestions, AutoPlanCheckClaimSections,
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

// TestPlanWithAnExpectedDigestRefusesAChangedDraft: a plan told the bytes it
// must find compares them with what it reads under the store's lock, and a
// draft that differs is refused before anything moves or is minted; the same
// draft with its own digest plans.
func TestPlanWithAnExpectedDigestRefusesAChangedDraft(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, autoPlanDraftRel, autoPlanDraft)
	reviewed := DraftDigest([]byte(autoPlanDraft))
	changed := strings.Replace(autoPlanDraft, "then z.", "then w.", 1)
	writeFile(t, root, autoPlanDraftRel, changed)
	_, err := Plan(root, "itd-10", PlanOptions{ExpectDigest: reviewed})
	if !errors.Is(err, ErrDraftChanged) {
		t.Fatalf("a draft whose bytes differ from the expected digest is refused as changed: %v", err)
	}
	if got, rerr := os.ReadFile(filepath.Join(root, autoPlanDraftRel)); rerr != nil || string(got) != changed {
		t.Fatalf("the refused plan leaves the draft byte-identical: %v", rerr)
	}
	if ents, _ := os.ReadDir(filepath.Join(root, ".abcd", "development", "specs")); len(ents) != 0 {
		t.Fatalf("the refused plan mints no spec: %v", ents)
	}
	if _, err := Plan(root, "itd-10", PlanOptions{ExpectDigest: DraftDigest([]byte(changed))}); err != nil {
		t.Fatalf("the draft its digest names plans: %v", err)
	}
	if _, err := Plan(root, "itd-10", PlanOptions{ExpectDigest: DraftDigest([]byte(changed))}); !errors.Is(err, ErrDraftChanged) {
		t.Fatalf("a record no longer in drafts/ is refused to a plan expecting a draft: %v", err)
	}
}

// TestPlanComparesTheExpectedDigestUnderTheLock: a draft rewritten after the
// plan's corpus load and before it takes the store's lock is refused as
// changed, so the compare is made on the bytes read under the lock, not on
// any read before it.
func TestPlanComparesTheExpectedDigestUnderTheLock(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, autoPlanDraftRel, autoPlanDraft)
	reviewed := DraftDigest([]byte(autoPlanDraft))
	changed := strings.Replace(autoPlanDraft, "then z.", "then w.", 1)
	beforeIntentMintLock = func() {
		beforeIntentMintLock = nil
		writeFile(t, root, autoPlanDraftRel, changed)
	}
	t.Cleanup(func() { beforeIntentMintLock = nil })
	_, err := Plan(root, "itd-10", PlanOptions{ExpectDigest: reviewed})
	if !errors.Is(err, ErrDraftChanged) {
		t.Fatalf("a draft rewritten just before the lock is refused as changed: %v", err)
	}
	if got, rerr := os.ReadFile(filepath.Join(root, autoPlanDraftRel)); rerr != nil || string(got) != changed {
		t.Fatalf("the refused plan leaves the rewritten draft as it is: %v", rerr)
	}
	if ents, _ := os.ReadDir(filepath.Join(root, ".abcd", "development", "specs")); len(ents) != 0 {
		t.Fatalf("the refused plan mints no spec: %v", ents)
	}
}

// TestAutoPlanChecksRefuseADraftPlanWouldRefuse: a draft `plan` itself would
// refuse (a spec_id already set, a slug not kebab-case) fails the plannable
// row at the start, so no review is spent on it.
func TestAutoPlanChecksRefuseADraftPlanWouldRefuse(t *testing.T) {
	for name, draft := range map[string]string{
		"a spec_id already set": strings.Replace(autoPlanDraft, "spec_id: null", "spec_id: spc-2610010000000001", 1),
		"a slug not kebab-case": strings.Replace(autoPlanDraft, "slug: alpha", "slug: Alpha_One", 1),
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, autoPlanDraftRel, draft)
			res, err := AutoPlanChecks(root, "itd-10")
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range res.Rows {
				if r.Name == AutoPlanCheckPlannable {
					if r.OK || r.Remedy == "" {
						t.Fatalf("the plannable row refuses it with a remedy: %+v", r)
					}
					return
				}
			}
			t.Fatalf("no plannable row: %+v", res.Rows)
		})
	}
}
