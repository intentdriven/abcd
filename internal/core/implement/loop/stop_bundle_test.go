package loop

import (
	"strings"
	"testing"
	"time"
)

// A stop of a run with members (spc-2611201455051897 step 1, criterion 5):
// the stop lets every issue the run holds go, plainly, and the run no longer
// records a reservation on any of them.

// TestAStopOfARunWithMembersReleasesEveryIssuePlainly: a stop that settles a
// run of three issues releases all three with no hand-back published, names
// each in what it released, and clears the run's reserved_at and every
// member's.
func TestAStopOfARunWithMembersReleasesEveryIssuePlainly(t *testing.T) {
	repo, bare := bundleRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	runID := bundleRun(t, repo, now.Add(-10*time.Minute))
	o, _, _ := stopped(t)
	o.Now = func() time.Time { return now }
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	for _, id := range bundleIssues {
		r := tipOf(t, bare, id)
		if !r.released {
			t.Fatalf("the stop releases %s: %+v", id, r)
		}
		if kind, _ := tombstoneTrailers(t, bare, id); kind != "" {
			t.Fatalf("the stop releases %s plainly, publishing no hand-back: %q", id, kind)
		}
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if st.ReservedAt != nil {
		t.Fatalf("the run still says it holds its lead: %v", st.ReservedAt)
	}
	for _, m := range st.Bundle {
		if m.ReservedAt != nil {
			t.Fatalf("the run still says it holds %s: %v", m.Issue, m.ReservedAt)
		}
	}
	released := linesAt(st, StageReservationReleased)
	if got := issuesOf(released); len(got) != 3 || got[0] != bundleLead || got[1] != bundleTwo || got[2] != bundleThree {
		t.Fatalf("each release's line names its issue: %+v", released)
	}
	c := controlOf(t, repo.Root(), runID)
	for _, id := range bundleIssues {
		if c.Stop == nil || !strings.Contains(c.Stop.Released, id) {
			t.Fatalf("the stop's account names %s: %+v", id, c.Stop)
		}
	}
}
