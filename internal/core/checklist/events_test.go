package checklist_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/checklist"
)

// TestRecordCompactionChecksTheFileItOpens: the sidecar the append opens is
// checked as its path is, so a second name for another file, or a FIFO, in the
// sidecar's place is left as it was and the compaction goes unrecorded, without
// waiting on the FIFO; a sidecar the hook made itself takes each new line.
func TestRecordCompactionChecksTheFileItOpens(t *testing.T) {
	t.Run("a second name for another file", func(t *testing.T) {
		root := newRepo(t)
		other := filepath.Join(t.TempDir(), "log.txt")
		plant(t, other, []byte("someone else's file\n"))
		if err := os.Link(other, abs(root, checklist.EventsRelPath)); err != nil {
			t.Skipf("no hard link across these directories: %v", err)
		}
		if err := checklist.RecordCompaction(root, "auto", time.Now()); !errors.Is(err, checklist.ErrEventsNotKept) {
			t.Fatalf("RecordCompaction over a second name = %v, want ErrEventsNotKept", err)
		}
		if got := readFile(t, other); !bytes.Equal(got, []byte("someone else's file\n")) {
			t.Fatalf("the other file was appended to: %q", got)
		}
	})
	t.Run("a FIFO", func(t *testing.T) {
		root := newRepo(t)
		p := abs(root, checklist.EventsRelPath)
		if err := mkfifo(p); err != nil {
			t.Skipf("no FIFO here: %v", err)
		}
		done := make(chan error, 1)
		go func() { done <- checklist.RecordCompaction(root, "auto", time.Now()) }()
		select {
		case err := <-done:
			if !errors.Is(err, checklist.ErrEventsNotKept) {
				t.Fatalf("RecordCompaction over a FIFO = %v, want ErrEventsNotKept", err)
			}
		case <-time.After(30 * time.Second):
			t.Fatal("the append waited on a FIFO in the sidecar's place")
		}
		if fi, err := os.Lstat(p); err != nil || fi.Mode()&os.ModeNamedPipe == 0 {
			t.Fatalf("the FIFO was replaced: %v %v", fi, err)
		}
	})
	t.Run("its own sidecar", func(t *testing.T) {
		root := newRepo(t)
		for range 2 {
			if err := checklist.RecordCompaction(root, "manual", time.Now()); err != nil {
				t.Fatal(err)
			}
		}
		if got := bytes.Count(readFile(t, abs(root, checklist.EventsRelPath)), []byte("\n")); got != 2 {
			t.Fatalf("want two lines in the hook's own sidecar, got %d", got)
		}
	})
}

// mkfifo makes a FIFO at path, for the tests that plant one where a checklist
// file belongs.
func mkfifo(path string) error { return syscall.Mkfifo(path, 0o600) }
