package loop

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// ceilingOnOrigin pushes to origin's default branch a commit whose
// .abcd/config.json holds body, on top of the branch as last fetched, and
// fetches it, leaving the checkout's own branch and working tree alone: the
// repository's ceiling as a reviewed change landing there would set it.
func ceilingOnOrigin(t *testing.T, repo *gittest.Repo, body string) {
	t.Helper()
	ceilingOnBranch(t, repo, "main", body)
}

// ceilingOnBranch is ceilingOnOrigin for the branch named branch on origin,
// built on origin's main as last fetched.
func ceilingOnBranch(t *testing.T, repo *gittest.Repo, branch, body string) {
	t.Helper()
	index := filepath.Join(t.TempDir(), "ceiling.index")
	git := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", repo.Root()}, args...)...)
		cmd.Env = append(gittest.Env(t), "GIT_INDEX_FILE="+index)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
		return strings.TrimSpace(string(out))
	}
	file := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	base := git("rev-parse", "refs/remotes/origin/main")
	git("read-tree", base)
	git("update-index", "--add", "--cacheinfo", "100644,"+git("hash-object", "-w", file)+",.abcd/config.json")
	c := git("commit-tree", git("write-tree"), "-p", base, "-m", "the repository's ceiling")
	git("push", "-q", "origin", c+":refs/heads/"+branch)
	git("fetch", "-q", "origin")
}

// TestADrainCountsAgainstTheConfiguredCap: with reservation.max_live 12 in
// the repository's file on origin's default branch, a drain holding ten does
// not wait at the cap: it counts against the same cap the reservation act
// judges.
func TestADrainCountsAgainstTheConfiguredCap(t *testing.T) {
	f := drainLandFixture(t)
	first := f.firstMove(t)
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	f.runID, f.issue = first.Start.RunID, secondIssue
	f.parkIssueLaneOnce(t)
	for i := 1; i <= claim.MaxLive-1; i++ {
		id := fmt.Sprintf("iss-26093000000002%02d", i)
		if _, err := claim.Reserve(f.repo.Root(), claim.Request{Record: id, Footprint: []string{fmt.Sprintf("other/f%d.go", i)}}, claim.Options{}); err != nil {
			t.Fatalf("Pat's reservation %d: %v", i, err)
		}
	}
	ceilingOnOrigin(t, f.repo, `{"reservation": {"max_live": 12}}`)
	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatalf("a drain under the repository's 12: %v", err)
	}
	st, _, err := readDrain(f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if st.Waiting == DrainWaitCap || strings.Contains(res.Next, "live reservations") {
		t.Fatalf("holding ten under the repository's cap of twelve, the drain waited at the cap: %q, next %s", st.Waiting, res.Next)
	}
}

// TestALookAlikeBranchNeverHoldsADrainAtTheCap: a branch pushed to origin
// under a conventional name, setting reservation.max_live 1, in a checkout
// that keeps neither origin/HEAD nor origin/main, so the resolver here guesses
// that branch for origin's default, never holds a drain whose person holds
// one at a wait: the drain's count reads the figure where origin says its
// default is, as the reservation act does, and origin's main sets none.
func TestALookAlikeBranchNeverHoldsADrainAtTheCap(t *testing.T) {
	f := drainLandFixture(t)
	first := f.firstMove(t)
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	f.runID, f.issue = first.Start.RunID, secondIssue
	f.parkIssueLaneOnce(t)
	ceilingOnBranch(t, f.repo, "develop", `{"reservation": {"max_live": 1}}`)
	f.repo.Git("update-ref", "-d", "refs/remotes/origin/HEAD")
	f.repo.Git("update-ref", "-d", "refs/remotes/origin/main")
	if got := gitutil.TrackedDefaultBranch(f.repo.Root(), claim.Remote); got != "refs/remotes/origin/develop" {
		t.Fatalf("the resolver here names %q; want it to guess origin/develop", got)
	}
	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatalf("a drain beside a look-alike branch: %v", err)
	}
	st, _, err := readDrain(f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if st.Waiting == DrainWaitCap || strings.Contains(res.Next, "live reservations") {
		t.Fatalf("holding one beside a look-alike branch setting 1, the drain waited at the cap: %q, next %s", st.Waiting, res.Next)
	}
}
