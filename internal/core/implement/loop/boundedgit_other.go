//go:build !unix

package loop

import (
	"os/exec"
	"syscall"
)

// stopWithDescendants is the unix stopWithDescendants where there is no
// process table to walk: cmd's Cancel asks git alone to stop, or kills it,
// and the sweep does nothing.
func stopWithDescendants(cmd *exec.Cmd) (sweep func()) {
	cmd.Cancel = func() error {
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
	return func() {}
}
