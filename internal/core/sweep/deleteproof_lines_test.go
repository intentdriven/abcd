package sweep

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/gittest"
)

// endingIn gives the bare repository at bare branches at sha named
// refs/heads/a/<n>/<padding>/refs/heads/<branch>, which `git ls-remote
// origin refs/heads/<branch>` lists beside the branch itself and sorts
// before it, until the listing runs past want bytes; with want 0 it gives
// one. They are made in one update-ref transaction.
func endingIn(t *testing.T, bare, sha, branch string, want int) {
	t.Helper()
	pad := strings.Repeat("x", 200)
	var in strings.Builder
	for n, listed := 0, 0; n == 0 || listed <= want; n++ {
		ref := fmt.Sprintf("refs/heads/a/%d/%s/refs/heads/%s", n, pad, branch)
		fmt.Fprintf(&in, "create %s %s\n", ref, sha)
		listed += len(sha) + 1 + len(ref) + 1
	}
	cmd := exec.Command("git", "-C", bare, "update-ref", "--stdin")
	cmd.Env, cmd.Stdin = gittest.Env(t), strings.NewReader(in.String())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("update-ref: %v\n%s", err, out)
	}
}

// TestARemoteDeleteIsJudgedAtTheBranchsOwnLine: an origin holding feat, with
// a pull request open on it, and more branches whose names end in
// refs/heads/feat than the old bound on the listing kept, sorted before
// feat's own line, never has the delete of feat proven safe: origin's tip of
// feat is read from feat's own line, and the open pull request refuses it.
// A branch origin does not hold, beside one such branch, still reads as
// absent, and its delete as removing nothing.
func TestARemoteDeleteIsJudgedAtTheBranchsOwnLine(t *testing.T) {
	repo, origin := proofRepo(t)
	branchWith(repo, "feat", "f.txt")
	repo.Git("push", "-q", "origin", "feat")
	tip := repo.Git("rev-parse", "feat")
	endingIn(t, origin, tip, "feat", proofListCap)
	if listed := repo.Git("ls-remote", "origin", "refs/heads/feat"); len(listed) <= proofListCap {
		t.Fatalf("origin's listing of feat runs to %d bytes; want more than %d", len(listed), proofListCap)
	}
	open := forgeSaying(`[{"number":7,"headRefName":"feat","headRefOid":"` + tip + `","baseRefName":"main","state":"OPEN"}]`)
	remote := func(b string) guard.DeleteTarget {
		return guard.DeleteTarget{Kind: guard.DeleteRemoteBranch, Dir: repo.Root(), Remote: "origin", Branch: b}
	}
	proofRefused(t, prove(t, DeleteProofOptions{Forge: open}, remote("feat")), "pull request #7 is open")
	got, found, err := lsRemoteTipContext(context.Background(), repo.Root(), "origin", "feat")
	if err != nil || !found || got != tip {
		t.Fatalf("lsRemoteTipContext = %q, %v, %v; want feat's tip %s", got, found, err, tip)
	}

	endingIn(t, origin, tip, "gone", 0)
	if tips, err := lsRemoteRefs(repo.Root(), "origin", pushDestinations("gone")); err != nil || len(tips) != 0 {
		t.Fatalf("a branch origin does not hold, beside one ending in its name: %v, %v; want none held", tips, err)
	}
	if p := prove(t, DeleteProofOptions{Forge: forgeSaying(`[]`)}, remote("gone")); !p.Safe {
		t.Errorf("a branch origin does not hold, beside one ending in its name: refused (%s), want safe", p.Failed)
	}
}

// TestAPushDeletesTheRefGitMatchesItsDestinationTo: git matches a push's
// destination against the remote's refs by the rev-parse rules, so
// `--delete heads/b` deletes refs/heads/b, and `--delete t` deletes the tag
// t where origin holds no branch t. The prover judges the ref so named: the
// branch behind heads/<b> meets every check a branch does (an open pull
// request, the default branch), a destination naming a tag or more than one
// ref is not proven, and a branch named plainly is judged as before.
func TestAPushDeletesTheRefGitMatchesItsDestinationTo(t *testing.T) {
	repo, _ := proofRepo(t)
	held := func(ref string) bool {
		return strings.TrimSpace(repo.Git("ls-remote", "origin", ref)) != ""
	}
	branchWith(repo, "b", "b.txt")
	repo.Git("push", "-q", "origin", "b")
	repo.Git("tag", "t", "main")
	repo.Git("push", "-q", "origin", "refs/tags/t")
	repo.Git("push", "-q", "origin", "--delete", "heads/b")
	if held("refs/heads/b") {
		t.Fatal("git push --delete heads/b left refs/heads/b on origin")
	}
	repo.Git("push", "-q", "origin", "--delete", "t")
	if held("refs/tags/t") {
		t.Fatal("git push --delete t left the tag t on origin")
	}

	repo.Git("push", "-q", "origin", "b", "refs/tags/t")
	tip := repo.Git("rev-parse", "b")
	open := forgeSaying(`[{"number":7,"headRefName":"b","headRefOid":"` + tip + `","baseRefName":"main","state":"OPEN"}]`)
	remote := func(b string) guard.DeleteTarget {
		return guard.DeleteTarget{Kind: guard.DeleteRemoteBranch, Dir: repo.Root(), Remote: "origin", Branch: b}
	}
	proofRefused(t, prove(t, DeleteProofOptions{Forge: open}, remote("heads/b")), "pull request #7 is open")
	proofRefused(t, prove(t, DeleteProofOptions{Forge: open}, remote("b")), "pull request #7 is open")
	proofRefused(t, prove(t, DeleteProofOptions{Forge: forgeSaying(`[]`)}, remote("heads/main")), "the default branch")
	proofRefused(t, prove(t, DeleteProofOptions{Forge: forgeSaying(`[]`)}, remote("t")), "refs/tags/t on origin, which is no branch")
	repo.Git("push", "-q", "origin", "main:refs/tags/b")
	proofRefused(t, prove(t, DeleteProofOptions{Forge: open}, remote("b")), "more than one ref")
	full := guard.DeleteTarget{Kind: guard.DeleteRemoteBranch, Dir: repo.Root(), Remote: "origin", Branch: "b", Ref: "refs/heads/b"}
	proofRefused(t, prove(t, DeleteProofOptions{Forge: open}, full), "pull request #7 is open")
	if p := prove(t, DeleteProofOptions{Forge: forgeSaying(`[]`)}, remote("heads/gone")); !p.Safe {
		t.Errorf("a destination naming no ref of origin's: refused (%s), want safe", p.Failed)
	}
}
