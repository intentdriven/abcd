package lint

import (
	"path/filepath"
	"strings"
	"testing"
)

// The content legs mark what they spoke about so the required-fields leg does
// not add a second, weaker finding on the same line (the call-site protocol in
// checkRecordSchema). The marks on id and slug were recorded as unreachable
// (iss-2608301634520703); they are not: an EMPTY quoted value (`id: ""`) is a
// present value to the filename legs, which report the disagreement and mark
// it, and absent to the required-fields leg, which would otherwise report the
// same line again as a missing property. This pins the protocol at both marks.
func TestFilenameLegsMarkWhatTheyJudged(t *testing.T) {
	for name, tc := range map[string]struct{ id, slug, field string }{
		"empty id":   {`""`, "a-finding", "id"},
		"empty slug": {"iss-1", `""`, "slug"},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			seedRecRoot(t, root)
			writeFile(t, root, "work/issues/open/iss-1-a-finding.md",
				"---\nschema_version: 1\nid: "+tc.id+"\nslug: "+tc.slug+
					"\nseverity: minor\ncategory: bug\nsource: user-observation\nfound_during: t\n---\n\nan issue\n")
			fs, err := Lint(schemaConfig(), root)
			if err != nil {
				t.Fatal(err)
			}
			line := 3
			if tc.field == "slug" {
				line = 4
			}
			rel := filepath.Join("work", "issues", "open", "iss-1-a-finding.md")
			if !findingWith(fs, rel, ruleRecordSchema, "frontmatter declares ''") {
				t.Fatalf("the filename leg did not report the empty %s: %+v", tc.field, fs)
			}
			for _, f := range fs {
				if f.RuleID == ruleRecordSchema && f.File == rel && f.Line == line &&
					strings.Contains(f.Message, "required property") {
					t.Errorf("the required-fields leg spoke a second time on a value a content leg judged: %+v", f)
				}
			}
		})
	}
}
