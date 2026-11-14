package loop

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/core/spotcheck"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The drain's hand-backs and its judgements a model answered, recorded for
// the spot-check as they happen (spc-2610301925429250 step 1), each as one
// file on the store's local copy, never failing or holding the drain.

// storeRecords is what the store's local copy holds, read with git alone:
// every record file, decoded.
func storeRecords(t *testing.T, repo *gittest.Repo) spotcheck.Records {
	t.Helper()
	var rec spotcheck.Records
	if repo.Git("for-each-ref", spotcheck.LocalRef) == "" {
		return rec
	}
	for _, p := range strings.Fields(repo.Git("ls-tree", "-r", "--name-only", spotcheck.LocalRef)) {
		raw := []byte(repo.Git("cat-file", "blob", spotcheck.LocalRef+":"+p))
		var err error
		switch {
		case strings.HasPrefix(p, spotcheck.HandBacksDir+"/"):
			var h spotcheck.HandBack
			err = json.Unmarshal(raw, &h)
			rec.HandBacks = append(rec.HandBacks, h)
		case strings.HasPrefix(p, spotcheck.DecisionsDir+"/"):
			var d spotcheck.Decision
			err = json.Unmarshal(raw, &d)
			rec.Decisions = append(rec.Decisions, d)
		default:
			rec.Unreadable = append(rec.Unreadable, p)
		}
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
	}
	return rec
}

// TestALanesHandBackIsRecordedForTheSpotCheck: the move that routes a lane's
// hand-back records it with its kind, route, the lane's run and the drain's
// rule, on the local copy alone: nothing reaches origin.
func TestALanesHandBackIsRecordedForTheSpotCheck(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first := drainedToLaneOnce(t, repo, DrainOptions{})
	handBackLaneOf(t, repo, first.Start.RunID, Options{}, LaneHandBack{Kind: HandBackDesignFinding, Reason: "the page needs a decision", Home: "the brief"})
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Routed) == 0 {
		t.Fatalf("the move routed no hand-back: %+v", res)
	}
	var got *spotcheck.HandBack
	for _, h := range storeRecords(t, repo).HandBacks {
		if h.Issue == secondIssue {
			got = &h
		}
	}
	if got == nil || got.From != "lane" || got.Kind != HandBackDesignFinding || got.Route != RouteHome ||
		got.RunID != first.Start.RunID || got.DrainRule != res.Rule || got.Reason != "the page needs a decision" {
		t.Fatalf("the recorded hand-back is %+v", got)
	}
	if out := gitIn(t, bare, "for-each-ref", spotcheck.BranchRef); out != "" {
		t.Fatalf("a record reached origin: %s", out)
	}
}

// TestARecordThatCannotBeWrittenNeverHoldsTheDrain: a store that cannot be
// read refuses the write; the move routes the hand-back as it would, says the
// record was not written, and the next move writes it from the drain's state.
func TestARecordThatCannotBeWrittenNeverHoldsTheDrain(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	first := drainedToLaneOnce(t, repo, DrainOptions{})
	handBackLaneOf(t, repo, first.Start.RunID, Options{}, LaneHandBack{Kind: HandBackDesignFinding, Reason: "the page needs a decision", Home: "the brief"})
	blob := strings.TrimSpace(repo.Git("hash-object", "-w", "commands/drain.md"))
	repo.Git("update-ref", spotcheck.LocalRef, blob)
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatalf("a store that cannot be written refused the move: %v", err)
	}
	if len(res.Routed) == 0 || !strings.Contains(res.Next, "record(s) of the spot-check were not written here") || !strings.Contains(res.Next, "a later move tries them again") {
		t.Fatalf("the move routed %d and said %q", len(res.Routed), res.Next)
	}
	if got := repo.Git("rev-parse", spotcheck.LocalRef); got != blob {
		t.Fatalf("a record was written over an unreadable store: %s", got)
	}
	repo.Git("update-ref", "-d", spotcheck.LocalRef)
	if _, err := Drain(repo.Root(), Options{}, DrainOptions{}); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, h := range storeRecords(t, repo).HandBacks {
		found = found || (h.Issue == secondIssue && h.Kind == HandBackDesignFinding)
	}
	if !found {
		t.Fatal("the next move did not write the record the first could not")
	}
}

