package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
