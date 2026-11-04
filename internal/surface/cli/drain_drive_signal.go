package cli

// drain_drive_signal.go is a drive's stop (spc-2610230813348813 step 5;
// dcn-2611041608057544, dcn-2611041550139264). The first interrupt, hang-up or
// termination asks for a stop and leaves the context live, so the runner's call
// in flight finishes: the launcher kills an agent's process group when its
// context ends, and only the second signal ends it. Agents and a preflight lead
// process groups of their own, so the terminal's interrupt reaches neither.

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// driveKillGrace is how long a second signal leaves the drive to kill the group
// and say so before the process exits whatever it is doing.
const driveKillGrace = 5 * time.Second

// driveKilledExit is the exit a second signal ends the process with.
const driveKilledExit = 130

// driveExit is os.Exit, a seam the tests replace.
var driveExit = os.Exit

// driveSignals is one drive's signal handling.
type driveSignals struct {
	// stop is closed by the first signal.
	stop chan struct{}
	// ctx ends on the second.
	ctx    context.Context
	cancel context.CancelFunc
	ch     chan os.Signal
	once   sync.Once
}

// watchDriveSignals starts the handling. first and second say what each signal
// did.
func watchDriveSignals(first, second func()) *driveSignals {
	ctx, cancel := context.WithCancel(context.Background())
	s := &driveSignals{stop: make(chan struct{}), ctx: ctx, cancel: cancel, ch: make(chan os.Signal, 2)}
	// The drive outlives a terminal that closed, so a write to it is no reason
	// to die.
	signal.Ignore(syscall.SIGPIPE)
	signal.Notify(s.ch, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	go func() {
		if _, ok := <-s.ch; !ok {
			return
		}
		close(s.stop)
		first()
		if _, ok := <-s.ch; !ok {
			return
		}
		second()
		cancel()
		time.AfterFunc(driveKillGrace, func() { driveExit(driveKilledExit) })
	}()
	return s
}

// close ends the handling.
func (s *driveSignals) close() {
	s.once.Do(func() {
		signal.Stop(s.ch)
		close(s.ch)
		s.cancel()
	})
}