// TestAModelsJudgementIsRecordedAndAPersonsIsNot: a judgement a routed
// model answered is the automated facilitator's decision, recorded with the
// question, the answer, the model and the drain's rule; one a person answered
// is not; a record already kept is not written again; and a checkout with no
// origin keeps none.
func TestAModelsJudgementIsRecordedAndAPersonsIsNot(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	t0 := time.Date(2026, 11, 13, 7, 0, 0, 0, time.UTC)
	st := DrainState{Rule: "adr-2609291342092738", Judgements: []DrainJudgement{
		{Issue: "iss-1", Answer: JudgementNo, Reason: "it changes neither", Applied: true, At: t0, Route: &runner.RouteRecord{Model: "model-a"}},
		{Issue: "iss-2", Answer: JudgementYes, Kind: HandBackUserVisible, Reason: "a person said so", Applied: true, At: t0},
	}}
	if note := spotcheckRecords(context.Background(), repo.Root(), st); note != "" {
		t.Fatalf("the write said %q", note)
	}
	ds := storeRecords(t, repo).Decisions
	if len(ds) != 1 {
		t.Fatalf("decisions = %+v, want the model's alone", ds)
	}
	d := ds[0]
	if d.Kind != spotcheck.KindDrainJudgement || d.Issue != "iss-1" || d.Answer != JudgementNo || d.Model != "model-a" ||
		d.Rule != st.Rule || d.Reason != "it changes neither" || !strings.HasPrefix(d.Question, "Does this remedy") {
		t.Fatalf("the recorded judgement is %+v", d)
	}
	tip := repo.Git("rev-parse", spotcheck.LocalRef)
	spotcheckRecords(context.Background(), repo.Root(), st)
	if repo.Git("rev-parse", spotcheck.LocalRef) != tip {
		t.Fatal("a record already kept was written again")
	}
	bare := gittest.NewRepo(t)
	bare.Write("a.txt", "a\n")
	bare.Commit("the base")
	if note := spotcheckRecords(context.Background(), bare.Root(), st); note != "" || bare.Git("for-each-ref", spotcheck.LocalRef) != "" {
		t.Fatalf("a checkout with no origin: note %q", note)
	}
}

// TestTheMovesNoteCountsOnlyWhatItTriedAndCouldNotWrite: a record the store
// cannot keep at all (a route with no time) is counted apart, the same at
// every move, and never as a record not written; a person's judgement and a
// record already kept are never counted.
func TestTheMovesNoteCountsOnlyWhatItTriedAndCouldNotWrite(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	t0 := time.Date(2026, 11, 13, 7, 0, 0, 0, time.UTC)
	st := DrainState{Rule: "adr-2609291342092738",
		HandBacks: []DrainRoute{
			{Issue: "iss-1", From: "lane", Kind: HandBackStuck, Route: RouteHome, At: &t0},
			{Issue: "iss-2", From: "lane", Kind: HandBackStuck, Route: RouteHome},
		},
		Judgements: []DrainJudgement{
			{Issue: "iss-3", Answer: JudgementYes, Kind: HandBackUserVisible, Reason: "a person said so", Applied: true, At: t0},
		}}
	for range 2 {
		note := spotcheckRecords(context.Background(), repo.Root(), st)
		if note != "1 record(s) of the drain's state cannot be kept by the spot-check's store (the hand-back of iss-2 carries no time)" {
			t.Fatalf("the move said %q", note)
		}
	}
	if rec := storeRecords(t, repo); len(rec.HandBacks) != 1 || len(rec.Decisions) != 0 {
		t.Fatalf("the store holds %+v", rec)
	}
	blob := strings.TrimSpace(repo.Git("hash-object", "-w", "commands/drain.md"))
	repo.Git("update-ref", spotcheck.LocalRef, blob)
	if note := spotcheckRecords(context.Background(), repo.Root(), st); !strings.HasPrefix(note, "1 record(s) of the spot-check were not written here") {
		t.Fatalf("over an unreadable store the move said %q", note)
	}
}
