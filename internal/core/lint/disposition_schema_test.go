package lint

import (
	"path/filepath"
	"testing"
)

// wellFormedDisposition is the control: a rule watched only failing is a rule
// that might refuse everything.
const wellFormedDisposition = "---\nschema_version: 1\nid: dsp-3\nitem: rdi-2\nstate: accepted\n" +
	"disposition_grounds: worth acting on\n---\n\n"

// A disposition states the item it answers twice — as the directory it is filed
// under and as its `item` field — exactly as an admission states its run twice.
// The admission store's contradiction was named and the disposition store's was
// not, so the same double claim was checked in one store and passed in its
// sibling (iss-2608301203525338).
func TestDispositionItemFieldMustAgreeWithItsBucket(t *testing.T) {
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/dispositions/rdi-2/dsp-3.md", wellFormedDisposition)
	writeFile(t, root, "work/issues/dispositions/rdi-2/dsp-4.md",
		"---\nschema_version: 1\nid: dsp-4\nitem: rdi-9\nstate: accepted\ndisposition_grounds: worth acting on\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join("work", "issues", "dispositions", "rdi-2", "dsp-4.md"), ruleRecordSchema,
		"item declares 'rdi-9' but the disposition is filed under 'rdi-2'") {
		t.Errorf("an item field contradicting its bucket must be a finding: %+v", fs)
	}
	if n := countRule(fs, ruleRecordSchema); n != 1 {
		t.Fatalf("expected exactly 1 finding (the contradiction), got %d: %+v", n, fs)
	}
}

// The bucket check stands down on an absent field and leaves it to the required
// leg, so the store declaring a bucketField has to declare its required set too:
// the one issueschema declares, which is the set the disposition writer refuses
// a record without. A disposition naming no state records no answer.
func TestDispositionRecordRequiresItsDeclaredFields(t *testing.T) {
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/dispositions/rdi-2/dsp-3.md", wellFormedDisposition)
	writeFile(t, root, "work/issues/dispositions/rdi-2/dsp-4.md",
		"---\nschema_version: 1\nid: dsp-4\nitem: rdi-2\ndisposition_grounds: worth acting on\n---\n\n")
	writeFile(t, root, "work/issues/dispositions/rdi-2/dsp-5.md",
		"---\nschema_version: 1\nid: dsp-5\nstate: accepted\ndisposition_grounds: worth acting on\n---\n\n")
	writeFile(t, root, "work/issues/dispositions/rdi-2/dsp-6.md",
		"---\nschema_version: 1\nid: dsp-6\nitem: rdi-2\nstate: accepted\nverdict: yes\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join("work", "issues", "dispositions", "rdi-2")
	if !findingWith(fs, filepath.Join(dir, "dsp-4.md"), ruleRecordSchema, "'state'") {
		t.Errorf("a disposition naming no state must be a finding: %+v", fs)
	}
	if !findingWith(fs, filepath.Join(dir, "dsp-5.md"), ruleRecordSchema, "'item'") {
		t.Errorf("a disposition naming no item must be a finding: %+v", fs)
	}
	if !findingWith(fs, filepath.Join(dir, "dsp-6.md"), ruleRecordSchema, "unknown frontmatter property 'verdict'") {
		t.Errorf("a key outside the disposition allow-list must be a finding: %+v", fs)
	}
	if n := countRule(fs, ruleRecordSchema); n != 3 {
		t.Fatalf("expected exactly 3 findings (one per malformed record), got %d: %+v", n, fs)
	}
}
