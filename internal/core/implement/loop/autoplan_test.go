package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/gittest"
)

const draftRel = ".abcd/development/intents/drafts/itd-10-alpha.md"

// draftIntent is a draft every auto-plan row passes: criteria, an answered
// mechanism, one scope condition not yet stamped, every question settled,
// pursued grounds, and its provenance. Each replacement pair is applied in
// turn, so a test breaks one row.
func draftIntent(replace ...string) string {
	s := "---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: standalone\norigin: researcher-authored\nproduction_mode: hand-written\n---\n# alpha\n\n" +
		"## Mechanism\n\nWe expect it to work because it is small; shown wrong if it is not.\n\n" +
		"## Scope Conditions\n\n- Holds for one repository with one person in it.\n\n" +
		"## Acceptance Criteria\n\n- Given x, when y, then z.\n\n" +
		settledQuestions +
		"## Grounds\n\n- pursued: we expect the loop to plan the draft unattended; shown wrong if a review needs a human\n"
	for i := 0; i+1 < len(replace); i += 2 {
		s = strings.Replace(s, replace[i], replace[i+1], 1)
	}
	return s
}

// draftRepo stands up a committed repository holding the draft alone, the
// local tier the run lives in, and a temporary HOME.
func draftRepo(t *testing.T, draft string) *gittest.Repo {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repo := gittest.NewRepo(t)
	repo.Write(".gitignore", ".abcd/.work.local/\n")
	repo.Write(draftRel, draft)
	repo.Commit("init")
	if err := os.MkdirAll(filepath.Join(repo.Root(), ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	return repo
}

// writeReturn writes a reviewer's return at the await's path.
func writeReturn(t *testing.T, root, rel, verdict string) string {
	t.Helper()
	p := rel
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, filepath.FromSlash(rel))
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("## Analysis\n\nRead the draft.\n\n### Verdict\n\n"+verdict+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// reviewBoth hands out both plan reviews and returns them with the verdicts
// given, the design lens first.
func reviewBoth(t *testing.T, root, id, design, record string) {
	t.Helper()
	for _, verdict := range []string{design, record} {
		res, err := advance(root, id, DefaultStages(), Options{})
		if err != nil {
			t.Fatal(err)
		}
		if res.Awaiting == nil || res.Awaiting.Role != RoleRuthless || !strings.Contains(res.Awaiting.Brief, PlanReviewDirName) {
			t.Fatalf("a plan review is handed to a fresh reviewer: %+v", res)
		}
		brief, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(res.Awaiting.Brief)))
		if err != nil || !strings.Contains(string(brief), draftRel) || !strings.Contains(string(brief), "### Verdict") {
			t.Fatalf("the brief names the draft and the verdict section: %v\n%s", err, brief)
		}
		p := writeReturn(t, root, res.Awaiting.Receipt, verdict)
		if _, err := Receipt(root, id, p, DefaultStages(), Options{}); err != nil {
			t.Fatal(err)
		}
	}
}

// TestAutoPlanRefusesAnythingLessThanTheSubstitute: a draft short of any part
// of adr-2610310235000654's record-only substitute is refused at the start,
// naming the row, and no state is written; so is an issue key.
func TestAutoPlanRefusesAnythingLessThanTheSubstitute(t *testing.T) {
	cases := []struct {
		name, key, draft, check, reason string
	}{
		{"an open question", "itd-10", draftIntent(settledQuestions, "## Open Questions\n\n- Which runner?\n\n"), intent.AutoPlanCheckOpenQuestions, "Which runner?"},
		{"no grounds", "itd-10", draftIntent("- pursued: we expect the loop to plan the draft unattended; shown wrong if a review needs a human\n", ""), intent.AutoPlanCheckGrounds, "no grounds"},
		{"declined grounds", "itd-10", draftIntent("- pursued:", "- declined:"), intent.AutoPlanCheckGrounds, "declined"},
		{"no origin", "itd-10", draftIntent("origin: researcher-authored\n", ""), intent.AutoPlanCheckProvenance, "origin"},
		{"an unknown production mode", "itd-10", draftIntent("production_mode: hand-written", "production_mode: guessed"), intent.AutoPlanCheckProvenance, "production_mode"},
		{"no criteria", "itd-10", draftIntent("- Given x, when y, then z.\n", ""), intent.AutoPlanCheckCriteria, "Acceptance Criteria"},
		{"an unanswered mechanism", "itd-10", draftIntent("We expect it to work because it is small; shown wrong if it is not.", intent.MechanismPrompt), intent.AutoPlanCheckClaimSections, "Mechanism"},
		{"no scope conditions", "itd-10", draftIntent("## Scope Conditions\n\n- Holds for one repository with one person in it.\n\n", ""), intent.AutoPlanCheckClaimSections, "Scope Conditions"},
		{"a hold", "itd-10", draftIntent("kind: standalone\n", "kind: standalone\nheld: \"awaiting a ruling\"\n"), intent.AutoPlanCheckHold, "awaiting a ruling"},
		{"an unknown intent", "itd-99", draftIntent(), intent.AutoPlanCheckDraft, "itd-99"},
		{"an issue", "iss-2609010000001234", draftIntent(), CheckKey, "nothing to plan"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := draftRepo(t, tc.draft)
			_, err := Start(repo.Root(), tc.key, Options{AutoPlan: true})
			r := mustRefusal(t, err)
			if r.Stage != "check" || r.Check != tc.check || !strings.Contains(r.Reason, tc.reason) {
				t.Fatalf("want the %s check refusing with %q, got %+v", tc.check, tc.reason, r)
			}
			runTierAbsent(t, repo.Root())
			if _, err := os.Stat(filepath.Join(repo.Root(), filepath.FromSlash(draftRel))); err != nil {
				t.Fatalf("a refused auto-plan leaves the draft where it was: %v", err)
			}
		})
	}
}

