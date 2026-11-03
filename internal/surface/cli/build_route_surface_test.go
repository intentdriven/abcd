package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// routeStepMachine enables opencode with a provider allowlist and routes no
// role, so only a step's --route can hand the implementer to it.
const routeStepMachine = `{"oracle":{"api":{"local":{"base_url":"http://localhost:11434/v1","models":["qwen3-coder","devstral"]}}},` +
	`"runner":{"opencode":{"model":"local/qwen3-coder"}}}`

// runToImplement starts a run and steps its lane to the implement stage,
// returning the run's state path.
func runToImplement(t *testing.T) string {
	t.Helper()
	repo := buildRepo(t)
	repo.Write("AGENTS.md", "# AGENTS.md\n\n- Run make check.\n")
	repo.Commit("conventions")
	var start struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal([]byte(mustImplement(t, "build", "itd-10", "--json")), &start); err != nil || start.RunID == "" {
		t.Fatalf("start: %v %+v", err, start)
	}
	for _, want := range []string{"worktree", "brief"} {
		if res := mustStep(t, "implement", "step", "--json"); res.PerformedStage != want {
			t.Fatalf("step = %+v, want %s", res, want)
		}
	}
	return filepath.Join(repo.Root(), filepath.FromSlash(loop.StateRelPath(start.RunID)))
}

// onlyGitOnPath leaves git's own directory alone on PATH, so no harness on the
// machine can be reached.
func onlyGitOnPath(t *testing.T) {
	t.Helper()
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
}

// TestAStepsRouteHandsTheRoleToTheRunnerItNames is iss-2610161801358637 at the
// front door: no file routes the implementer, and `implement step --route
// implementer=runner:opencode@<provider>/<model>` hands it to opencode for
// that step. The runner is not on PATH, so the lane pauses rather than handing
// the role to the host (iss-2611012047534449), naming opencode as the runner
// asked for, which only the --route made it.
func TestAStepsRouteHandsTheRoleToTheRunnerItNames(t *testing.T) {
	runToImplement(t)
	writeMachineConfig(t, routeStepMachine)
	onlyGitOnPath(t)
	var res struct {
		Awaiting *struct {
			Role string `json:"role"`
		} `json:"awaiting"`
		Paused *struct {
			Role   string `json:"role"`
			Paused struct {
				Runner string `json:"runner"`
				Reason string `json:"reason"`
			} `json:"paused"`
		} `json:"paused"`
	}
	out := mustImplement(t, "implement", "step", "--json", "--route", "implementer=runner:opencode@local/devstral")
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatal(err)
	}
	if res.Awaiting != nil || res.Paused == nil || res.Paused.Role != "implementer" ||
		res.Paused.Paused.Runner != "opencode" || res.Paused.Paused.Reason != "absent" {
		t.Fatalf("the --route asked opencode for the implementer, and the lane paused: %s", out)
	}
}

// TestARestartTakesARouteAndConsultsTheRunner (iss-2611012047534449): a
// --route with --restart is no longer refused; the restarted implementer is
// started on the route's runner at once, and, the runner absent, the lane
// pauses with the restart's aside saved, named in the text.
func TestARestartTakesARouteAndConsultsTheRunner(t *testing.T) {
	runToImplement(t)
	writeMachineConfig(t, routeStepMachine)
	onlyGitOnPath(t)
	// The implementer is handed to the host, as before any route existed.
	if res := mustStep(t, "implement", "step", "--json"); res.Awaiting == nil || res.Awaiting.Role != "implementer" {
		t.Fatalf("the host is handed the implementer: %+v", res)
	}
	text := mustImplement(t, "implement", "step", "--restart", "lane-1", "--route", "implementer=runner:opencode@local/devstral")
	for _, want := range []string{"restart: lane-1", "PAUSED: lane-1's implementer waits for the opencode runner", "retried: lane-1's implementer on the opencode runner", "Do not start an agent"} {
		if !strings.Contains(text, want) {
			t.Fatalf("the text names %q:\n%s", want, text)
		}
	}
}

