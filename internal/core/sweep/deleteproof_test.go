package sweep

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/gittest"
)

// proofRepo is sweepRepo with an origin that holds main, fetched, so origin's
// default branch is tracked as a checkout's is.
func proofRepo(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	repo, _ := sweepRepo(t)
	origin := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, origin)
	repo.Git("remote", "add", "origin", origin)
	repo.Git("push", "-q", "origin", "main")
	repo.Git("fetch", "-q", "origin")
	return repo, origin
}

// publish pushes main to origin and fetches it back.
func publish(repo *gittest.Repo) {
	repo.Git("push", "-q", "origin", "main")
	repo.Git("fetch", "-q", "origin")
}

// branchWith cuts branch from main and commits rel on it, leaving main
// checked out.
func branchWith(repo *gittest.Repo, branch, rel string) {
	repo.Git("switch", "-q", "-c", branch)
	repo.Write(rel, rel+"\n")
	repo.Git("add", "-A")
	repo.Git("commit", "-q", "-m", "change "+rel)
	repo.Git("switch", "-q", "main")
}

// forgeSaying is a forge client that answers every listing with out.
func forgeSaying(out string) func(string, ...string) (string, error) {
	return func(string, ...string) (string, error) { return out, nil }
}

// idleProbe is the liveness probe's options for a world nothing holds.
func idleProbe() ProbeOptions {
	return ProbeOptions{Reservations: claim.Snapshot{FetchedAt: time.Unix(1, 0)}, Processes: cwds(), Now: stale()}
}

func prove(t *testing.T, o DeleteProofOptions, target guard.DeleteTarget) guard.DeleteProof {
	t.Helper()
	return NewDeleteProver(o).ProveDelete(target)
}

func proofRefused(t *testing.T, p guard.DeleteProof, want string) {
	t.Helper()
	if p.Safe {
		t.Fatalf("proven safe, want refused for %q", want)
	}
	if !strings.Contains(p.Failed, want) {
		t.Errorf("failed %q, want it to say %q", p.Failed, want)
	}
}

// TestDeleteProofJudgesALocalBranchOffline pins the local branch's proof: a
// branch merged into origin's default branch as last fetched, by a
// fast-forward or a squash, is safe; one holding work it lacks, the default
// branch, one checked out in a worktree and abcd's coordination branches are
// refused. No forge is asked.
func TestDeleteProofJudgesALocalBranchOffline(t *testing.T) {
	repo, _ := proofRepo(t)
	local := func(b string) guard.DeleteTarget {
		return guard.DeleteTarget{Kind: guard.DeleteLocalBranch, Dir: repo.Root(), Branch: b}
	}
	o := DeleteProofOptions{Forge: func(string, ...string) (string, error) {
		t.Fatal("a local branch's proof asked the forge")
		return "", nil
	}}

	branchWith(repo, "ff", "ff.txt")
	repo.Git("merge", "-q", "--ff-only", "ff")
	branchWith(repo, "squashed", "s1.txt")
	repo.Git("merge", "-q", "--squash", "squashed")
	repo.Git("commit", "-q", "-m", "squash")
	publish(repo)
	branchWith(repo, "open-work", "w.txt")
	repo.Git("branch", "abcd-reserved/iss-1")
	repo.Git("branch", "checked-out")
	repo.Git("worktree", "add", "-q", filepath.Join(t.TempDir(), "co"), "checked-out")

	for _, b := range []string{"ff", "squashed", "no-such-branch"} {
		if p := prove(t, o, local(b)); !p.Safe {
			t.Errorf("%s: refused (%s), want safe", b, p.Failed)
		}
	}
	proofRefused(t, prove(t, o, local("open-work")), "holds work origin/main does not")
	proofRefused(t, prove(t, o, local("main")), "the default branch")
	proofRefused(t, prove(t, o, local("checked-out")), "checked out in the worktree")
	p := prove(t, o, local("abcd-reserved/iss-1"))
	proofRefused(t, p, "reservation branch")
	if !strings.Contains(p.Settle, "abcd reserve --sweep --apply") {
		t.Errorf("the reservation refusal does not name the sweep: %q", p.Settle)
	}
}

