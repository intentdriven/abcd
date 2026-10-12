package lint

import (
	"path/filepath"
	"testing"
)

// TestRecordSchemaJudgesFindingKey (itd-2609291923559186 G1): a record the
// consistency pass filed carries `finding_key:`, a key capture writes in one
// shape only. The well-formed key is clean; any other value is reported in the
// reader's terms, because capture's reader refuses and skips the record, so the
// committed-ledger gate refuses what capture refuses.
func TestRecordSchemaJudgesFindingKey(t *testing.T) {
	issues := "work/issues"
	keyed := func(id, slug, key string) string {
		return "---\nschema_version: 1\nid: " + id + "\nslug: " + slug +
			"\nseverity: minor\ncategory: inconsistency\nsource: agent-finding\nfound_during: t\nfinding_key: " + key + "\n---\n\nan issue\n"
	}
	cases := []struct {
		name, file, rec, substr string
	}{
		{"well-formed", "iss-5-a.md", keyed("iss-5", "a", "cfk-0123456789abcdef"), ""},
		{"well-formed, quoted", "iss-6-b.md", keyed("iss-6", "b", `"cfk-0123456789abcdef"`), ""},
		{"too short", "iss-7-c.md", keyed("iss-7", "c", "cfk-0123"), "invalid finding_key 'cfk-0123'"},
		{"upper-case hex", "iss-8-d.md", keyed("iss-8", "d", "cfk-0123456789ABCDEF"), "invalid finding_key"},
		{"another family's id", "iss-9-e.md", keyed("iss-9", "e", "iss-0123456789abcdef"), "invalid finding_key"},
		{"a list", "iss-10-f.md", keyed("iss-10", "f", "[]"), "invalid finding_key"},
	}
	root := t.TempDir()
	seedRecRoot(t, root)
	for _, c := range cases {
		writeFile(t, root, issues+"/open/"+c.file, c.rec)
	}
	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		file := filepath.Join(issues, "open", c.file)
		if c.substr == "" {
			if findingWith(fs, file, ruleRecordSchema, "") {
				t.Errorf("%s: a well-formed finding_key drew a finding: %+v", c.name, fs)
			}
			continue
		}
		if !findingWith(fs, file, ruleRecordSchema, c.substr) {
			t.Errorf("%s: want a record_schema finding containing %q on %s: %+v", c.name, c.substr, file, fs)
		}
	}
}
