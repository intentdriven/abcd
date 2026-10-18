package loop

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
)

// A drain keeps its lane's lease every move, and the landing checks the
// reservation before its push (spc-2610161044461448 step 7;
// iss-2610161015393523; dcn-2610161040374343): a drain move, waiting or not,
// renews the lane in progress at half its lease though no step of the lane
// ran, and a lane whose reservation is now another person's pushes nothing,
// opens no pull request and is handed back.

// TestAWaitingDrainMoveRenewsTheLaneInProgress: the drain is paused by its
// pace window, the lane in progress is past half its lease and no step has
// run; the drain's move opens nothing and renews the lane's reservation.
func TestAWaitingDrainMoveRenewsTheLaneInProgress(t *testing.T) {
	t0 := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
	c := &clock{t: t0}
	o := Options{Now: c.now}
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first := drainToLane(t, repo, o, DrainOptions{})
	if first.Start == nil || first.Lane == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	ref := claim.RefPrefix + secondIssue
	before := originReservation(t, bare, ref)
	st, live, err := readDrain(repo.Root())
	if err != nil || !live {
		t.Fatalf("a live drain: %v", err)
	}
	until := t0.Add(4 * time.Hour)
	st.NextEligibleAt = &until
	if err := writeDrain(repo.Root(), st); err != nil {
		t.Fatal(err)
	}
	c.t = t0.Add(61 * time.Minute)
	res, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Start != nil || !strings.Contains(res.Next, "nothing before") {
		t.Fatalf("the paused drain opens nothing: %+v", res)
	}
	after := originReservation(t, bare, ref)
	if after.commit == before.commit || !after.claimed.Equal(c.t) || after.holder != "octo" || after.released {
		t.Fatalf("the waiting move renews the lane's reservation: before %+v, after %+v", before, after)
	}
	if note := recordNote(t, repo, first.Start.RunID, StageReservationRenewed); !strings.Contains(note, "renewed the reservation on "+secondIssue) {
		t.Fatalf("the lane's run record logs the renewal: %q", note)
	}
}

// TestALaneWhoseReservationIsAnothersPushesNothingAndIsHandedBack: between the
// records and the push, the issue's reservation becomes Quinn's; the push step
// refuses, pushes nothing and opens no pull request, and the drain's next move
// hands the lane back at once.
// pushReadyLane drives the drain's documentation-issue lane to its push: a
// fix commit and a receipt resolving the issue, a passing round, the records
// committed and the preflight receipt minted for its head.
func pushReadyLane(t *testing.T, f *landFixture) Lane {
	t.Helper()
	first := drainToLane(t, f.repo, Options{}, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	f.runID, f.issue = first.Start.RunID, secondIssue
	repo := f.repo
	stepTo(t, repo, f.runID, f.stages, StageImplement)
	if res, err := advance(repo.Root(), f.runID, f.stages, Options{}); err != nil || res.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
	l := currentLane(t, repo, f.runID)
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), f.runID, l.ID)
	sha := laneCommit(t, repo, l, "fix.txt")
	rc := goodReceipt(t, f.runID, l, dir, sha)
	rc.Model = "claude-test-5"
	rc.Resolves = []Resolution{{Issue: f.issue, Commit: sha, Note: "the page names the flag", Impact: "fix",
		Grounds: "pursued: the page names the flag; shown wrong if it still does not"}}
	if _, err := Receipt(repo.Root(), f.runID, writeReceipt(t, dir, rc), f.stages, Options{}); err != nil {
		t.Fatal(err)
	}
	passRound(t, repo, f.runID, f.stages, RoleRuthless, RoleSecurity)
	stepTo(t, repo, f.runID, f.stages, StageLand)
	f.step(t)
	f.step(t)
	l = currentLane(t, repo, f.runID)
	if l.Landing == nil || !l.Landing.RecordsDone || l.Landing.Pushed != "" {
		t.Fatalf("the lane is at its push: %+v", l.Landing)
	}
	preflighted(t, l, l.HeadSHA)
	return l
}