// TestDeleteProofJudgesARemoteBranchAtOriginsTip pins the remote branch's
// proof: judged at the tip origin holds now, refused while a pull request is
// open on it, while the forge cannot answer, and when origin's tip moved past
// what this checkout fetched; a branch the forge records merged at that tip
// is safe, as is one origin does not hold.
func TestDeleteProofJudgesARemoteBranchAtOriginsTip(t *testing.T) {
	repo, origin := proofRepo(t)
	remote := func(b string) guard.DeleteTarget {
		return guard.DeleteTarget{Kind: guard.DeleteRemoteBranch, Dir: repo.Root(), Remote: "origin", Branch: b}
	}
	branchWith(repo, "feat", "f.txt")
	repo.Git("push", "-q", "origin", "feat")
	tip := repo.Git("rev-parse", "feat")

	none := forgeSaying(`[]`)
	proofRefused(t, prove(t, DeleteProofOptions{Forge: none}, remote("feat")), "holds work origin/main does not")
	proofRefused(t, prove(t, DeleteProofOptions{}, remote("feat")), "no forge was asked")
	failing := func(string, ...string) (string, error) { return "", errors.New("gh: not signed in") }
	proofRefused(t, prove(t, DeleteProofOptions{Forge: failing}, remote("feat")), "could not be asked")
	open := forgeSaying(`[{"number":7,"headRefName":"feat","headRefOid":"` + tip + `","baseRefName":"main","state":"OPEN"}]`)
	proofRefused(t, prove(t, DeleteProofOptions{Forge: open}, remote("feat")), "pull request #7 is open")
	merged := forgeSaying(`[{"number":7,"headRefName":"feat","headRefOid":"` + tip + `","baseRefName":"main","state":"MERGED"}]`)
	if p := prove(t, DeleteProofOptions{Forge: merged}, remote("feat")); !p.Safe {
		t.Errorf("merged at origin's tip: refused (%s), want safe", p.Failed)
	}
	if p := prove(t, DeleteProofOptions{Forge: none}, remote("gone")); !p.Safe {
		t.Errorf("a branch origin does not hold: refused (%s), want safe", p.Failed)
	}
	proofRefused(t, prove(t, DeleteProofOptions{Forge: merged}, guard.DeleteTarget{
		Kind: guard.DeleteRemoteBranch, Dir: repo.Root(), Remote: "upstream", Branch: "feat"}), "only a delete on origin")

	// Another copy moves origin's feat past the merged tip: the work the
	// delete would take is a commit this checkout has never seen.
	other := filepath.Join(t.TempDir(), "other")
	repo.Git("clone", "-q", "--branch", "feat", origin, other)
	repo.Git("-C", other, "commit", "-q", "--allow-empty", "-m", "later")
	repo.Git("-C", other, "push", "-q", "origin", "feat")
	proofRefused(t, prove(t, DeleteProofOptions{Forge: merged}, remote("feat")), "has not fetched")
	repo.Git("fetch", "-q", "origin")
	proofRefused(t, prove(t, DeleteProofOptions{Forge: merged}, remote("feat")), "holds work origin/main does not")

	unreachable := func(string, string, string) (string, bool, error) { return "", false, errors.New("no route") }
	proofRefused(t, prove(t, DeleteProofOptions{Forge: merged, RemoteTip: unreachable}, remote("feat")), "could not be read")
}