// TestAStepsRouteRefusalsExitTwoAndWriteNothing: a runner the machine has not
// enabled, a model its provider does not list, a tier route, a role the step
// does not dispatch, and a --route on a step that dispatches no role are each
// refused at exit 2 before the step runs, naming the fix, and the run's state
// is left byte for byte; so is a --route beside --accept-config-change, which
// writes its acceptance before the step is judged.
func TestAStepsRouteRefusalsExitTwoAndWriteNothing(t *testing.T) {
	path := runToImplement(t)
	writeMachineConfig(t, routeStepMachine)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"runner not enabled", []string{"--route", "implementer=runner:claude"}, "not enabled on this machine"},
		{"model off the allowlist", []string{"--route", "implementer=runner:opencode@local/llama-9000"}, "local's list"},
		{"tier route", []string{"--route", "implementer=frontier"}, "names a tier"},
		{"no runner rung", []string{"--route", "implementer=opencode@local/devstral"}, "runner:<name>"},
		{"role not dispatched", []string{"--route", "scribe=runner:opencode"}, `not "scribe"`},
		{"a step that dispatches no role", []string{"--route", "implementer=runner:opencode", "--discard", "lane-1"}, "--discard"},
		// --accept-config-change writes its acceptance before the step is
		// judged, so a --route beside it would be refused after a write.
		{"with --accept-config-change", []string{"--route", "security-reviewer=runner:opencode", "--accept-config-change"}, "--accept-config-change"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ref := refusalDocs(t, 2, append([]string{"implement", "step", "--json"}, tc.args...)...)
			if !strings.Contains(ref["reason"].(string), tc.want) {
				t.Fatalf("the refusal names %q: %+v", tc.want, ref)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("a refused --route changed the run's state")
			}
		})
	}
}

// TestARoutedReceiptsLineNamesTheRoute: the human rendering of a receipt a
// runner ran under a step's --route names the --route, and one run on the
// configured route reads as before.
func TestARoutedReceiptsLineNamesTheRoute(t *testing.T) {
	r := &runner.RouteRecord{Asked: "opencode", Ran: "opencode", Model: "local/devstral", Override: "implementer=runner:opencode@local/devstral"}
	if got := routeSuffix(r); !strings.HasSuffix(got, "model local/devstral; under --route implementer=runner:opencode@local/devstral)") {
		t.Fatalf("suffix = %q", got)
	}
	r.Override = ""
	if got := routeSuffix(r); got != " — ran on the opencode runner (asked opencode; model local/devstral)" {
		t.Fatalf("suffix = %q", got)
	}
}

// stateUnchanged asserts the run's state at path still reads before.
func stateUnchanged(t *testing.T, path string, before []byte) {
	t.Helper()
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("a refused --route changed the run's state")
	}
}

