//go:build unix

package matrix

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// A run's folder carried from elsewhere is listed without opening anything
// that is not a directory: a FIFO where a listing or an entry stands is
// refused, never opened, so it cannot hang the re-run.
func TestARunFolderHoldingAFIFOIsRefusedNotOpened(t *testing.T) {
	for _, at := range []string{"trials", "trials/feature-a", "trials/feature-a/host-big/1.json"} {
		t.Run(strings.ReplaceAll(at, "/", "_"), func(t *testing.T) {
			dir := t.TempDir()
			fifo := filepath.Join(dir, filepath.FromSlash(at))
			if err := os.MkdirAll(filepath.Dir(fifo), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := syscall.Mkfifo(fifo, 0o600); err != nil {
				t.Skipf("mkfifo: %v", err)
			}
			d, err := openRunDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer d.Close()
			done := make(chan error, 1)
			go func() {
				_, err := readTrials(d, "mx-261023012949-ab12", "trials")
				done <- err
			}()
			select {
			case err := <-done:
				if err == nil {
					t.Fatalf("a FIFO at %s is read as a run's trials; want it refused", at)
				}
			case <-time.After(5 * time.Second):
				// Unblock the open so the test process can end.
				if w, err := os.OpenFile(fifo, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
					w.Close()
				}
				t.Fatalf("listing a run's folder with a FIFO at %s hangs", at)
			}
		})
	}
}
