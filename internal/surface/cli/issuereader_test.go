package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// The CLI registers the issue ledger's reader, so record_schema's
// reader-parity leg runs in the gate: a single-quoted severity, which the reader
// refuses and skips while every re-derived leg reads it as `minor`, is a
// finding here. Without the registration the leg is silent and this fails.
func TestCLIRegistersTheLedgerReader(t *testing.T) {
	root := t.TempDir()
	rel := filepath.Join("work", "issues", "open", "iss-5-a-slug.md")
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nschema_version: 1\nid: iss-5\nslug: a-slug\nseverity: minor\ncategory: bug\n" +
		"source: user-observation\nfound_during: t\n---\n\nan issue\n"
	body = strings.Replace(body, "severity: minor", "severity: minor\n  stray", 1)
	if err := os.WriteFile(filepath.Join(root, rel), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := lint.Config{Rules: map[string]lint.RuleConfig{
		"record_schema": {Enabled: true, Severity: "blocker", RecordStores: map[string]string{"iss": "work/issues"}},
	}}
	fs, err := lint.Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if strings.Contains(f.Message, "ledger reader refuses") {
			return
		}
	}
	t.Fatalf("the reader-parity leg did not run in the CLI: %+v", fs)
}

// The CLI registers the doc-fidelity flags reader, so a config arming
// brief_flag_landed under `abcd lint` reports a review flag whose replacement
// its chapter does not carry, as record-lint does (iss-2610050259233425).
func TestCLIRegistersTheBriefFlagReader(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		if err := os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".abcd/development/brief/04-surfaces/10-docs.md", "the wording as committed\n")
	write(".abcd/work/brief-review-flags.json", `{"schema_version": 1, "flags": [{"chapter": "10-docs.md", `+
		`"sentence": "s", "replacement": "the drafted wording", "commit": "c", "applied": "t"}]}`+"\n")
	cfg := lint.Config{Rules: map[string]lint.RuleConfig{
		"brief_flag_landed": {Enabled: true, Severity: "blocker"},
	}}
	fs, err := lint.Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if f.RuleID == "brief_flag_landed" && strings.Contains(f.Message, "the drafted wording") {
			return
		}
	}
	t.Fatalf("the brief flag reader did not run in the CLI: %+v", fs)
}
