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
	res, err := Drive(context.Background(), root, id, steps, o, Runners{Config: cfg, Transcripts: &memTranscripts{}})
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
