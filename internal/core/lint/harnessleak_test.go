package lint

import (
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/testsecret"
)

// The session-identifier fixture is GENERATED AT RUNTIME, seeded per test, for
// the reason the scanner's own fixtures are (internal/adapter/scanner/
// harnessleak_test.go, and the secret-shaped-fixtures-at-runtime principle):
// this repository scans full history and main cannot be force-pushed, so an
// identifier written down as a literal is in the history for good. Nothing here
// is split, waived, or hidden behind a reserved documentation host — the
// generated value is not real, so it needs no escape and the rule under test can
// read this file and find nothing.
func synthSessionURL(t *testing.T, seed uint64) string {
	t.Helper()
	return "https://agent-host.dev/code/session_" + testsecret.Synthetic(seed, 22)
}

// harnessFooter is the banned attribution shape, spelled out. It needs no
// evasion either: a Go source line puts prose before the match, and the pattern
// requires a footer to occupy its own line.
const harnessFooter = "Generated with [Some Tool](https://tool.dev)"

func harnessLeakCfg() Config {
	return Config{
		Roots: []string{"docs"},
		Rules: map[string]RuleConfig{
			ruleHarnessLeak: {Enabled: true, Severity: severityBlocker},
		},
	}
}

// TestHarnessLeakInLintedProse is the "any committed or posted text, not only a
// freshly created pull-request body" half of the criterion: the same class the
// scanner defines reaches the documents the lint walks.
func TestHarnessLeakInLintedProse(t *testing.T) {
	root := t.TempDir()
	sessionURL := synthSessionURL(t, 17)
	writeFile(t, root, filepath.Join("docs", "release.md"),
		"# Release\n\nShipped.\n\n"+harnessFooter+"\n")
	writeFile(t, root, filepath.Join("docs", "run.md"),
		"# Run\n\nRecorded at "+sessionURL+"\n")

	fs, err := Lint(harnessLeakCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleHarnessLeak); n != 2 {
		t.Fatalf("expected both leak shapes to be flagged, got %d: %+v", n, fs)
	}
	if !hasFinding(fs, filepath.Join("docs", "release.md"), ruleHarnessLeak, 5) {
		t.Errorf("expected the footer finding on its own line; got %+v", fs)
	}
	if !messageContains(fs, "Assisted-by") {
		t.Errorf("expected the message to name the sanctioned alternative; got %+v", fs)
	}
}

// TestHarnessLeakSparesFencedExamples: a fenced example is quoted material, and
// the whole point of a fence in this corpus is to show a literal. A gate that
// cannot be written about is one people route around.
func TestHarnessLeakSparesFencedExamples(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, filepath.Join("docs", "policy.md"),
		"# Policy\n\nThe banned shape:\n\n```\n"+harnessFooter+"\n```\n\nUse the trailer instead.\n")

	fs, err := Lint(harnessLeakCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleHarnessLeak); n != 0 {
		t.Fatalf("expected a fenced example to be spared, got %d: %+v", n, fs)
	}
}

// TestHarnessLeakHonoursTheWaiver: the line-scoped escape the privacy rule
// teaches works here too, so one deliberately illustrative line does not force a
// whole page into a fence.
func TestHarnessLeakHonoursTheWaiver(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, filepath.Join("docs", "policy.md"),
		"# Policy\n\n"+harnessFooter+" <!-- abcd-lint:allow -->\n")

	fs, err := Lint(harnessLeakCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleHarnessLeak); n != 0 {
		t.Fatalf("expected the waiver to be honoured, got %d: %+v", n, fs)
	}
}

// TestHarnessLeakDisabledByDefault: the rule is opt-in like every other, so a
// config that does not name it changes nothing.
func TestHarnessLeakDisabledByDefault(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, filepath.Join("docs", "release.md"), "# Release\n\n"+harnessFooter+"\n")

	fs, err := Lint(Config{Roots: []string{"docs"}}, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleHarnessLeak); n != 0 {
		t.Fatalf("expected an unconfigured rule to be inert, got %d: %+v", n, fs)
	}
}

// TestHarnessLeakSecondMatchOnALine: a benign leftmost candidate must not disarm
// the pattern for the rest of the line. Stopping at the first skipped match made
// this blocker strictly weaker than the scanner it shares a definition with —
// the exact drift the class exists to prevent.
func TestHarnessLeakSecondMatchOnALine(t *testing.T) {
	root := t.TempDir()
	sessionURL := synthSessionURL(t, 23)
	writeFile(t, root, filepath.Join("docs", "run.md"),
		"# Run\n\nBackground https://agent-host.dev/blog/using-agent-session-management-and-1m and the run at "+sessionURL+"\n")

	fs, err := Lint(harnessLeakCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleHarnessLeak); n != 1 {
		t.Fatalf("a skipped leftmost candidate hid a real session URL; got %d: %+v", n, fs)
	}
}

// The issue ledger is committed free text a verb writes from operator input,
// and it sits outside the record's Roots, so a harness_leak rooted at the
// durable record alone never read it (iss-2608301306580014). The rule's own
// extra_roots arm it over the ledger for the leak class alone, and a tree the
// Roots walk already read is not read twice.
func TestHarnessLeakReadsItsExtraRoots(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, filepath.Join("docs", "run.md"), "# Run\n\nRecorded at "+synthSessionURL(t, 31)+"\n")
	writeFile(t, root, filepath.Join("work", "issues", "resolved", "iss-1-a.md"),
		"---\nid: \"iss-1\"\nresolution: \"fixed in the run at "+synthSessionURL(t, 37)+"\"\n---\n\nBody.\n")
	writeFile(t, root, filepath.Join("work", "reviews", "r.md"), "# Review\n\nAt "+synthSessionURL(t, 41)+"\n")

	cfg := harnessLeakCfg()
	rc := cfg.Rules[ruleHarnessLeak]
	rc.ExtraRoots = []string{"work", "docs"}
	rc.Exempt = []string{"work/reviews/*"}
	cfg.Rules[ruleHarnessLeak] = rc
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasFinding(fs, filepath.Join("work", "issues", "resolved", "iss-1-a.md"), ruleHarnessLeak, 3) {
		t.Errorf("a session URL in a ledger record is not flagged: %+v", fs)
	}
	if n := countRule(fs, ruleHarnessLeak); n != 2 {
		t.Fatalf("want one finding in the ledger and one in docs, the exempt review and the doubly-declared docs "+
			"tree drawing no second one; got %d: %+v", n, fs)
	}

	// Without the extra root the ledger is outside the rule, as it was.
	fs, err = Lint(harnessLeakCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleHarnessLeak); n != 1 {
		t.Fatalf("want the docs finding alone without extra_roots; got %d: %+v", n, fs)
	}
}

// The repository's own record-lint configuration arms harness_leak over the
// working tier, where the issue ledger lives (iss-2608301306580014).
func TestRecordLintArmsHarnessLeakOverTheLedger(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("..", "..", "..", ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatal(err)
	}
	rc, ok := cfg.Rules[ruleHarnessLeak]
	if !ok || !rc.Enabled {
		t.Fatal("record-lint.json must enable harness_leak")
	}
	for _, r := range rc.ExtraRoots {
		if r == ".abcd/work" {
			return
		}
	}
	t.Fatalf("record-lint.json harness_leak extra_roots = %v, want .abcd/work, the tree the issue ledger lives in", rc.ExtraRoots)
}
