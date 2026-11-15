package sweep

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/report"
	"github.com/intentdriven/abcd/internal/gittest"
)

// kindOrder is each row kind's place in the spec's Rows order: the worktrees
// with the strays and the embargo store, then the local branches, the
// branches on origin, the reservations and the inbox.
var kindOrder = map[RowKind]int{
	RowWorktree: 0, RowStray: 0, RowEmbargoStore: 0,
	RowBranch: 1, RowRemoteBranch: 2, RowReservation: 3, RowInbox: 4,
}

// everyKind is a repository with one item of each kind a sweep acts on: a
// merged lane worktree, a merged local branch, a merged branch on origin, a
// spent reservation (judged by a fake) and a waiting inbox report, in a
// checkout standing in for abcd's own. It answers the options a whole sweep
// runs with, their forge answering both the pull-request listings and the
// workflow's calls, and the workflow fake.
func everyKind(t *testing.T) (*gittest.Repo, Options, *workflowForge) {
	t.Helper()
	repo, _ := originRepo(t)
	root := gittest.RootCommit(t, repo.Root())
	t.Cleanup(report.SetAbcdRootCommitForTest(root))
	prev := promoteRootCommit
	promoteRootCommit = root
	t.Cleanup(func() { promoteRootCommit = prev })

	mergedAdd(t, repo, "wt")
	branchWith(repo, "loc", "loc.txt")
	repo.Git("merge", "-q", "--ff-only", "loc")
	tip := mergedOnOrigin(repo, "feat", "feat.txt")
	fileReport(t, "capture refuses a slug with a digit first")
	noWaitBetweenRunListings(t)

	prs := &fakeForge{prs: []pullRequest{mergedPR(4, "feat", tip)}}
	wf := &workflowForge{runs: []string{runListing(t, 10), runListing(t, 11)}}
	o := sweepOpts()
	o.Forge = func(dir string, args ...string) (string, error) {
		if len(args) > 0 && args[0] == "pr" {
			return prs.gh(dir, args...)
		}
		return wf.gh(dir, args...)
	}
	o.reservations = spentAndKept().judge
	return repo, o, wf
}

// TestOneSweepRunsEveryHalfInTheSpecsOrder: a whole dry run lists every kind
// of item, in the order of the spec's Rows, dispatching nothing; the applied
// run acts on one item of each kind.
func TestOneSweepRunsEveryHalfInTheSpecsOrder(t *testing.T) {
	repo, o, wf := everyKind(t)

	dry, err := runSweep(context.Background(), repo.Root(), o, false)
	if err != nil {
		t.Fatal(err)
	}
	place, seen := 0, map[int]bool{}
	for _, r := range dry.Rows {
		p, ok := kindOrder[r.Kind]
		if !ok || p < place {
			t.Fatalf("row %s (%s) comes out of the spec's order: %+v", r.Name, r.Kind, dry.Rows)
		}
		place, seen[p] = p, true
	}
	if len(seen) != 5 {
		t.Fatalf("the dry run lists %d of the five kinds: %+v", len(seen), dry.Rows)
	}
	if len(wf.dispatches()) != 0 || dry.Applied || dry.ExitCode() != ExitActed {
		t.Fatalf("a dry run dispatched %d times, applied %v, exits %d", len(wf.dispatches()), dry.Applied, dry.ExitCode())
	}

	res, err := runSweep(context.Background(), repo.Root(), o, true)
	if err != nil {
		t.Fatalf("%v: %+v", err, res.Rows)
	}
	acted := map[RowKind]Outcome{}
	for _, r := range res.Rows {
		if r.Outcome != OutcomeKept {
			acted[r.Kind] = r.Outcome
		}
	}
	want := map[RowKind]Outcome{RowWorktree: OutcomeRemoved, RowBranch: OutcomeDeleted, RowRemoteBranch: OutcomeDeleted,
		RowReservation: OutcomeDispatched, RowInbox: OutcomePromoted}
	for k, o := range want {
		if acted[k] != o {
			t.Fatalf("no %s row was %s: %+v", k, o, res.Rows)
		}
	}
	if len(wf.dispatches()) != 1 || !strings.Contains(res.Workflow, "run 11") || res.ExitCode() != ExitActed {
		t.Fatalf("dispatched %d times, workflow %q, exit %d", len(wf.dispatches()), res.Workflow, res.ExitCode())
	}
}

// TestAHalfThatRefusesLeavesTheOthersToAct: a reservation judgement that
// fails refuses that half alone; the inbox after it is still promoted, and
// the run carries the refusal, so it exits 2. Options without the build
// loop's reader refuse the whole run before any half runs.
func TestAHalfThatRefusesLeavesTheOthersToAct(t *testing.T) {
	repo, o, wf := everyKind(t)
	j := spentAndKept()
	j.err = errors.New("the reservations could not be fetched")
	o.reservations = j.judge

	res, err := runSweep(context.Background(), repo.Root(), o, true)
	if err == nil || res.ExitCode() != ExitRefused || !strings.Contains(res.Refusal, "judging the reservations: the reservations could not be fetched") {
		t.Fatalf("err %v, exit %d, refusal %q", err, res.ExitCode(), res.Refusal)
	}
	var promoted bool
	for _, r := range res.Rows {
		if r.Kind == RowReservation {
			t.Fatalf("a refused half has rows: %+v", r)
		}
		promoted = promoted || (r.Kind == RowInbox && r.Outcome == OutcomePromoted)
	}
	if !promoted || len(wf.dispatches()) != 0 {
		t.Fatalf("the inbox was promoted: %v; dispatches %d; rows %+v", promoted, len(wf.dispatches()), res.Rows)
	}

	o.Held = nil
	res, err = runSweep(context.Background(), repo.Root(), o, true)
	if err == nil || res.ExitCode() != ExitRefused || len(res.Rows) != 0 {
		t.Fatalf("a run with no reader of the loop's holds: err %v, exit %d, rows %+v", err, res.ExitCode(), res.Rows)
	}
}
