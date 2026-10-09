package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestImplementStepRestartIsWired: `implement step --restart <lane>` reaches
// the loop's restart. A lane with no implementer out is refused (exit 2), and
// the flag beside --release or --discard is refused as two decisions; a lane
// whose implementer is out is restarted, its uncommitted work saved aside: the
// text names the aside and the fresh agent to start, and --json carries the
// aside and the re-told await. --yielded is refused without --restart.
func TestImplementStepRestartIsWired(t *testing.T) {
	repo := buildRepo(t)
	repo.Write("AGENTS.md", "# AGENTS.md\n\n- Run make check.\n")
	repo.Commit("conventions")
	mustImplement(t, "build", "itd-10", "--json")

	ref := refusalDocs(t, 2, "implement", "step", "--restart", "lane-1", "--json")
	if ref["stage"] != "restart" || !strings.Contains(ref["reason"].(string), "no implementer out") {
		t.Fatalf("a lane with no implementer out is refused naming it: %v", ref)
	}
	for _, other := range []string{"--release", "--discard"} {
		ref := refusalDocs(t, 2, "implement", "step", "--restart", "lane-1", other, "lane-1", "--json")
		if !strings.Contains(ref["reason"].(string), "one decision each") {
			t.Fatalf("--restart beside %s is refused: %v", other, ref)
		}
	}
	// A blank --yielded is refused as blank, never read as an agent that died.
	for _, blank := range []string{"", "   ", " \t "} {
		ref = refusalDocs(t, 2, "implement", "step", "--restart", "lane-1", "--yielded", blank, "--json")
		if !strings.Contains(ref["reason"].(string), "--yielded is blank") {
			t.Fatalf("--yielded %q is refused as blank: %v", blank, ref)
		}
	}
	ref = refusalDocs(t, 2, "implement", "step", "--yielded", "NETWORK: git push", "--json")
	if !strings.Contains(ref["reason"].(string), "--yielded") {
		t.Fatalf("--yielded without --restart is refused: %v", ref)
	}

	mustStep(t, "implement", "step", "--json")
	mustStep(t, "implement", "step", "--json")
	await := mustStep(t, "implement", "step", "--json")
	if await.Awaiting == nil || await.Awaiting.Role != "implementer" {
		t.Fatalf("the lane awaits its implementer: %+v", await)
	}
	wt := laneWorktreeOf(t)
	if err := os.WriteFile(filepath.Join(wt, "half.txt"), []byte("half done\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	text := mustImplement(t, "implement", "step", "--restart", "lane-1")
	for _, want := range []string{"restart: lane-1", "agent died", "aside: .abcd/.work.local/run/", "1 file(s)", "start a fresh implementer"} {
		if !strings.Contains(text, want) {
			t.Fatalf("the text names %q:\n%s", want, text)
		}
	}
	if _, err := os.Lstat(filepath.Join(wt, "half.txt")); !os.IsNotExist(err) {
		t.Fatalf("the lane's worktree is reset: %v", err)
	}

	if err := os.WriteFile(filepath.Join(wt, "again.txt"), []byte("again\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var res struct {
		Lane     string `json:"lane"`
		Awaiting *struct {
			Role string `json:"role"`
		} `json:"awaiting"`
		Aside struct {
			Path  string   `json:"path"`
			Why   string   `json:"why"`
			Files []string `json:"files"`
		} `json:"aside"`
	}
	// A second restart in the same second takes its own aside.
	out := mustImplement(t, "implement", "step", "--restart", "lane-1", "--yielded", "NETWORK: gh pr view 7", "--json")
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("--restart --json is one object: %v\n%s", err, out)
	}
	if res.Lane != "lane-1" || res.Awaiting == nil || res.Awaiting.Role != "implementer" ||
		!strings.Contains(res.Aside.Path, "/lane-1/aside/") || res.Aside.Why != "agent yielded: NETWORK: gh pr view 7" ||
		len(res.Aside.Files) != 1 || res.Aside.Files[0] != "again.txt" {
		t.Fatalf("--json carries the re-told await and the aside: %+v", res)
	}
}

// laneWorktreeOf is lane-1's worktree, as the one run's state names it.
func laneWorktreeOf(t *testing.T) string {
	t.Helper()
	runs, err := filepath.Glob(filepath.Join(".abcd", ".work.local", "run", "run-*", "state.json"))
	if err != nil || len(runs) != 1 {
		t.Fatalf("one run: %v %v", runs, err)
	}
	data, err := os.ReadFile(runs[0])
	if err != nil {
		t.Fatal(err)
	}
	var st struct {
		Lanes []struct {
			Worktree string `json:"worktree"`
		} `json:"lanes"`
	}
	if err := json.Unmarshal(data, &st); err != nil || len(st.Lanes) == 0 {
		t.Fatalf("the state names the lane: %v", err)
	}
	return st.Lanes[0].Worktree
}
