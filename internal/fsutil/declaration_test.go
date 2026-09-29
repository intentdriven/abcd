//go:build unix

package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

// writeDeclaration writes body at dir/name, owner-only writable.
func writeDeclaration(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// A declaration that passes every guard is read unchanged: the control for the
// replacements in replaced_test.go, so a refusal there is the replacement's and
// not the fixture's.
func TestReadDeclarationReadsTheVettedFile(t *testing.T) {
	path := writeDeclaration(t, t.TempDir(), "decl", "vetted\n")
	raw, refusal, err := ReadDeclaration(path, 1024)
	if err != nil || refusal != DeclarationOK {
		t.Fatalf("an owned, owner-only-writable regular file must be read: refusal %d, err %v", refusal, err)
	}
	if string(raw) != "vetted\n" {
		t.Fatalf("read %q, want the vetted bytes", raw)
	}
}
