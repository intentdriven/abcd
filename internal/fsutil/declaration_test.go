//go:build unix

package fsutil

import (
	"errors"
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
// swap below, so the refusal there is the swap's and not the fixture's.
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

// The owner and permission guards judge the file an lstat saw; the bytes come
// from a later open. A file renamed into place between the two — by anyone
// with write on the declaration's directory — was never vetted, so it is
// refused rather than read as the caller's word (iss-2609251537550065). The
// swapped-in file here would pass every guard on its own, so only the identity
// check between the lstat and the opened descriptor can refuse it.
func TestReadDeclarationRefusesAFileSwappedInAfterVetting(t *testing.T) {
	dir := t.TempDir()
	path := writeDeclaration(t, dir, "decl", "vetted\n")
	other := writeDeclaration(t, dir, "other", "swapped\n")
	prev := declarationVetted
	t.Cleanup(func() { declarationVetted = prev })
	declarationVetted = func(p string) {
		if err := os.Rename(other, p); err != nil {
			t.Fatalf("swap: %v", err)
		}
	}
	raw, refusal, err := ReadDeclaration(path, 1024)
	if refusal == DeclarationOK || err == nil {
		t.Fatalf("a file swapped in after vetting must be refused; read %q", raw)
	}
	if raw != nil {
		t.Fatalf("a refused declaration returns no bytes; got %q", raw)
	}
	if refusal != DeclarationUnreadable || !errors.Is(err, ErrDeclarationSwapped) {
		t.Fatalf("the swap must be named: refusal %d, err %v", refusal, err)
	}
}