// TestAStepsRouteForARoleTheStepDoesNotHandOutIsRefused: a --route the step
// would not use is refused at exit 2 before anything is written, never
// dropped: a step that performs a stage abcd runs itself hands no role; a
// step that hands out the implementer refuses a reviewer's route, naming the
// implementer; and a step that re-tells an await already out hands no role.
func TestAStepsRouteForARoleTheStepDoesNotHandOutIsRefused(t *testing.T) {
	repo := buildRepo(t)
	repo.Write("AGENTS.md", "# AGENTS.md\n\n- Run make check.\n")
	repo.Commit("conventions")
	var start struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal([]byte(mustImplement(t, "build", "itd-10", "--json")), &start); err != nil || start.RunID == "" {
		t.Fatalf("start: %v %+v", err, start)
	}
	path := filepath.Join(repo.Root(), filepath.FromSlash(loop.StateRelPath(start.RunID)))
	writeMachineConfig(t, routeStepMachine)
	refused := func(route string, want ...string) {
		t.Helper()
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		ref := refusalDocs(t, 2, "implement", "step", "--json", "--route", route)
		for _, w := range want {
			if !strings.Contains(ref["reason"].(string), w) {
				t.Fatalf("--route %s: the refusal names %q: %+v", route, w, ref)
			}
		}
		stateUnchanged(t, path, before)
	}
	refused("implementer=runner:opencode", "hands no role", "worktree")
	for _, want := range []string{"worktree", "brief"} {
		if res := mustStep(t, "implement", "step", "--json"); res.PerformedStage != want {
			t.Fatalf("step = %+v, want %s", res, want)
		}
	}
	refused("security-reviewer=runner:opencode", "security-reviewer", "lane-1's implementer")
	res := mustStep(t, "implement", "step", "--json")
	if res.Awaiting == nil || res.Awaiting.Role != "implementer" {
		t.Fatalf("the host is handed the implementer: %+v", res)
	}
	// The host's agent has written its receipt, so the role is the host's to
	// hand back, and no --route retries it (iss-2611012047534449).
	if err := os.WriteFile(res.Awaiting.Receipt, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	refused("implementer=runner:opencode", "hands no role", "lane-1's implementer")
}

// TestAStepsRouteHandsAReviewerToTheRunnerItNames: at the validators, a
// --route for the reviewer the step hands out hands it to the runner it
// names (absent from PATH here, so the lane pauses naming opencode); a route
// for the other reviewer is refused, naming the one the step hands out.
func TestAStepsRouteHandsAReviewerToTheRunnerItNames(t *testing.T) {
	repo := buildRepo(t)
	repo.Write("AGENTS.md", "# AGENTS.md\n\n- Run make check.\n")
	repo.Commit("conventions")
	// The run's first ruthless reviewer is asked for the middle class, which
	// this machine maps to the model the --route names: a --route naming
	// another model than the reviewer's class is refused before anything
	// launches (dcn-2610292236161673 (b), internal/core/runner).
	writeMachineConfig(t, `{"models":{"classes":{"middle":"local/devstral","below":"local/qwen3-coder"}}}`)
	laneAtValidate(t, repo)
	writeMachineConfig(t, routeStepMachine)
	onlyGitOnPath(t)
	ref := refusalDocs(t, 2, "implement", "step", "--json", "--route", "security-reviewer=runner:opencode")
	if !strings.Contains(ref["reason"].(string), "lane-1's ruthless-reviewer") {
		t.Fatalf("the refusal names the reviewer the step hands out: %+v", ref)
	}
	var res struct {
		Paused *struct {
			Role   string `json:"role"`
			Paused struct {
				Runner string `json:"runner"`
			} `json:"paused"`
		} `json:"paused"`
	}
	out := mustImplement(t, "implement", "step", "--json", "--route", "ruthless-reviewer=runner:opencode@local/devstral")
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatal(err)
	}
	if res.Paused == nil || res.Paused.Role != "ruthless-reviewer" || res.Paused.Paused.Runner != "opencode" {
		t.Fatalf("the --route asked opencode for the ruthless reviewer: %s", out)
	}
}

// TestAStepsRouteSilencesTheSetupWarningOfTheRouteItReplaces: the warning
// that the repository keeps a role on the host over the person's runner
// route is not printed when the step's --route replaces that role's route,
// since it would contradict what then runs.
func TestAStepsRouteSilencesTheSetupWarningOfTheRouteItReplaces(t *testing.T) {
	runToImplement(t)
	writeMachineConfig(t, `{"oracle":{"api":{"local":{"base_url":"http://localhost:11434/v1","models":["qwen3-coder","devstral"]}}},`+
		`"roles":{"implementer":{"runner":"opencode"}},"runner":{"opencode":{"model":"local/qwen3-coder"}}}`)
	if err := os.WriteFile(filepath.Join(".abcd", "config.json"), []byte(`{"roles":{"implementer":{"runner":"host"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	onlyGitOnPath(t)
	code, out, errOut := implementCLI(t, "implement", "step", "--json", "--route", "implementer=runner:opencode@local/devstral")
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	if strings.Contains(errOut, "the repository's route stands") {
		t.Fatalf("the warning for the route the --route replaced was printed:\n%s", errOut)
	}
}
