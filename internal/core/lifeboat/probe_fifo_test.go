//go:build unix

package lifeboat

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestListDirDoesNotBlockOnFifo proves SourceContext.ListDir opens directory
// entries non-blocking: a FIFO planted at a path a probe adapter lists (a
// statically-planted trap, no race) must not hang the probe. Before the fix
// ListDir opened with a plain O_RDONLY, so opening the FIFO blocked in the
// kernel until a writer appeared and abcd disembark probe/plan/pack hung.
func TestListDirDoesNotBlockOnFifo(t *testing.T) {
	dir := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(dir, "docs"), 0o644); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatalf("newSourceContext: %v", err)
	}

	done := make(chan []string, 1)
	go func() { done <- ctx.ListDir("docs") }()
	select {
	case names := <-done:
		// A FIFO is not a directory, so the listing is empty — but promptly.
		if len(names) != 0 {
			t.Fatalf("ListDir on a FIFO returned entries: %v", names)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("ListDir hung on a FIFO (open must not block)")
	}
}

// TestReadLifeboatFileDoesNotBlockOnFifo proves the guarded lifeboat read refuses
// a FIFO promptly rather than blocking the open. A vetted regular file in an
// untrusted lifeboat can be swapped for a FIFO between the manifest walk and the
// read; the read must return an error, not hang embark.
func TestReadLifeboatFileDoesNotBlockOnFifo(t *testing.T) {
	dir := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(dir, "trap.md"), 0o644); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatalf("OpenRoot: %v", err)
	}
	defer root.Close()

	done := make(chan error, 1)
	go func() {
		_, e := readLifeboatFile(root, "trap.md")
		done <- e
	}()
	select {
	case e := <-done:
		if e == nil {
			t.Fatal("readLifeboatFile must refuse a FIFO, not read it")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("readLifeboatFile hung on a FIFO (open must not block)")
	}
}

// TestWalkDescentDoesNotBlockOnFifo is iss-337: both walks descend into a child
// directory by name after the parent's ReadDir reported it as a directory, and a
// hostile tree can swap that directory for a FIFO in the window between the two.
// os.Root.OpenRoot opens its final component with neither O_DIRECTORY nor
// O_NONBLOCK, so the descent would block in the kernel until a writer appeared.
// The race itself cannot be scheduled deterministically, so the test hands the
// descent the state the race produces — a FIFO where a directory was listed —
// and requires a prompt refusal. A real directory still opens (the ok: side).
func TestWalkDescentDoesNotBlockOnFifo(t *testing.T) {
	dir := t.TempDir()
	fifo := filepath.Join(dir, "swapped")
	if err := syscall.Mkfifo(fifo, 0o644); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "real"), 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatalf("OpenRoot: %v", err)
	}
	defer root.Close()

	type result struct {
		sub *os.Root
		err error
	}
	done := make(chan result, 1)
	go func() {
		sub, e := openWalkDir(root, "swapped")
		done <- result{sub, e}
	}()
	select {
	case r := <-done:
		if r.err == nil {
			r.sub.Close()
			t.Fatal("openWalkDir opened a FIFO as a directory; it must refuse it")
		}
	case <-time.After(3 * time.Second):
		// Release the blocked open so the leaked goroutine can finish.
		if w, err := os.OpenFile(fifo, os.O_RDWR, 0); err == nil {
			w.Close()
		}
		t.Fatal("openWalkDir hung on a FIFO (the descent must not block)")
	}

	sub, err := openWalkDir(root, "real")
	if err != nil {
		t.Fatalf("openWalkDir refused a real directory: %v", err)
	}
	sub.Close()
}

// TestWalkFilesStartDoesNotBlockOnFifo is the start-directory half of iss-337: a
// FIFO planted at the start path of a walk must return promptly with nothing,
// not block the open that descends into the start.
func TestWalkFilesStartDoesNotBlockOnFifo(t *testing.T) {
	dir := t.TempDir()
	fifo := filepath.Join(dir, "trap")
	if err := syscall.Mkfifo(fifo, 0o644); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatalf("newSourceContext: %v", err)
	}
	defer ctx.Close()

	done := make(chan []string, 1)
	go func() {
		paths, _ := ctx.WalkFiles("trap")
		done <- paths
	}()
	select {
	case paths := <-done:
		if len(paths) != 0 {
			t.Fatalf("WalkFiles from a FIFO start returned %v, want nothing", paths)
		}
	case <-time.After(3 * time.Second):
		if w, err := os.OpenFile(fifo, os.O_RDWR, 0); err == nil {
			w.Close()
		}
		t.Fatal("WalkFiles hung on a FIFO start (the descent must not block)")
	}
}
