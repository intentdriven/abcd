package runner

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
)

// twoModels is a provider block listing two models, so an override can pick
// the one the machine's runner model is not.
const twoModels = `"oracle":{"api":{"local":{"base_url":"http://localhost:11434/v1","models":["qwen3-coder","devstral"]}}}`

const overrideMachine = `{` + twoModels + `,"runner":{"claude":{},"opencode":{"model":"local/qwen3-coder"}}}`

// flagRoutes parses texts as the build loop's step does.
func flagRoutes(t *testing.T, texts ...string) []oracle.FlagRoute {
	t.Helper()
	routes, err := oracle.ParseRunnerRoutes(texts, []string{roleImplementer, "ruthless-reviewer", "security-reviewer"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return routes
}

// TestOverrideRoutesARoleToARunnerWithItsModel is iss-2610161801358637: one
// invocation's --route hands a role to a runner with a model of its own, from
// the flag layer, while every other role and the runner's machine model stay
// as the machine file says.
func TestOverrideRoutesARoleToARunnerWithItsModel(t *testing.T) {
	c := mustLoad(t, overrideMachine, "")
	text := "ruthless-reviewer=runner:opencode@local/devstral"
	if err := c.Override(flagRoutes(t, text)); err != nil {
		t.Fatal(err)
	}
	r := c.RouteFor("ruthless-reviewer")
	if r.Runner != OpenCode || r.Layer != layered.Flag || r.Origin != text || r.Override() != text {
		t.Fatalf("route = %+v", r)
	}
	if got := c.RouteFor("security-reviewer"); got.Runner != Host || got.Override() != "" {
		t.Fatalf("an unnamed role moved: %+v", got)
	}
	if rc, _ := c.Runner(OpenCode); rc.Model != "local/qwen3-coder" {
		t.Fatalf("the machine's model changed: %+v", rc)
	}
}

// TestOverrideRefusals: a runner this build does not ship, one the machine has
// not enabled, and a model its provider does not list are each refused before
// anything runs, naming the fix.
func TestOverrideRefusals(t *testing.T) {
	cases := []struct {
		name, machine, route string
		want                 []string
	}{
		{"unknown runner", overrideMachine, "implementer=runner:aider",
			[]string{"a runner (a 5-byte value, not quoted) this build does not ship", "claude, opencode"}},
		{"runner not enabled", `{` + twoModels + `,"runner":{"claude":{}}}`, "implementer=runner:opencode@local/devstral",
			[]string{"not enabled", "runner.opencode", abcdhome.Display("config.json"), "claude"}},
		{"model off the allowlist", overrideMachine, "implementer=runner:opencode@local/llama-9000",
			[]string{"a model (a 16-byte value, not quoted)", "local's list", "qwen3-coder, devstral", abcdhome.Display("config.json")}},
		{"provider not configured", overrideMachine, "implementer=runner:opencode@openrouter/qwen/qwen3-coder",
			[]string{"provider (a 10-byte value, not quoted) is not configured", abcdhome.Display("config.json")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := mustLoad(t, tc.machine, "")
			err := c.Override(flagRoutes(t, tc.route))
			if err == nil {
				t.Fatalf("--route %s admitted", tc.route)
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not name %q", err, w)
				}
			}
			if r := c.RouteFor("implementer"); r.Runner != Host {
				t.Fatalf("a refused override still routed the role: %+v", r)
			}
		})
	}
}

// TestOverrideRunsTheNamedModelAndRecordsTheRoute: the dispatch launches the
// runner with the model the --route names, the role's receipt records the
// --route verbatim, and a role the flag does not name runs the machine model.
func TestOverrideRunsTheNamedModelAndRecordsTheRoute(t *testing.T) {
	f := newFake(t, "ok", Claude, OpenCode)
	c := mustLoad(t, routed(overrideMachine), "")
	text := "implementer=runner:opencode@local/devstral"
	if err := c.Override(flagRoutes(t, text)); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, c, true)
	out, err := h.d.Dispatch(context.Background(), f.request(roleImplementer))
	if err != nil {
		t.Fatal(err)
	}
	if out.Handoff || out.Answer == nil {
		t.Fatalf("outcome = %+v", out)
	}
	if !slices.Contains(f.argv(t, OpenCode), "--model=local/devstral") {
		t.Fatalf("opencode argv = %q, want the --route's model", f.argv(t, OpenCode))
	}
	// opencode's run mode names no served model, so the asked side carries
	// the --route's model and the served one stays empty (itd-166).
	if r := out.Receipt.Route; r.Asked != OpenCode || r.Ran != OpenCode || r.Model != "" || r.ModelAsked != "local/devstral" || r.Override != text {
		t.Fatalf("route = %+v, want opencode asked for local/devstral under %s", r, text)
	}

	// The machine-routed reviewer, which the flag does not name, keeps the
	// machine's model and records no override.
	out, err = h.d.Dispatch(context.Background(), f.request("ruthless-reviewer"))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(f.argv(t, OpenCode), "--model=local/qwen3-coder") {
		t.Fatalf("opencode argv = %q, want the machine's model", f.argv(t, OpenCode))
	}
	if out.Receipt.Route.Override != "" || out.Receipt.Route.ModelAsked != "local/qwen3-coder" {
		t.Fatalf("route = %+v", out.Receipt.Route)
	}
}

// TestOverrideWithoutAModelKeepsTheMachineModel: a runner route with no @
// hands the role to the runner on the model the machine file names.
func TestOverrideWithoutAModelKeepsTheMachineModel(t *testing.T) {
	f := newFake(t, "ok", Claude, OpenCode)
	c := mustLoad(t, overrideMachine, "")
	if err := c.Override(flagRoutes(t, "security-reviewer=runner:opencode")); err != nil {
		t.Fatal(err)
	}
	out, err := newHarness(t, c, true).d.Dispatch(context.Background(), f.request("security-reviewer"))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(f.argv(t, OpenCode), "--model=local/qwen3-coder") || out.Receipt.Route.Override != "security-reviewer=runner:opencode" {
		t.Fatalf("argv %q, route %+v", f.argv(t, OpenCode), out.Receipt.Route)
	}
}

// TestOverrideDropsTheDiagnosticsOfTheRolesItReplaces: a setup diagnostic
// about a role's configured route (the repository keeping it on the host over
// the person's runner route) is not said once a --route replaces that route,
// since it would contradict what then runs; another role's stays.
func TestOverrideDropsTheDiagnosticsOfTheRolesItReplaces(t *testing.T) {
	machine := `{` + twoModels + `,"roles":{"implementer":{"runner":"opencode"},"security-reviewer":{"runner":"opencode"}},` +
		`"runner":{"opencode":{"model":"local/qwen3-coder"}}}`
	repo := `{"roles":{"implementer":{"runner":"host"},"security-reviewer":{"runner":"host"}}}`
	c := mustLoad(t, machine, repo)
	if len(c.Diagnostics) != 2 {
		t.Fatalf("diagnostics = %q, want one per displaced route", c.Diagnostics)
	}
	if err := c.Override(flagRoutes(t, "implementer=runner:opencode@local/devstral")); err != nil {
		t.Fatal(err)
	}
	if len(c.Diagnostics) != 1 || !strings.Contains(c.Diagnostics[0], "roles.security-reviewer.runner") {
		t.Fatalf("only the unreplaced role's diagnostic stands: %q", c.Diagnostics)
	}
}
