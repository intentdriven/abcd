package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	put(filepath.Join(r.Home, ".abcd", "config.json"), machine)
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

// TestRoleRoutedByRepoToMachineRunner: the repository names the runner for a
// role; the machine enables the runner and its model route.
func TestRoleRoutedByRepoToMachineRunner(t *testing.T) {
	c := mustLoad(t,
		`{`+localProvider+`,"runner":{"fallback_host":"claude","claude":{},"opencode":{"model":"local/qwen3-coder"}}}`,
		`{"roles":{"ruthless-reviewer":{"runner":"opencode"}}}`)
	r := c.RouteFor("ruthless-reviewer")
	if r.Runner != OpenCode || r.Layer != layered.Repo || r.Origin != ".abcd/config.json" {
		t.Fatalf("route = %+v", r)
	}
	rc, ok := c.Runner(OpenCode)
	if !ok || rc.Model != "local/qwen3-coder" {
		t.Fatalf("runner = %+v, %v", rc, ok)
	}
	if c.FallbackHost() != Claude {
		t.Fatalf("fallback host = %q", c.FallbackHost())
	}
}

// TestModelOffTheAllowlistIsRefused is criterion 5: a runner whose model
// route its provider does not list is refused when the configuration is read,
// before any runner can be launched, naming the list.
func TestModelOffTheAllowlistIsRefused(t *testing.T) {
	for name, model := range map[string]string{
		"not listed":           "local/llama-3",
		"provider unknown":     "elsewhere/qwen3-coder",
		"not provider/model":   "qwen3-coder",
		"denied vendor prefix": "local/anthropic/claude-opus",
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

// TestRunnerBlocksAreTheMachines: a repository cannot enable a runner or set
// its model: which harness abcd launches, and on whose credential, is the
// person's own machine's to say.
func TestRunnerBlocksAreTheMachines(t *testing.T) {
	_, err := Load(roots(t, "", `{"runner":{"claude":{}}}`))
	if err == nil || !strings.Contains(err.Error(), "~/.abcd/config.json") {
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
	c := mustLoad(t, `{"runner":{"claude":{}}}`,
		`{"roles":{"not-an-agent":{"runner":"claude"},"implementer":{"runner":"claude"}}}`)
	if len(c.Diagnostics) != 1 || !strings.Contains(c.Diagnostics[0], "not-an-agent") {
		t.Fatalf("diagnostics = %q", c.Diagnostics)
	}
	if r := c.RouteFor("implementer"); r.Runner != Claude {
		t.Fatalf("implementer route = %+v", r)
	}
}
