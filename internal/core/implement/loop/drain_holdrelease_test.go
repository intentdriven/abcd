package loop

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The drain's own judgement hold (spc-2610161044461448 step 2;
// iss-2610161015387318): the reservation the drain takes on an issue before
// it asks a judgement over it is recorded in the judgement it awaits, and it
// is released, by a tombstone over exactly that commit, on every path that
// leaves it unused: only the judgement still awaited, or the lane opened on
// it, keeps it. A reservation this person made by hand is never released for
// a judgement.

// countPushes makes every push repo makes to origin append a line to a file,
// through the receive-pack its remote names, and returns the count reader.
func countPushes(t *testing.T, repo *gittest.Repo) func() int {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "pushes")
	script := filepath.Join(dir, "receive-pack")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho push >> '"+log+"'\nexec git receive-pack \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	repo.Git("config", "remote.origin.receivepack", script)
	return func() int {
		data, err := os.ReadFile(log)
		if err != nil {
			return 0
		}
		return strings.Count(string(data), "push\n")
	}
}

// refUpdates makes origin log every update a push makes to its refs, through
// its own update hook, and returns the reader: each commit pushed to ref, in
// the order pushed.
func refUpdates(t *testing.T, bare string) func(ref string) []string {
	t.Helper()
	log := filepath.Join(t.TempDir(), "updates")
	hook := filepath.Join(bare, "hooks", "update")
	if err := os.MkdirAll(filepath.Dir(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hook, []byte("#!/bin/sh\necho \"$1 $3\" >> '"+log+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return func(ref string) []string {
		data, _ := os.ReadFile(log)
		var out []string
		for _, line := range strings.Split(string(data), "\n") {
			if r, sha, ok := strings.Cut(line, " "); ok && r == ref {
				out = append(out, sha)
			}
		}
		return out
	}
}

// releasedOver fails unless origin's reservation of id is a tombstone pushed
// over taken, the commit the drain's own reservation made: the release
// reached origin, and released that reservation and no other.
func releasedOver(t *testing.T, bare, id, taken string) {
	t.Helper()
	r := originReservation(t, bare, claim.RefPrefix+id)
	if !r.released || r.parent != taken {
		t.Fatalf("origin's reservation of %s = %+v; want a tombstone over the drain's %s", id, r, taken)
	}
}

// heldStill fails unless origin's reservation of id is still commit taken.
func heldStill(t *testing.T, bare, id, taken string) {
	t.Helper()
	if r := originReservation(t, bare, claim.RefPrefix+id); r.commit != taken || r.released {
		t.Fatalf("origin's reservation of %s = %+v; want it still %s", id, r, taken)
	}
}

// askFirst is the drain's first move over the fixture, which reserves the
// documentation issue and asks the judgement over it, and the commit that
// reservation made.
func askFirst(t *testing.T, repo *gittest.Repo, bare string, o Options, d DrainOptions) (DrainResult, string) {
	t.Helper()
	res, err := Drain(repo.Root(), o, d)
	if err != nil {
		t.Fatal(err)
	}
	if res.Judging == nil || res.Judging.Issue != secondIssue || res.Judging.Hold == nil {
		t.Fatalf("the first move asks over %s on a reservation it took: %+v", secondIssue, res.Judging)
	}
	taken := refOf(t, bare, secondIssue)
	if taken == "" || res.Judging.Hold.Commit != taken {
		t.Fatalf("the judgement records the reservation the drain took: %+v, origin %s", res.Judging.Hold, taken)
	}
	return res, taken
}

// fileFirstOnOrigin files a tech-debt issue on origin alone, the group the
// drain order takes first.
func fileFirstOnOrigin(t *testing.T, bare, id string) {
	t.Helper()
	pushFromOtherCopy(t, bare, "file "+id, func(dir string) {
		if _, err := capture.Capture(capture.CaptureRequest{RepoRoot: dir, Text: "The drain page misses a flag (" + id + ")",
			Severity: capture.SeverityMinor, Category: capture.Category("tech-debt"), Source: "manual-test", Slug: "flag",
			FoundDuring: "the loop's tests", FoundAt: "commands/drain.md", ForceID: id, Remedy: "Name the new flag."}); err != nil {
			t.Fatalf("capture on origin: %v", err)
		}
	})
}

// TestANoWhileTheOrderNowStartsElsewhereReleasesTheJudgedIssue: a no is
// answered over X while an issue new on origin now heads the order; the move
// asks over that issue instead, and X, which no lane opens on, is released.
func TestANoWhileTheOrderNowStartsElsewhereReleasesTheJudgedIssue(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first, taken := askFirst(t, repo, bare, Options{}, DrainOptions{})
	fileFirstOnOrigin(t, bare, originIssue)
	next := judge(t, repo, Options{}, first, JudgementNo, "", "the remedy changes neither")
	if next.Judging == nil || next.Judging.Issue != originIssue {
		t.Fatalf("the move asks over the issue now heading the order: %+v", next.Judging)
	}
	releasedOver(t, bare, secondIssue, taken)
	if !strings.Contains(next.Next, "to judge "+secondIssue) {
		t.Fatalf("the move says it released %s's reservation: %q", secondIssue, next.Next)
	}
}

// TestANoAnsweredOnceTheWindowHasElapsedReleasesTheIssue: a no answered once
// the window has elapsed opens no lane, so the reservation taken for it is
// released; after the pause the stored no opens the lane on a reservation of
// its own.
func TestANoAnsweredOnceTheWindowHasElapsedReleasesTheIssue(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	c := &clock{t: time.Now().UTC().Truncate(time.Second)}
	pace := "1/5"
	first, taken := askFirst(t, repo, bare, Options{Now: c.now, Pace: &pace}, DrainOptions{})
	c.t = c.t.Add(2 * time.Minute)
	paused := judge(t, repo, Options{Now: c.now}, first, JudgementNo, "", "the remedy changes neither")
	if paused.NextEligibleAt == nil || paused.Start != nil {
		t.Fatalf("the window has elapsed, so no lane opens: %+v", paused)
	}
	releasedOver(t, bare, secondIssue, taken)

	c.t = paused.NextEligibleAt.Add(time.Second)
	after, err := Drain(repo.Root(), Options{Now: c.now}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if after.Start == nil || after.Lane == nil || after.Lane.Issue != secondIssue {
		t.Fatalf("after the pause the stored no opens the lane: %+v", after)
	}
	if r := originReservation(t, bare, claim.RefPrefix+secondIssue); r.released || r.commit == taken {
		t.Fatalf("the lane reserves the issue afresh: %+v", r)
	}
}

// TestAJudgedIssueThatLeavesTheEligibleSetUnansweredIsReleased: X is raised
// past the rule's severities on origin before any answer, and so is the only
// other issue; nothing eligible is left, the drain ends, and X is released.
func TestAJudgedIssueThatLeavesTheEligibleSetUnansweredIsReleased(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	_, taken := askFirst(t, repo, bare, Options{}, DrainOptions{})
	pushFromOtherCopy(t, bare, "raise both", func(dir string) {
		setSeverity(t, dir, secondIssue, "major")
		setSeverity(t, dir, eligibleIssue, "major")
	})
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Stopped != DrainStoppedEmpty || res.Judging != nil {
		t.Fatalf("nothing eligible is left: %+v", res)
	}
	releasedOver(t, bare, secondIssue, taken)
}

// TestAnOutageStopReleasesTheJudgedIssue: a run's outage give-up stops the
// drain with a judgement awaited; the reservation taken for it is released.
func TestAnOutageStopReleasesTheJudgedIssue(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	c := &clock{t: time.Now().UTC().Truncate(time.Second)}
	o := Options{Now: c.now}
	_, taken := askFirst(t, repo, bare, o, DrainOptions{})
	giveUp(t, joinShared(t, repo.Root(), c.now), "host-a", func(d time.Duration) { c.t = c.t.Add(d) })
	res, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Stopped != DrainStoppedOutage || res.Judging != nil {
		t.Fatalf("the drain stops on the outage, awaiting nothing: %+v", res)
	}
	releasedOver(t, bare, secondIssue, taken)
}

// TestTheCapStopReleasesTheJudgedIssue: a drain at its cap with a judgement
// awaited stops, and the reservation taken for the judgement is released.
func TestTheCapStopReleasesTheJudgedIssue(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	_, taken := askFirst(t, repo, bare, Options{}, DrainOptions{Max: 1})
	st, _, err := readDrain(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	st.Lanes = append(st.Lanes, DrainLane{Issue: eligibleIssue, RunID: "run-2610160000000001", OpenedAt: st.StartedAt, Outcome: DrainLaneDone})
	if err := writeDrain(repo.Root(), st); err != nil {
		t.Fatal(err)
	}
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Stopped != DrainStoppedCap || res.Judging != nil {
		t.Fatalf("the drain stops at its cap, awaiting nothing: %+v", res)
	}
	releasedOver(t, bare, secondIssue, taken)
}

// TestAnOriginGiveUpReleasesTheJudgedIssue: origin's default branch cannot be
// read through the outage schedule while a judgement is awaited; every wait
// keeps the reservation taken for it, and the give-up releases it.
func TestAnOriginGiveUpReleasesTheJudgedIssue(t *testing.T) {
	t0 := time.Now().UTC().Truncate(time.Second)
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	_, taken := askFirst(t, repo, bare, Options{Now: func() time.Time { return t0 }}, DrainOptions{})
	gitIn(t, bare, "branch", "-m", "main", "trunk")
	gitIn(t, bare, "symbolic-ref", "HEAD", "refs/heads/trunk")
	now := t0.Add(time.Minute)
	for i := 0; i < 20; i++ {
		res, err := Drain(repo.Root(), Options{Now: func() time.Time { return now }}, DrainOptions{})
		if err != nil {
			t.Fatalf("the move at %s: %v", now, err)
		}
		if res.Stopped == DrainStoppedOrigin {
			if res.Judging != nil {
				t.Fatalf("the drain that gave up still awaits a judgement: %+v", res.Judging)
			}
			releasedOver(t, bare, secondIssue, taken)
			return
		}
		heldStill(t, bare, secondIssue, taken)
		if res.OriginWait == nil {
			t.Fatalf("the move at %s waits on origin: %+v", now, res)
		}
		now = res.OriginWait.NextProbeAt
	}
	t.Fatal("the drain never gave up on origin")
}

// TestAFailedJudgementRequestReleasesTheReservationItTook: the request cannot
// be written, so the move refuses, and the reservation it took just before is
// released rather than left to hold others off.
func TestAFailedJudgementRequestReleasesTheReservationItTook(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	if err := os.MkdirAll(filepath.Join(repo.Root(), filepath.FromSlash(DrainJudgementRequestRel), "blocked"), 0o755); err != nil {
		t.Fatal(err)
	}
	updates := refUpdates(t, bare)
	if _, err := Drain(repo.Root(), Options{}, DrainOptions{}); err == nil {
		t.Fatal("a request that cannot be written refuses the move")
	}
	// The move's two pushes: the drain's reservation, then the tombstone
	// over exactly that commit.
	pushed := updates(claim.RefPrefix + secondIssue)
	if len(pushed) != 2 {
		t.Fatalf("origin took %d update(s) to %s; want the drain's reservation and its tombstone", len(pushed), secondIssue)
	}
	releasedOver(t, bare, secondIssue, pushed[0])
	if p := originReservation(t, bare, claim.RefPrefix+secondIssue); p.holder != heldBy(t, patEmail) || p.commit != pushed[1] {
		t.Fatalf("the tombstone is this person's, the move's last push: %+v", p)
	}
}

// TestAStartErrorReleasesTheReservationTakenForTheJudgement: a no opens the
// lane, and Start fails with an error that is no refusal; the move fails, and
// the reservation the drain took for the judgement is released.
func TestAStartErrorReleasesTheReservationTakenForTheJudgement(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first, taken := askFirst(t, repo, bare, Options{}, DrainOptions{})
	was := startLane
	t.Cleanup(func() { startLane = was })
	startLane = func(string, string, Options) (StartResult, error) {
		return StartResult{}, errors.New("the disk is full")
	}
	path := writeAnswer(t, repo, first.Judging, judgementAnswer(first.Judging, JudgementNo, "", "the remedy changes neither"))
	if _, err := Drain(repo.Root(), Options{}, DrainOptions{Judgement: path}); err == nil || !strings.Contains(err.Error(), "disk is full") {
		t.Fatalf("the start's error = %v; want it returned", err)
	}
	releasedOver(t, bare, secondIssue, taken)
}

// TestAnUnreadableViewIsNamedAndTheDrainsHoldStillReleased: the move's view
// of the reservations cannot be read; the move says so, and a yes still
// releases the reservation the judgement records.
func TestAnUnreadableViewIsNamedAndTheDrainsHoldStillReleased(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first, taken := askFirst(t, repo, bare, Options{}, DrainOptions{})
	was := drainView
	t.Cleanup(func() { drainView = was })
	drainView = func(string, time.Time) (claim.Snapshot, error) {
		return claim.Snapshot{}, errors.New("the tracking refs are unreadable")
	}
	next := judge(t, repo, Options{}, first, JudgementYes, HandBackTrustRule, "the remedy moves a trust boundary")
	releasedOver(t, bare, secondIssue, taken)
	if r := routeOf(t, next.HandBacks, secondIssue); !strings.Contains(r.Wrote, "released the reservation on "+secondIssue) {
		t.Fatalf("the route says the reservation was released: %q", r.Wrote)
	}
	if !strings.Contains(next.Next, "the tracking refs are unreadable") {
		t.Fatalf("the move names the view it could not read: %q", next.Next)
	}
}

// TestAYesLeavesAReservationThisPersonMadeByHand: a reservation this person
// made of X with `abcd reserve` before the drain, and one renewed by hand
// after the drain took it, are this person's, not the drain's: a yes pushes
// nothing over either, and the move's one push is the next issue's
// reservation.
func TestAYesLeavesAReservationThisPersonMadeByHand(t *testing.T) {
	t.Run("before the drain", func(t *testing.T) {
		repo, bare := reservingDrainRepo(t, "commands/drain.md")
		if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands"}}, claim.Options{}); err != nil {
			t.Fatal(err)
		}
		mine := refOf(t, bare, secondIssue)
		first, err := Drain(repo.Root(), Options{}, DrainOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if first.Judging == nil || first.Judging.Issue != secondIssue || first.Judging.Hold != nil {
			t.Fatalf("the drain asks over X on the person's own reservation, recording none of its own: %+v", first.Judging)
		}
		pushes := countPushes(t, repo)
		next := judge(t, repo, Options{}, first, JudgementYes, HandBackTrustRule, "the remedy moves a trust boundary")
		// The move goes on to the next issue, whose reservation is its one push.
		if n := pushes(); n != 1 || next.Judging == nil || next.Judging.Issue != eligibleIssue {
			t.Fatalf("the yes move pushed %d time(s), judging %+v; want the next issue's reservation alone", n, next.Judging)
		}
		heldStill(t, bare, secondIssue, mine)
	})
	t.Run("renewed after the drain took it", func(t *testing.T) {
		repo, bare := reservingDrainRepo(t, "commands/drain.md")
		first, _ := askFirst(t, repo, bare, Options{}, DrainOptions{})
		if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands"}}, claim.Options{}); err != nil {
			t.Fatal(err)
		}
		mine := refOf(t, bare, secondIssue)
		pushes := countPushes(t, repo)
		next := judge(t, repo, Options{}, first, JudgementYes, HandBackTrustRule, "the remedy moves a trust boundary")
		if n := pushes(); n != 1 || next.Judging == nil || next.Judging.Issue != eligibleIssue {
			t.Fatalf("the yes move pushed %d time(s), judging %+v; want the next issue's reservation alone", n, next.Judging)
		}
		heldStill(t, bare, secondIssue, mine)
		if r := routeOf(t, next.HandBacks, secondIssue); !strings.Contains(r.Wrote, "left the reservation on "+secondIssue+" as it stands") {
			t.Fatalf("the route says the reservation was left: %q", r.Wrote)
		}
	})
}

// TestAPollAfterTheHoldWasReleasedByHandReservesAgainBeforeAsking: this
// person releases the drain's reservation by hand while the judgement is
// awaited; the next poll does not ask on the hold it recorded, which origin no
// longer shows, but reserves again first and records that reservation, so
// origin shows a live drain hold over the person's tombstone.
func TestAPollAfterTheHoldWasReleasedByHandReservesAgainBeforeAsking(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	_, taken := askFirst(t, repo, bare, Options{}, DrainOptions{})
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Release: true}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	tombstone := refOf(t, bare, secondIssue)
	poll, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	r := originReservation(t, bare, claim.RefPrefix+secondIssue)
	if r.released || r.parent != tombstone || r.holder != heldBy(t, patEmail) {
		t.Fatalf("origin's reservation after the poll = %+v; want a live one over the tombstone %s", r, tombstone)
	}
	if h := poll.Judging.Hold; poll.Judging.Issue != secondIssue || h == nil || h.Commit != r.commit || h.Commit == taken {
		t.Fatalf("the poll records the reservation it took again as the drain's hold: %+v", poll.Judging)
	}
}

// TestAPanicReleasesTheReservationTakenForTheJudgement: a no opens the lane,
// and Start panics; the panic still reaches the caller, and the reservation
// the drain took for the judgement, on which no lane opened, is released on
// the way out.
func TestAPanicReleasesTheReservationTakenForTheJudgement(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first, taken := askFirst(t, repo, bare, Options{}, DrainOptions{})
	was := startLane
	t.Cleanup(func() { startLane = was })
	startLane = func(string, string, Options) (StartResult, error) { panic("the start blew up") }
	path := writeAnswer(t, repo, first.Judging, judgementAnswer(first.Judging, JudgementNo, "", "the remedy changes neither"))
	func() {
		defer func() {
			if r := recover(); r != "the start blew up" {
				t.Fatalf("the panic reached the caller as %v", r)
			}
		}()
		_, _ = Drain(repo.Root(), Options{}, DrainOptions{Judgement: path})
	}()
	releasedOver(t, bare, secondIssue, taken)
}

// TestAMovesNextJoinsOnlyTheClausesItHas: the clauses a move adds to its next
// step are joined by "; " with no empty clause between or around them, so a
// next step that was empty never starts with the separator.
func TestAMovesNextJoinsOnlyTheClausesItHas(t *testing.T) {
	for _, c := range []struct {
		parts []string
		want  string
	}{
		{[]string{"", "released the reservation"}, "released the reservation"},
		{[]string{"ask the judgement", "", "released the reservation"}, "ask the judgement; released the reservation"},
		{[]string{"", ""}, ""},
		{[]string{"the view could not be read", ""}, "the view could not be read"},
	} {
		if got := joinNext(c.parts...); got != c.want {
			t.Errorf("joinNext(%q) = %q; want %q", c.parts, got, c.want)
		}
	}
}
