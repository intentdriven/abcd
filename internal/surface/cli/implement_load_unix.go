//go:build unix

package cli

import (
	"os"
	"syscall"
)

// signalledBy names the signal that killed a finished process, and reports
// false for one that exited on its own.
func signalledBy(ps *os.ProcessState) (int, bool) {
	ws, ok := ps.Sys().(syscall.WaitStatus)
	if !ok || !ws.Signaled() {
		return 0, false
	}
	return int(ws.Signal()), true
}
