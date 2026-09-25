package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// record-lint registers the issue ledger's reader, so record_schema's
// reader-parity leg runs in the gate: a single-quoted severity, which the reader
// refuses and skips while every re-derived leg reads it as `minor`, is a
// finding here. Without the registration the leg is silent and this fails.
func TestRecordLintRegistersTheLedgerReader(t *testing.T) {
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
	t.Fatalf("the reader-parity leg did not run in record-lint: %+v", fs)
}
