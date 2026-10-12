//go:build unix

package loop

import (
	"os/exec"
	"sync/atomic"
	"syscall"
	"time"
)

// stopAsGroup makes cmd's deadline stop reach every process cmd starts, not
// git alone: cmd runs in a process group of its own (Setpgid), its Cancel
// sends SIGTERM to that group (git removes its lock files on it, and a hook
// and the programs the hook started are stopped with it), and the sweep it
// returns, run once Wait has returned, sends SIGKILL to the group when
// commitStopGrace has passed since the stop, or sooner if the group is
// already empty. The sweep does nothing when the deadline was not reached: a
// program a hook leaves running behind a git that finished is the
// repository's, and is left alone.
//
// A process the group's processes start in a group or session of their own
// (a daemon) is outside the group and is not reached; neither is one a hook
// asks another process to start (a gpg-agent's pinentry).
func stopAsGroup(cmd *exec.Cmd) (sweep func()) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	var stopped atomic.Int64
	cmd.Cancel = func() error {
		stopped.Store(time.Now().UnixNano())
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
	return func() {
		at := stopped.Load()
		if at == 0 || cmd.Process == nil {
			return
		}
		group := -cmd.Process.Pid
		for deadline := time.Unix(0, at).Add(commitStopGrace); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
			if syscall.Kill(group, 0) != nil {
				return
			}
		}
		_ = syscall.Kill(group, syscall.SIGKILL)
	}
}