// TestDeleteProofJudgesAWorktreeRemoval pins the worktree's proof: a clean
// worktree on a merged branch nothing holds is safe; one holding an
// untracked or an ignored file, one a process works in, the main working
// tree and a path git does not list are refused.
func TestDeleteProofJudgesAWorktreeRemoval(t *testing.T) {
	repo, _ := proofRepo(t)
	wt := func(p string) guard.DeleteTarget {
		return guard.DeleteTarget{Kind: guard.DeleteWorktree, Dir: repo.Root(), Path: p}
	}
	add := func(name string) string {
		p := filepath.Join(t.TempDir(), name)
		repo.Git("worktree", "add", "-q", "-b", name, p, "main")
		repo.Git("-C", p, "commit", "-q", "--allow-empty", "-m", name)
		return p
	}
	done := add("done")
	repo.Git("merge", "-q", "--ff-only", "done")
	publish(repo)
	idle := DeleteProofOptions{Probe: idleProbe()}
	if p := prove(t, idle, wt(done)); !p.Safe {
		t.Fatalf("a clean, merged, idle worktree: refused (%s), want safe", p.Failed)
	}

	busy := idle
	busy.Probe.Processes = cwds(filepath.Join(done, "sub"))
	proofRefused(t, prove(t, busy, wt(done)), "it may be in use")

	unfetched := idle
	unfetched.Probe.Reservations = claim.Snapshot{}
	p := prove(t, unfetched, wt(done))
	proofRefused(t, p, "never been fetched")
	if !strings.Contains(p.Settle, "abcd peers") {
		t.Errorf("the unfetched reservations refusal does not say how to fetch them: %q", p.Settle)
	}

	if err := os.MkdirAll(filepath.Join(done, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(done, ".abcd", ".work.local", "NEXT.md"), "handover\n")
	proofRefused(t, prove(t, idle, wt(done)), "git has not committed")

	unmerged := add("unmerged")
	proofRefused(t, prove(t, idle, wt(unmerged)), "holds work origin/main does not")
	untracked := add("untracked")
	repo.Git("merge", "-q", "--ff-only", "untracked")
	publish(repo)
	write(t, filepath.Join(untracked, "notes.txt"), "notes\n")
	proofRefused(t, prove(t, idle, wt(untracked)), "git has not committed")

	proofRefused(t, prove(t, idle, wt(repo.Root())), "main working tree")
	proofRefused(t, prove(t, idle, wt(t.TempDir())), "not a worktree git lists")
}

// TestDeleteProofKeepsABranchItsDefaultHolds pins ancestry alone as proof: a
// branch whose tip the default branch already holds deletes nothing it
// lacks, whether it was merged by a fast-forward with no pull request,
// pushed to origin so, or cut and never committed to. A local branch the
// repository does not hold is safe even where a branch below its name exists.
func TestDeleteProofKeepsABranchItsDefaultHolds(t *testing.T) {
	repo, _ := proofRepo(t)
	branchWith(repo, "ff", "ff.txt")
	repo.Git("push", "-q", "origin", "ff")
	repo.Git("merge", "-q", "--ff-only", "ff")
	publish(repo)
	repo.Git("branch", "fresh")
	branchWith(repo, "nest/sub", "n.txt")

	none := DeleteProofOptions{Forge: forgeSaying(`[]`)}
	for _, target := range []guard.DeleteTarget{
		{Kind: guard.DeleteRemoteBranch, Dir: repo.Root(), Remote: "origin", Branch: "ff"},
		{Kind: guard.DeleteLocalBranch, Dir: repo.Root(), Branch: "ff"},
		{Kind: guard.DeleteLocalBranch, Dir: repo.Root(), Branch: "fresh"},
		{Kind: guard.DeleteLocalBranch, Dir: repo.Root(), Branch: "nest"},
	} {
		if p := prove(t, none, target); !p.Safe {
			t.Errorf("%s: refused (%s), want safe", target, p.Failed)
		}
	}
	proofRefused(t, prove(t, none, guard.DeleteTarget{Kind: guard.DeleteLocalBranch, Dir: repo.Root(), Branch: "nest/sub"}),
		"holds work origin/main does not")
}

// TestDeleteProofRefusesOnceItsTimeRunsOut pins the budget: a forge slower
// than what is left of it is no answer, and a target asked after it lapsed is
// refused unread, since a host that times out the hook would run the delete.
func TestDeleteProofRefusesOnceItsTimeRunsOut(t *testing.T) {
	repo, _ := proofRepo(t)
	branchWith(repo, "feat", "f.txt")
	repo.Git("push", "-q", "origin", "feat")
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	slow := func(string, ...string) (string, error) { <-release; return `[]`, nil }
	p := NewDeleteProver(DeleteProofOptions{Forge: slow, Budget: 50 * time.Millisecond})
	target := guard.DeleteTarget{Kind: guard.DeleteRemoteBranch, Dir: repo.Root(), Remote: "origin", Branch: "feat"}
	proofRefused(t, p.ProveDelete(target), "could not be asked")
	proofRefused(t, p.ProveDelete(target), "ran out")
}
