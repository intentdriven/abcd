package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// writeMachineConfig writes the machine layer's config under the test's HOME.
func writeMachineConfig(t *testing.T, body string) {
	t.Helper()
	dir := abcdhome.Path(os.Getenv("HOME"))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestBuildRefusesARunnerModelOffItsAllowlistBeforeTheRun is
// itd-2609201916056194 criterion 5 at the front door: a runner whose model
// route its provider does not list is refused when the lane is about to
// start, naming the key, and no run is created.
func TestBuildRefusesARunnerModelOffItsAllowlistBeforeTheRun(t *testing.T) {
	repo := buildRepo(t)
	writeMachineConfig(t, `{"oracle":{"api":{"local":{"base_url":"http://localhost:11434/v1","models":["qwen3-coder"]}}},`+
		`"runner":{"opencode":{"model":"local/llama-3"}}}`)
	ref := refusalDocs(t, 2, "build", "itd-10", "--json")
	if ref["stage"] != "runner" || !strings.Contains(ref["reason"].(string), "runner.opencode.model") {
		t.Fatalf("the refusal names the runner's model key: %+v", ref)
	}
	runDirAbsent(t, repo.Root())
}

// TestAStepWhoseRunnerIsAbsentPausesTheLaneAndCountsIt is criteria 3 and 4 at
// the front door, under iss-2611012047534449: `implement step` at a stage
// whose role is routed to a runner that is not on PATH hands the role to no
// one, the host included, names it as paused with why, and a later step
// retries it once; `implement status` and `implement record` count each
// failed start per runner and per role. PATH holds git's own directory alone,
// so no harness on the machine can be reached.
func TestAStepWhoseRunnerIsAbsentPausesTheLaneAndCountsIt(t *testing.T) {
	repo := buildRepo(t)
	repo.Write("AGENTS.md", "# AGENTS.md\n\n- Run make check.\n")
	repo.Commit("conventions")
	mustImplement(t, "build", "itd-10", "--json")
	for _, want := range []string{"worktree", "brief"} {
		if res := mustStep(t, "implement", "step", "--json"); res.PerformedStage != want {
			t.Fatalf("step = %+v, want %s", res, want)
		}
	}
	writeMachineConfig(t, `{"roles":{"implementer":{"runner":"opencode"}},"runner":{"opencode":{}}}`)
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	if git, err = filepath.EvalSymlinks(git); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(git))
	for _, h := range []string{"opencode", "claude"} {
		if p, err := exec.LookPath(h); err == nil {
			t.Fatalf("a real %s is reachable at %s; the test runs no real harness", h, p)
		}
	}

	var res struct {
		Awaiting *struct {
			Role string `json:"role"`
		} `json:"awaiting"`
		Paused *struct {
			Role   string `json:"role"`
			Paused struct {
				Runner string `json:"runner"`
				Reason string `json:"reason"`
				Detail string `json:"detail"`
			} `json:"paused"`
		} `json:"paused"`
		Fallback *struct{} `json:"fallback"`
	}
	if err := json.Unmarshal([]byte(mustImplement(t, "implement", "step", "--json")), &res); err != nil {
		t.Fatal(err)
	}
	if res.Awaiting != nil || res.Fallback != nil {
		t.Fatalf("the host is handed nothing: %+v", res)
	}
	if p := res.Paused; p == nil || p.Role != "implementer" || p.Paused.Runner != "opencode" || p.Paused.Reason != "absent" ||
		!strings.Contains(p.Paused.Detail, "not on PATH") {
		t.Fatalf("the step names the paused role and why: %+v", res.Paused)
	}
	text := mustImplement(t, "implement", "step")
	for _, want := range []string{"PAUSED: lane-1's implementer waits for the opencode runner", "retried: lane-1's implementer on the opencode runner", "Do not start an agent"} {
		if !strings.Contains(text, want) {
			t.Fatalf("a later step retries the paused role once and says so (%q): %s", want, text)
		}
	}
	status := mustImplement(t, "implement", "status")
	if !strings.Contains(status, "fallbacks: 2 (by runner: opencode 2; by role: implementer 2)") {
		t.Fatalf("status counts each failed start per runner and per role:\n%s", status)
	}
	var rec struct {
		FallbackCounts struct {
			Total    int            `json:"total"`
			ByRunner map[string]int `json:"by_runner"`
			ByRole   map[string]int `json:"by_role"`
		} `json:"fallback_counts"`
	}
	if err := json.Unmarshal([]byte(mustImplement(t, "implement", "record", "--json")), &rec); err != nil {
		t.Fatal(err)
	}
	if c := rec.FallbackCounts; c.Total != 2 || c.ByRunner["opencode"] != 2 || c.ByRole["implementer"] != 2 {
		t.Fatalf("the run record counts the fallback: %+v", c)
	}
	if text := mustImplement(t, "implement", "record"); !strings.Contains(text, "fallbacks: 2 (by runner: opencode 2; by role: implementer 2)") {
		t.Fatalf("the record's text counts the fallback:\n%s", text)
	}
}

// TestARepositoryRouteToARunnerIsSkippedAtTheFrontDoor is rulings RN2 and OC2
// (2026-10-02) at the front door: a route the repository commits that hands a
// role to a runner is skipped with a warning on stderr naming its file, its
// key and the machine's file, and the host is handed the role as if it were
// unrouted: no runner is asked, so no fallback is recorded.
func TestARepositoryRouteToARunnerIsSkippedAtTheFrontDoor(t *testing.T) {
	repo := buildRepo(t)
	repo.Write("AGENTS.md", "# AGENTS.md\n\n- Run make check.\n")
	repo.Commit("conventions")
	mustImplement(t, "build", "itd-10", "--json")
	for _, want := range []string{"worktree", "brief"} {
		if res := mustStep(t, "implement", "step", "--json"); res.PerformedStage != want {
			t.Fatalf("step = %+v, want %s", res, want)
		}
	}
	writeMachineConfig(t, `{"runner":{"opencode":{}}}`)
	repo.Write(".abcd/config.json", `{"roles":{"implementer":{"runner":"opencode"}}}`)
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	if git, err = filepath.EvalSymlinks(git); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(git))

	code, out, errOut := implementCLI(t, "implement", "step", "--json")
	if code != 0 {
		t.Fatalf("step exited %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	var res struct {
		Awaiting *struct {
			Role string `json:"role"`
		} `json:"awaiting"`
		Fallback *struct {
			Asked string `json:"asked"`
		} `json:"fallback"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatal(err)
	}
	if res.Awaiting == nil || res.Awaiting.Role != "implementer" || res.Fallback != nil {
		t.Fatalf("the host is handed the implementer as if unrouted: %+v (fallback %+v)", res, res.Fallback)
	}
	for _, want := range []string{".abcd/config.json (repo layer)", "roles.implementer.runner", "skipped", abcdhome.Display("config.json")} {
		if !strings.Contains(errOut, want) {
			t.Errorf("stderr does not name %q:\n%s", want, errOut)
		}
	}
}
