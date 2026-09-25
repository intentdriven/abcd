package actionsexpr

import "testing"

// TestEvalIfReadsTheStatusFunctionsFromTheContext holds the status-check
// functions to the run state the caller states, and to failing closed when it
// states none: a job condition that reads cancelled() in a context that does
// not say whether the run was cancelled has no answer, and guessing one is how
// a condition that publishes on a red gate would read as safe.
func TestEvalIfReadsTheStatusFunctionsFromTheContext(t *testing.T) {
	run := map[string]any{"success()": false, "failure()": true, "cancelled()": false}
	for _, tc := range []struct {
		cond string
		want bool
	}{
		{"!cancelled()", true},
		{"cancelled()", false},
		{"failure()", true},
		{"success()", false},
		{"always()", true},
		{"${{ !cancelled() && failure() }}", true},
	} {
		got, err := EvalIf(tc.cond, run)
		if err != nil {
			t.Errorf("EvalIf(%q): %v", tc.cond, err)
			continue
		}
		if got != tc.want {
			t.Errorf("EvalIf(%q) = %v, want %v", tc.cond, got, tc.want)
		}
	}
	if _, err := EvalIf("!cancelled()", map[string]any{}); err == nil {
		t.Error("EvalIf read cancelled() from a context that does not state it; it must fail closed")
	}
}

// TestEvalIfAppliesTheImplicitSuccessCheck holds GitHub's rule that a
// condition naming no status function is evaluated as `success() && (...)`:
// that is why a job whose need was skipped or failed is itself skipped even
// when its own condition reads true, and a test that evaluated the condition
// alone would report such a job as running.
func TestEvalIfAppliesTheImplicitSuccessCheck(t *testing.T) {
	red := map[string]any{"success()": false, "failure()": true, "cancelled()": false, "inputs.create_tag": true}
	green := map[string]any{"success()": true, "failure()": false, "cancelled()": false, "inputs.create_tag": true}
	if got, err := EvalIf("inputs.create_tag", red); err != nil || got {
		t.Errorf("EvalIf(inputs.create_tag) after a failed need = %v, %v; want false, the implicit success() skips it", got, err)
	}
	if got, err := EvalIf("inputs.create_tag", green); err != nil || !got {
		t.Errorf("EvalIf(inputs.create_tag) after green needs = %v, %v; want true", got, err)
	}
	// Naming any status function turns the implicit check off.
	if got, err := EvalIf("always() && inputs.create_tag", red); err != nil || !got {
		t.Errorf("EvalIf(always() && inputs.create_tag) after a failed need = %v, %v; want true", got, err)
	}
}

// TestEvalIfRefusesAValueGitHubWouldReadAsAString holds the one spelling of a
// condition that does not mean what it reads: an expression followed by more
// text is a non-empty string, which GitHub treats as true whatever the
// expression says. Evaluating the expression part alone would pass it.
func TestEvalIfRefusesAValueGitHubWouldReadAsAString(t *testing.T) {
	if _, err := EvalIf("${{ cancelled() }} && failure()", map[string]any{"cancelled()": false, "failure()": false}); err == nil {
		t.Error("EvalIf accepted an expression followed by text; GitHub reads that as a non-empty string, always true")
	}
}
