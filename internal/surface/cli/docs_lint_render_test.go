package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDocsLintRenderSanitisesConfigFields pins that the findings renderer
// sanitises every config-derived field. A banned token's id is free text from
// the committed config — a trust boundary (LoadConfig's own contract) — so a
// hostile clone must not be able to put a raw terminal escape on the finding
// line through it.
func TestDocsLintRenderSanitisesConfigFields(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{
	  "roots": ["docs"],
	  "banned_tokens": [
	    {"id": "evil\u001b[2Jtoken", "pattern": "forbidden-word", "message": "no", "severity": "warn", "successor": "allowed-word", "allow_context": ["nowhere-real"]}
	  ]
	}`
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "docs-lint.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "docs", "page.md"), []byte("uses the forbidden-word here\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	Run([]string{"docs", "lint"}, &stdout, &stderr)
	out := stdout.String() + stderr.String()
	if !strings.Contains(out, "evil") {
		t.Fatalf("expected the finding to render (config id present), got:\n%s", out)
	}
	if strings.ContainsRune(out, 0x1b) {
		t.Fatalf("a raw ESC from the config's token id reached the terminal:\n%q", out)
	}
}

// TestDocsLintWithNoRulesSaysNothingWasChecked is iss-2609150805167646's first
// remedy (loud-staging): a config that arms no rule runs nothing, so the verb must
// say so rather than print "0 finding(s), 0 blocker(s)", a count that implies a
// check happened. The probe is the report's own: a present-tense violation inside
// a configured root, under the config the scaffold used to write. The JSON
// carries the number of checks that ran, so a caller can tell the two apart too.
func TestDocsLintWithNoRulesSaysNothingWasChecked(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	for _, d := range []string{".abcd", "docs"} {
		if err := os.MkdirAll(filepath.Join(repo, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	empty := `{"roots": ["docs"], "banned_tokens": [], "rules": {}, "exempt_paths": [], "exempt_if_status": []}`
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "docs-lint.json"), []byte(empty), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "docs", "page.md"), []byte("Previously this was different.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	Run([]string{"docs", "lint"}, &stdout, &stderr)
	out := stdout.String() + stderr.String()
	if !strings.Contains(out, "nothing was checked") {
		t.Errorf("a config with no rules must say nothing was checked, got:\n%s", out)
	}
	if strings.Contains(out, "finding(s)") {
		t.Errorf("a lint that ran no rule must not report a finding count, got:\n%s", out)
	}

	stdout.Reset()
	stderr.Reset()
	Run([]string{"docs", "lint", "--json"}, &stdout, &stderr)
	var res struct {
		Checks *int `json:"checks"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		t.Fatalf("--json output does not parse: %v\n%s", err, stdout.String())
	}
	if res.Checks == nil || *res.Checks != 0 {
		t.Errorf("--json must carry checks: 0 for a config that armed nothing, got %s", stdout.String())
	}
}

// TestDocsLintCountsTheChecksItRan is the ok side of the no-rules render: an
// armed config keeps the finding-count summary and reports how many checks ran.
func TestDocsLintCountsTheChecksItRan(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	for _, d := range []string{".abcd", "docs"} {
		if err := os.MkdirAll(filepath.Join(repo, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	armed := `{"roots": ["docs"], "banned_tokens": [
	  {"id": "present_tense/previously", "pattern": "(?i)\\bpreviously\\b", "message": "no", "severity": "warn", "successor": "present tense", "allow_context": ["docs-lint: allow"]}
	], "rules": {"links_resolve": {"enabled": true, "severity": "blocker"}, "harness_leak": {"enabled": false, "severity": "blocker"}}}`
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "docs-lint.json"), []byte(armed), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "docs", "page.md"), []byte("Previously this was different.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	Run([]string{"docs", "lint"}, &stdout, &stderr)
	out := stdout.String() + stderr.String()
	if !strings.Contains(out, "1 finding(s), 0 blocker(s)") || strings.Contains(out, "nothing was checked") {
		t.Errorf("an armed config must report its findings, got:\n%s", out)
	}
	stdout.Reset()
	stderr.Reset()
	Run([]string{"docs", "lint", "--json"}, &stdout, &stderr)
	var res struct {
		Checks *int `json:"checks"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		t.Fatalf("--json output does not parse: %v\n%s", err, stdout.String())
	}
	// One token and one enabled rule; the disabled rule is not a check that ran.
	if res.Checks == nil || *res.Checks != 2 {
		t.Errorf("--json must count the armed checks (want 2), got %s", stdout.String())
	}
}
