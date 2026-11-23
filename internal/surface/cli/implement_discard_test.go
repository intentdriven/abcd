package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
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

// TestImplementStepDiscardLeavesAReservationMadeAgainAfterARelease: through
// the CLI, a run's reservation released by hand and made again from this copy
// is not the run's: a discard leaves it standing, though the step's driver
// claim writes the run's state just before the discard (it replaces a driver
// that is gone).
func TestImplementStepDiscardLeavesAReservationMadeAgainAfterARelease(t *testing.T) {
	repo := buildRepo(t)
	repo.Write("AGENTS.md", "# AGENTS.md\n\n- Run make check.\n")
	repo.Write(buildSpec, "---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n\n## Summary\n\nA written design record.\n\n"+
		"## Footprint\n\n- packages: internal/widget\n\n## Steps\n\n1. The state file\n")
	repo.Commit("conventions")
	bare := filepath.Join(t.TempDir(), "origin.git")
	gitCmd(t, filepath.Dir(bare), "init", "-q", "--bare", bare)
	gitCmd(t, bare, "symbolic-ref", "HEAD", "refs/heads/main")
	repo.Git("config", "user.name", "Pat Example")
	repo.Git("config", "user.email", octoMail)
	repo.Git("remote", "add", "origin", bare)
	repo.Git("push", "-q", "origin", "main")
	repo.Git("fetch", "-q", "origin")
	var start struct {
		RunID string `json:"run_id"`
		State string `json:"state"`
	}
	if err := json.Unmarshal([]byte(mustImplement(t, "build", "itd-10", "--json")), &start); err != nil {
		t.Fatal(err)
	}
	mustStep(t, "implement", "step", "--json") // worktree
	mustStep(t, "implement", "step", "--json") // brief
	tip := func() string {
		return strings.TrimSpace(gitCmd(t, bare, "for-each-ref", "--format=%(objectname)", claim.RefPrefix+"itd-10"))
	}
	if tip() == "" {
		t.Fatal("premise: the build reserved itd-10 on origin")
	}
	mustImplement(t, "reserve", "itd-10", "--release")
	mustImplement(t, "reserve", "itd-10", "--footprint", "internal/widget")
	mine := tip()

	ctl := filepath.Join(repo.Root(), ".abcd", ".work.local", "run", start.RunID, "control.json")
	raw, err := os.ReadFile(ctl)
	if err != nil {
		t.Fatal(err)
	}
	var c map[string]any
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	c["driver"] = map[string]any{"pid": 2147483646, "start_unix": 1, "at": "2026-01-01T00:00:00Z"}
	if raw, err = json.Marshal(c); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ctl, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	out := mustImplement(t, "implement", "step", "--run", start.RunID, "--discard", "lane-1", "--json")
	if !strings.Contains(out, `"stage": "discarded"`) && !strings.Contains(out, `"stage":"discarded"`) {
		t.Fatalf("the lane is discarded: %s", out)
	}
	if got := tip(); got != mine {
		t.Fatalf("the reservation made again is left standing at %s; origin now holds %s\n%s", mine, got, out)
	}
	state, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(start.State)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(state), "as it stands") {
		t.Fatalf("the run record says the reservation was left as it stands:\n%s", state)
	}
}
