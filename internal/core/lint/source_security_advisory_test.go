package lint

import (
	"path/filepath"
	"testing"
)

// TestRecordSchemaAcceptsSecurityAdvisorySource: record-lint's issue schema
// reads the same source enum the ledger writer does, so a record carrying
// `source: security-advisory` passes and one carrying an unknown source is
// still refused (spc-2610120557432880, step 2).
func TestRecordSchemaAcceptsSecurityAdvisorySource(t *testing.T) {
	root := t.TempDir()
	issues := "work/issues"
	writeFile(t, root, issues+"/open/iss-1-ok.md",
		"---\nschema_version: 1\nid: iss-1\nslug: ok\nseverity: major\ncategory: security\nsource: security-advisory\nfound_during: t\n---\n\nan issue\n")
	writeFile(t, root, issues+"/open/iss-2-bad.md",
		"---\nschema_version: 1\nid: iss-2\nslug: bad\nseverity: major\ncategory: security\nsource: security-advisories\nfound_during: t\n---\n\nan issue\n")
	writeFile(t, root, "rec/decisions/adrs/0001-model.md", "---\nid: adr-1\n---\n# ADR-1\n")
	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if f.RuleID == ruleRecordSchema && f.File == filepath.Join(issues, "open", "iss-1-ok.md") {
			t.Errorf("a security-advisory record was refused: %+v", f)
		}
	}
	if !findingWith(fs, filepath.Join(issues, "open", "iss-2-bad.md"), ruleRecordSchema, "source") {
		t.Errorf("an unknown source passed: %+v", fs)
	}
}
