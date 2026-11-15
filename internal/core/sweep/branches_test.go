package sweep

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// branchFixture is proofRepo holding one local branch of each kind the
// local-branch rows judge, origin's main published with every merge:
//
//   - ff, merged by a fast-forward of main: deletable;
//   - squashed, merged by a squash: merged by content, which `git branch -d`
//     would refuse, since main does not hold its commit;
//   - open-work, holding work main lacks;
//   - checked-out, checked out in a worktree;
//   - abcd-reserved/iss-1, a reservation ref;
//   - ahead, with a commit its upstream on origin lacks.
func branchFixture(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	r, o := proofRepo(t)
	branchWith(r, "ff", "ff.txt")
	r.Git("merge", "-q", "--ff-only", "ff")
	branchWith(r, "squashed", "s1.txt")
	r.Git("merge", "-q", "--squash", "squashed")
	r.Git("commit", "-q", "-m", "squash")
	branchWith(r, "ahead", "ahead.txt")
	r.Git("push", "-q", "-u", "origin", "ahead")
	r.Git("switch", "-q", "ahead")
	r.Write("ahead2.txt", "more\n")
	r.Git("add", "-A")
	r.Git("commit", "-q", "-m", "more")
	r.Git("switch", "-q", "main")
	publish(r)
	branchWith(r, "open-work", "w.txt")
	r.Git("branch", "abcd-reserved/iss-1")
	r.Git("branch", "checked-out")
	r.Git("worktree", "add", "-q", filepath.Join(t.TempDir(), "co"), "checked-out")
	return r, o
}

func branchRowFor(t *testing.T, res Result, name string) Row {
	t.Helper()
	for _, r := range res.Rows {
		if r.Kind == RowBranch && r.Name == name {
			return r
		}
	}
	t.Fatalf("no branch row named %q in %+v", name, res.Rows)
	return Row{}
}

// TestTheBranchDryRunJudgesEachLocalBranchAndWritesNothing: every local
// branch has a row, the merged one with no worktree is the one delete, each
// keep names its reason, and not one byte of the checkout, its git dir or
// origin changes.
func TestTheBranchDryRunJudgesEachLocalBranchAndWritesNothing(t *testing.T) {
	repo, origin := branchFixture(t)
	roots := []string{repo.Root(), origin}
	before := treeHash(t, roots...)
	res, err := sweepBranches(context.Background(), repo.Root(), false)
	if err != nil {
		t.Fatal(err)
	}
	after := treeHash(t, roots...)
	for p, h := range before {
		if after[p] != h {
			t.Errorf("the dry run changed %s", p)
		}
	}
	for p := range after {
		if _, ok := before[p]; !ok {
			t.Errorf("the dry run made %s", p)
		}
	}

	want := map[string]string{
		"main":                "the default branch",
		"squashed":            "`git branch -d` checks, so it would refuse",
		"open-work":           "unmerged",
		"checked-out":         "checked out in the worktree",
		"abcd-reserved/iss-1": "reservation branch",
		"ahead":               "1 commit(s) not on its upstream origin/ahead",
	}
	if len(res.Rows) != len(want)+1 {
		t.Fatalf("want %d rows, got %+v", len(want)+1, res.Rows)
	}
	for name, reason := range want {
		kept(t, branchRowFor(t, res, name), reason)
	}
	ff := branchRowFor(t, res, "ff")
	if ff.Action != ActionDelete || !strings.Contains(ff.Reason, "merged, judged by ancestry") || !hasEvidence(ff, KindMerged, Proven) {
		t.Fatalf("a merged branch with no worktree: %+v; want it deleted", ff)
	}
	if ff.Outcome != "" || res.Applied {
		t.Fatalf("a dry run carries no outcome: %+v", res)
	}
	if res.ExitCode() != ExitActed {
		t.Fatalf("a dry run that would delete exits %d", res.ExitCode())
	}
}

// TestApplyDeletesAMergedBranchWithBranchDashD: the one deletable branch is
// deleted by `git branch -d` and named; every kept branch stands.
func TestApplyDeletesAMergedBranchWithBranchDashD(t *testing.T) {
	repo, _ := branchFixture(t)
	res, err := sweepBranches(context.Background(), repo.Root(), true)
	if err != nil {
		t.Fatal(err)
	}
	ff := branchRowFor(t, res, "ff")
	if ff.Outcome != OutcomeDeleted || !strings.Contains(ff.Reason, "deleted with `git branch -d`") {
		t.Fatalf("the merged branch: %+v; want it deleted and named", ff)
	}
	if out := repo.Git("branch", "--list", "ff"); out != "" {
		t.Fatalf("ff still stands: %q", out)
	}
	for _, name := range []string{"main", "squashed", "open-work", "checked-out", "abcd-reserved/iss-1", "ahead"} {
		r := branchRowFor(t, res, name)
		if r.Outcome != OutcomeKept {
			t.Errorf("%s: %+v; want it kept", name, r)
		}
		if out := repo.Git("branch", "--list", name); out == "" {
			t.Errorf("%s was deleted", name)
		}
	}
	if res.ExitCode() != ExitActed {
		t.Fatalf("an applied run that deleted a branch exits %d", res.ExitCode())
	}
}

// TestARefusalOfGitBranchDashDIsKeptAndIsTheRunsFault: when the repository
// changes after the judgement so that git no longer finds the branch merged,
// `git branch -d` refuses; the branch stands, its row names git's refusal,
// and the run exits as a fault.
func TestARefusalOfGitBranchDashDIsKeptAndIsTheRunsFault(t *testing.T) {
	repo, _ := branchFixture(t)
	before := repo.Git("rev-parse", "ff~1")
	beforeBranchDelete = func(string) { repo.Git("switch", "-q", "--detach", before) }
	t.Cleanup(func() { beforeBranchDelete = nil })

	res, err := sweepBranches(context.Background(), repo.Root(), true)
	if err == nil || res.ExitCode() != ExitRefused || res.Refusal == "" {
		t.Fatalf("a refused delete is the run's fault: err %v, exit %d, refusal %q", err, res.ExitCode(), res.Refusal)
	}
	ff := branchRowFor(t, res, "ff")
	if ff.Outcome != OutcomeKept || !strings.Contains(ff.Reason, "`git branch -d` refused") || !strings.Contains(ff.Reason, "not fully merged") {
		t.Fatalf("the refused branch: %+v; want it kept, naming git's refusal", ff)
	}
	if out := repo.Git("branch", "--list", "ff"); out == "" {
		t.Fatal("the branch git refused to delete is gone")
	}
}

// TestABranchThatMovedAfterItsJudgementIsKept: a branch whose tip moves
// between the judgement and the delete is left for the next run to judge, and
// is no fault.
func TestABranchThatMovedAfterItsJudgementIsKept(t *testing.T) {
	repo, _ := branchFixture(t)
	beforeBranchDelete = func(string) { repo.Git("branch", "-f", "ff", "main~2") }
	t.Cleanup(func() { beforeBranchDelete = nil })
	res, err := sweepBranches(context.Background(), repo.Root(), true)
	if err != nil {
		t.Fatal(err)
	}
	ff := branchRowFor(t, res, "ff")
	if ff.Outcome != OutcomeKept || !strings.Contains(ff.Reason, "moved since it was judged") {
		t.Fatalf("the moved branch: %+v", ff)
	}
	if res.ExitCode() != ExitNothing {
		t.Fatalf("a run that deleted nothing exits %d", res.ExitCode())
	}
}
