package interview

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestAWorktreePathHoldingANewlineStopsTheInterview: the pre-push gate reads
// git's worktree listing a line at a time, so a worktree whose path holds a
// newline is read there as more than one worktree, and the second can be one
// holding a receipt for HEAD that git itself lists nowhere. The tripwire
// cannot judge the gate as the hook will, so such a path stops the interview,
// saying so in plain words: one a role plants by hand during a turn, its
// gitdir naming the turn directory on its second line beside a receipt
// there; one git itself adds during a turn; and one standing before the
// interview, which stops it before anything is drawn.
func TestAWorktreePathHoldingANewlineStopsTheInterview(t *testing.T) {
	for _, c := range []struct {
		name string
		// during is what the second turn does, given the run, its HEAD and
		// the repository's common directory and the turn's script.
		during func(t *testing.T, r *writtenRun, head, common, script string)
		// before adds the worktree before the run.
		before bool
	}{
		{name: "an entry planted with a newline in its gitdir, a receipt in the turn directory", during: func(t *testing.T, r *writtenRun, head, common, script string) {
			stubAlso(t, script, 2, map[string]string{
				filepath.Join(common, "worktrees", "planted", "gitdir"):    "/nonexistent\nworktree " + turnDirToken + "/.git\n",
				filepath.Join(common, "worktrees", "planted", "HEAD"):      head + "\n",
				filepath.Join(common, "worktrees", "planted", "commondir"): "../..\n",
				"turns/.git":                        "gitdir: " + filepath.Join(common, "worktrees", "planted") + "\n",
				"turns/" + receiptsRel + "/" + head: "commit " + head + "\n",
			})
		}},
		{name: "a worktree git adds at a path holding a newline", during: func(t *testing.T, r *writtenRun, _, _, script string) {
			at := filepath.Join(t.TempDir(), "a\nworktree "+filepath.Join(t.TempDir(), "b"))
			stubPeer(t, script, 2, r.git.Env(), stubCmd{Dir: r.repo, Script: "git worktree add -q --detach '" + at + "'"})
		}},
		{name: "a worktree at a path holding a newline before the interview", before: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
			r := newWrittenRun(t, routedToClaude)
			ignoring(".abcd/.work.local/\n")(r)
			r.git.Commit("base")
			head := r.git.Git("rev-parse", "HEAD")
			common := filepath.Join(r.repo, ".git")
			drawn := 1
			if c.before {
				r.git.Git("worktree", "add", "-q", "--detach", filepath.Join(t.TempDir(), "a\nb"))
				drawn = 0
			} else {
				c.during(t, r, head, common, script)
			}
			_, err := r.w.Run(context.Background())
			if err == nil || !strings.Contains(err.Error(), "newline") {
				t.Fatalf("err = %v, want the interview stopped on a worktree path holding a newline", err)
			}
			if len(r.asked) != drawn || len(r.finished) != 0 {
				t.Fatalf("asked %d, finished %d; nothing more is drawn or finished", len(r.asked), len(r.finished))
			}
		})
	}
}