func TestALaneWhoseReservationIsAnothersPushesNothingAndIsHandedBack(t *testing.T) {
	f := drainLandFixture(t)
	l := pushReadyLane(t, f)
	repo := f.repo

	// The reservation becomes Quinn's.
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Release: true}, claim.Options{}); err != nil {
		t.Fatalf("Pat's release: %v", err)
	}
	q := otherCopy(t, f.bare, quinnEmail)
	if _, err := claim.Reserve(q, claim.Request{Record: secondIssue, Footprint: []string{"AGENTS.md"}}, claim.Options{}); err != nil {
		t.Fatalf("Quinn's reservation: %v", err)
	}

	_, err := advance(repo.Root(), f.runID, f.stages, Options{})
	r, ok := AsRefusal(err)
	if !ok || !strings.Contains(r.Reason, "quinn") || !strings.Contains(r.Reason, "does not push") {
		t.Fatalf("the push refuses on Quinn's reservation: %v", err)
	}
	if got := gitIn(t, f.bare, "for-each-ref", "--format=%(refname)", "refs/heads/"+l.Branch); got != "" {
		t.Fatalf("nothing is pushed: origin holds %s", got)
	}
	l = currentLane(t, repo, f.runID)
	if l.PR != 0 || l.Landing.Pushed != "" {
		t.Fatalf("no pull request is opened: %+v", l)
	}
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	route := routeOf(t, res.Routed, secondIssue)
	if route.Kind != claim.HandBackStuck || !strings.Contains(route.Reason, "quinn") {
		t.Fatalf("the drain hands the lane back at once: %+v", route)
	}
	if tip := originReservation(t, f.bare, claim.RefPrefix+secondIssue); tip.holder != "quinn" || tip.released {
		t.Fatalf("Quinn's reservation is left as it stands: %+v", tip)
	}
}

// TestOriginUnreadAtThePushFenceIsAWaitNotARefusal: origin cannot be read
// when the push fence fetches the lane's reservation, whether its address is
// gone or it refuses the connection. However often the host steps the lane,
// nothing is logged as its refusal, and the drain's next move, once origin
// answers, leaves the lane in progress rather than handing it back as stuck.
// A network failure also waits on the shared outage probe, as the push's
// own does.
func TestOriginUnreadAtThePushFenceIsAWaitNotARefusal(t *testing.T) {
	closed := func(t *testing.T) string {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		addr := ln.Addr().String()
		if err := ln.Close(); err != nil {
			t.Fatal(err)
		}
		return "http://" + addr + "/origin.git"
	}
	for name, tc := range map[string]struct {
		url     func(t *testing.T, bare string) string
		network bool
	}{
		"origin's address is gone":      {url: func(t *testing.T, _ string) string { return filepath.Join(t.TempDir(), "gone.git") }},
		"origin refuses the connection": {url: func(t *testing.T, _ string) string { return closed(t) }, network: true},
	} {
		t.Run(name, func(t *testing.T) {
			f := drainLandFixture(t)
			l := pushReadyLane(t, f)
			repo := f.repo
			// The address is changed as a person would, accepting the
			// changed configuration the run's watch names.
			setOrigin := func(url string) {
				repo.Git("remote", "set-url", "origin", url)
				if _, err := AcceptConfigChange(repo.Root(), f.runID, Options{}); err != nil {
					t.Fatal(err)
				}
			}
			setOrigin(tc.url(t, f.bare))
			for n := range 4 {
				_, err := advance(repo.Root(), f.runID, f.stages, Options{})
				r, ok := AsRefusal(err)
				if !ok || !r.Contention {
					t.Fatalf("step %d waits on origin: %v", n+1, err)
				}
				if n == 0 && !strings.Contains(r.Reason, "could not read the reservation on "+secondIssue) {
					t.Fatalf("the push fence names the reservation it could not read: %v", err)
				}
				if n == 0 && tc.network != (r.outage != nil || r.Outage != nil) {
					t.Fatalf("only a lost network waits on the shared outage probe (network %v): %+v", tc.network, r)
				}
			}
			if got := currentLane(t, repo, f.runID); got.Refused != nil {
				t.Fatalf("origin unread is no refusal of the lane: %+v", got.Refused)
			}
			setOrigin(f.bare)
			res, err := Drain(repo.Root(), Options{}, DrainOptions{})
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range res.Routed {
				if r.Issue == secondIssue {
					t.Fatalf("the lane origin kept waiting is not handed back: %+v", r)
				}
			}
			if res.Lane == nil || res.Lane.RunID != f.runID || !strings.Contains(res.Next, "drive "+secondIssue) {
				t.Fatalf("the lane stays in progress: %+v", res)
			}
			if got := gitIn(t, f.bare, "for-each-ref", "--format=%(refname)", "refs/heads/"+l.Branch); got != "" {
				t.Fatalf("nothing was pushed while origin was unread: %s", got)
			}
		})
	}
}

