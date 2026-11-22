package matrix

import (
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/gittest"
)

const reportIntent = "itd-2610161733316518"

// pinOf is the cell's pin in rec.
func pinOf(t *testing.T, rec RunRecord, cell string) CellPin {
	t.Helper()
	for _, p := range rec.Cells {
		if p.Cell == cell {
			return p
		}
	}
	t.Fatalf("the run pins no cell %s", cell)
	return CellPin{}
}

// codeTrial is copy repeat of cell on task feature-a as status, judged as such:
// a passed or failed copy carries its score and a compliance result that ranks
// it, an unexamined one a result that could not examine it, and a metered
// copy its cost.
func codeTrial(t *testing.T, rec RunRecord, cell string, repeat int, status TrialStatus) TrialRecord {
	t.Helper()
	pin := pinOf(t, rec, cell)
	tr := TrialRecord{SchemaVersion: TrialSchema, Run: rec.RunID, Key: TrialKey{Task: "feature-a", Cell: cell, Repeat: repeat},
		Kind: KindCode, Intent: reportIntent, FrozenCommit: rec.FrozenCommit, Source: pin.Source, Runner: pin.Runner,
		HarnessVersion: pin.HarnessVersion, ModelAsked: pin.Model, ModelServed: pin.Model, Status: status,
		StartedAt: t0, FinishedAt: t0.Add(3 * time.Minute), ModelTimeMS: 120000, GateWaitMS: 60000}
	if pin.Source.Metered() && status != StatusNotStarted {
		tr.Cost = &oracle.Cost{Amount: 0.5, Unit: oracle.CostUSD}
	}
	switch status {
	case StatusPassed:
		tr.Score, tr.Compliance = &Score{Passed: 4, Total: 4}, compliantResult()
	case StatusFailed:
		tr.Score, tr.Compliance = &Score{Passed: 1, Total: 4}, compliantResult()
	case StatusUnexamined:
		tr.Compliance = resultWith(RuleOwnCopyOnly, StateUnexamined)
		tr.Reason = "the compliance check could not examine the copy"
	case StatusNotStarted:
		tr.Reason, tr.FinishedAt, tr.ModelTimeMS, tr.GateWaitMS = "the provider was down before the copy began", time.Time{}, 0, 0
	}
	if pin.Source == SourceHost && tr.Score != nil {
		// A host copy's trial is built from the host's receipt alone.
		tr.Score.Detail = hostScoreDetail
		tr.Unpinned = hostTrialUnpinned(CopyReceipt{ModelServed: tr.ModelServed, HarnessVersion: tr.HarnessVersion,
			ModelTimeMS: tr.ModelTimeMS, GateWaitMS: tr.GateWaitMS}, true)
	}
	return tr
}

// reportLab is a run of goodDeclaration, each edit made to its inputs, with
// host-big's three copies of feature-a passing and or-cheap's two passing, one
// failing, one never started and one unexamined, and reviews of them.
func reportLab(t *testing.T, edits ...func(*runInputs)) (*gittest.Repo, string, lab.Minted, RunRecord) {
	t.Helper()
	r, home, m, rec, _ := checkLab(t, edits...)
	trials := []TrialRecord{
		codeTrial(t, rec, "host-big", 1, StatusPassed),
		codeTrial(t, rec, "host-big", 2, StatusPassed),
		codeTrial(t, rec, "host-big", 3, StatusPassed),
		codeTrial(t, rec, "or-cheap", 1, StatusPassed),
		codeTrial(t, rec, "or-cheap", 2, StatusFailed),
		codeTrial(t, rec, "or-cheap", 3, StatusPassed),
		codeTrial(t, rec, "or-cheap", 4, StatusNotStarted),
		codeTrial(t, rec, "or-cheap", 5, StatusUnexamined),
	}
	for _, tr := range trials {
		if err := writeTrial(r.Root(), m.ID, tr); err != nil {
			t.Fatalf("writeTrial(%s): %v", tr.Key, err)
		}
	}
	copies := []ReviewedCopy{
		{Label: "copy-00000001", Key: TrialKey{"feature-a", "or-cheap", 2}, Truth: TruthFailing},
		{Label: "copy-00000002", Key: TrialKey{"feature-a", "or-cheap", 1}, Truth: TruthPassing},
		{Label: "copy-00000003", Key: TrialKey{"feature-a", "host-big", 1}, Truth: TruthPassing},
	}
	verdicts := []ReviewVerdict{
		{Reviewer: "local-rev", Label: "copy-00000001", Defective: true},
		{Reviewer: "local-rev", Label: "copy-00000002", Defective: true},
		{Reviewer: "local-rev", Label: "copy-00000003", Defective: false},
	}
	if err := recordReviews(r.Root(), m.ID, rec.RunID, "feature-a", copies, verdicts); err != nil {
		t.Fatalf("recordReviews: %v", err)
	}
	return r, home, m, rec
}

