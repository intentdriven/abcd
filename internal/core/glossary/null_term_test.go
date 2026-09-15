package glossary

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReadTermRejectsNullFields (iss-2608270908339164) proves a YAML null in a
// term file's `term` or `status` field is treated as MISSING, not accepted as a
// literal value. readTerm gated on TrimSpace-empty only, so `term: NULL` /
// `status: ~` passed the presence check and minted a phantom term whose name was
// the literal "NULL" — a YAML null becoming data rather than a diagnosis. The fix
// treats frontmatter.IsNull(value) as an absent field.
func TestReadTermRejectsNullFields(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{"null term", "---\nterm: NULL\nstatus: draft\ndefinition: a thing\n---\n"},
		{"tilde term", "---\nterm: ~\nstatus: draft\ndefinition: a thing\n---\n"},
		{"null status", "---\nterm: widget\nstatus: NULL\ndefinition: a thing\n---\n"},
		{"null definition", "---\nterm: widget\nstatus: draft\ndefinition: null\n---\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "widget.md"), []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}
			root, err := os.OpenRoot(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			term, err := readTerm(root, "widget.md")
			if err == nil {
				t.Fatalf("a term file with a null field minted a term %+v; a YAML null must be a missing field, not data", term)
			}
		})
	}
}
