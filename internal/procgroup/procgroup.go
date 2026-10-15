// Package procgroup is the one way abcd bounds a command together with every
// process it starts: the command leads its own process group, and the bound
// kills that group through the pid the command's own handle holds, never by a
// pattern and never another process.
package procgroup

import (
	"os/exec"
	"syscall"
	"time"
)

// Bound makes cmd, built with exec.CommandContext, lead its own process group,
// and makes the context's end kill the whole group: a pipeline, a recipe's
// shell or a background child the command started ends with it rather than
// outliving the bound. The kill is addressed to the group through the pid
// cmd's handle holds while the command is not yet reaped, so the group id is
// still its own. grace bounds the wait for the command's output after it exits
// or is killed (exec.Cmd.WaitDelay), so a process that left the group and held
// the output pipe cannot make the bound unbounded.
//
// A group of its own is out of the terminal's foreground group, so an
// interrupt typed at the terminal does not reach it: a caller that should stop
// on one ends the context on it (signal.NotifyContext).
func Bound(cmd *exec.Cmd, grace time.Duration) {
	Lead(cmd)
	cmd.Cancel = func() error { return Kill(cmd) }
	cmd.WaitDelay = grace
}

// Lead makes cmd lead its own process group once started, for a caller that
// bounds it itself (its own timer and select) and stops it with Kill.
func Lead(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// Kill sends SIGKILL to the process group cmd leads (Lead), addressed through
// the pid cmd's handle holds. Call it only after cmd started and before Wait
// has reaped it, so the group id is still its own.
func Kill(cmd *exec.Cmd) error {
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
