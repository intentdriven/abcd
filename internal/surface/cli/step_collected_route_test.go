package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// TestARouteNoRunnerReportedIsNotShownAsARun: a receipt a step collected names
// the runner asked for and no runner that ran it, and the text says that rather
// than naming a blank runner.
func TestARouteNoRunnerReportedIsNotShownAsARun(t *testing.T) {
	r := &runner.RouteRecord{Asked: runner.Claude}
	if got := routeSuffix(r); strings.Contains(got, "ran on") || !strings.Contains(got, "routed to the claude runner") {
		t.Fatalf("routeSuffix = %q", got)
	}
	var b bytes.Buffer
	renderStepResult(&b, "step", loop.StepResult{RunID: "run-2611040000000001", Lane: "lane-1", Route: r})
	if got := b.String(); strings.Contains(got, "ran on") || !strings.Contains(got, "routed to the claude runner") {
		t.Fatalf("the step's text = %q", got)
	}
}

// TestAPausedRoleSaysWhatItsRunnerDid: only a runner that is absent is known not
// to have started the role.
func TestAPausedRoleSaysWhatItsRunnerDid(t *testing.T) {
	for reason, want := range map[runner.Reason]string{
		runner.ReasonInvalid: "whose answer was refused",
		runner.ReasonFailed:  "which did not bring it back",
		runner.ReasonAbsent:  "which did not start it",
	} {
		var b bytes.Buffer
		renderStepResult(&b, "step", loop.StepResult{RunID: "run-2611040000000001", Lane: "lane-1",
			Paused: &loop.Await{Role: "implementer", Paused: &loop.RunnerPause{Runner: runner.Claude, Reason: reason, Detail: "d"}}})
		if got := b.String(); !strings.Contains(got, want) {
			t.Fatalf("%s: the step's text = %q; want %q", reason, got, want)
		}
	}
}
