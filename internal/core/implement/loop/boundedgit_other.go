//go:build !unix

package loop

import (
	"os/exec"
	"syscall"
)

// stopAsGroup is the unix stopAsGroup where there are no process groups to
// signal: cmd's Cancel asks git alone to stop, or kills it, and the sweep does
// nothing.
func stopAsGroup(cmd *exec.Cmd) (sweep func()) {
	cmd.Cancel = func() error {
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
	return func() {}
}
