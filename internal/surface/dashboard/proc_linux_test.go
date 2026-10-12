package dashboard

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
	"testing"
)

// TestReadProcessTakesAnExitRaceForAGoneProcess drives readProcess's two
// reads of /proc through the failure a read racing the exit returns, ESRCH,
// and holds each to errProcessGone, so Stop sees the server gone rather than
// failing.
func TestReadProcessTakesAnExitRaceForAGoneProcess(t *testing.T) {
	esrch := func(op, path string) error { return &fs.PathError{Op: op, Path: path, Err: syscall.ESRCH} }
	for _, c := range []struct {
		name string
		stat func(string) ([]byte, error)
		exe  func(string) (string, error)
	}{
		{"stat", func(p string) ([]byte, error) { return nil, esrch("read", p) }, os.Readlink},
		{"exe", os.ReadFile, func(p string) (string, error) { return "", esrch("readlink", p) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			stat, exe := readStat, readExe
			t.Cleanup(func() { readStat, readExe = stat, exe })
			readStat, readExe = c.stat, c.exe
			if _, err := readProcess(os.Getpid()); !errors.Is(err, errProcessGone) {
				t.Fatalf("readProcess with an ESRCH from the %s read = %v, want errProcessGone", c.name, err)
			}
		})
	}
}
