package lint

import (
	"path/filepath"
	"strings"
	"testing"
)

// withField returns a valid issue record carrying one extra frontmatter line.
func withField(id, slug, line string) string {
	return strings.Replace(validIssue(id, slug), "\nseverity:", "\n"+line+"\nseverity:", 1)
}

// readerParityPhrase opens the reader-parity leg's finding (checkIssueReaderParity).
const readerParityPhrase = "capture's ledger reader refuses this record"

// TestRecordSchemaReportsEveryRecordTheLedgerReaderSkips (iss-2609261631132673):
// a committed issue the ledger reader skips is one every capture surface leaves
// out, so record_schema reports it, in the reader's own words. The reasons
// below were silent here while `abcd <id>` named them: a malformed item in an
// id list, and a list naming one record twice. The finding carries the
// reader's message rather than a lint-side restatement, because the verdict
// comes from the reader itself (the reader-parity leg, asked of the one
// judgement core/issuerecord holds) — a well-formed record stays silent.
func TestRecordSchemaReportsEveryRecordTheLedgerReaderSkips(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/README.md", "# record\n")
	open := "work/issues/open"
	writeFile(t, root, open+"/iss-1-fine.md", validIssue("iss-1", "fine"))
	cases := []struct {
		name, content, reason string
	}{
		{"iss-2-dup.md", withField("iss-2", "dup", "duplicates: [bogus]"),
			`malformed frontmatter: "duplicates" item "bogus" does not match iss-N or itd-N`},
		{"iss-3-blocked.md", withField("iss-3", "blocked", "blocked_by: [bogus]"),
			`malformed frontmatter: "blocked_by" item "bogus" does not match iss-N`},
		{"iss-4-twice.md", withField("iss-4", "twice", "related_issues: [iss-1, iss-1]"),
			"invariant violation: related_issues contains duplicate items"},
	}
	for _, c := range cases {
		writeFile(t, root, open+"/"+c.name, c.content)
	}
	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		file := filepath.Join(open, c.name)
		if !findingWith(fs, file, ruleRecordSchema, readerParityPhrase) ||
			!findingWith(fs, file, ruleRecordSchema, c.reason) {
			t.Errorf("%s: no record_schema finding in the reader's words %q: %+v", c.name, c.reason, fs)
		}
	}
	for _, f := range fs {
		if f.File == filepath.Join(open, "iss-1-fine.md") {
			t.Errorf("a record the reader takes was reported: %+v", f)
		}
	}
}
