package loop

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/core/spotcheck"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The merge gate's decision at a lane's landing, recorded for the
// spot-check (spc-2610301925429250 step 1): made from the decision alone,
// never changing what the lane does or asks.

// gateRecordsOf are the merge-gate records the store's local copy holds,
// read with git alone.
func gateRecordsOf(t *testing.T, repo *gittest.Repo) []spotcheck.Decision {
	t.Helper()
	var out []spotcheck.Decision
	for _, d := range storeRecords(t, repo).Decisions {
		if d.Kind == spotcheck.KindMergeGate {
			out = append(out, d)
		}
	}
	return out
}

// parkedAuto walks a lane in auto mode to its pull request with its records
// round failed, so the gate leaves it open and the lane parks, with rec as
// the lane's recorder (nil turns the record off) and before run just ahead
// of the steps that push and ask the gate.
func parkedAuto(t *testing.T, rec func(root, site, pr, head string, d landing.MergeDecision) string, before func(f *landFixture)) (*landFixture, Lane) {
	t.Helper()
	f := autoFixture(t)
	f.validated(t)
	f.step(t)
	f.step(t)
	f.recordsRound(t, "FIX FIRST", "APPROVE")
	f.step(t)
	saved := recordLaneGate
	t.Cleanup(func() { recordLaneGate = saved })
	recordLaneGate = rec
	if rec == nil {
		recordLaneGate = func(string, string, string, string, landing.MergeDecision) string { return "" }
	}
	if before != nil {
		before(f)
	}
	l := f.pushToArm(t)
	if l.Landing == nil || l.Landing.Armed || !strings.HasPrefix(l.Landing.Merge, "left open: ") {
		t.Fatalf("the lane parks at its pull request: %+v\n%s", l.Landing, f.ghLog(t))
	}
	return f, l
}

// laneNote is a parked lane's reason with its pushed head written as HEAD,
// its run as RUN and the spot-check's note taken out, so two fixtures'
// reasons compare.
func laneNote(f *landFixture, l Lane) string {
	m := l.Landing.Merge
	if i := strings.Index(m, "; the spot-check's "); i >= 0 {
		if j := strings.Index(m[i:], "waits on it"); j >= 0 {
			m = m[:i] + m[i+j+len("waits on it"):]
		}
	}
	h := l.Landing.Pushed
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(m, h, "HEAD"), h[:12], "HEAD"), f.runID, "RUN")
}

// stepParked takes one step of a parked lane, which waits for its merge and
// says so as a contention.
func stepParked(t *testing.T, f *landFixture) {
	t.Helper()
	if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err != nil {
		if r, ok := AsRefusal(err); !ok || !r.Contention {
			t.Fatalf("a parked lane's step: %v", err)
		}
	}
}

func ghCalls(f *landFixture, t *testing.T) int {
	return len(strings.Split(strings.TrimSpace(f.ghLog(t)), "\n"))
}

// TestAParkedLanesGateDecisionIsRecorded: in auto mode the gate's decision
// over the lane's pull request is recorded with the lane site, the pull
// request, the pushed head and the answer as the gate gave it, once however
// often the parked lane asks again; the lane parks with the same reason and
// asks the forge as often as with no record; and a store the write cannot
// read changes nothing but the note in the reason.
func TestAParkedLanesGateDecisionIsRecorded(t *testing.T) {
	var handed []landing.MergeDecision
	f, l := parkedAuto(t, func(root, site, pr, head string, d landing.MergeDecision) string {
		handed = append(handed, d)
		return landing.RecordMergeGate(root, site, pr, head, d)
	}, nil)
	recs := gateRecordsOf(t, f.repo)
	if len(handed) != 1 || len(recs) != 1 {
		t.Fatalf("handed %+v, recorded %+v", handed, recs)
	}
	r, d := recs[0], handed[0]
	if r.Site != spotcheck.SiteLane || r.PR != l.PR || r.Head != l.Landing.Pushed || r.Arm || r.Arm != d.Arm || r.Method != d.Method || r.Why != strings.ReplaceAll(d.Why, ".abcd/config.json", "<removed>") || r.Unread != d.Unread {
		t.Fatalf("the record is %+v; the gate decided %+v", r, d)
	}
	// The three facts the gate set as it decided, on the gate's answer and on
	// the record made from it.
	if d.Mode != landing.ModeAuto || !d.AuthorCanWrite || d.ApprovalCounted {
		t.Fatalf("the gate's facts are %+v", d)
	}
	if r.Mode != d.Mode || r.AuthorCanWrite == nil || *r.AuthorCanWrite != d.AuthorCanWrite || r.ApprovalCounted == nil || *r.ApprovalCounted != d.ApprovalCounted {
		t.Fatalf("the record's facts are %+v; the gate's %+v", r, d)
	}
	if !strings.HasPrefix(l.Landing.Merge, "left open: "+d.Why) {
		t.Fatalf("the lane parked on %q; the gate decided %q", l.Landing.Merge, d.Why)
	}
	calls := ghCalls(f, t)
	// The parked lane asks again on its next step: still the one record.
	stepParked(t, f)
	again := ghCalls(f, t) - calls
	if n := len(gateRecordsOf(t, f.repo)); n != 1 || len(handed) != 2 {
		t.Fatalf("the gate asked again kept %d records (handed %d)", n, len(handed))
	}

	g, plain := parkedAuto(t, nil, nil)
	if got := ghCalls(g, t); got != calls {
		t.Fatalf("the lane with the record asked the forge %d times; without it %d", calls, got)
	}
	if laneNote(g, plain) != laneNote(f, l) || strings.Contains(l.Landing.Merge, "spot-check") {
		t.Fatalf("the lane with the record parked on %q; without it %q", l.Landing.Merge, plain.Landing.Merge)
	}
	stepParked(t, g)
	if got := ghCalls(g, t) - calls; got != again {
		t.Fatalf("asked again, the lane with the record asked the forge %d times; without it %d", again, got)
	}
	if len(gateRecordsOf(t, g.repo)) != 0 {
		t.Fatal("the lane with the record off wrote one")
	}

	var blob string
	h, failed := parkedAuto(t, landing.RecordMergeGate, func(f *landFixture) {
		blob = strings.TrimSpace(f.repo.Git("hash-object", "-w", "AGENTS.md"))
		f.repo.Git("update-ref", spotcheck.LocalRef, blob)
	})
	if !strings.Contains(failed.Landing.Merge, "the spot-check's record of this decision was not written") {
		t.Fatalf("the lane over an unwritable store parked on %q", failed.Landing.Merge)
	}
	if laneNote(h, failed) != laneNote(g, plain) || failed.Landing.Armed != plain.Landing.Armed || ghCalls(h, t) != calls {
		t.Fatalf("the lane over an unwritable store parked on %q, asking %d times; with no record %q, %d", failed.Landing.Merge, ghCalls(h, t), plain.Landing.Merge, calls)
	}
	if got := h.repo.Git("rev-parse", spotcheck.LocalRef); got != blob {
		t.Fatalf("a record was written over an unreadable store: %s", got)
	}
}
