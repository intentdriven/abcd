package loop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
