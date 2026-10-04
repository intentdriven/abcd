package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
)

// localProvider is a machine provider block that holds no key: the allowlist a
// runner's model route is admitted against.
const localProvider = `"oracle":{"api":{"local":{"base_url":"http://localhost:11434/v1","models":["qwen3-coder"]}}}`

// roots writes the machine and repository config files (either may be "") and
// returns the roots that read them.
func roots(t *testing.T, machine, repo string) layered.Roots {
	t.Helper()
	r := layered.Roots{Repo: t.TempDir(), Home: t.TempDir()}
	put := func(path, body string) {
		if body == "" {
			return
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	put(abcdhome.Path(r.Home, "config.json"), machine)
	put(filepath.Join(r.Repo, ".abcd", "config.json"), repo)
	return r
}

func mustLoad(t *testing.T, machine, repo string) *Config {
	t.Helper()
	c, err := Load(roots(t, machine, repo))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return c
}

// TestUnsetRoleIsHost is criterion 2's configuration half: with nothing
// configured every role routes to the host, from the bundled default.
func TestUnsetRoleIsHost(t *testing.T) {
	c := mustLoad(t, "", "")
	r := c.RouteFor("ruthless-reviewer")
	if r.Runner != Host || r.Layer != layered.Bundled {
		t.Fatalf("route = %+v, want the bundled host", r)
	}
}

// TestRoleRoutedByMachineToItsRunner: the person's own machine names the
// runner for a role, enables the runner and sets its model route.
func TestRoleRoutedByMachineToItsRunner(t *testing.T) {
	c := mustLoad(t,
		`{`+localProvider+`,"roles":{"ruthless-reviewer":{"runner":"opencode"}},`+
			`"runner":{"fallback_host":"claude","claude":{},"opencode":{"model":"local/qwen3-coder"}}}`, "")
	r := c.RouteFor("ruthless-reviewer")
	if r.Runner != OpenCode || r.Layer != layered.Machine || r.Origin != "~/.abcd.noindex/config.json" {
		t.Fatalf("route = %+v", r)
	}
	if len(c.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %q", c.Diagnostics)
	}
	rc, ok := c.Runner(OpenCode)
	if !ok || rc.Model != "local/qwen3-coder" {
		t.Fatalf("runner = %+v, %v", rc, ok)
	}
	if c.FallbackHost() != Claude {
		t.Fatalf("fallback host = %q", c.FallbackHost())
	}
}

// TestRepoRouteToARunnerIsSkipped is rulings RN2 and OC2 (2026-10-02): only a
// personal route may hand a role to a runner, claude or opencode alike. A
// route the repository commits is skipped with a diagnostic naming its file,
// its key and the machine's file where the person may set it, and the role
// runs on the host as if unrouted, even when the machine enables the runner.
func TestRepoRouteToARunnerIsSkipped(t *testing.T) {
	for _, name := range runnerNames {
		c := mustLoad(t, `{"runner":{"claude":{},"opencode":{}}}`,
			`{"roles":{"ruthless-reviewer":{"runner":"`+name+`"}}}`)
		if r := c.RouteFor("ruthless-reviewer"); r.Runner != Host || r.Layer != layered.Bundled {
			t.Errorf("%s: route = %+v, want the bundled host", name, r)
		}
		if len(c.Diagnostics) != 1 {
			t.Fatalf("%s: diagnostics = %q, want one", name, c.Diagnostics)
		}
		for _, want := range []string{".abcd/config.json (repo layer)", "roles.ruthless-reviewer.runner",
			"~/.abcd.noindex/config.json", "skipped", name} {
			if !strings.Contains(c.Diagnostics[0], want) {
				t.Errorf("%s: diagnostic %q lacks %q", name, c.Diagnostics[0], want)
			}
		}
	}
}

// TestMachineRouteAppliesUnderASkippedRepoRoute: the skipped repository route
// falls through to the next layer, so the person's own route for the role
// applies.
func TestMachineRouteAppliesUnderASkippedRepoRoute(t *testing.T) {
	c := mustLoad(t, `{"roles":{"scribe":{"runner":"claude"}},"runner":{"claude":{},"opencode":{}}}`,
		`{"roles":{"scribe":{"runner":"opencode"}}}`)
	if r := c.RouteFor("scribe"); r.Runner != Claude || r.Layer != layered.Machine {
		t.Fatalf("route = %+v, want the machine's claude route", r)
	}
	if len(c.Diagnostics) != 1 || !strings.Contains(c.Diagnostics[0], "roles.scribe.runner") {
		t.Fatalf("diagnostics = %q", c.Diagnostics)
	}
}

// TestRepoRouteToHostStands: a repository may keep a role on the host, which
// spends nothing of the person's, over a machine route to a runner. The route
// stands, but not in silence: a checkout that cancels the person's own choice
// says so, naming the repository's file, the role and the machine route it
// displaced (sec-runnerSeal, 2026-10-02).
func TestRepoRouteToHostStands(t *testing.T) {
	c := mustLoad(t, `{"roles":{"scribe":{"runner":"claude"}},"runner":{"claude":{}}}`,
		`{"roles":{"scribe":{"runner":"host"}}}`)
	if r := c.RouteFor("scribe"); r.Runner != Host || r.Layer != layered.Repo {
		t.Fatalf("route = %+v, want the repository's host route", r)
	}
	if len(c.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %q, want one naming the displaced machine route", c.Diagnostics)
	}
	for _, want := range []string{".abcd/config.json (repo layer)", "roles.scribe.runner", `"host"`,
		"~/.abcd.noindex/config.json (machine layer)", `"claude"`} {
		if !strings.Contains(c.Diagnostics[0], want) {
			t.Errorf("diagnostic %q lacks %q", c.Diagnostics[0], want)
		}
	}
}

// TestRepoRouteToHostDisplacingNothingIsSilent: a repository route to the host
// that displaces no runner route, because the machine sets none or sets the
// host too, is the default restated and says nothing.
func TestRepoRouteToHostDisplacingNothingIsSilent(t *testing.T) {
	for name, machine := range map[string]string{
		"no machine route":      `{"runner":{"claude":{}}}`,
		"machine route to host": `{"roles":{"scribe":{"runner":"host"}},"runner":{"claude":{}}}`,
	} {
		c := mustLoad(t, machine, `{"roles":{"scribe":{"runner":"host"}}}`)
		if r := c.RouteFor("scribe"); r.Runner != Host || r.Layer != layered.Repo {
			t.Errorf("%s: route = %+v, want the repository's host route", name, r)
		}
		if len(c.Diagnostics) != 0 {
			t.Errorf("%s: diagnostics = %q, want none", name, c.Diagnostics)
		}
	}
}

// TestFlagRouteIsPersonal: the flag layer is the person's own invocation, for
// one run, so a route there is personal and may hand a role to a runner. No
// front door sets it for a role today; the rule is held here so a flag added
// later is not silently treated as the repository's.
func TestFlagRouteIsPersonal(t *testing.T) {
	r := roots(t, `{"runner":{"claude":{}}}`, `{"roles":{"scribe":{"runner":"opencode"}}}`)
	s, err := layered.Load(layered.Config, r)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetFlag("roles.scribe.runner", "claude", "--runner scribe=claude"); err != nil {
		t.Fatal(err)
	}
	c := &Config{roles: map[string]Route{}}
	if err := c.readRoles(s); err != nil {
		t.Fatal(err)
	}
	if got := c.RouteFor("scribe"); got.Runner != Claude || got.Layer != layered.Flag {
		t.Fatalf("route = %+v, want the flag's claude route", got)
	}
}

// TestModelOffTheAllowlistIsRefused is criterion 5: a runner whose model
// route its provider does not list is refused when the configuration is read,
// before any runner can be launched, naming the list.
func TestModelOffTheAllowlistIsRefused(t *testing.T) {
	for name, model := range map[string]string{
		"not listed":         "local/llama-3",
		"provider unknown":   "elsewhere/qwen3-coder",
		"not provider/model": "qwen3-coder",
	} {
		_, err := Load(roots(t, `{`+localProvider+`,"runner":{"opencode":{"model":"`+model+`"}}}`, ""))
		if err == nil {
			t.Errorf("%s: %q admitted", name, model)
			continue
		}
		if !strings.Contains(err.Error(), "runner.opencode.model") {
			t.Errorf("%s: refusal does not name the key: %v", name, err)
		}
	}
}

// TestAllowlistAloneDecides follows adr-2609300107513982: abcd bundles no
// vendor denylist, so a vendor-prefixed model a provider lists is admitted,
// and only an oracle.denylist entry the configuration writes refuses it.
func TestAllowlistAloneDecides(t *testing.T) {
	const listed = `"oracle":{"api":{"local":{"base_url":"http://localhost:11434/v1","models":["anthropic/claude-opus"]}}`
	c := mustLoad(t, `{`+listed+`},"runner":{"opencode":{"model":"local/anthropic/claude-opus"}}}`, "")
	if rc, ok := c.Runner(OpenCode); !ok || rc.Model != "local/anthropic/claude-opus" {
		t.Fatalf("runner = %+v, %v; a listed model is admitted by the allowlist alone", rc, ok)
	}
	_, err := Load(roots(t,
		`{`+listed+`,"denylist":["anthropic/*"]},"runner":{"opencode":{"model":"local/anthropic/claude-opus"}}}`, ""))
	if err == nil || !strings.Contains(err.Error(), "oracle.denylist") {
		t.Fatalf("err = %v, want the configured denylist entry to refuse the route", err)
	}
}

// TestRunnerBlocksAreTheMachines: a repository cannot enable a runner or set
// its model: which harness abcd launches, and on whose credential, is the
// person's own machine's to say.
func TestRunnerBlocksAreTheMachines(t *testing.T) {
	_, err := Load(roots(t, "", `{"runner":{"claude":{}}}`))
	if err == nil || !strings.Contains(err.Error(), "~/.abcd.noindex/config.json") {
		t.Fatalf("err = %v, want a refusal naming the machine's file", err)
	}
}

// TestConfigRefusals: every other fault is loud.
func TestConfigRefusals(t *testing.T) {
	for name, tc := range map[string][2]string{
		"unknown runner":           {`{"runner":{"claude":{}}}`, `{"roles":{"scribe":{"runner":"gemini"}}}`},
		"unknown role key":         {"", `{"roles":{"scribe":{"runnr":"host"}}}`},
		"unknown runner key":       {`{"runner":{"claude":{"binary":"/tmp/x"}}}`, ""},
		"fallback not enabled":     {`{"runner":{"fallback_host":"opencode","claude":{}}}`, ""},
		"fallback host is host":    {`{"runner":{"fallback_host":"host","claude":{}}}`, ""},
		"runner not a string":      {"", `{"roles":{"scribe":{"runner":1}}}`},
		"unknown runner namespace": {`{"runner":{"aider":{}}}`, ""},
	} {
		if _, err := Load(roots(t, tc[0], tc[1])); err == nil {
			t.Errorf("%s: loaded", name)
		}
	}
}

// TestRoleOutsideTheRosterIsADiagnostic: a role no agent answers to is named
// and skipped, as the oracle's routes are; the rest apply.
func TestRoleOutsideTheRosterIsADiagnostic(t *testing.T) {
	c := mustLoad(t, `{"roles":{"not-an-agent":{"runner":"claude"},"implementer":{"runner":"claude"}},"runner":{"claude":{}}}`, "")
	if len(c.Diagnostics) != 1 || !strings.Contains(c.Diagnostics[0], "not-an-agent") {
		t.Fatalf("diagnostics = %q", c.Diagnostics)
	}
	if r := c.RouteFor("implementer"); r.Runner != Claude {
		t.Fatalf("implementer route = %+v", r)
	}
}