func codeCell(t *testing.T, rep Report, cell string) CodeCell {
	t.Helper()
	if len(rep.Code) != 1 {
		t.Fatalf("the report holds %d code tasks; want feature-a alone", len(rep.Code))
	}
	for _, c := range rep.Code[0].Cells {
		if c.Cell == cell {
			return c
		}
	}
	t.Fatalf("the code report shows no cell %s: %+v", cell, rep.Code[0].Cells)
	return CodeCell{}
}

// A finished code task's report shows each builder's pass rate on the hidden
// tests with its interval, its pass^k, the reviewers' confirmed findings, its
// cost per passing build stated per source with a note on what is
// comparable, and its model time excluding the time spent waiting on gates;
// a copy never started, or one the compliance check could not examine, is
// counted apart, never in the rate.
func TestTheCodeReportShowsEachBuildersPassRateReviewsCostAndTime(t *testing.T) {
	r, _, m, rec := reportLab(t)
	rep, err := BuildReport(r.Root(), m.ID, rec.RunID)
	if err != nil {
		t.Fatalf("BuildReport: %v", err)
	}
	cheap := codeCell(t, rep, "or-cheap")
	if cheap.PassAt1.Passed != 2 || cheap.PassAt1.Attempts != 3 || cheap.PassAt1.Interval == nil {
		t.Fatalf("or-cheap's pass@1 = %+v; want 2 of 3 with an interval", cheap.PassAt1)
	}
	if cheap.NotStarted != 1 || cheap.Unexamined != 1 {
		t.Fatalf("or-cheap sets apart %d not started and %d unexamined; want one each", cheap.NotStarted, cheap.Unexamined)
	}
	if len(cheap.PassHatK) != 2 || cheap.PassHatK[1].K != 2 {
		t.Fatalf("or-cheap's pass^k = %+v; want k of 1 and 2", cheap.PassHatK)
	}
	if cheap.Reviews.Confirmed != 1 || cheap.Reviews.Unconfirmed != 1 {
		t.Fatalf("or-cheap's reviews = %+v; want one confirmed finding and one flag on a passing copy", cheap.Reviews)
	}
	if len(cheap.Costs) != 1 || cheap.Costs[0].Unit != oracle.CostUSD || cheap.Costs[0].PerPass == nil ||
		*cheap.Costs[0].PerPass != 1.0 || cheap.Costs[0].Comparable == "" {
		t.Fatalf("or-cheap's cost = %+v; want 2.00 usd over two passes, 1.00 a passing build, with a note", cheap.Costs)
	}
	if cheap.ModelTimeMS != 120000 || cheap.GateWaitMS != 60000 {
		t.Fatalf("or-cheap's time = %d model, %d gates; want 120000 a copy of model time, gate waits apart", cheap.ModelTimeMS, cheap.GateWaitMS)
	}
	big := codeCell(t, rep, "host-big")
	if big.PassAt1.Passed != 3 || len(big.Costs) != 1 || big.Costs[0].PerPass != nil || !strings.Contains(big.Costs[0].Comparable, "host") {
		t.Fatalf("host-big = %+v; want 3 of 3 and a host cost that is not comparable with a metered one", big)
	}
	if len(rep.Code[0].Reviewers) != 1 || rep.Code[0].Reviewers[0].Reviewer != "local-rev" {
		t.Fatalf("the reviewers' scores = %+v; want local-rev's", rep.Code[0].Reviewers)
	}
}

// A comparison resting on fewer repeats than its power estimate asks for says
// the smallest difference it could have detected rather than ranking; with
// the repeats it asks for, it ranks.
func TestAComparisonUnderItsPowerEstimateStatesTheSmallestDetectableDifference(t *testing.T) {
	r, _, m, rec := reportLab(t)
	rep, err := BuildReport(r.Root(), m.ID, rec.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Code[0].Comparisons) != 1 {
		t.Fatalf("comparisons = %+v; want host-big against or-cheap", rep.Code[0].Comparisons)
	}
	c := rep.Code[0].Comparisons[0]
	if c.Ranked || c.Ahead != "" || c.MinDetectable <= 0 || !strings.Contains(c.Note, "smallest difference") {
		t.Fatalf("an unsized comparison = %+v; want it unranked, stating the smallest difference it could detect", c)
	}
	if c.Kind != "model and harness" {
		t.Fatalf("host-big against or-cheap compares %q; want model and harness, across sources", c.Kind)
	}

	// The same copies, with a smallest difference declared that three
	// repeats each can detect: the comparison is ranked.
	r2, _, m2, rec2 := reportLab(t, func(in *runInputs) {
		in.Declaration = []byte(strings.Replace(string(in.Declaration), `"max": 20}`, `"max": 20, "min_difference": 0.95}`, 1))
	})
	rep2, err := BuildReport(r2.Root(), m2.ID, rec2.RunID)
	if err != nil {
		t.Fatal(err)
	}
	c2 := rep2.Code[0].Comparisons[0]
	if !c2.Ranked || c2.Needed == 0 || c2.Needed > 3 {
		t.Fatalf("a comparison at its power estimate = %+v; want it ranked", c2)
	}

	// Asked to detect a small difference, three repeats each fall short.
	r3, _, m3, rec3 := reportLab(t, func(in *runInputs) {
		in.Declaration = []byte(strings.Replace(string(in.Declaration), `"max": 20}`, `"max": 20, "min_difference": 0.1}`, 1))
	})
	rep3, err := BuildReport(r3.Root(), m3.ID, rec3.RunID)
	if err != nil {
		t.Fatal(err)
	}
	c3 := rep3.Code[0].Comparisons[0]
	if c3.Ranked || c3.Needed <= 3 || !strings.Contains(c3.Note, "fewer than") {
		t.Fatalf("a comparison under its power estimate = %+v; want it unranked, naming the repeats it needed", c3)
	}
}

