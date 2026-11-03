package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// record-lint registers the pending-reference lookup (iss-2611031731385043):
// an id the tree does not resolve is looked for on the open pull requests'
// branches before it is refused. With no forge client to ask, the refusal
// stands and says why the lookup could not help, which it can only say if the
// lookup ran. Without the registration the reason is absent and this fails.
func TestRecordLintRegistersThePendingLookup(t *testing.T) {
	empty := t.TempDir()
	t.Setenv("PATH", empty)
	t.Setenv("HOME", empty)
	root := t.TempDir()
	rel := filepath.Join("work", "issues", "open", "iss-5-a-slug.md")
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: iss-5\n---\n\nThe keys stay a fallback, per dcn-2611031615094092.\n"
	if err := os.WriteFile(filepath.Join(root, rel), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := lint.Config{Rules: map[string]lint.RuleConfig{
		"prose_citation_resolves": {Enabled: true, Severity: "blocker", RecordStores: map[string]string{"iss": "work/issues"}},
	}}
	fs, err := lint.Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if f.RuleID == "prose_citation_resolves" && strings.Contains(f.Message, "`gh` is not installed") {
			return
		}
	}
	t.Fatalf("the pending lookup did not run in record-lint: %+v", fs)
}
