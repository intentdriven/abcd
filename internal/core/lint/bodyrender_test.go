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

// Two record_schema legs over the issue store are asked of seams a front door
// registers, because this package cannot import the ledger reader or the site
// renderer. A caller that registers neither ran every other leg and those two
// not at all, with no signal; an armed rule over issue records now says which
// seam is missing, once, instead of passing as though it had asked.
func TestRecordSchemaNamesAnUnregisteredSeam(t *testing.T) {
	prevReader, prevBody := issueReadRefusal, recordBodyCheck
	t.Cleanup(func() { issueReadRefusal, recordBodyCheck = prevReader, prevBody })
	reader := func(content, status, path string) error { return nil }
	body := func(rel, content string) error { return nil }

	root := t.TempDir()
	seedRecRoot(t, root)
	writeFile(t, root, filepath.Join("work", "issues", "open", "iss-5-a-slug.md"), validIssue("iss-5", "a-slug"))

	for name, tc := range map[string]struct {
		reader func(content, status, path string) error
		body   func(rel, content string) error
		want   []string
		absent []string
	}{
		"both registered": {reader, body, nil, []string{"SetIssueReader", "SetRecordBodyCheck"}},
		"no reader":       {nil, body, []string{"SetIssueReader"}, []string{"SetRecordBodyCheck"}},
		"no body check":   {reader, nil, []string{"SetRecordBodyCheck"}, []string{"SetIssueReader"}},
		"neither":         {nil, nil, []string{"SetIssueReader", "SetRecordBodyCheck"}, nil},
	} {
		t.Run(name, func(t *testing.T) {
			issueReadRefusal, recordBodyCheck = tc.reader, tc.body
			fs, err := Lint(schemaConfig(), root)
			if err != nil {
				t.Fatal(err)
			}
			var seam []Finding
			for _, f := range fs {
				if f.RuleID == ruleRecordSchema && strings.Contains(f.Message, "not registered") {
					seam = append(seam, f)
				}
			}
			if tc.want == nil {
				if len(seam) != 0 {
					t.Fatalf("every seam is registered, so nothing is named: %+v", seam)
				}
				return
			}
			if len(seam) != 1 {
				t.Fatalf("want one finding naming the unregistered seam(s), got %+v (all: %+v)", seam, fs)
			}
			for _, w := range tc.want {
				if !strings.Contains(seam[0].Message, w) {
					t.Errorf("the finding must name %s: %s", w, seam[0].Message)
				}
			}
			for _, a := range tc.absent {
				if strings.Contains(seam[0].Message, a) {
					t.Errorf("the finding names %s, which is registered: %s", a, seam[0].Message)
				}
			}
		})
	}
}