// writingRun records a writing task's judging in rec's lab by judges member
// judges, each ranking their subset in the order shown.
func writingRun(t *testing.T, r *gittest.Repo, m lab.Minted, rec RunRecord, judges int) {
	t.Helper()
	in := writingInputs(2, 3, judges)
	in.Run = rec.RunID
	for i := range in.Copies {
		in.Copies[i].Key.Cell = []string{"host-big", "or-cheap"}[i/3]
	}
	p, k := draw(t, in, seedA)
	if err := writeJudging(r.Root(), m.ID, p, k); err != nil {
		t.Fatal(err)
	}
	var vs []Verdict
	for i, s := range p.Slots {
		vs = append(vs, verdictOn(s, JudgeID{GitHub: []string{"member-one", "member-two", "member-three", "member-four"}[i]}, t0))
	}
	if _, err := recordVerdicts(r.Root(), m.ID, rec.RunID, "doc-page", verdictFile(t, k.Packet, vs...), ""); err != nil {
		t.Fatal(err)
	}
}

// A written doc's report rates each model from all the judges' rankings with
// its interval, gives the judges' agreement and each copy's length; with
// fewer than three judges it gives each judge's ranking and the good-enough
// counts only.
func TestTheWritingReportRatesModelsAndFallsBackBelowThreeJudges(t *testing.T) {
	r, _, m, rec := reportLab(t)
	writingRun(t, r, m, rec, 3)
	rep, err := BuildReport(r.Root(), m.ID, rec.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Writing) != 1 {
		t.Fatalf("writing reports = %+v; want doc-page's", rep.Writing)
	}
	w := rep.Writing[0]
	if w.Members != 3 || w.Floor != "" || len(w.Ratings) != 2 || w.Agreement == nil || len(w.Copies) == 0 {
		t.Fatalf("doc-page judged by three = %+v; want ratings, agreement and lengths", w)
	}
	for _, g := range w.GoodEnough {
		if g.Of == 0 {
			t.Fatalf("good-enough counts = %+v; want every model's marks counted", w.GoodEnough)
		}
	}
	for _, c := range w.Copies {
		if c.Words == 0 {
			t.Fatalf("copy %s shows no length", c.Key)
		}
	}

	r2, _, m2, rec2 := reportLab(t)
	writingRun(t, r2, m2, rec2, 2)
	rep2, err := BuildReport(r2.Root(), m2.ID, rec2.RunID)
	if err != nil {
		t.Fatal(err)
	}
	w2 := rep2.Writing[0]
	if w2.Members != 2 || w2.Floor == "" || w2.Ratings != nil || w2.Agreement != nil || len(w2.Rankings) != 2 || len(w2.GoodEnough) != 2 {
		t.Fatalf("doc-page judged by two = %+v; want each judge's ranking and the good-enough counts only", w2)
	}
}

// Every report lists the run's limitations (what it could not pin, examine or
// detect at its repeat count) and the improvements it suggests for the next
// run.
func TestTheReportListsLimitationsAndSuggestedImprovements(t *testing.T) {
	r, _, m, rec := reportLab(t, func(in *runInputs) { in.PromptSHA256 = nil })
	writingRun(t, r, m, rec, 2)
	rep, err := BuildReport(r.Root(), m.ID, rec.RunID)
	if err != nil {
		t.Fatal(err)
	}
	all := strings.Join(rep.Limitations, "\n")
	for _, want := range []string{
		"could not be pinned",        // an unpinned field of the run record
		"asserted, not proven",       // the host cell's isolation
		"network policy",             // asserted for every copy
		"smallest difference",        // an unsized comparison
		"model and harness together", // the harness confound
		"could not examine",          // the unexamined copy
		"never started",              // the copy an outage stopped
		"fewer than three judges",    // the writing floor
		"guess",                      // people are not asked to guess
		"fidelity",                   // the check task not yet judged
	} {
		if !strings.Contains(all, want) {
			t.Errorf("the limitations say nothing of %q:\n%s", want, all)
		}
	}
	improve := strings.Join(rep.Improvements, "\n")
	for _, want := range []string{"min_difference", "three judges", "transcript", "pin"} {
		if !strings.Contains(improve, want) {
			t.Errorf("the improvements suggest nothing of %q:\n%s", want, improve)
		}
	}
}
