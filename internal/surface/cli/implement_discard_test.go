package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestImplementStepDiscardEndsALaneAtWork (iss-2611231404319245): through
// the CLI, `implement step --discard` ends a lane at its implement stage with
// no agent out in a run no other process drives: it names the bundle its
// commit was saved to, in the run's own directory, and the lane's worktree
// and branch are gone.
func TestImplementStepDiscardEndsALaneAtWork(t *testing.T) {
	repo := buildRepo(t)
	repo.Write("AGENTS.md", "# AGENTS.md\n\n- Run make check.\n")
	repo.Commit("conventions")
	var start struct {
		RunID string `json:"run_id"`
		State string `json:"state"`
	}
	if err := json.Unmarshal([]byte(mustImplement(t, "build", "itd-10", "--json")), &start); err != nil {
		t.Fatal(err)
	}
	mustStep(t, "implement", "step", "--json") // worktree
	mustStep(t, "implement", "step", "--json") // brief
	raw, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(start.State)))
	if err != nil {
		t.Fatal(err)
	}
	var st struct {
		Lanes []struct {
			Stage, Worktree, Branch string
		} `json:"lanes"`
	}
	if err := json.Unmarshal(raw, &st); err != nil || len(st.Lanes) != 1 || st.Lanes[0].Stage != "implement" {
		t.Fatalf("premise: one lane at implement, no agent out yet: %v %+v", err, st)
	}
	wt, branch := st.Lanes[0].Worktree, st.Lanes[0].Branch
	if err := os.WriteFile(filepath.Join(wt, "partial.txt"), []byte("partial\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	repo.Git("-C", wt, "add", "partial.txt")
	repo.Git("-C", wt, "commit", "-q", "-m", "partial work")

	out := mustImplement(t, "implement", "step", "--run", start.RunID, "--discard", "lane-1", "--json")
	var res struct {
		Stage  string `json:"stage"`
		Bundle string `json:"bundle"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if res.Stage != "discarded" || !strings.HasPrefix(res.Bundle, ".abcd/.work.local/run/"+start.RunID+"/lane-1-") {
		t.Fatalf("the lane is discarded, its commit saved to a bundle in the run's directory: %s", out)
	}
	if heads := repo.Git("bundle", "list-heads", filepath.Join(repo.Root(), filepath.FromSlash(res.Bundle))); !strings.Contains(heads, "refs/heads/"+branch) {
		t.Fatalf("the bundle holds the lane's branch: %s", heads)
	}
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Fatalf("the lane's worktree is removed: %v", err)
	}
	if out := repo.Git("branch", "--list", branch); strings.TrimSpace(out) != "" {
		t.Fatalf("the lane's branch is deleted: %q", out)
	}
}
