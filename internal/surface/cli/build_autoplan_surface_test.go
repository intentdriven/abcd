package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/core/occupancy"
	"github.com/intentdriven/abcd/internal/core/sessionname"
)

// TestBuildAutoPlanIsWiredAndRefusesAnythingLess is step 11 at the surface:
// `abcd build <itd-N> --auto-plan` starts the auto-plan lane on a recorded
// draft, refuses a draft short of the substitute at exit 2 naming the row in
// --json and writing nothing, and goes with no --release.
func TestBuildAutoPlanIsWiredAndRefusesAnythingLess(t *testing.T) {
	repo := buildRepo(t)
	const draft = ".abcd/development/intents/drafts/itd-11-beta.md"
	body := "---\nid: itd-11\nslug: beta\nspec_id: null\nkind: standalone\norigin: researcher-authored\nproduction_mode: hand-written\n---\n# beta\n\n" +
		"## Scope Conditions\n\nNone stated.\n\n## Acceptance Criteria\n\n- Given x, when y, then z.\n\n" +
		"## Open Questions\n\n- Which runner?\n" + cliGroundsSection
	repo.Write(draft, body)
	repo.Commit("draft")

	ref := refusalDocs(t, 2, "build", "itd-11", "--auto-plan", "--json")
	if ref["stage"] != "check" || ref["check"] != "open_questions" || !strings.Contains(ref["reason"].(string), "Which runner?") {
		t.Fatalf("refusal = %v", ref)
	}
	runDirAbsent(t, repo.Root())

	repo.Write(draft, strings.Replace(body, "- Which runner?\n", "_None open._\n", 1))
	repo.Commit("settle")
	out := mustImplement(t, "build", "itd-11", "--auto-plan", "--json")
	var res struct {
		Lane struct {
			Stage    string `json:"stage"`
			AutoPlan *struct {
				Draft string `json:"draft"`
			} `json:"auto_plan"`
		} `json:"lane"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if res.Lane.Stage != "plan-review" || res.Lane.AutoPlan == nil || res.Lane.AutoPlan.Draft != draft {
		t.Fatalf("the auto-plan lane opens at its plan review: %s", out)
	}
	step := mustImplement(t, "implement", "step", "--json")
	if !strings.Contains(step, "ruthless-reviewer") || !strings.Contains(step, "plan-review") {
		t.Fatalf("the step hands the draft to a plan reviewer: %s", step)
	}
	if _, err := os.Stat(filepath.Join(repo.Root(), filepath.FromSlash(draft))); err != nil {
		t.Fatalf("nothing is planned before the reviews: %v", err)
	}

	if text := mustImplement(t, "implement", "status"); !strings.Contains(text, "auto-plan of "+draft+": plan reviews design pending, record pending") {
		t.Fatalf("status renders the auto-plan's reviews:\n%s", text)
	}
	code, _, errOut := implementCLI(t, "build", "itd-10", "--release", "--auto-plan")
	if code != 2 || !strings.Contains(errOut, "--auto-plan") {
		t.Fatalf("--release takes no --auto-plan: exit %d\n%s", code, errOut)
	}
}

// TestAnAutoPlanStepRefusesASharedMainCheckoutBeforeItPlans: the step that
// runs an auto-plan lane's plan keeps the record writers' rule, judged under
// the run's lock just before the plan: in a main checkout two live sessions
// started in it backs off (exit 3) and plans nothing, the steps before it run,
// and once the checkout is one session's the same step plans.
func TestAnAutoPlanStepRefusesASharedMainCheckoutBeforeItPlans(t *testing.T) {
	repo := buildRepo(t)
	const draft = ".abcd/development/intents/drafts/itd-11-beta.md"
	repo.Write(draft, "---\nid: itd-11\nslug: beta\nspec_id: null\nkind: standalone\norigin: researcher-authored\nproduction_mode: hand-written\n---\n# beta\n\n"+
		"## Scope Conditions\n\nNone stated.\n\n## Acceptance Criteria\n\n- Given x, when y, then z.\n\n"+
		"## Open Questions\n\n_None open._\n"+cliGroundsSection)
	repo.Commit("draft")
	mustImplement(t, "build", "itd-11", "--auto-plan", "--json")
	for range 2 {
		var res struct {
			Awaiting *struct {
				Receipt string `json:"receipt"`
			} `json:"awaiting"`
		}
		out := mustImplement(t, "implement", "step", "--json")
		if err := json.Unmarshal([]byte(out), &res); err != nil || res.Awaiting == nil {
			t.Fatalf("a plan review is handed out: %v %s", err, out)
		}
		p := filepath.Join(repo.Root(), filepath.FromSlash(res.Awaiting.Receipt))
		if err := os.WriteFile(p, []byte("## Analysis\n\nRead it.\n\n### Verdict\n\nSHIP\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		mustImplement(t, "implement", "receipt", p)
	}

	f := newNameFixture(t)
	f.procs = []machineload.Proc{
		{PID: 7001, PPID: 1, Name: "claude", Age: time.Hour},
		{PID: 7002, PPID: 1, Name: "claude", Age: time.Hour},
	}
	started := time.Now().Add(-time.Hour).Unix()
	tab, _ := f.machine().ReadTable()
	for id, pid := range map[string]int{"s-one": 7001, "s-two": 7002} {
		if err := occupancy.Note(f.home, id, repo.Root(), sessionname.Host{PID: pid, StartUnix: started}, tab); err != nil {
			t.Fatal(err)
		}
	}
	// The plan review's close and the plan's first check write no record,
	// so neither is held to the rule.
	mustImplement(t, "implement", "step")
	mustImplement(t, "implement", "step")
	code, _, errOut := implementCLI(t, "implement", "step")
	if code != 3 || !strings.Contains(errOut, "live sessions work in this main checkout") {
		t.Fatalf("the plan step backs off in a shared main checkout: exit %d\n%s", code, errOut)
	}
	if _, err := os.Stat(filepath.Join(repo.Root(), filepath.FromSlash(draft))); err != nil {
		t.Fatalf("the refused step plans nothing: %v", err)
	}
	sessionNameMachine = func() (sessionname.Machine, bool) { return sessionname.Machine{}, false }
	if out := mustImplement(t, "implement", "step"); !strings.Contains(out, "is planned") {
		t.Fatalf("the same step plans once the checkout is not shared: %s", out)
	}
	if _, err := os.Stat(filepath.Join(repo.Root(), ".abcd", "development", "intents", "planned", "itd-11-beta.md")); err != nil {
		t.Fatalf("the draft is planned: %v", err)
	}
}
