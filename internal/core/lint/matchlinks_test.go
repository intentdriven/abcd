package lint

import (
	"path/filepath"
	"testing"
)

// The filing-time match's typed links (itd-2609212137116617) are claims that
// another record exists, like every other cross-reference: a `duplicates:` or
// `refines:` naming a record the corpus does not hold is a finding, and one
// naming a present issue or intent is silent.
func TestRecordSchemaResolvesMatchLinks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/intents/drafts/itd-17-present.md", "---\nid: itd-17\nkind: null\nspec_id: null\n---\n# present\n")
	writeFile(t, root, "rec/intents/drafts/itd-18-linked.md",
		"---\nid: itd-18\nkind: null\nspec_id: null\nduplicates: [itd-17]\nrefines: [itd-99]\n---\n# linked\n")
	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 1 {
		t.Fatalf("want exactly the one phantom refines target, got %d: %+v", n, fs)
	}
	if !findingWith(fs, filepath.Join("rec/intents/drafts", "itd-18-linked.md"), ruleRecordSchema, "refines names 'itd-99'") {
		t.Fatalf("no finding naming the phantom refines target: %+v", fs)
	}
}
