package lint

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// record_schema asks the site renderer whether an issue record's body renders,
// so a construct the site render refuses is refused at the record gate, in the
// change that writes it, instead of at the far end of preflight
// (iss-2608301350287219). The renderer is registered by the front doors; this
// test registers a stand-in that refuses an indented code block, the construct
// the record was filed about, and restores the registration after.
func TestRecordSchemaRefusesAnIssueBodyTheSiteCannotRender(t *testing.T) {
	prev := recordBodyCheck
	t.Cleanup(func() { recordBodyCheck = prev })
	SetRecordBodyCheck(func(rel, content string) error {
		for _, l := range strings.Split(content, "\n") {
			if strings.HasPrefix(l, "    ") {
				return errors.New(rel + ": indented code block")
			}
		}
		return nil
	})
	root := t.TempDir()
	seedRecRoot(t, root)
	bad := filepath.Join("work", "issues", "open", "iss-5-a-slug.md")
	good := filepath.Join("work", "issues", "open", "iss-6-b-slug.md")
	writeFile(t, root, bad, validIssue("iss-5", "a-slug")+"\n    indented code\n")
	writeFile(t, root, good, validIssue("iss-6", "b-slug"))
	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, bad, ruleRecordSchema, "indented code block") {
		t.Errorf("a body the site renderer refuses must be a finding naming the construct: %+v", fs)
	}
	if findingWith(fs, good, ruleRecordSchema, "") {
		t.Errorf("a body that renders draws nothing: %+v", fs)
	}
}
