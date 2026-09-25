package lint

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
)

// The committed-ledger gate refuses exactly what capture's ledger reader
// refuses, spelling by spelling. Each case is a valid issue record with one line
// changed; `refused` is what the reader does with it (List skips a refused
// record, making it invisible to every capture surface), and the gate must
// agree in both directions: a finding where the reader refuses, and none where
// it reads the record (iss-2608300205044566, iss-2608300234598982,
// iss-2608300244483405, iss-2608301519255156).
func TestRecordSchemaAgreesWithTheLedgerReader(t *testing.T) {
	const good = "severity: minor\n"
	cases := []struct {
		name    string
		old     string
		new     string
		refused bool
	}{
		{"single-quoted severity", good, "severity: 'minor'\n", true},
		{"single-quoted lapsed_at", good, good + "lapsed_at: '2026-08-28T00:00:00Z'\n", true},
		{"double-quoted escape the reader decodes", good, "severity: \"min\\or\"\n", false},
		{"unicode-space-led bogus key", good, good + " bogus: x\n", true},
		{"indented comment as a key's only continuation", good, good + "blocked_by:\n  # none yet\n", true},
		{"stray indented line after a valued key", good, "severity: minor\n  stray\n", true},
		{"stray indented line after a null key", good, good + "blocked_by: null\n  stray\n", true},
		{"schema_version 2", "schema_version: 1\n", "schema_version: 2\n", true},
		{"quoted schema_version", "schema_version: 1\n", "schema_version: \"1\"\n", true},
		{"schema_version with a trailing comment", "schema_version: 1\n", "schema_version: 1 # v1\n", false},
		{"found_during as a list", "found_during: t\n", "found_during: [a]\n", true},
		{"the valid record itself", good, good, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			seedRecRoot(t, root)
			body := validIssue("iss-5", "a-slug")
			if !strings.Contains(body, c.old) {
				t.Fatalf("fixture lacks %q", c.old)
			}
			rel := filepath.Join("work", "issues", "open", "iss-5-a-slug.md")
			content := strings.Replace(body, c.old, c.new, 1)
			// The expectation is the reader's own verdict, asserted rather than
			// assumed, so a case cannot encode a guess about what the reader does.
			if refused := capture.ReadRefusal(content, "open", rel) != nil; refused != c.refused {
				t.Fatalf("fixture expectation is wrong: the reader refuses=%v", refused)
			}
			writeFile(t, root, rel, content)
			fs, err := Lint(schemaConfig(), root)
			if err != nil {
				t.Fatal(err)
			}
			got := findingWith(fs, rel, ruleRecordSchema, "")
			if got != c.refused {
				t.Fatalf("reader refuses=%v, gate reports=%v: %+v", c.refused, got, fs)
			}
		})
	}
}
