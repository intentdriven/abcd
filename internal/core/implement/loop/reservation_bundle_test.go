package loop

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The reservation acts over a run with members (spc-2611201455051897 step 1,
// criterion 5): every member holds its own reservation on origin, renewed,
// fenced before the push and the arming, and released as the lane ends, each
// line naming its issue; the widening judges the branch against the members'
// holds together. Each runs against the fixture's bare origin.

// tipOf is the reservation on id as origin holds it.
func tipOf(t *testing.T, bare, id string) reservation {
	t.Helper()
	return originReservation(t, bare, claim.RefPrefix+id)
}

// tipsOf is the reservation on each bundle issue as origin holds it.
func tipsOf(t *testing.T, bare string) map[string]reservation {
	t.Helper()
	out := map[string]reservation{}
	for _, id := range bundleIssues {
		out[id] = tipOf(t, bare, id)
	}
	return out
}

// bundleContext is the step's context over the run as it stands, at now.
func bundleContext(t *testing.T, repo *gittest.Repo, runID string, now time.Time) Context {
	t.Helper()
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	return Context{RepoRoot: repo.Root(), RunDir: runRel(runID), State: st, Now: now}
}

// linesAt is each record line at stage, in order.
func linesAt(st State, stage string) []Entry {
	var out []Entry
	for _, e := range st.Record {
		if e.Stage == stage {
			out = append(out, e)
		}
	}
	return out
}

// issuesOf is the issue each line names, in order.
func issuesOf(lines []Entry) []string {
	out := make([]string, len(lines))
	for i, e := range lines {
		out[i] = e.Issue
	}
	return out
}

// bundleBranch gives the run's lane the branch named off main that changes
// each of files, and returns the lane as the context carries it.
func bundleBranch(t *testing.T, repo *gittest.Repo, c *Context, branch string, files ...string) Lane {
	t.Helper()
	repo.Git("checkout", "-q", "-b", branch, "main")
	for _, f := range files {
		p := filepath.Join(repo.Root(), filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("corrected\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	repo.Git("add", "--", ".")
	repo.Git("commit", "-q", "-m", "the lane's fix")
	repo.Git("checkout", "-q", "main")
	c.State.Lanes[0].Branch = branch
	return c.State.Lanes[0]
}

// TestARunWithMembersRenewsEachAtHalfItsLease: a renewal finds each issue of a
// run with members half through its lease and renews every one, each line
// naming its issue.
func TestARunWithMembersRenewsEachAtHalfItsLease(t *testing.T) {
	repo, bare := bundleRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	runID := bundleRun(t, repo, now.Add(-61*time.Minute))
	before := tipsOf(t, bare)
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if note := renewReservation(repo.Root(), &st, now, nil); note != "" {
		t.Fatalf("the renewal says nothing went wrong: %q", note)
	}
	after := tipsOf(t, bare)
	for _, id := range bundleIssues {
		if after[id].parent != before[id].commit || after[id].released || !after[id].claimed.Equal(now) {
			t.Fatalf("%s is renewed as its tip's child at the step: before %+v, after %+v", id, before[id], after[id])
		}
	}
	renewed := linesAt(st, StageReservationRenewed)
	if !slices.Equal(issuesOf(renewed), bundleIssues) {
		t.Fatalf("each renewal's line names its issue: %+v", renewed)
	}
	for i, e := range renewed {
		if !strings.Contains(e.Note, bundleIssues[i]) {
			t.Fatalf("the line for %s names it: %q", bundleIssues[i], e.Note)
		}
	}
	// Under half its lease nothing more is pushed.
	if note := renewReservation(repo.Root(), &st, now.Add(time.Minute), nil); note != "" || len(linesAt(st, StageReservationRenewed)) != 3 {
		t.Fatalf("a renewal under half its lease pushes nothing: %q %+v", note, linesAt(st, StageReservationRenewed))
	}
}

// TestAWaitersRenewalRenewsARunWhoseMemberAloneIsDue: a waiter's renewal
// renews a run with members when one member alone is half through its lease.
func TestAWaitersRenewalRenewsARunWhoseMemberAloneIsDue(t *testing.T) {
	repo, bare := bundleRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	runID := bundleRun(t, repo, now.Add(-61*time.Minute))
	// The lead and the first member were renewed a minute ago; the third member
	// alone is half through its lease.
	for _, id := range []string{bundleLead, bundleTwo} {
		if _, err := claim.Reserve(repo.Root(), claim.Request{Record: id, Renew: true}, clockAt(now.Add(-time.Minute))); err != nil {
			t.Fatal(err)
		}
	}
	before := tipsOf(t, bare)
	got := RenewRunsIn(repo.Root(), runID, now)
	after := tipsOf(t, bare)
	if after[bundleThree].parent != before[bundleThree].commit {
		t.Fatalf("the member half through its lease is renewed: %+v %+v", after[bundleThree], got)
	}
	for _, id := range []string{bundleLead, bundleTwo} {
		if after[id].commit != before[id].commit {
			t.Fatalf("%s, under half its lease, is not renewed: %+v", id, after[id])
		}
	}
}

// TestTheFencesRefuseWhenAMembersHoldIsAnothers: the push fence and the land
// fence each refuse, naming the member, when a member's hold has moved to
// another copy, and push nothing of their act.
func TestTheFencesRefuseWhenAMembersHoldIsAnothers(t *testing.T) {
	repo, bare := bundleRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	runID := bundleRun(t, repo, now.Add(-30*time.Minute))
	p2 := otherCopy(t, bare, patEmail)
	if res, err := claim.Reserve(p2, claim.Request{Record: bundleThree, Footprint: []string{"docs/c.md"}, TakeOver: true}, clockAt(now.Add(-time.Minute))); err != nil || res.Act != claim.ActTakenOver {
		t.Fatalf("P's other copy takes the third member over: %+v %v", res, err)
	}
	taken := tipOf(t, bare, bundleThree)
	c := bundleContext(t, repo, runID, now)
	lane := bundleBranch(t, repo, &c, "build/bundle-fence", "docs/a.md")
	fences := map[string]func() error{
		"push": func() error { _, _, err := pushFence(c, lane); return err },
		"land": func() error { _, _, err := landFence(c, lane, fenceLease); return err },
	}
	for name, f := range fences {
		err := f()
		r, ok := AsRefusal(err)
		if !ok || !r.Contention || !strings.Contains(r.Reason, "the reservation on "+bundleThree+" moved to your copy on another computer") {
			t.Fatalf("the %s fence refuses naming the member: %v", name, err)
		}
		if got := tipOf(t, bare, bundleThree); got.commit != taken.commit {
			t.Fatalf("the %s fence moved the other copy's hold: %+v", name, got)
		}
	}
}

// TestTheReleaseOnLandingTombstonesEveryMember: the lane that completes a run
// with members tombstones the lead's and every member's reservation.
func TestTheReleaseOnLandingTombstonesEveryMember(t *testing.T) {
	repo, bare := bundleRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	runID := bundleRun(t, repo, now.Add(-10*time.Minute))
	c := bundleContext(t, repo, runID, now)
	clause := tidyReservation(c, c.State.Lanes[0])
	for id, r := range tipsOf(t, bare) {
		if !r.released {
			t.Fatalf("%s is tombstoned on landing: %+v", id, r)
		}
		if !strings.Contains(clause, id) {
			t.Fatalf("the landing's clause names %s: %q", id, clause)
		}
	}
}

// TestTheReleaseOnHandBackReleasesEachIssueWithItsKind: a run with members
// handed back releases each issue it holds with the hand-back's kind, each
// line naming its issue.
func TestTheReleaseOnHandBackReleasesEachIssueWithItsKind(t *testing.T) {
	repo, bare := bundleRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	runID := bundleRun(t, repo, now.Add(-10*time.Minute))
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	releaseOnHandBack(repo.Root(), &st, HandBack{Kind: HandBackStuck}, now)
	for _, id := range bundleIssues {
		if kind, _ := tombstoneTrailers(t, bare, id); kind != HandBackStuck {
			t.Fatalf("%s is tombstoned with the hand-back's kind: %q", id, kind)
		}
	}
	if got := issuesOf(linesAt(st, StageReservationReleased)); !slices.Equal(got, bundleIssues) {
		t.Fatalf("each release's line names its issue: %v", got)
	}
}

// TestTheWideningJudgesTheMembersTogether: the branch is judged against the
// holds of the lead and its members together, so a member's page is held, and
// a path none of them holds widens the hold of the first issue the run holds:
// the lead's, or, with the lead handed back alone, the first member's.
func TestTheWideningJudgesTheMembersTogether(t *testing.T) {
	repo, bare := bundleRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	runID := bundleRun(t, repo, now.Add(-10*time.Minute))
	c := bundleContext(t, repo, runID, now)
	lane := bundleBranch(t, repo, &c, "build/bundle-one", "docs/a.md", "docs/b.md", "docs/d.md")
	h := widenLaneChanges(c, lane)
	if h.err != nil || h.gap != "" || !slices.Equal(h.paths, []string{"docs/d.md"}) || h.key != bundleLead {
		t.Fatalf("only the path no member holds is unheld, and widens the lead's hold: %+v", h)
	}
	if fp := footprintOf(t, bare, bundleLead); !slices.Equal(fp, []string{"docs/a.md", "docs/d.md"}) {
		t.Fatalf("the lead's hold is widened: %v", fp)
	}
	for _, id := range []string{bundleTwo, bundleThree} {
		if fp := footprintOf(t, bare, id); !slices.Equal(fp, []string{bundleFiles[id]}) {
			t.Fatalf("%s's hold is not widened: %v", id, fp)
		}
	}
	if widened := linesAt(State{Record: h.lines}, StageWidenedTo); len(widened) != 1 || widened[0].Issue != bundleLead {
		t.Fatalf("the widening's line names the lead: %+v", h.lines)
	}

	// With the lead handed back alone, the first member's hold widens.
	c2 := bundleContext(t, repo, runID, now)
	c2.State.LeadHandBack = &HandBack{At: now, Kind: "decision"}
	lane2 := bundleBranch(t, repo, &c2, "build/bundle-two", "docs/b.md", "docs/e.md")
	h2 := widenLaneChanges(c2, lane2)
	if h2.err != nil || h2.key != bundleTwo || !slices.Equal(h2.paths, []string{"docs/e.md"}) {
		t.Fatalf("the first member's hold widens with the lead handed back alone: %+v", h2)
	}
	if fp := footprintOf(t, bare, bundleTwo); !slices.Equal(fp, []string{"docs/b.md", "docs/e.md"}) {
		t.Fatalf("the first member's hold is widened: %v", fp)
	}
}

// TestWidenOntoMovesTheWidenedPathsBeforeTheFirstHoldIsReleased: the paths
// widened onto one issue's hold are widened onto another's, with the first
// hold left standing for its own release.
func TestWidenOntoMovesTheWidenedPathsBeforeTheFirstHoldIsReleased(t *testing.T) {
	repo, bare := bundleRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	runID := bundleRun(t, repo, now.Add(-10*time.Minute))
	c := bundleContext(t, repo, runID, now)
	lane := bundleBranch(t, repo, &c, "build/bundle-onto", "docs/a.md", "docs/d.md")
	h := widenLaneChanges(c, lane)
	if h.err != nil || len(h.lines) == 0 {
		t.Fatalf("premise: the lead's hold is widened: %+v", h)
	}
	c.State.Record = append(c.State.Record, h.lines...)
	leadTip := tipOf(t, bare, bundleLead)
	lines, err := widenOnto(c, lane, bundleLead, bundleTwo)
	if err != nil {
		t.Fatal(err)
	}
	if fp := footprintOf(t, bare, bundleTwo); !slices.Equal(fp, []string{"docs/b.md", "docs/d.md"}) {
		t.Fatalf("the widened path moves onto the second hold: %v", fp)
	}
	if got := tipOf(t, bare, bundleLead); got.commit != leadTip.commit || got.released {
		t.Fatalf("the first hold stands until its own release: %+v", got)
	}
	if widened := linesAt(State{Record: lines}, StageWidenedTo); len(widened) != 1 || widened[0].Note != "docs/d.md" || widened[0].Issue != bundleTwo {
		t.Fatalf("the moved path's line names the second issue: %+v", lines)
	}
	c.State.Record = append(c.State.Record, lines...)
	if got := widenedPathsFor(c.State, bundleTwo); !slices.Equal(got, []string{"docs/d.md"}) {
		t.Fatalf("the second issue's hold now carries the widened path: %v", got)
	}
}

// TestLetGoAtLandReadsEveryMember: a run waiting at land is let go only when
// no issue of it is held, the lead's reservation and every member's.
func TestLetGoAtLandReadsEveryMember(t *testing.T) {
	repo, _ := bundleRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	runID := bundleRun(t, repo, now.Add(-10*time.Minute))
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	lane := st.Lanes[0]
	lane.Stage = StageLand
	st.Record = append(st.Record, Entry{At: now, Stage: StageReservationReleased, Note: "released", Issue: bundleLead})
	st.ReservedAt = nil
	if letGoAtLand(repo.Root(), st, lane) {
		t.Fatal("a run whose members still hold reservations is not let go")
	}
	for i := range st.Bundle {
		st.Bundle[i].ReservedAt = nil
	}
	if !letGoAtLand(repo.Root(), st, lane) {
		t.Fatal("a run holding no issue is let go")
	}
}

// TestThePausesFingerprintZeroesEveryMembersReservation: a member's
// reservation is renewed as the run's is, so it is no change to the pause's
// fingerprint, while a member's footprint is.
func TestThePausesFingerprintZeroesEveryMembersReservation(t *testing.T) {
	repo, _ := bundleRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	runID := bundleRun(t, repo, now.Add(-10*time.Minute))
	before, err := runFingerprint(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		later := now
		st.Bundle[0].ReservedAt, st.Bundle[1].ReservedAt = &later, nil
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	if after, err := runFingerprint(repo.Root(), runID); err != nil || after != before {
		t.Fatalf("a member's reservation is no change: %v", err)
	}
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		st.Bundle[0].Footprint = []string{"docs/z.md"}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	if after, err := runFingerprint(repo.Root(), runID); err != nil || after == before {
		t.Fatalf("a member's footprint is a change: %v", err)
	}
}

// TestTheChecksReadEveryMember: the start's checks of a run with members run
// the open-issue and peers checks for each member, so a member that is no open
// issue refuses the start, naming it.
func TestTheChecksReadEveryMember(t *testing.T) {
	repo, _ := bundleRepo(t)
	res, err := checkMembers(repo.Root(), bundleLead, []string{bundleTwo, bundleThree}, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("a run of eligible members passes its checks: %+v", res.Checks)
	}
	for _, id := range []string{bundleTwo, bundleThree} {
		var eligible, peers bool
		for _, c := range res.Checks {
			if strings.Contains(c.Detail, id) {
				eligible = eligible || c.Name == CheckEligible
				peers = peers || c.Name == CheckPeers
			}
		}
		if !eligible || !peers {
			t.Fatalf("%s takes the open-issue and the peers check: %+v", id, res.Checks)
		}
	}
	const gone = "iss-2611201500000009"
	res, err = checkMembers(repo.Root(), bundleLead, []string{bundleTwo, gone}, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK || !slices.ContainsFunc(res.Checks, func(c CheckRow) bool { return !c.OK && strings.Contains(c.Detail, gone) }) {
		t.Fatalf("a member that is no open issue refuses the start naming it: %+v", res.Checks)
	}
	if plain, err := check(repo.Root(), bundleLead, "", nil); err != nil || len(plain.Checks) >= len(res.Checks) {
		t.Fatalf("a run with no members takes the lead's checks alone: %+v %v", plain.Checks, err)
	}
}

// TestAParkedRunWhoseMemberAloneIsDueIsRenewed: a drain's move and a drive's
// wait each renew a run parked at land whose member alone is half through its
// lease, the lead's own renewal not yet due, and leave the lead's as it is.
func TestAParkedRunWhoseMemberAloneIsDueIsRenewed(t *testing.T) {
	for name, renew := range map[string]func(t *testing.T, f *landFixture, at time.Time){
		"a drain's move": func(t *testing.T, f *landFixture, at time.Time) {
			if _, err := Drain(f.repo.Root(), Options{Now: func() time.Time { return at }}, DrainOptions{}); err != nil {
				t.Fatal(err)
			}
		},
		"a drive's wait": func(t *testing.T, f *landFixture, at time.Time) {
			if notes := renewParkedDue(f.repo.Root(), at, nil); len(notes) != 0 {
				t.Fatalf("the renewal says %q", notes)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			f, l := parkDrainLane(t)
			f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "OPEN", Mergeable: "MERGEABLE", MergeStateStatus: "BLOCKED"})
			lead := originReservation(t, f.bare, claim.RefPrefix+secondIssue)
			held := lead.claimed
			if _, err := claim.Reserve(f.repo.Root(), claim.Request{Record: bundleTwo, Footprint: []string{"docs/member.md"}, Lease: 2 * time.Hour}, clockAt(held)); err != nil {
				t.Fatal(err)
			}
			if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
				st.Bundle = append(st.Bundle, Member{Issue: bundleTwo, Footprint: []string{"docs/member.md"}, ReservedAt: &held})
				st.Record = append(st.Record, Entry{At: held, Stage: StageReservation, Note: "reserved " + bundleTwo, Issue: bundleTwo})
				return true, nil
			}); err != nil {
				t.Fatal(err)
			}
			member := tipOf(t, f.bare, bundleTwo)
			at := held.Add(61 * time.Minute)

			renew(t, f, at)

			if tip := tipOf(t, f.bare, bundleTwo); tip.parent != member.commit || tip.released || !tip.claimed.Equal(at) {
				t.Fatalf("the member half through its lease is renewed: before %+v, now %+v", member, tip)
			}
			if tip := originReservation(t, f.bare, claim.RefPrefix+secondIssue); tip.commit != lead.commit {
				t.Fatalf("the lead, under half its lease, is not renewed: before %+v, now %+v", lead, tip)
			}
			st, err := ReadState(f.repo.Root(), f.runID)
			if err != nil {
				t.Fatal(err)
			}
			if got := issuesOf(linesAt(st, StageReservationRenewed)); !slices.Contains(got, bundleTwo) {
				t.Fatalf("the member's renewal is logged naming it: %+v", linesAt(st, StageReservationRenewed))
			}
		})
	}
}