// TestADraftWithoutAutoPlanIsStillRefused: the flag is the only door; a plain
// build of a draft refuses at the readiness gate, as it always has.
func TestADraftWithoutAutoPlanIsStillRefused(t *testing.T) {
	repo := draftRepo(t, draftIntent())
	_, err := Start(repo.Root(), "itd-10", Options{})
	if r := mustRefusal(t, err); r.Check != CheckReady || !strings.Contains(r.Reason, "draft") {
		t.Fatalf("a draft built without --auto-plan is refused at ready: %+v", r)
	}
	runTierAbsent(t, repo.Root())
}

// TestAutoPlanPlansADraftEndToEnd is step 11 played by a host: the run's one
// lane hands the draft to two fresh reviewers, one per lens, records the
// verdict the loop parsed from each return, then checks the substitute again
// and plans the draft; the run is then complete and says what is left.
func TestAutoPlanPlansADraftEndToEnd(t *testing.T) {
	repo := draftRepo(t, draftIntent())
	root := repo.Root()
	start, err := Start(root, "itd-10", Options{AutoPlan: true})
	if err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(root, start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Lanes) != 1 || st.Lanes[0].Stage != StagePlanReview || st.Lanes[0].AutoPlan == nil || len(st.Pending) != 0 {
		t.Fatalf("an auto-plan run is one lane at its plan review: %+v", st)
	}
	ap := st.Lanes[0].AutoPlan
	data, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(draftRel)))
	if ap.Draft != draftRel || ap.Digest != intent.DraftDigest(data) || len(ap.Reviews) != 2 ||
		ap.Reviews[0].Lens != LensDesign || ap.Reviews[1].Lens != LensRecord {
		t.Fatalf("the lane records the draft, its digest and the two lenses: %+v", ap)
	}

	if PlanStageNext(root, start.RunID) {
		t.Fatal("the plan is not next while the reviews are owed")
	}
	reviewBoth(t, root, start.RunID, "SHIP", "SHIP")
	res, err := advance(root, start.RunID, DefaultStages(), Options{})
	if err != nil || res.PerformedStage != StagePlanReview || res.Stage != StagePlan {
		t.Fatalf("both reviews passing completes the plan review: %+v %v", res, err)
	}
	if !PlanStageNext(root, start.RunID) {
		t.Fatal("the plan is next once both reviews passed, so the front door holds it to the shared-checkout rule")
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(draftRel))); err != nil {
		t.Fatalf("nothing is planned before the plan stage: %v", err)
	}
	res, err = advance(root, start.RunID, DefaultStages(), Options{})
	if err != nil || res.PerformedStage != StagePlan || !res.Complete {
		t.Fatalf("the plan stage plans and the run is complete: %+v %v", res, err)
	}
	planned := intent.PlannedPathOf(draftRel)
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(planned))); err != nil {
		t.Fatalf("the draft moved to planned/: %v", err)
	}
	st, err = ReadState(root, start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	ap = st.Lanes[0].AutoPlan
	if st.Lanes[0].Stage != StageDone || ap.Planned != planned || ap.Spec == "" {
		t.Fatalf("the lane records what the plan made: %+v %+v", st.Lanes[0], ap)
	}
	if !ap.Reviews[0].Pass || ap.Reviews[0].Verdict != "SHIP" || !ap.Reviews[1].Pass {
		t.Fatalf("the verdicts are the ones the loop parsed: %+v", ap.Reviews)
	}
	if !strings.Contains(res.Next, ap.Spec) || !strings.Contains(res.Next, "abcd build itd-10") {
		t.Fatalf("the complete run says what is left: %q", res.Next)
	}
	rec := recordOf(st)
	if rec.Lanes[0].AutoPlan == nil || len(rec.Lanes[0].Verdicts) != 2 {
		t.Fatalf("the run record names the auto-plan and both verdicts: %+v", rec.Lanes[0])
	}
}

