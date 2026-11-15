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

// TestGuardPageDocumentsTheQuestionVerb (spc-2610301917245442, step 1): the
// verb is wired on both front doors, so the plugin's guard page carries a
// `question` section naming its JSON, its exit codes and its two input forms,
// and the generated command-line reference documents `abcd guard question`
// with its help line in the house form.
func TestGuardPageDocumentsTheQuestionVerb(t *testing.T) {
	page, err := os.ReadFile(filepath.Join(repoRootFromPackage, "commands", "guard.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(page)
	at := strings.Index(text, "\n## `question`")
	if at < 0 {
		t.Fatal("commands/guard.md has no `question` section")
	}
	section := text[at+1:]
	if end := strings.Index(section, "\n## "); end >= 0 {
		section = section[:end]
	}
	for _, want := range []string{"guard question", "--ask", "abcd/question-check/v1", "`ready`", "`rows`", "`version`", "Exit codes", "`2`"} {
		if !strings.Contains(section, want) {
			t.Errorf("the guard page's `question` section does not name %s", want)
		}
	}
	ref, err := os.ReadFile(filepath.Join(repoRootFromPackage, "docs", "reference", "cli", "commands.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ref), "abcd guard question") {
		t.Error("docs/reference/cli/commands.md does not document abcd guard question")
	}
	if !strings.Contains(string(ref), "Check a draft question as the hook will judge it: Writes nothing; refuses") {
		t.Error("the reference does not carry the verb's help line in the house form")
	}
}

// handWritten is page with every generated block cut out, so a test reads
// only what a person wrote.
func handWritten(page string) string {
	var b strings.Builder
	for {
		at := strings.Index(page, "<!-- generated:")
		if at < 0 {
			b.WriteString(page)
			return b.String()
		}
		b.WriteString(page[:at])
		end := strings.Index(page[at:], "<!-- /generated -->")
		if end < 0 {
			return b.String()
		}
		page = page[at+end+len("<!-- /generated -->"):]
	}
}

// TestWriterPageStatesItsRefusalsAndFallbacks (spc-2610301917245442, step 4;
// approach 9): the hand-written part of the question-writing agent's page
// states its three refusals, each returned to the calling agent only (a
// missing or contradictory part of the brief, the mode in force among them,
// an `unnamed` addressee under `badge: true` included, but not one under
// `badge: false`; the check giving no ready input, on its
// exit 2 or a `refuse-away`; five checks spent with the last findings), and
// the caller's `abcd mode managed` when it abandons a question after one;
// the draft piped as one line of JSON; the `questions` array of `ready` as
// what is returned; the `checked_by` field with its three values;
// the five-check bound; that it never refuses for length, splitting into
// numbered parts with every answer's gain and cost kept; in abcd's own
// source checkout, the verb run from the source tree; that a hand count is
// made at abcd's defaults, reading neither the repository's values nor the
// person's, and the output says so (spc-2610301950037469, approach 10;
// itd-2610301914350541 decision 10); and that an exit 0 is done only with
// `abcds` true, an `admit-unchecked` pass being a header to fix.
func TestWriterPageStatesItsRefusalsAndFallbacks(t *testing.T) {
	page, err := os.ReadFile(filepath.Join(repoRootFromPackage, "agents", "question-drafter.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(strings.Fields(handWritten(string(page))), " ")
	for name, phrases := range map[string][]string{
		"the refusal of a missing or contradictory part": {"a part of the brief is missing or contradicts another"},
		"the refusal when no ready input comes back":     {"cannot give a ready input", "exits 2", "`refuse-away`", "never a reason to count by hand"},
		"unnamed: an unset mode, or no badge":            {"names a person (`product-thinker` or `facilitator`) other than the one you were handed", "the check's `badge` says why", "With `badge: true`", "the caller did not set the mode", "That is refusal cause 1", "naming the person you were handed and the mode in force", "With `badge: false`", "`unnamed` is no disagreement"},
		"the abandoned question's mode":                  {"abandons the question after your refusal", "`abcd mode managed`"},
		"the draft's one line of JSON":                   {"valid JSON on one line", "written as `\\n`", "can never be the delimiter line"},
		"the questions array, not ready":                 {"the `questions` array of the check's `ready` input"},
		"the refusal after five checks":                  {"five checks", "the check's last findings"},
		"refusals go to the caller only":                 {"to the agent that called you", "never to the person"},
		"checked_by and its three values":                {"`checked_by`", "`abcd`", "`hook-only`", "`hand`"},
		"an older binary is hook-only":                   {"has no `guard question`", "`checked_by: hook-only`", "count nothing by hand"},
		"a host with no shell counts by hand":            {"no shell", "`checked_by: hand`"},
		"the five-check bound":                           {"at most five checks to a part"},
		"never refused for length":                       {"never refuse for length", "numbered parts", `"None of these"`, "Q1b", "every answer's gain and cost"},
		"the source tree in abcd's checkout":             {"abcd's own source checkout", "`scripts/abcd.sh guard question`", "`version` it reports"},
		"what it returns":                                {"`questions`", "`rows`", "`version`", "`ready`"},
		"a hand count is at abcd's defaults":             {"A hand count is made at abcd's defaults", "neither the repository's values nor the person's were read", "Say so in a `note` beside `checked_by`"},
		"an unchecked pass is not done":                  {"exits 0 with a `ready` input and `abcds` `true`", "An exit 0 with `hook` `admit-unchecked` is not done", "`abcds` is `false`", "Fix the header to abcd's number"},
	} {
		for _, p := range phrases {
			if !strings.Contains(text, p) {
				t.Errorf("agents/question-drafter.md does not state %s: missing %q", name, p)
			}
		}
	}
	if strings.Contains(text, "`remaining`") {
		t.Error("the writer page still returns `remaining`: a part that does not fit is split into numbered parts, never handed back unasked")
	}
	if strings.Contains(text, "prints `managed`") || strings.Contains(text, "carries abcd's block") {
		t.Error("the writer page still judges an `unnamed` tab by `abcd mode` or abcd's block in a root file; the check's `badge` is the State value the hook decides with")
	}
}
