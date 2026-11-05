package loop

import (
	"context"
	"testing"

	"github.com/intentdriven/abcd/internal/core/runner"
)

// The launch ledger's seam in the loop (itd-2611031342011261): a role the step
// starts on a runner is listed against its run, lane and await, and a role
// handed to the host is listed by its placeholder, once.
func TestARoutedRoleIsListedAgainstItsRunLaneAndAwait(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	newDriveEnv(t, "ok", "claude")
	cfg := runnerConfig(t, `{"roles":{"implementer":{"runner":"claude"}},"runner":{"claude":{}}}`, "")
	if _, err := Drive(context.Background(), root, id, steps, o, Runners{Config: cfg, Transcripts: &memTranscripts{}}); err != nil {
		t.Fatal(err)
	}
	got, err := cfg.Ledger().Read(runner.LedgerQuery{Run: id})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 {
		t.Fatalf("listing = %+v", got)
	}
	e := got.Entries[0]
	if e.Role != RoleImplementer || e.Runner != runner.Claude || e.Run != id || e.Lane == "" || e.Await == "" ||
		e.Repo != root || e.Outcome != runner.OutcomeFinished || e.Model != "fake-model" {
		t.Fatalf("entry = %+v", e)
	}
}

func TestARoleOnTheHostIsListedByItsPlaceholder(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	newDriveEnv(t, "ok", "claude")
	cfg := runnerConfig(t, "", "")
	res, err := driveStep(context.Background(), root, id, steps, o, Runners{Config: cfg, Transcripts: &memTranscripts{}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Awaiting == nil {
		t.Fatalf("the step handed nothing out: %+v", res)
	}
	got, err := cfg.Ledger().Read(runner.LedgerQuery{Run: id})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 || got.Entries[0].Route != runner.RouteHost || got.Entries[0].Outcome != runner.OutcomeHanded ||
		got.Entries[0].Lane != res.Lane {
		t.Fatalf("listing = %+v", got)
	}
}

// driveStep is what `abcd implement step` does: Drive, DriveReview, ListHostRole.
func driveStep(ctx context.Context, root, id string, steps Stages, o Options, rs Runners, rr *ReviewRoute) (StepResult, error) {
	res, err := Drive(ctx, root, id, steps, o, rs)
	if err != nil {
		return res, err
	}
	if res, err = DriveReview(ctx, root, id, res, steps, Options{}, rr); err != nil {
		return res, err
	}
	ListHostRole(ctx, root, id, res, rs.Config)
	return res, nil
}

// A reviewer on the host route that oracle.review then answers never reached
// the host: the listing holds no placeholder for it. One the route does not
// answer does reach the host, and is listed once.
func TestAReviewTheRouteAnswersIsNotListedAsTheHosts(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	oneReviewerPerRole(t)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	implemented(t, repo, id, stages, "one.txt")
	cfg := runnerConfig(t, "", "")
	rs := Runners{Config: cfg, Transcripts: &memTranscripts{}}
	listed := func() []runner.LedgerEntry {
		t.Helper()
		got, err := cfg.Ledger().Read(runner.LedgerQuery{Run: id})
		if err != nil {
			t.Fatal(err)
		}
		return got.Entries
	}

	res, err := driveStep(context.Background(), repo.Root(), id, stages, Options{}, rs, rpRoute(t, rpServer(verdictFor)))
	if err != nil {
		t.Fatal(err)
	}
	if res.Route == nil || res.Route.Ran != "rp" {
		t.Fatalf("the route did not answer the review: %+v", res)
	}
	if got := listed(); len(got) != 0 {
		t.Fatalf("a review the route answered is listed as the host's: %+v", got)
	}

	// The next review finds no server, so the host's agent has it.
	res, err = driveStep(context.Background(), repo.Root(), id, stages, Options{}, rs, &ReviewRoute{Name: "rp", Absent: "no server on this machine"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Fallback == nil || res.Awaiting == nil {
		t.Fatalf("the review did not fall back: %+v", res)
	}
	got := listed()
	if len(got) != 1 || got[0].Route != runner.RouteHost || got[0].Outcome != runner.OutcomeHanded || got[0].Role != res.Awaiting.Role {
		t.Fatalf("the review left with the host is not listed once: %+v", got)
	}
}
