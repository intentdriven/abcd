package lint

import (
	"fmt"
	"path/filepath"
	"testing"
)

// TestRecordSchemaAcceptsTheRemedyField is the committed-ledger half of the
// `remedy:` field (itd-82 decision 6): the gate accepts a record carrying it,
// so a record capture writes is never one record-lint refuses, and it refuses a
// remedy that is not a string, the verdict the ledger reader reaches.
func TestRecordSchemaAcceptsTheRemedyField(t *testing.T) {
	root := t.TempDir()
	issues := "work/issues"
	head := "---\nschema_version: 1\nid: %s\nslug: ok\nseverity: minor\ncategory: bug\nsource: user-observation\nfound_during: t\n"
	writeFile(t, root, issues+"/open/iss-1-ok.md",
		fmt.Sprintf(head, "iss-1")+"remedy: \"guard the nil map before the write\"\n---\n\nan issue\n")
	writeFile(t, root, issues+"/open/iss-2-ok.md",
		fmt.Sprintf(head, "iss-2")+"remedy: [one, two]\n---\n\nan issue\n")
	// The configured root must exist, or the lint refuses the configuration.
	writeFile(t, root, "rec/decisions/adrs/0001-model.md", "---\nid: adr-1\n---\n# ADR-1\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if findingWith(fs, filepath.Join(issues, "open", "iss-1-ok.md"), ruleRecordSchema, "") {
		t.Errorf("record_schema refused a string remedy: %+v", fs)
	}
	if !findingWith(fs, filepath.Join(issues, "open", "iss-2-ok.md"), ruleRecordSchema, "remedy") {
		t.Errorf("record_schema accepted a list-valued remedy: %+v", fs)
	}
}
