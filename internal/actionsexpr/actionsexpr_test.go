package actionsexpr

import (
	"math"
	"testing"
)

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

// TestLooseEqualFollowsGitHubsCoercionTable holds `==` to GitHub's loose
// equality as the Actions runner evaluates it: operands of different types are
// both coerced to a number (null 0, true 1, false 0, a string parsed the way the
// runner's ExpressionUtility.ParseNumber parses it, and anything else NaN), NaN
// equals nothing, and strings compare ignoring case (iss-2609251616248870).
func TestLooseEqualFollowsGitHubsCoercionTable(t *testing.T) {
	ctx := map[string]any{
		"success()": true,
		// The expression grammar has no unary minus and no Infinity literal,
		// so the numbers a literal cannot spell come from the context.
		"n.minus_one":  -1.0,
		"n.minus_half": -0.5,
		"n.minus_16":   -16.0,
		"n.two_pow_32": 4294967296.0,
		"n.inf":        math.Inf(1),
		"n.minus_inf":  math.Inf(-1),
	}
	for _, tc := range []struct {
		expr string
		want bool
	}{
		{"true == 'true'", false}, // 1 vs NaN
		{"false == 'false'", false},
		{"true == '1'", true},
		{"false == '0'", true},
		{"false == ''", true},
		{"true == 1", true},
		{"false == 0", true},
		{"true == 2", false},
		{"null == ''", true},
		{"null == 0", true},
		{"null == false", true},
		{"null == '0'", true},
		{"null == 'a'", false},
		{"null == null", true},
		{"1 == '1'", true},
		{"1 == '1.0'", true},
		{"1 == '1e0'", true},
		{"0 == ''", true},
		{"1 == 'abc'", false},
		{"1 == '+1'", true}, // the runner allows a leading sign
		// The runner's ParseNumber (actions/runner, ExpressionUtility.cs)
		// trims whitespace first; a string of only whitespace is 0.
		{"1 == ' 1 '", true},
		{"1 == '\t1\n'", true},
		{"0 == '   '", true},
		// A decimal point may lead or trail its digits, but not stand alone.
		{"0.5 == '.5'", true},
		{"5 == '5.'", true},
		{"n.minus_half == '-.5'", true},
		{"50 == '5.e1'", true},
		{"0 == '.'", false},
		{"0 == '+'", false},
		{"0 == '-'", false},
		{"1 == '1e'", false},
		{"10 == '1,0'", false},
		{"1000 == '1_000'", false},
		// .NET's double parser accepts trailing NULs after a number.
		{"1 == '1\x00\x00'", true},
		{"1 == '1 \x00'", false},
		// 0x hex and 0o octal, lower-case prefix only, parsed as a 32-bit
		// two's-complement integer; wider values and a sign are NaN.
		{"16 == '0x10'", true},
		{"255 == '0xfF'", true},
		{"1 == '0x000000001'", true},
		{"n.minus_one == '0xffffffff'", true},
		{"n.two_pow_32 == '0x100000000'", false},
		{"16 == '0X10'", false},
		{"n.minus_16 == '-0x10'", false},
		{"0 == '0x'", false},
		{"0 == '0xg'", false},
		{"8 == '0o10'", true},
		{"n.minus_one == '0o37777777777'", true},
		{"n.two_pow_32 == '0o40000000000'", false},
		{"8 == '0O10'", false},
		{"0 == '0o8'", false},
		// Infinity in either sign, and the out-of-range decimal that is one.
		{"n.inf == 'Infinity'", true},
		{"n.inf == 'infinity'", true},
		{"n.inf == '+Infinity'", true},
		{"n.minus_inf == '-Infinity'", true},
		{"n.minus_inf == ' -INFINITY '", true},
		{"n.inf == '1e999'", true},
		{"n.inf == 'Inf'", false},
		{"0 == 'NaN'", false},
		{"'abc' == 'ABC'", true},
		{"'abc' != 'abd'", true},
		{"'1' == '1.0'", false}, // same type: compared as strings
		{"true == true", true},
		{"1 == 1", true},
	} {
		got, err := EvalIf(tc.expr, ctx)
		if err != nil {
			t.Errorf("EvalIf(%q): %v", tc.expr, err)
			continue
		}
		if got != tc.want {
			t.Errorf("EvalIf(%q) = %v, want %v", tc.expr, got, tc.want)
		}
	}
}