// TestAPausedDrainDoesNotRenewALaneHandedBack: a lane handed back whose
// release origin refused, whether by its own step or as stuck, still holds
// its reservation while the drain is paused and has not routed it; a move
// past half the lease renews nothing of it.
func TestAPausedDrainDoesNotRenewALaneHandedBack(t *testing.T) {
	for name, handBack := range map[string]func(t *testing.T, repo *gittest.Repo, runID string, c *clock){
		"by its own step": func(t *testing.T, repo *gittest.Repo, runID string, c *clock) {
			handBackLaneOf(t, repo, runID, Options{Now: c.now}, LaneHandBack{Kind: HandBackDesignFinding, Reason: "the page needs a decision", Home: "the brief"})
		},
		"as stuck": func(t *testing.T, repo *gittest.Repo, runID string, c *clock) {
			o := Options{Now: c.now}
			for range 3 {
				refuseStep(t, repo, runID, briefRefusing(sameRefusal), o)
			}
			b := stuckBounds{refusals: PaceValue{Value: BundledStuckRefusals}, minutes: PaceValue{Value: BundledStuckMinutes}}
			if _, err := handBackStuck(repo.Root(), runID, b, c.t); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			t0 := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
			c := &clock{t: t0}
			o := Options{Now: c.now}
			repo, bare, runID := stuckDrain(t, c)
			hook := filepath.Join(bare, "hooks", "pre-receive")
			refuseReservations := "#!/bin/sh\nwhile read old new ref; do case \"$ref\" in refs/heads/abcd-reserved/*) echo 'origin is closed' >&2; exit 1;; esac; done\nexit 0\n"
			if err := os.WriteFile(hook, []byte(refuseReservations), 0o755); err != nil {
				t.Fatal(err)
			}
			handBack(t, repo, runID, c)
			if note := recordNote(t, repo, runID, StageReservationReleaseFailed); note == "" {
				t.Fatal("the hand-back's release was refused")
			}
			if err := os.Remove(hook); err != nil {
				t.Fatal(err)
			}
			ref := claim.RefPrefix + secondIssue
			before := originReservation(t, bare, ref)
			st, live, err := readDrain(repo.Root())
			if err != nil || !live {
				t.Fatalf("a live drain: %v", err)
			}
			until := t0.Add(4 * time.Hour)
			st.NextEligibleAt = &until
			if err := writeDrain(repo.Root(), st); err != nil {
				t.Fatal(err)
			}
			c.t = t0.Add(61 * time.Minute)
			res, err := Drain(repo.Root(), o, DrainOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(res.Next, "nothing before") {
				t.Fatalf("the drain is paused: %+v", res)
			}
			if after := originReservation(t, bare, ref); after.commit != before.commit {
				t.Fatalf("the handed-back lane's reservation is not renewed: before %+v, after %+v", before, after)
			}
			if note := recordNote(t, repo, runID, StageReservationRenewed); note != "" {
				t.Fatalf("the run record logs no renewal: %q", note)
			}
		})
	}
}
