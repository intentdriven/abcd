package loop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestANewLaneRunsTheRepositorysPostCheckoutHook is iss-2610120640452939: the
// person's own `git worktree add` runs the repository's post-checkout hook
// (the one `git lfs install` installs), so the lane abcd adds runs it too, in
// the lane's worktree, on the lane's branch.
func TestANewLaneRunsTheRepositorysPostCheckoutHook(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX hook script")
	}
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	mark := filepath.Join(t.TempDir(), "post-checkout")
	hook := filepath.Join(repo.Root(), ".git", "hooks", "post-checkout")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\npwd -P > "+mark+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	res, err := advance(repo.Root(), start.RunID, DefaultStages(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.PerformedStage != StageWorktree {
		t.Fatalf("want the worktree stage performed: %+v", res)
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(mark)
	if err != nil {
		t.Fatal("adding the lane's worktree did not run the repository's post-checkout hook")
	}
	want, err := filepath.EvalSymlinks(st.Lanes[0].Worktree)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(got)) != want {
		t.Fatalf("the post-checkout hook ran in %q, want the lane's worktree %q", strings.TrimSpace(string(got)), want)
	}
}

// shortWorktreeAddTimeout bounds the lane's `git worktree add` at d for the
// rest of the test.
func shortWorktreeAddTimeout(t *testing.T, d time.Duration) {
	t.Helper()
	was := worktreeAddTimeout
	worktreeAddTimeout = d
	t.Cleanup(func() { worktreeAddTimeout = was })
}

// TestALaneCheckoutHasItsOwnBound is iss-2610120640452939's review finding:
// the lane's checkout scales with the tree (a Git LFS smudge downloads every
// object), so it is not held to the commit deadline, which a hook or a
// signing prompt waits on. A checkout longer than the commit deadline is made;
// one past its own, much larger bound is refused naming the checkout and that
// bound, not a signing prompt.
func TestALaneCheckoutHasItsOwnBound(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX hook script")
	}
	t.Run("longer than the commit deadline", func(t *testing.T) {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		hook := filepath.Join(repo.Root(), ".git", "hooks", "post-checkout")
		if err := os.WriteFile(hook, []byte("#!/bin/sh\nsleep 3\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		shortCommitTimeout(t, time.Second)
		start, err := Start(repo.Root(), "itd-10", Options{})
		if err != nil {
			t.Fatal(err)
		}
		res, err := advance(repo.Root(), start.RunID, DefaultStages(), Options{})
		if err != nil {
			t.Fatalf("a checkout longer than the commit deadline is made: %v", err)
		}
		if res.PerformedStage != StageWorktree {
			t.Fatalf("want the worktree stage performed: %+v", res)
		}
	})
	t.Run("past its own bound", func(t *testing.T) {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		sleeper(t, filepath.Join(repo.Root(), ".git", "hooks", "post-checkout"))
		shortWorktreeAddTimeout(t, time.Second)
		start, err := Start(repo.Root(), "itd-10", Options{})
		if err != nil {
			t.Fatal(err)
		}
		began := time.Now()
		_, err = advance(repo.Root(), start.RunID, DefaultStages(), Options{})
		if took := time.Since(began); took > 30*time.Second {
			t.Fatalf("the step took %s; the checkout's bound did not stop it", took)
		}
		r := mustRefusal(t, err)
		if !strings.Contains(r.Reason, "checkout") || !strings.Contains(r.Reason, "within 1s") || strings.Contains(r.Reason, "signing prompt") {
			t.Fatalf("the refusal names the checkout and its bound, not a signing prompt: %+v", r)
		}
	})
}

// TestARestartsResetRunsTheRepositorysHooks is iss-2610120640452939's review
// finding: the person's own `git reset --hard` runs the repository's
// reference-transaction hook (it moves HEAD and ORIG_HEAD) and its
// post-index-change hook (it writes the index), so the reset a restart makes
// in the lane's worktree runs them too, as the person's own reset does.
func TestARestartsResetRunsTheRepositorysHooks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX hook script")
	}
	repo, runID, _, _, _ := deadLane(t)
	mark := filepath.Join(t.TempDir(), "hooks-ran")
	for _, h := range []string{"reference-transaction", "post-index-change"} {
		body := "#!/bin/sh\necho " + h + " >> " + mark + "\ncat >/dev/null 2>&1 || true\n"
		if err := os.WriteFile(filepath.Join(repo.Root(), ".git", "hooks", h), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// The hooks are added mid-run, so the run tripwire would stop the restart
	// over them: the person who installed them accepts them first
	// (configwatch.go). Only the restart's reset is then left to run them.
	if _, err := AcceptConfigChange(repo.Root(), runID, Options{}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(mark); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if _, err := Restart(repo.Root(), runID, "lane-1", "", Options{}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(mark)
	for _, h := range []string{"reference-transaction", "post-index-change"} {
		if !strings.Contains(string(got), h) {
			t.Fatalf("the restart's reset did not run the repository's %s hook (ran: %q)", h, got)
		}
	}
}
