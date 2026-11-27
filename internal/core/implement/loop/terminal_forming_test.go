//go:build unix

package loop

import (
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The drive's let-go of a bundle the drain is forming
// (spc-2611201455051897 step 4): it releases the lead's hold and every
// member's plainly and clears `forming`, as it releases a judgement's hold;
// and a let-go the drive's comparison stops names each of those holds with
// when it lapses.

// formingDrain is a live drain on the bundle fixture forming a bundle of the
// lead and its two members, each reserved on origin by the drain at c's time
// and recorded with the commit that reservation made.
func formingDrain(t *testing.T) (repo *gittest.Repo, bare string, c *clock, taken map[string]string) {
	t.Helper()
	repo, bare = bundleRepo(t)
	c = &clock{t: time.Now().UTC().Truncate(time.Second)}
	taken = map[string]string{}
	holds := map[string]*DrainHold{}
	for _, id := range bundleIssues {
		res, err := claim.Reserve(repo.Root(), claim.Request{Record: id, Footprint: []string{bundleFiles[id]}, Drain: true}, clockAt(c.t))
		if err != nil {
			t.Fatalf("reserving %s: %v", id, err)
		}
		taken[id] = res.Commit
		holds[id] = &DrainHold{Commit: res.Commit, ClaimedAt: res.ClaimedAt, ExpiresAt: res.ExpiresAt}
	}
	st := DrainState{SchemaVersion: DrainSchemaVersion, StartedAt: c.t, UpdatedAt: c.t,
		BundleMax: &PaceValue{Value: 5, Layer: "repo", Origin: ".abcd/config.json"},
		Forming: &DrainForming{Lead: bundleLead, Hold: holds[bundleLead], Members: []DrainFormingMember{
			{Issue: bundleTwo, Hold: holds[bundleTwo]}, {Issue: bundleThree, Hold: holds[bundleThree]},
		}}}
	writeDrainAt(t, repo.Root(), st, DrainSchemaVersion)
	return repo, bare, c, taken
}

// sayInto is a terminal's Say that keeps every line.
func sayInto(lines *[]string) func(string) {
	return func(s string) { *lines = append(*lines, s) }
}

// TestTheDrivesLetGoReleasesEveryFormingHoldPlainly: a drive that ends while
// the drain forms a bundle of a lead and two members releases all three
// holds plainly, each over the commit the drain took, publishing no
// hand-back, and clears `forming`. Watched failing before letGo read
// `forming`: all three were still held on origin.
func TestTheDrivesLetGoReleasesEveryFormingHoldPlainly(t *testing.T) {
	repo, bare, c, taken := formingDrain(t)
	var said []string
	tm := newTerminal(repo.Root(), TerminalOptions{Options: Options{Now: c.now}, Say: sayInto(&said)})
	tm.letGo()
	for _, id := range bundleIssues {
		releasedOver(t, bare, id, taken[id])
		if kind, _ := tombstoneTrailers(t, bare, id); kind != "" {
			t.Fatalf("the let-go releases %s plainly, publishing no hand-back: %q", id, kind)
		}
		if !slices.ContainsFunc(said, func(s string) bool { return strings.HasPrefix(s, id+": released the reservation on "+id) }) {
			t.Fatalf("the let-go says it released %s:\n%s", id, strings.Join(said, "\n"))
		}
	}
	st, live, err := readDrain(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if !live || st.Forming != nil {
		t.Fatalf("the let-go clears forming and leaves the drain live: live %v, forming %+v", live, st.Forming)
	}
}

// TestALetGoTheComparisonStopsNamesEveryFormingHold: a let-go the drive's
// comparison stops, because a setting what the drive holds reads changed,
// releases nothing and names the lead's hold and each member's, with when
// it lapses, what renews it, and how the person releases it. Watched
// failing before leftHeld read `forming`: no hold was named.
func TestALetGoTheComparisonStopsNamesEveryFormingHold(t *testing.T) {
	repo, bare, c, taken := formingDrain(t)
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	repo.Git("config", "core.sshCommand", "/bin/true")
	var said []string
	tm := newTerminal(repo.Root(), TerminalOptions{Options: Options{Now: c.now}, Say: sayInto(&said)})
	tm.o.Options.remote = id
	tm.letGo()
	all := strings.Join(said, "\n")
	if !strings.Contains(all, "nothing this drive holds was released: releasing is a push") {
		t.Fatalf("the let-go says nothing was released:\n%s", all)
	}
	line := regexp.MustCompile("(?m)^(\\S+): its reservation was not released and lapses at (\\S+) " + regexp.QuoteMeta(unlessFormingRenewed) +
		"; once what the drive holds is as it should be, release it with `abcd reserve (\\S+) --release`, or start the drive again, which takes the run up$")
	var named []string
	for _, m := range line.FindAllStringSubmatch(all, -1) {
		if m[1] != m[3] {
			t.Fatalf("a line names one issue and its release: %q", m[0])
		}
		if at, err := time.Parse(time.RFC3339, m[2]); err != nil || !at.After(c.t) {
			t.Fatalf("the lease's end is a time to come: %q %v", m[2], err)
		}
		named = append(named, m[1])
	}
	if !slices.Equal(named, bundleIssues) {
		t.Fatalf("each forming hold is named, the lead's first: %v\n%s", named, all)
	}
	for _, id := range bundleIssues {
		heldStill(t, bare, id, taken[id])
	}
	if st, _, err := readDrain(repo.Root()); err != nil || st.Forming == nil {
		t.Fatalf("a let-go that releases nothing keeps forming: %+v %v", st.Forming, err)
	}
}

// TestTheDriveNamesAFinishedBundleByWhatItCarried: the drive's line for a
// finished lane names it as the move's note does (laneIssues): a lane of one
// by its issue and run, a bundle with its members, and a bundle whose lead was
// handed back alone by its run and members, never a blank.
func TestTheDriveNamesAFinishedBundleByWhatItCarried(t *testing.T) {
	var lines []string
	tm := newTerminal(t.TempDir(), TerminalOptions{Stages: DefaultStages(), Say: sayInto(&lines)})
	tm.sayMove(DrainResult{Finished: []DrainFinished{
		{Issue: "iss-1", RunID: "run-1", Done: true},
		{Issue: "iss-2", RunID: "run-2", Done: true, Bundle: []string{"iss-3", "iss-4"}},
		{Issue: "", RunID: "run-5", Done: true, Bundle: []string{"iss-6", "iss-7"}},
	}})
	want := []string{
		"finished iss-1's run run-1",
		"finished iss-2's run run-2 with its member(s) iss-3 and iss-4",
		"finished run run-5 with its member(s) iss-6 and iss-7",
	}
	if !slices.Equal(lines, want) {
		t.Fatalf("the drive names each finished lane by what it carried:\n got %q\nwant %q", lines, want)
	}
}

// TestALetGoAfterABundleOpensLeavesItsLanesHolds: the move that opens a
// bundle's lane clears `forming` in the same write that records the lane, so
// a drive's let-go after it finds no forming hold to release: the lead's and
// the member's holds, now the lane's, stay held at the commits the drain
// took while the lane is parked at its pull request.
func TestALetGoAfterABundleOpensLeavesItsLanesHolds(t *testing.T) {
	repo, bare := bundleDrainRepo(t)
	bundleMachineConfig(t, `{"drain":{"bundle_max":2}}`)
	first := mustDrain(t, repo, Options{}, DrainOptions{})
	lead, m1, _, _ := orderOf(t, first)
	second := judge(t, repo, Options{}, first, JudgementNo, "", noReason)
	third := judge(t, repo, Options{}, second, JudgementNo, "", noReason)
	if third.Lane == nil || !slices.Equal(third.Lane.Bundle, []string{m1}) {
		t.Fatalf("the bundle's lane opens: %+v", third)
	}
	taken := map[string]string{lead: refOf(t, bare, lead), m1: refOf(t, bare, m1)}
	st := formedState(t, repo)
	if st.Forming != nil || len(st.Lanes) != 1 || st.Lanes[0].RunID != third.Lane.RunID {
		t.Fatalf("one write records the lane and clears forming: %+v", st)
	}
	// Parked at its pull request, the lane keeps its own holds through a let-go.
	st.Lanes[0].Outcome = DrainLanePullRequest
	if err := writeDrain(repo.Root(), st); err != nil {
		t.Fatal(err)
	}
	var said []string
	newTerminal(repo.Root(), TerminalOptions{Say: sayInto(&said)}).letGo()
	for id, commit := range taken {
		heldStill(t, bare, id, commit)
	}
	if len(said) != 0 {
		t.Fatalf("the let-go releases nothing of the lane's:\n%s", strings.Join(said, "\n"))
	}
}
