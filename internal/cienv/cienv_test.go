package cienv

import (
	"strings"
	"testing"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

// TestRunnerNamesTheVariable: GITHUB_ACTIONS=true or a CI value other than
// empty, "false" or "0" is a runner, and the reason names the variable.
func TestRunnerNamesTheVariable(t *testing.T) {
	for _, tc := range []struct {
		env  map[string]string
		want string
	}{
		{map[string]string{"GITHUB_ACTIONS": "true"}, "GITHUB_ACTIONS=true"},
		{map[string]string{"CI": "true"}, "CI=true"},
		{map[string]string{"CI": "1"}, "CI=1"},
		{map[string]string{"CI": "\x1b[31m" + strings.Repeat("x", 40)}, "CI=?[31m" + strings.Repeat("x", 27)},
	} {
		got, ok := Runner(env(tc.env))
		if !ok || got != tc.want {
			t.Errorf("%v: Runner = %q, %v; want %q, true", tc.env, got, ok, tc.want)
		}
	}
}

// TestRunnerFalseValuesAreNotCI: an empty, "false" or "0" CI is not a runner,
// and GITHUB_ACTIONS counts only as "true".
func TestRunnerFalseValuesAreNotCI(t *testing.T) {
	for _, m := range []map[string]string{{}, {"CI": ""}, {"CI": "false"}, {"CI": "0"}, {"GITHUB_ACTIONS": "false"}} {
		if got, ok := Runner(env(m)); ok || got != "" {
			t.Errorf("%v: Runner = %q, %v; want not a runner", m, got, ok)
		}
	}
}