// TestTheGateCountsAReceiptExactlyWhenTheHookDoes: the tripwire's reading of
// the pre-push gate takes a receipt for HEAD as the hook's own lookup
// (scripts/preflight-receipt.sh check) takes it, by the same name and the same
// checks, so it reads the gate open exactly when that lookup, run here on the
// same repository, says so. A receipt the hook refuses and the reading
// counted would hide a later opening, since only shut to open stops the
// interview: one in a worktree whose commit tracks another file in its local
// tier, and one in a listed worktree git cannot list files in.
func TestTheGateCountsAReceiptExactlyWhenTheHookDoes(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("no bash to run the hook's own receipt lookup")
	}
	lookup, err := filepath.Abs(filepath.Join("..", "..", "..", "scripts", "preflight-receipt.sh"))
	if err != nil {
		t.Fatal(err)
	}
	receipt := func(t *testing.T, dir, head string) string {
		p := filepath.Join(dir, filepath.FromSlash(receiptsRel), head)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("commit "+head+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	laneGit := func(t *testing.T, r *writtenRun, lane string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", lane, "-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture", "-c", "commit.gpgsign=false"}, args...)...)
		cmd.Env = r.git.Env()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	for _, c := range []struct {
		name  string
		setup func(t *testing.T, r *writtenRun, head, lane string)
	}{
		{name: "no receipt", setup: func(*testing.T, *writtenRun, string, string) {}},
		{name: "a receipt in this checkout's own tier", setup: func(t *testing.T, r *writtenRun, head, _ string) { receipt(t, r.repo, head) }},
		{name: "a receipt in a sibling's tier", setup: func(t *testing.T, _ *writtenRun, head, lane string) { receipt(t, lane, head) }},
		{name: "a receipt that is a link to a file", setup: func(t *testing.T, _ *writtenRun, head, lane string) {
			p := receipt(t, lane, head)
			if err := os.Rename(p, p+".real"); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(p+".real", p); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "a receipt that is a directory", setup: func(t *testing.T, _ *writtenRun, head, lane string) {
			if err := os.MkdirAll(filepath.Join(lane, filepath.FromSlash(receiptsRel), head), 0o700); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "a receipt in a sibling whose commit tracks another file in its tier", setup: func(t *testing.T, r *writtenRun, head, lane string) {
			if err := os.WriteFile(filepath.Join(lane, ".abcd", ".work.local", "kept"), []byte("kept\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			laneGit(t, r, lane, "add", "-f", ".abcd/.work.local/kept")
			laneGit(t, r, lane, "commit", "-q", "-m", "track the tier")
			receipt(t, lane, head)
		}},
		{name: "a receipt reached through .abcd linked to a directory the sibling tracks", setup: func(t *testing.T, r *writtenRun, head, lane string) {
			if err := os.MkdirAll(filepath.Join(lane, "store", ".work.local"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(lane, "store", ".work.local", "kept"), []byte("kept\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			laneGit(t, r, lane, "add", "-f", "store/.work.local/kept")
			laneGit(t, r, lane, "commit", "-q", "-m", "track the store")
			if err := os.RemoveAll(filepath.Join(lane, ".abcd")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("store", filepath.Join(lane, ".abcd")); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(lane, ".abcd", ".work.local"), 0o700); err != nil {
				t.Fatal(err)
			}
			receipt(t, lane, head)
		}},
		{name: "a receipt in a listed worktree git cannot list files in", setup: func(t *testing.T, _ *writtenRun, head, lane string) {
			if err := os.Remove(filepath.Join(lane, ".git")); err != nil {
				t.Fatal(err)
			}
			receipt(t, lane, head)
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := newWrittenRun(t, "")
			ignoring(".abcd/.work.local/\n")(r)
			r.git.Commit("base")
			head := r.git.Git("rev-parse", "HEAD")
			lane := laneFor(t, r, "lane")
			c.setup(t, r, head, lane)
			cmd := exec.Command(bash, lookup, "check", head)
			cmd.Dir, cmd.Env = r.repo, r.git.Env()
			out, err := cmd.CombinedOutput()
			var ee *exec.ExitError
			if err != nil && (!errors.As(err, &ee) || ee.ExitCode() != 1) {
				t.Fatalf("the hook's receipt lookup did not answer: %v\n%s", err, out)
			}
			hookOpen := err == nil
			st, err := readTree(r.repo, "")
			if err != nil {
				t.Fatal(err)
			}
			if goOpen := strings.HasPrefix(st[pushGateKey], gateOpen+"\x00"); goOpen != hookOpen {
				t.Fatalf("the hook's lookup reads the gate open %v, the tripwire %v (%q)", hookOpen, goOpen, st[pushGateKey])
			}
		})
	}
}

// TestAPeersServerInfoLeavesTheInterviewRunning: a peer's git gc, or the
// repack a commit's auto-gc runs, rewrites the common directory's info/refs
// for dumb-transport clients, listing every ref. Neither git, which never
// reads it for a local command, nor the pre-push gate reads it, and it runs
// nothing, so a peer rewriting it while the role writes leaves the interview
// running to its outcome; info/exclude there is still watched.
func TestAPeersServerInfoLeavesTheInterviewRunning(t *testing.T) {
	script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubDone)
	r := newWrittenRun(t, routedToClaude)
	r.git.Commit("base")
	peer := laneFor(t, r, "lane")
	stubPeer(t, script, 1, r.git.Env(), stubCmd{Dir: peer, Script: fixtureGit + " commit -q --allow-empty -m peer && git update-server-info"})
	res, err := r.w.Run(context.Background())
	if err != nil {
		t.Fatalf("err = %v, want the interview run to its outcome past a peer's update-server-info", err)
	}
	if _, err := os.Stat(filepath.Join(r.repo, ".git", "info", "refs")); err != nil {
		t.Fatalf("the peer wrote no info/refs, so the case proves nothing: %v", err)
	}
	if len(res.Changed) != 0 || len(r.asked) != 1 || len(r.finished) != 1 {
		t.Fatalf("changed %v, asked %d, finished %d", res.Changed, len(r.asked), len(r.finished))
	}
	st, err := readTree(r.repo, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := st[".git/info/exclude"]; !ok {
		t.Fatalf("info/exclude is not read: %v", st)
	}
}
