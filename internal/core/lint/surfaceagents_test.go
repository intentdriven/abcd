package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The register sweep reaches agents/ as well as commands/ and skills/
// (iss-2610110456358519): a harness registers every markdown file directly under
// agents/ as an invocable agent, so a page added without a register row, or a
// row left behind after its page was deleted, is a surface the register
// misstates.
func TestSurfaceCoverageSweepsAgentPages(t *testing.T) {
	const reg = "rec/registry.md"
	register := func(agentRows ...string) string {
		s := "# Surfaces\n\n" +
			"| # | Command | Status | File |\n" +
			"|---|---|---|---|\n" +
			"| 1 | `/abcd:ahoy` | shipped | [`a.md`](a.md) |\n\n" +
			"Example of the agents table, which the sweep must not read:\n\n" +
			"```\n| Agent | Serves | Purpose |\n|---|---|---|\n| `example` | — | Fenced |\n```\n\n" +
			"| Agent | Serves | Purpose |\n" +
			"|---|---|---|\n"
		for _, r := range agentRows {
			s += r + "\n"
		}
		return s
	}
	setup := func(t *testing.T, agentRows ...string) string {
		root := t.TempDir()
		writeFile(t, root, "commands/ahoy.md", "# ahoy\n")
		writeFile(t, root, "agents/reviewer.md", "# reviewer\n")
		writeFile(t, root, "agents/drafter.md", "# drafter\n")
		// A subdirectory is an agent's fixtures, not an agent page.
		writeFile(t, root, "agents/reviewer/fixtures/canary.md", "# canary\n")
		writeFile(t, root, reg, register(agentRows...))
		return root
	}
	run := func(t *testing.T, root string) []Finding {
		t.Helper()
		fs, err := Lint(Config{Rules: map[string]RuleConfig{
			"surface_coverage": {
				Enabled: true, Severity: "blocker",
				CommandsDir: "commands", AgentsDir: "agents",
				Registry: filepath.FromSlash(reg),
			},
		}}, root)
		if err != nil {
			t.Fatal(err)
		}
		return fs
	}
	rowReviewer := "| [`reviewer`](../agents/reviewer.md) | build | Reviews a diff |"
	rowDrafter := "| [`drafter`](../agents/drafter.md) | — | Drafts a question |"

	t.Run("every page has a row and every row a page", func(t *testing.T) {
		fs := run(t, setup(t, rowReviewer, rowDrafter))
		if n := countRule(fs, "surface_coverage"); n != 0 {
			t.Fatalf("a register naming every agent page should be clean, got %d findings: %+v", n, fs)
		}
	})

	t.Run("an agent page added without a row fails", func(t *testing.T) {
		root := setup(t, rowReviewer, rowDrafter)
		writeFile(t, root, "agents/newcomer.md", "# newcomer\n")
		fs := run(t, root)
		if !hasFinding(fs, filepath.Join("agents", "newcomer.md"), "surface_coverage", 0) {
			t.Fatalf("an agent page with no register row raised no finding: %+v", fs)
		}
		if n := countRule(fs, "surface_coverage"); n != 1 {
			t.Fatalf("expected exactly 1 finding, got %d: %+v", n, fs)
		}
		for _, f := range fs {
			if !strings.HasPrefix(f.Message, SurfaceCoverageLabel) || !strings.Contains(f.Message, "newcomer") {
				t.Errorf("finding %q must carry the row-level label and name the agent", f.Message)
			}
		}
	})

	t.Run("an agent page deleted with its row left fails", func(t *testing.T) {
		root := setup(t, rowReviewer, rowDrafter)
		if err := os.Remove(filepath.Join(root, "agents", "drafter.md")); err != nil {
			t.Fatal(err)
		}
		fs := run(t, root)
		// The drafter row is the agents table's second data row: line 18.
		if !hasFinding(fs, filepath.FromSlash(reg), "surface_coverage", 18) {
			t.Fatalf("an agent row whose page is gone raised no finding on its line: %+v", fs)
		}
		if n := countRule(fs, "surface_coverage"); n != 1 {
			t.Fatalf("expected exactly 1 finding, got %d: %+v", n, fs)
		}
	})

	t.Run("a register with no agents table fails for every page", func(t *testing.T) {
		root := setup(t)
		writeFile(t, root, reg, "# Surfaces\n\n| # | Command | Status | File |\n|---|---|---|---|\n| 1 | `/abcd:ahoy` | shipped | [`a.md`](a.md) |\n")
		fs := run(t, root)
		for _, name := range []string{"reviewer.md", "drafter.md"} {
			if !hasFinding(fs, filepath.Join("agents", name), "surface_coverage", 0) {
				t.Errorf("agents/%s raised no finding against a register with no agents table: %+v", name, fs)
			}
		}
	})

	t.Run("unarmed without agents_dir", func(t *testing.T) {
		root := setup(t)
		fs, err := Lint(Config{Rules: map[string]RuleConfig{
			"surface_coverage": {Enabled: true, Severity: "blocker", CommandsDir: "commands", Registry: filepath.FromSlash(reg)},
		}}, root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, "surface_coverage"); n != 0 {
			t.Fatalf("with no agents_dir the agent sweep must not run, got %d findings: %+v", n, fs)
		}
	})
}

// The repository's own config arms the agent sweep, and its register names
// every agent page: without agents_dir the sweep above checks nothing here.
func TestShippedConfigArmsTheAgentSweep(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(repoRootFromPackage, ".abcd/record-lint.json"))
	if err != nil {
		t.Fatal(err)
	}
	sc, ok := cfg.Rules["surface_coverage"]
	if !ok || !sc.Enabled || sc.AgentsDir != "agents" {
		t.Fatalf("surface_coverage must be enabled with agents_dir \"agents\" in .abcd/record-lint.json; got enabled=%v agents_dir=%q", sc.Enabled, sc.AgentsDir)
	}
	fs, err := checkSurfaceCoverage(repoRootFromPackage, sc)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if strings.HasPrefix(f.File, "agents"+string(filepath.Separator)) || strings.Contains(f.Message, "agent") {
			t.Errorf("the register and agents/ disagree: %s:%d %s", f.File, f.Line, f.Message)
		}
	}
}
