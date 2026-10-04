package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// The pacing intent at the surface (itd-2609201925079472): `abcd build`'s
// --pace and --sub-agents, the pace the run record names, and the window clock
// `abcd implement step` keeps.

type paceJSON struct {
	Pace *struct {
		WorkMinutes struct {
			Value         int
			Layer, Origin string
		} `json:"work_minutes"`
		PauseMinutes struct {
			Value         int
			Layer, Origin string
		} `json:"pause_minutes"`
		SubAgents struct {
			Value         int
			Layer, Origin string
		} `json:"sub_agents"`
	} `json:"pace"`
	RunID string `json:"run_id"`
}

func buildPace(t *testing.T, args ...string) paceJSON {
	t.Helper()
	var res paceJSON
	out := mustImplement(t, append([]string{"build", "itd-10", "--json"}, args...)...)
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if res.Pace == nil {
		t.Fatalf("the build names the run's pace: %s", out)
	}
	return res
}

// TestBuildRunsOnTheBundledPaceAndNamesIt is criterion 1 at the surface: with
// no flag and no configuration the run is paced 120/300 with two lanes, and
// the build and the status name the bundled layer.
func TestBuildRunsOnTheBundledPaceAndNamesIt(t *testing.T) {
	buildRepo(t)
	res := buildPace(t)
	p := res.Pace
	if p.WorkMinutes.Value != 120 || p.PauseMinutes.Value != 300 || p.SubAgents.Value != 2 ||
		p.WorkMinutes.Layer != "bundled" || p.SubAgents.Layer != "bundled" {
		t.Fatalf("pace = %+v", *p)
	}
	status := mustImplement(t, "implement", "status")
	if !strings.Contains(status, "pace:    120/300 minutes, 2 sub-agents (bundled)") {
		t.Fatalf("the status names the pace and its layer:\n%s", status)
	}
	if !strings.Contains(status, "pace 120/300 minutes, 2 sub-agents (bundled)") {
		t.Fatalf("the run record names the pace and its layer:\n%s", status)
	}
}

// TestBuildPaceFlagsWinOverTheConfiguration is criteria 2 and 3 at the
// surface: the repository's configuration wins over the machine's, and the
// flags win over both, each named in the text.
func TestBuildPaceFlagsWinOverTheConfiguration(t *testing.T) {
	repo := buildRepo(t)
	home := os.Getenv("HOME")
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abcdhome.Path(home, "config.json"), []byte(`{"pace": {"work_minutes": 60, "pause_minutes": 120, "sub_agents": 1}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	repo.Write(".abcd/config.json", `{"pace": {"work_minutes": 100, "pause_minutes": 200, "sub_agents": 4}}`)
	repo.Commit("pace")

	code, out, errOut := implementCLI(t, "build", "itd-10", "--pace", "90/240", "--sub-agents", "3")
	if code != 0 || !strings.Contains(out, "pace:    90/240 minutes, 3 sub-agents (work from --pace 90/240, the flag layer, pause from --pace 90/240, the flag layer, sub-agents from --sub-agents 3, the flag layer)") {
		t.Fatalf("the flags win and are named: exit %d\n%s\n%s", code, out, errOut)
	}
	if err := os.RemoveAll(filepath.Join(repo.Root(), ".abcd", ".work.local", "run")); err != nil {
		t.Fatal(err)
	}
	p := buildPace(t).Pace
	if p.WorkMinutes.Value != 100 || p.WorkMinutes.Layer != "repo" || p.WorkMinutes.Origin != ".abcd/config.json" || p.SubAgents.Value != 4 {
		t.Fatalf("without a flag the repository's file wins over the machine's: %+v", *p)
	}
}

// TestBuildRefusesAMalformedPace is criterion 9 at the surface: a malformed
// --pace or --sub-agents is refused at exit 2, naming the value and the
// accepted form, and writes nothing.
func TestBuildRefusesAMalformedPace(t *testing.T) {
	repo := buildRepo(t)
	for _, args := range [][]string{{"--pace", "90"}, {"--sub-agents", "two"}, {"--pace", "0/300"}} {
		ref := refusalDocs(t, 2, append([]string{"build", "itd-10", "--json"}, args...)...)
		if ref["stage"] != "pace" || !strings.Contains(ref["reason"].(string), args[1]) ||
			!strings.Contains(ref["remedy"].(string), "<work-minutes>/<pause-minutes>") {
			t.Fatalf("%v: refusal = %v", args, ref)
		}
		runDirAbsent(t, repo.Root())
	}
	code, _, errOut := implementCLI(t, "build", "itd-10", "--pace", "90")
	if code != 2 || !strings.Contains(errOut, "refused at pace") || !strings.Contains(errOut, "<work-minutes>/<pause-minutes>") {
		t.Fatalf("text refusal: exit %d\n%s", code, errOut)
	}
	runDirAbsent(t, repo.Root())
}

// TestAnElapsedWindowPausesTheRunAtTheSurface is criteria 4 and 5 at the
// surface: once the run's window has elapsed, `implement step` starts nothing,
// exits 0 naming next_eligible_at, and the status says the run is paused; a
// step before that time is refused at exit 3 naming it.
func TestAnElapsedWindowPausesTheRunAtTheSurface(t *testing.T) {
	repo := buildRepo(t)
	run := buildPace(t, "--pace", "60/30").RunID
	statePath := filepath.Join(repo.Root(), ".abcd", ".work.local", "run", run, "state.json")
	raw, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	var st map[string]any
	if err := json.Unmarshal(raw, &st); err != nil {
		t.Fatal(err)
	}
	st["window_started_at"] = time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	rewritten, _ := json.MarshalIndent(st, "", "  ")
	if err := os.WriteFile(statePath, rewritten, 0o600); err != nil {
		t.Fatal(err)
	}

	var res struct {
		PerformedStage string `json:"performed_stage"`
		NextEligibleAt string `json:"next_eligible_at"`
		Next           string `json:"next"`
	}
	out := mustImplement(t, "implement", "step", "--json")
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatal(err)
	}
	if res.PerformedStage != "" || res.NextEligibleAt == "" || !strings.Contains(res.Next, res.NextEligibleAt) {
		t.Fatalf("an elapsed window performs nothing and names next_eligible_at: %s", out)
	}
	status := mustImplement(t, "implement", "status")
	if !strings.Contains(status, "paused until "+res.NextEligibleAt) {
		t.Fatalf("the status names the pause:\n%s", status)
	}
	ref := refusalDocs(t, 3, "implement", "step", "--json")
	if ref["stage"] != "pause" || !strings.Contains(ref["reason"].(string), res.NextEligibleAt) {
		t.Fatalf("a step inside the pause is refused naming the time: %v", ref)
	}
	code, _, errOut := implementCLI(t, "implement", "step")
	if code != 3 || !strings.Contains(errOut, res.NextEligibleAt) {
		t.Fatalf("text: exit %d\n%s", code, errOut)
	}
}
