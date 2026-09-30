package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// record-lint registers the intent package's supersession chain reader, so
// stale_edge follows a superseded record to its successor. Without the
// registration the rule reports the missing seam instead, and this fails.
func TestRecordLintRegistersTheSupersessionChain(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("rec/intents/superseded/itd-2-x.md", "---\nid: itd-2\nslug: x\nsuperseded_by: itd-3\n---\n\n# itd-2\n")
	write("rec/intents/planned/itd-3-x.md", "---\nid: itd-3\nslug: x\n---\n\n# itd-3\n")
	write("rec/intents/drafts/itd-5-x.md", "---\nid: itd-5\nslug: x\nblocked_by: [itd-2]\n---\n\n# itd-5\n")
	cfg := lint.Config{Rules: map[string]lint.RuleConfig{
		"record_schema": {RecordStores: map[string]string{"itd": "rec/intents"}},
		"stale_edge":    {Enabled: true, Severity: "warn"},
	}}
	fs, err := lint.Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if f.RuleID == "stale_edge" && strings.Contains(f.Message, "ends at itd-3 (planned)") {
			return
		}
	}
	t.Fatalf("stale_edge did not follow the chain through the registered reader: %+v", fs)
}
