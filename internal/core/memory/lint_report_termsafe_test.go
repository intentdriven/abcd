package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// lint_report_termsafe_test.go — iss-2609020239068243. Lint builds the
// degraded-scanner MR001 message from openStoreRedactor's error, whose text
// carries a pattern name read from the per-repo .abcd/config/pii.json, and
// renderLintReportMD wrote that message into report.md with no termsafe pass
// while the CLI render sanitises the same finding. A hostile or careless
// pattern name then reached a file the operator opens in a pager.

// hostileRunes is a terminal title-set sequence plus a colour escape and a
// bidi override: every one is masked by termsafe.Sanitize.
const hostileRunes = "\x1b]0;pwned\x07\x1b[31m\u202e"

func mustBeTerminalSafe(t *testing.T, label, text string) {
	t.Helper()
	for _, r := range []string{"\x1b", "\x07", "\u202e"} {
		if strings.Contains(text, r) {
			t.Errorf("%s carries the raw control rune %q:\n%q", label, r, text)
		}
	}
}

func TestLintReportMDSanitisesADegradedScannerPatternName(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(Dir(repo), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(repo, ".abcd", "config")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	// A NEW pattern with no regex degrades the scanner, and the reason names it.
	pii := `{"patterns": {"evil` + strings.NewReplacer("\x1b", `\u001b`, "\x07", `\u0007`, "\u202e", `\u202e`).Replace(hostileRunes) + `": {"regex": ""}}}`
	if err := os.WriteFile(filepath.Join(cfg, "pii.json"), []byte(pii), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow})
	if err != nil {
		t.Fatalf("lint: %v", err)
	}
	var degraded bool
	for _, f := range res.Findings {
		if f.Code == "MR001" && strings.Contains(f.Message, "secret scan unavailable") && strings.Contains(f.Message, "pwned") {
			degraded = true
		}
	}
	if !degraded {
		t.Fatalf("fixture drift: no degraded-scanner MR001 naming the pattern: %+v", res.Findings)
	}
	raw, err := os.ReadFile(filepath.Join(res.ReportDir, "report.md"))
	if err != nil {
		t.Fatal(err)
	}
	mustBeTerminalSafe(t, "report.md", string(raw))
}

// Every free-text field a finding carries into report.md is sanitised: the
// file, the message and the suggestion, and the store path in the header.
func TestRenderLintReportMDSanitisesEveryFreeTextField(t *testing.T) {
	out := renderLintReportMD(map[string]any{
		"store_path": "store" + hostileRunes,
		"summary":    map[string]any{"blockers": 1},
		"findings": []any{map[string]any{
			"code": "MR001", "severity": "blocker", "line": 3,
			"file":       "page" + hostileRunes + ".md",
			"message":    "message" + hostileRunes,
			"suggestion": "suggestion" + hostileRunes,
		}},
	})
	mustBeTerminalSafe(t, "report.md", out)
	for _, want := range []string{"store", "page", "message", "suggestion"} {
		if !strings.Contains(out, want) {
			t.Errorf("report.md lost the field text %q:\n%s", want, out)
		}
	}
}
