package lint

import (
	"path/filepath"
	"testing"
)

// readingLinkRecord is a well-formed detection reading record with the extra
// frontmatter lines given.
func readingLinkRecord(id, extra string) string {
	return "---\nschema_version: 1\nid: " + id + "\nrun: rdg-1\nmanifest: sha256:beef\nposition: detection\n" +
		"regime: registrative\npattern: a stated constraint\ntension: t\nconstraint_in_play: c\nwhy_a_tension: w\n" +
		extra + "---\n\n"
}

// A reading record carries the filing-time match's typed links (ruling DQ2b,
// adr-2609300821558671). The committed-tree gate accepts them, and resolves
// each id it names: a link to a reading item the ledger holds is clean, and a
// link to one it does not is a finding, as it is on an issue.
func TestReadingRecordTypedLinksAreAcceptedAndResolved(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	writeFile(t, root, "work/issues/readings/rdg-1/rdi-2.md", readingLinkRecord("rdi-2", ""))
	writeFile(t, root, "work/issues/readings/rdg-1/rdi-3.md", readingLinkRecord("rdi-3", "duplicates: [rdi-2]\nrefines: [rdi-2]\n"))

	fs, err := Lint(readingSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 0 {
		t.Fatalf("a reading record linking an item the ledger holds must be clean, got %d finding(s): %+v", n, fs)
	}

	writeFile(t, root, "work/issues/readings/rdg-1/rdi-4.md", readingLinkRecord("rdi-4", "duplicates: [rdi-9]\n"))
	fs, err = Lint(readingSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join("work", "issues", "readings", "rdg-1", "rdi-4.md"), ruleRecordSchema, "rdi-9") {
		t.Fatalf("a link to a reading item the ledger does not hold must be reported: %+v", fs)
	}
}
