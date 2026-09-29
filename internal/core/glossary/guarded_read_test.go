package glossary

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// The glossary store is committed and travels with a clone, and the index walk
// reads every term file it finds. A bare os.ReadFile there hangs every verb that
// builds the index on a committed FIFO, and reads an out-of-tree file as a term
// through a committed symlink (iss-2609012037125129, the sibling of
// GHSA-fh9j-8xmg-m33f). The term read goes through fsutil.ReadGuardedInRoot with
// a term-family cap (maxTermBytes) since 746a5d2c; these pin that it stays so.

// termStore lays out one bounded context holding a valid term, and returns the
// repository root and the context directory.
func termStore(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	ctx := filepath.Join(root, filepath.FromSlash(DirRelPath), "core")
	if err := os.MkdirAll(ctx, 0o755); err != nil {
		t.Fatal(err)
	}
	valid := "---\nterm: widget\nstatus: settled\ndefinition: a thing\n---\n"
	if err := os.WriteFile(filepath.Join(ctx, "widget.md"), []byte(valid), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, ctx
}

// scanWithin runs Scan with a deadline, so a read that blocks fails the test
// instead of hanging the suite.
func scanWithin(t *testing.T, root string) error {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		_, err := Scan(root)
		done <- err
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("Scan blocked on a term file: the read is not O_NONBLOCK-guarded")
		return nil
	}
}

func TestScanRefusesAFIFOTermFile(t *testing.T) {
	root, ctx := termStore(t)
	if err := syscall.Mkfifo(filepath.Join(ctx, "fifo.md"), 0o644); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}
	if err := scanWithin(t, root); err == nil {
		t.Fatal("Scan read a FIFO as a term file")
	}
}

func TestScanRefusesASymlinkedTermFile(t *testing.T) {
	root, ctx := termStore(t)
	outside := filepath.Join(t.TempDir(), "secret.md")
	if err := os.WriteFile(outside, []byte("---\nterm: leaked\nstatus: settled\ndefinition: out of tree\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(ctx, "leaked.md")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	if err := scanWithin(t, root); err == nil {
		t.Fatal("Scan read an out-of-tree file through a symlinked term file")
	}
}

func TestScanRefusesAnOversizedTermFile(t *testing.T) {
	root, ctx := termStore(t)
	big := make([]byte, maxTermBytes+1)
	copy(big, "---\nterm: huge\nstatus: settled\ndefinition: x\n---\n")
	if err := os.WriteFile(filepath.Join(ctx, "huge.md"), big, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := scanWithin(t, root); err == nil {
		t.Fatal("Scan read a term file past the term-family cap")
	}
}