// TestAFailingPlanReviewHandsTheDraftBack: a review that does not pass hands
// the lane back with its findings, and the draft is not planned.
func TestAFailingPlanReviewHandsTheDraftBack(t *testing.T) {
	repo := draftRepo(t, draftIntent())
	root := repo.Root()
	start, err := Start(root, "itd-10", Options{AutoPlan: true})
	if err != nil {
		t.Fatal(err)
	}
	reviewBoth(t, root, start.RunID, "SHIP", "FIX FIRST")
	res, err := advance(root, start.RunID, DefaultStages(), Options{})
	if err != nil || res.HandBack == nil || res.HandBack.Verdict != HandBackNotPlanned || len(res.HandBack.Findings) != 1 {
		t.Fatalf("a failing review hands the draft back with its findings: %+v %v", res, err)
	}
	if !strings.Contains(res.Next, "still a draft") {
		t.Fatalf("the hand-back says the draft is unplanned: %q", res.Next)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(draftRel))); err != nil {
		t.Fatalf("the draft is not planned: %v", err)
	}
	if _, err := advance(root, start.RunID, DefaultStages(), Options{}); err == nil {
		t.Fatal("a handed-back auto-plan starts nothing further")
	}
}

// TestAReturnWithNoVerdictIsRefusedAndRecordsNothing: only the loop records a
// verdict, from a return it can read as one.
func TestAReturnWithNoVerdictIsRefusedAndRecordsNothing(t *testing.T) {
	repo := draftRepo(t, draftIntent())
	root := repo.Root()
	start, err := Start(root, "itd-10", Options{AutoPlan: true})
	if err != nil {
		t.Fatal(err)
	}
	res, err := advance(root, start.RunID, DefaultStages(), Options{})
	if err != nil || res.Awaiting == nil {
		t.Fatal(res, err)
	}
	p := filepath.Join(root, filepath.FromSlash(res.Awaiting.Receipt))
	if err := os.WriteFile(p, []byte("Looks fine. Verdict: SHIP\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Receipt(root, start.RunID, p, DefaultStages(), Options{}); err == nil {
		t.Fatal("a return with no Verdict section is refused")
	}
	st, _ := ReadState(root, start.RunID)
	if st.Lanes[0].AutoPlan.Reviews[0].Verdict != "" {
		t.Fatalf("a refused return records no verdict: %+v", st.Lanes[0].AutoPlan.Reviews[0])
	}
}

// TestADraftChangedAfterTheStartIsHandedBackUnplanned: the reviews and the
// plan are bound to the bytes the start checked.
func TestADraftChangedAfterTheStartIsHandedBackUnplanned(t *testing.T) {
	repo := draftRepo(t, draftIntent())
	root := repo.Root()
	start, err := Start(root, "itd-10", Options{AutoPlan: true})
	if err != nil {
		t.Fatal(err)
	}
	reviewBoth(t, root, start.RunID, "SHIP", "SHIP")
	if _, err := advance(root, start.RunID, DefaultStages(), Options{}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(draftRel)), []byte(draftIntent("then z.", "then w.")), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := advance(root, start.RunID, DefaultStages(), Options{})
	if err != nil || res.HandBack == nil || !strings.Contains(res.HandBack.Reason, "changed") {
		t.Fatalf("a changed draft is handed back: %+v %v", res, err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(draftRel))); err != nil {
		t.Fatalf("the changed draft is not planned: %v", err)
	}
}

// TestAutoPlanOnAPlannedIntentBuildsIt: a planned intent has nothing to plan,
// so the flag leaves the build as it is, and the record says so.
func TestAutoPlanOnAPlannedIntentBuildsIt(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	res, err := Start(repo.Root(), "itd-10", Options{AutoPlan: true})
	if err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Lanes[0].Stage != StageWorktree || st.Lanes[0].AutoPlan != nil {
		t.Fatalf("a planned intent is built: %+v", st.Lanes[0])
	}
	found := false
	for _, e := range st.Record {
		found = found || strings.Contains(e.Note, "--auto-plan")
	}
	if !found {
		t.Fatalf("the record says the flag had nothing to plan: %+v", st.Record)
	}
}

// TestAStoredAutoPlanIsHeldToWhatTheLoopWrites: a file of version 16 carrying
// an auto-plan record, and a forged verdict, are each refused as the run's
// state.
func TestAStoredAutoPlanIsHeldToWhatTheLoopWrites(t *testing.T) {
	repo := draftRepo(t, draftIntent())
	root := repo.Root()
	start, err := Start(root, "itd-10", Options{AutoPlan: true})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, filepath.FromSlash(StateRelPath(start.RunID)))
	good, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for name, edit := range map[string]func(doc map[string]any){
		"version 16": func(doc map[string]any) { doc["schema_version"] = schemaVersionUnplanned },
		"a forged verdict": func(doc map[string]any) {
			rv := doc["lanes"].([]any)[0].(map[string]any)["auto_plan"].(map[string]any)["reviews"].([]any)[0].(map[string]any)
			rv["verdict"], rv["pass"] = "FIX FIRST", true
		},
		"a draft outside drafts/": func(doc map[string]any) {
			doc["lanes"].([]any)[0].(map[string]any)["auto_plan"].(map[string]any)["draft"] = "AGENTS.md"
		},
	} {
		t.Run(name, func(t *testing.T) {
			var doc map[string]any
			if err := json.Unmarshal(good, &doc); err != nil {
				t.Fatal(err)
			}
			edit(doc)
			out, _ := json.Marshal(doc)
			if err := os.WriteFile(path, out, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := ReadState(root, start.RunID); err == nil {
				t.Fatal("the forged state must be refused")
			}
		})
	}
}

// TestADraftChangedDuringTheReviewsIsHandedBack: the plan review itself
// checks the bytes, so a draft edited while its reviewers read it is handed
// back before any further reviewer is started on a text the first never read.
func TestADraftChangedDuringTheReviewsIsHandedBack(t *testing.T) {
	repo := draftRepo(t, draftIntent())
	root := repo.Root()
	start, err := Start(root, "itd-10", Options{AutoPlan: true})
	if err != nil {
		t.Fatal(err)
	}
	res, err := advance(root, start.RunID, DefaultStages(), Options{})
	if err != nil || res.Awaiting == nil {
		t.Fatal(res, err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(draftRel)), []byte(draftIntent("then z.", "then w.")), 0o600); err != nil {
		t.Fatal(err)
	}
	p := writeReturn(t, root, res.Awaiting.Receipt, "SHIP")
	if _, err := Receipt(root, start.RunID, p, DefaultStages(), Options{}); err != nil {
		t.Fatal(err)
	}
	res, err = advance(root, start.RunID, DefaultStages(), Options{})
	if err != nil || res.HandBack == nil || !strings.Contains(res.HandBack.Reason, "changed") {
		t.Fatalf("a draft changed during the reviews is handed back: %+v %v", res, err)
	}
}
