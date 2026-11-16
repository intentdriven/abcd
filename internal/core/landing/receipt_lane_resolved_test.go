package landing

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// laneName is the store name the implement loop gives a lane's worktree, as
// the layouts below name theirs.
const laneName = "run-1234567890123456-lane-1"

// laneWorktree adds a worktree of f's repository on a new branch at a lane's
// name in a store of its own, and returns its path.
func laneWorktree(t *testing.T, f *fixture, branch string) string {
	t.Helper()
	lane := filepath.Join(t.TempDir(), "store", laneName)
	f.repo.Git("worktree", "add", "-q", "-b", branch, lane, "HEAD")
	return lane
}

// relistedLane builds a drive lane's worktree whose own .git file names a
// repository nested inside the lane, so the worktrees git lists from inside
// the lane are two paths that do not end in the lane's name: lane/x, the
// nested repository on lanebr, and lane/s, a symlink to the lane itself. Both
// lie in the lane. It returns the lane and lane/x.
func relistedLane(t *testing.T, f *fixture) (lane, x string) {
	t.Helper()
	lane = laneWorktree(t, f, "lanebr")
	x = filepath.Join(lane, "x")
	if err := os.MkdirAll(x, 0o755); err != nil {
		t.Fatal(err)
	}
	xr := gittest.NewRepoAt(t, x)
	xr.Git("fetch", "-q", "--update-head-ok", f.repo.Root(), "+refs/heads/lanebr:refs/heads/lanebr")
	xr.Git("symbolic-ref", "HEAD", "refs/heads/lanebr")
	xr.Git("reset", "-q", "--hard")
	xr.Git("remote", "add", "origin", f.bare)
	xr.Git("fetch", "-q", "origin")
	w := filepath.Join(x, ".git", "worktrees", "w")
	if err := os.MkdirAll(w, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"HEAD":      "ref: refs/heads/lanebr\n",
		"commondir": "../..\n",
		"gitdir":    filepath.Join(lane, "s", ".git") + "\n",
	} {
		if err := os.WriteFile(filepath.Join(w, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(".", filepath.Join(lane, "s")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(x, ".git", "info"), 0o755); err != nil {
		t.Fatal(err)
	}
	exclude, err := os.OpenFile(filepath.Join(x, ".git", "info", "exclude"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := exclude.WriteString("x/\ns\n"); err != nil {
		t.Fatal(err)
	}
	if err := exclude.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lane, ".git"), []byte("gitdir: "+w+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "read-tree", "lanebr")
	cmd.Dir = lane
	cmd.Env = append(gittest.Env(t), "GIT_DIR="+w, "GIT_WORK_TREE="+lane)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git read-tree: %v\n%s", err, out)
	}
	wts, err := gitutil.ListWorktrees(lane, gitutil.MaxWorktreeListing)
	if err != nil {
		t.Fatal(err)
	}
	var listed []string
	for _, wt := range wts {
		listed = append(listed, filepath.Base(wt.Path))
	}
	if strings.Join(listed, " ") != "x s" {
		t.Fatalf("git lists %v from the lane; want x and s", listed)
	}
	return lane, x
}

// landFrom lands from the checkout at cwd, branch "" being its own, with an
// open pull request at the fixture's head, and returns the enqueue step.
func landFrom(t *testing.T, f *fixture, cwd, branch string) Step {
	t.Helper()
	f.forge.head = f.head()
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewBlocked, viewQueued}
	o := f.options()
	o.Branch = branch
	res, err := Land(cwd, o)
	if err != nil {
		t.Fatalf("land from %s: %v (steps %+v)", cwd, err, res.Steps)
	}
	return step(t, res, "enqueue")
}

// wantNotArmed fails unless the enqueue step armed nothing and its note says
// every phrase in want.
func wantNotArmed(t *testing.T, f *fixture, s Step, want ...string) {
	t.Helper()
	if f.forge.called("pr", "merge", "7", "--auto") != 0 || s.Status != StatusLeft {
		t.Fatalf("enqueue step %+v, calls %v: want nothing armed", s, f.forge.calls)
	}
	for _, w := range want {
		if !strings.Contains(s.Note, w) {
			t.Fatalf("enqueue note = %q; want it to say %q", s.Note, w)
		}
	}
}

func TestAPathInsideADriveLaneIsToldResolved(t *testing.T) {
	store := t.TempDir()
	lane := filepath.Join(store, laneName)
	mine := filepath.Join(store, "mine")
	for _, d := range []string{filepath.Join(lane, "x", "y"), mine} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(".", filepath.Join(lane, "s")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(lane, filepath.Join(mine, "into")); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]bool{
		lane:                                  true,
		filepath.Join(lane, "x"):              true,
		filepath.Join(lane, "x", "y"):         true,
		filepath.Join(lane, "s"):              true,
		filepath.Join(lane, "not-yet"):        true,
		filepath.Join(mine, "into"):           true,
		filepath.Join(mine, "into", "x"):      true,
		mine:                                  false,
		filepath.Join(store, "lane-1"):        false,
		"":                                    false,
		"/home/u/run-0000000000000001-lane-0": false,
	} {
		if got := insideDriveLane(path); got != want {
			t.Errorf("insideDriveLane(%q) = %v, want %v", path, got, want)
		}
	}
}

// Landing from a drive lane's worktree whose own listing names the nested
// repository and a symlink back to the lane, neither path ending in the
// lane's name, counts no review receipt in either and names the lane: on its
// current branch, and on the branch named.
func TestLandFromARelistedLaneCountsNoReceipt(t *testing.T) {
	for _, branch := range []string{"", "lanebr"} {
		t.Run("branch="+branch, func(t *testing.T) {
			f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
			lane, x := relistedLane(t, f)
			writeReceiptAt(t, x, f.head(), fullPairBody(f.head()), 0o600)
			s := landFrom(t, f, lane, branch)
			wantNotArmed(t, f, s, laneName, "is a drive lane's worktree or lies inside one", "main checkout or your own worktree")
		})
	}
}

// A repository nested inside a drive lane's worktree, with its own .git and
// the real origin, is the lane's: landing from it counts no receipt there.
func TestLandFromARepositoryNestedInALaneCountsNoReceipt(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	lane := filepath.Join(t.TempDir(), "store", laneName)
	f.repo.Git("worktree", "add", "-q", "--detach", lane, "HEAD")
	sub := filepath.Join(lane, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	sr := gittest.NewRepoAt(t, sub)
	sr.Git("remote", "add", "origin", f.bare)
	sr.Git("fetch", "-q", "origin")
	sr.Git("fetch", "-q", f.repo.Root(), "+refs/heads/feat/x:refs/heads/feat/x")
	sr.Git("checkout", "-q", "feat/x")
	writeReceiptAt(t, sub, f.head(), fullPairBody(f.head()), 0o600)
	s := landFrom(t, f, sub, "")
	wantNotArmed(t, f, s, filepath.Join(laneName, "sub"), "is a drive lane's worktree or lies inside one")
}

// Landing from the main checkout a branch a drive lane's worktree holds counts
// no receipt, even the main checkout's own, and says the branch is checked out
// in the lane, which lands it itself, rather than to run it from the main
// checkout, where the person already is.
func TestLandOfABranchALaneHoldsSaysTheDriveLandsIt(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	f.reviewed(t)
	laneWorktree(t, f, "lanebr")
	s := landFrom(t, f, f.repo.Root(), "lanebr")
	wantNotArmed(t, f, s, "the branch is checked out in a drive lane's worktree", laneName, "the drive lands that branch itself", "handed back and its worktree removed first")
	if strings.Contains(s.Note, "run it from the main checkout") {
		t.Fatalf("enqueue note = %q; want no bidding to run it from where it ran", s.Note)
	}
}

// A receipt or an approval in the relisted lane vouches for nothing, asked by
// either listed path, and an approval there is no person's to the landings
// reader, the worktrees listed by it or passed in.
func TestARelistedLanesRecordsVouchForNothing(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	lane, x := relistedLane(t, f)
	head := f.head()
	writeReceiptAt(t, x, head, fullPairBody(head), 0o600)
	writeApprovalAt(t, x, head)
	for _, dir := range []string{filepath.Join(lane, "s"), x, lane} {
		if vouched, err := headVouched(dir, head); err != nil || vouched {
			t.Errorf("headVouched(%s) = %v, %v; want nothing vouched in a lane", dir, vouched, err)
		}
	}
	ctx := context.Background()
	listed, err := gitutil.ListWorktrees(lane, gitutil.MaxWorktreeListing)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		root string
		wts  []gitutil.Worktree
	}{{lane, nil}, {lane, listed}, {f.repo.Root(), listed}} {
		if p, err := ApprovalRecord(ctx, c.root, head, c.wts); err != nil || p != "" {
			t.Errorf("ApprovalRecord(%s, %d listed) = %q, %v; want no approval in a lane", c.root, len(c.wts), p, err)
		}
	}
	for _, root := range []string{filepath.Join(lane, "s"), x} {
		if _, err := RecordApproval(root, head); err == nil || !strings.Contains(err.Error(), ErrDriveLaneRecord.Error()) {
			t.Errorf("RecordApproval(%s) = %v; want it refused in a lane", root, err)
		}
	}
}
