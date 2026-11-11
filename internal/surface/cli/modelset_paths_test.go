package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
)

// TestADriveRefusesAnOldClassNameWithTheMapping: a drive reads the class map
// first, so an old class name in the machine's file is refused there with the
// whole mapping, as the review stage refuses it, and nothing starts.
func TestADriveRefusesAnOldClassNameWithTheMapping(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	w.machineConfig(t, `{"runner":{"claude":{}},"models":{"classes":{"research":"anthropic/claude-fable-5","middle":"anthropic/claude-opus-5-5"}}}`)
	text := refusedDrive(t, "drain", "--drive", "--yes")
	for _, want := range []string{"models.classes.middle", "`top` is now `research`", "`middle` is `top`", "`below` is `default`",
		"`smallest` is `small`", "nothing started"} {
		if !strings.Contains(text, want) {
			t.Errorf("the refusal %q does not name %q", text, want)
		}
	}
	noDrainBegun(t, w.repo)
	if w.terminalCalls != 0 {
		t.Fatal("nothing runs")
	}
}

// TestADriveWhoseReviewerCannotLaunchNamesItsClassWholly: the refusal of a
// reviewer no route here can launch names its class with the model and the
// effort, as every surface shows a class, never the class name alone.
func TestADriveWhoseReviewerCannotLaunchNamesItsClassWholly(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	w.machineConfig(t, `{"oracle":{"denylist":["anthropic/*"]},"runner":{"claude":{}}}`)
	text := refusedDrive(t, "drain", "--drive", "--yes")
	want := "research (" + loop.BundledClasses[loop.ClassResearch] + ", effort " + loop.BundledEffort + ")"
	if !strings.Contains(text, want) {
		t.Fatalf("the refusal %q does not name %q", text, want)
	}
}

// TestImplementStatusJSONCarriesEffortsAndTheJudge: `abcd implement status
// --json` carries each reviewer's class, model and effort, and the judge with
// its status, as the text does.
func TestImplementStatusJSONCarriesEffortsAndTheJudge(t *testing.T) {
	buildRepo(t)
	mustImplement(t, "build", "itd-10")
	out := mustImplement(t, "implement", "status", "--json")
	var res struct {
		Runs []struct {
			Review *struct {
				Roles []struct {
					Role      string
					Reviewers []struct{ Class, Model, Effort string }
				}
				Judge *struct{ Class, Model, Effort, Status string }
			}
		}
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if len(res.Runs) != 1 || res.Runs[0].Review == nil || len(res.Runs[0].Review.Roles) != 2 {
		t.Fatalf("the status carries the run's review set: %s", out)
	}
	rv := res.Runs[0].Review
	for _, r := range rv.Roles {
		if len(r.Reviewers) != 2 || r.Reviewers[0].Class != loop.ClassResearch || r.Reviewers[1].Class != loop.ClassDefault {
			t.Fatalf("%s runs research and default: %s", r.Role, out)
		}
		for _, c := range r.Reviewers {
			if c.Model != loop.BundledClasses[c.Class] || c.Effort != loop.BundledEffort {
				t.Fatalf("each reviewer carries its model and effort: %s", out)
			}
		}
	}
	if j := rv.Judge; j == nil || j.Class != loop.ClassTop || j.Model != loop.BundledClasses[loop.ClassTop] ||
		j.Effort != loop.BundledEffort || j.Status != loop.JudgeWaiting {
		t.Fatalf("the judge carries its model, effort and status: %s", out)
	}
}
