package intent

import (
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/grounds"
)

const (
	pickIntentClear = "---\nid: itd-10\n---\n# a\n\n## Acceptance Criteria\n\n" +
		"- **Given** x, **when** y, **then** z.\n- **Given** p, **when** q,\n  **then** r.\n\n## Grounds\n"
	pickIntentHalf = "---\nid: itd-11\n---\n# b\n\n## Acceptance Criteria\n\n" +
		"- **Given** x, **when** y, **then** z.\n- The page states the flags.\n"
	pickSpecFootprint = "---\nid: spc-1\n---\n# s\n\n## Summary\n\nWritten.\n\n## Footprint\n\n" +
		"- packages: internal/core/intent, internal/core/spec\n- tests: the score over fixtures\n"
	pickSpecBare = "---\nid: spc-2\n---\n# s\n\n## Summary\n\nWritten.\n"
)

// TestReadinessScoresTheThreeParts is spec scope 2: criteria clarity from the
// Given-When-Then bullets (a clause on a continuation line counts), the test
// path and the footprint from the spec's `## Footprint`, at equal weight.
func TestReadinessScoresTheThreeParts(t *testing.T) {
	s := readiness(pickIntentClear, pickSpecFootprint)
	if s.Criteria.Points != PartPoints || s.TestPath.Points != PartPoints || s.Footprint.Points != PartPoints/2 {
		t.Fatalf("parts = %+v", s)
	}
	if s.Total != PartPoints+PartPoints+PartPoints/2 || s.NoFootprint {
		t.Fatalf("total = %d, no footprint = %v", s.Total, s.NoFootprint)
	}
	if h := readiness(pickIntentHalf, pickSpecFootprint); h.Criteria.Points != PartPoints/2 ||
		!strings.Contains(h.Criteria.Detail, "1 of 2") {
		t.Fatalf("half the criteria in Given-When-Then form: %+v", h.Criteria)
	}
}

// TestReadinessWithoutFootprintReadsZeroAndSaysSo is criterion 8: a spec with
// no `## Footprint` section scores zero on the test path and the footprint,
// and the reason says the spec carries no footprint.
func TestReadinessWithoutFootprintReadsZeroAndSaysSo(t *testing.T) {
	s := readiness(pickIntentClear, pickSpecBare)
	if s.TestPath.Points != 0 || s.Footprint.Points != 0 || !s.NoFootprint {
		t.Fatalf("an absent footprint reads zero and is flagged: %+v", s)
	}
	if s.Total != PartPoints {
		t.Fatalf("total = %d, want the criteria part alone", s.Total)
	}
	p, _ := Choose([]PickCandidate{{ID: "itd-10", Score: s}})
	text := PickEntryText("run-2609291600001234", time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC), p)
	if !strings.Contains(text, "itd-10 100 (criteria 100, test path 0, footprint 0, its spec carries no footprint)") {
		t.Fatalf("the reason says the spec carries no footprint: %s", text)
	}
}

// TestChooseTakesTheOldestOfATieAndSaysSo is criterion 3: candidates whose
// scores tie are placed by age, and the reason says the tie was broken by age.
func TestChooseTakesTheOldestOfATieAndSaysSo(t *testing.T) {
	same := readiness(pickIntentClear, pickSpecBare)
	p, ok := Choose([]PickCandidate{
		{ID: "itd-2609211116005482", Score: same},
		{ID: "itd-50", Score: same},
		{ID: "itd-2609201916151817", Score: same},
	})
	if !ok || p.Chosen.ID != "itd-50" || p.RunnerUp == nil || p.RunnerUp.ID != "itd-2609201916151817" || !p.TieBrokenByAge {
		t.Fatalf("the oldest of a tie is taken, the ordinal before every timestamp id: %+v", p)
	}
	text := PickEntryText("run-2609291600001234", time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC), p)
	if !strings.Contains(text, "the tie was broken by age") {
		t.Fatalf("the reason says the tie was broken by age: %s", text)
	}
}

// TestPickEntryIsOneValidRunMarkedGround is criterion 2's entry: the text
// opens with the run's marker, names every candidate with its score, the rule,
// the runner-up and why it lost, and the falsifier, and passes the grounds
// writer's own gate.
func TestPickEntryIsOneValidRunMarkedGround(t *testing.T) {
	p, _ := Choose([]PickCandidate{
		{ID: "itd-11", Score: readiness(pickIntentHalf, pickSpecBare)},
		{ID: "itd-10", Score: readiness(pickIntentClear, pickSpecFootprint)},
	})
	text := PickEntryText("run-2609291600001234", time.Date(2026, 9, 29, 23, 0, 0, 0, time.UTC), p)
	g, err := grounds.New(grounds.Pursued, text)
	if err != nil {
		t.Fatalf("the entry passes the grounds writer's gate: %v", err)
	}
	if !strings.HasPrefix(text, "picked by run run-2609291600001234 on 2026-09-29; ") || !isRunPick(g) {
		t.Fatalf("the entry opens with the run's marker: %s", text)
	}
	for _, want := range []string{"itd-10 250", "itd-11 50", "rule: " + PickRule,
		"runner-up: itd-11, which lost on score (50 against 250)", "falsifier: " + PickFalsifier} {
		if !strings.Contains(text, want) {
			t.Fatalf("the entry names %q: %s", want, text)
		}
	}
	if isRunPick(grounds.Grounds{Token: grounds.Pursued, Text: "we picked by run of luck the thing"}) {
		t.Fatal("a human's entry is not a run's pick")
	}
}

// TestReadyGroundsReportsTheHumanEntryPastARunPick is criterion 2's gate half:
// a run-marked entry below the human's is counted but never reported as the
// most recent conjecture.
func TestReadyGroundsReportsTheHumanEntryPastARunPick(t *testing.T) {
	root := t.TempDir()
	p, _ := Choose([]PickCandidate{{ID: "itd-10", Score: readiness(pickIntentClear, pickSpecBare)}})
	pick := PickEntryText("run-2609291600001234", time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC), p)
	writeFile(t, root, plannedDir+"/itd-10-alpha.md",
		plannedUnlinked("itd-10", "alpha")+
			"\n## Grounds\n\n- pursued: we expect a stamped identity to survive rewording\n- pursued: "+pick+"\n")

	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	g := checkByName(t, res, CheckGrounds)
	if !g.OK || !strings.Contains(g.Detail, "2 recorded ground(s)") {
		t.Fatalf("both entries are counted: %+v", g)
	}
	if !strings.Contains(g.Detail, "most recent pursued: we expect a stamped identity") {
		t.Fatalf("the human's entry is the most recent conjecture: %q", g.Detail)
	}
	if !strings.Contains(g.Detail, "1 run pick entry") {
		t.Fatalf("the run's entry is named as skipped: %q", g.Detail)
	}
}
