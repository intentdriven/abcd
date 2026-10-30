package oracle

import (
	"strings"
	"testing"
)

// TestRunnerRouteParses is iss-2610161801358637's grammar: the --route a
// runner-dispatching invocation takes names the role, the runner and, after
// @, the <provider>/<model> route it asks for; without @ the runner keeps
// its machine model. A model may carry its own slashes and colons.
func TestRunnerRouteParses(t *testing.T) {
	dispatched := []string{"implementer", "ruthless-reviewer"}
	cases := []struct{ text, role, runner, model string }{
		{"implementer=runner:opencode@openrouter/qwen/qwen3-coder", "implementer", "opencode", "openrouter/qwen/qwen3-coder"},
		{"ruthless-reviewer=runner:opencode@local/qwen2.5-coder:32b", "ruthless-reviewer", "opencode", "local/qwen2.5-coder:32b"},
		{"implementer=runner:claude", "implementer", "claude", ""},
	}
	for _, tc := range cases {
		routes, err := ParseRunnerRoutes([]string{tc.text}, dispatched)
		if err != nil {
			t.Fatalf("--route %s: %v", tc.text, err)
		}
		fr := routes[0]
		if fr.Agent != tc.role || fr.Runner != tc.runner || fr.Model != tc.model || fr.Text != tc.text {
			t.Fatalf("--route %s parsed as %+v", tc.text, fr)
		}
	}
}

// TestRunnerRouteRefusals: a runner route is refused before anything runs
// when it is malformed, names a role the invocation does not dispatch,
// repeats a role, carries settings, or is a tier route where only a runner
// route applies; each refusal names the grammar or what to do instead.
func TestRunnerRouteRefusals(t *testing.T) {
	dispatched := []string{"implementer", "ruthless-reviewer"}
	cases := []struct{ name, route, want string }{
		{"role not dispatched", "scribe=runner:opencode", `this invocation dispatches implementer, ruthless-reviewer, not "scribe"`},
		{"no runner name", "implementer=runner:", RunnerRouteSyntax},
		{"empty model", "implementer=runner:opencode@", "an empty model after @"},
		{"model without provider", "implementer=runner:opencode@qwen3-coder", "<provider>/<model>"},
		{"model with empty name", "implementer=runner:opencode@local/", "<provider>/<model>"},
		{"settings", "implementer=runner:opencode@local/qwen3-coder?seed=1", "takes no settings"},
		{"tier route", "implementer=frontier", "runs its roles on the host or a runner"},
		{"tier route with connection", "implementer=economy@openrouter", "names a tier"},
		{"no route", "implementer", RunnerRouteSyntax},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseRunnerRoutes([]string{tc.route}, dispatched)
			if err == nil {
				t.Fatalf("--route %s admitted", tc.route)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not name %q", err, tc.want)
			}
		})
	}
	t.Run("the same role twice", func(t *testing.T) {
		_, err := ParseRunnerRoutes([]string{"implementer=runner:claude", "implementer=runner:opencode"}, dispatched)
		if err == nil || !strings.Contains(err.Error(), "more than once") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("too long", func(t *testing.T) {
		_, err := ParseRunnerRoutes([]string{"implementer=runner:opencode@local/" + strings.Repeat("m", MaxRouteBytes)}, dispatched)
		if err == nil || !strings.Contains(err.Error(), "at most") {
			t.Fatalf("err = %v", err)
		}
	})
}

// TestTierVerbRefusesARunnerRoute: a delegating verb sends its agent to the
// harness or a provider connection and launches no runner, so a runner route
// there is refused naming where it applies, never read as a tier.
func TestTierVerbRefusesARunnerRoute(t *testing.T) {
	_, err := ParseRoutes([]string{"scribe=runner:opencode@local/qwen3-coder"}, []string{"scribe"}, NoConnections{})
	if err == nil {
		t.Fatal("a runner route was admitted by a verb that launches no runner")
	}
	for _, want := range []string{"names a runner", "abcd implement step"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not name %q", err, want)
		}
	}
}

// TestAMalformedRouteNamesTheGrammarItsCallerTakes: a route that is not a
// runner route on `implement step` (the runner: rung left out) names the
// runner grammar, never a tier it did not ask for; a malformed route on a
// verb that takes tiers names the tier grammar alone, never the runner rung
// that verb refuses.
func TestAMalformedRouteNamesTheGrammarItsCallerTakes(t *testing.T) {
	for _, text := range []string{"implementer=opencode@local/devstral", "implementer=opencode", "implementer"} {
		_, err := ParseRunnerRoutes([]string{text}, []string{"implementer"})
		if err == nil || !strings.Contains(err.Error(), RunnerRouteSyntax) || strings.Contains(err.Error(), "tier") {
			t.Errorf("--route %s on a runner verb: %v", text, err)
		}
	}
	for _, text := range []string{"scribe", "scribe=", "=frontier"} {
		_, err := ParseRoutes([]string{text}, []string{"scribe"}, NoConnections{})
		if err == nil || !strings.Contains(err.Error(), RouteSyntax) || strings.Contains(err.Error(), "runner") {
			t.Errorf("--route %s on a tier verb: %v", text, err)
		}
	}
	_, err := ParseRoutes([]string{"scribe=runner:"}, []string{"scribe"}, NoConnections{})
	if err == nil || !strings.Contains(err.Error(), "names a runner") {
		t.Errorf("a malformed runner rung on a tier verb is refused as a runner route: %v", err)
	}
}
