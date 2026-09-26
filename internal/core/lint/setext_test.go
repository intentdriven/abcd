package lint

import (
	"path/filepath"
	"testing"
)

// A bare `---` directly under a paragraph line is not a thematic break in
// CommonMark: it underlines the line above into a setext heading, which the
// record explorer then renders as a heading nobody wrote (iss-2608221342508878).
// record_schema reports it on a record body; a `---` after a blank line, inside
// a fence, or under a list item or heading is a thematic break and passes.
func TestRecordSchemaReportsASetextUnderlineInARecordBody(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	mk := func(n, body string) string {
		return "---\nschema_version: 1\nid: iss-" + n + "\nslug: s\nseverity: minor\ncategory: bug\nsource: user-observation\nfound_during: t\n---\n\n" + body
	}
	writeFile(t, root, "work/issues/open/iss-1-s.md", mk("1", "a paragraph\n---\nmore\n"))
	writeFile(t, root, "work/issues/open/iss-2-s.md", mk("2", "a paragraph\n\n---\n\n## Evidence\n---\n- item\n---\n```\ntext\n---\n```\n"))
	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasFinding(fs, filepath.Join("work", "issues", "open", "iss-1-s.md"), ruleRecordSchema, 12) {
		t.Errorf("the setext underline on line 12 was not reported: %+v", fs)
	}
	for _, f := range fs {
		if f.File == filepath.Join("work", "issues", "open", "iss-2-s.md") && f.RuleID == ruleRecordSchema {
			t.Errorf("a thematic break was reported: %+v", f)
		}
	}
}
