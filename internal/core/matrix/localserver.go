package matrix

// localserver.go holds the one local inference server a run serves its local
// cells from (decision 12, spc-2610161949559050 step 3): one at a time, started
// as one owned process group and stopped as one when its cells end, through
// the handle the run holds and never by a pattern. The machine serves one
// local model at a time (the intent's scope conditions), so a second cell's
// server is refused while the first's runs rather than started beside it.

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/intentdriven/abcd/internal/procgroup"
)

// serverGrace bounds the wait for a stopped server's output pipes.
const serverGrace = 5 * time.Second

// localServer is the run's one local inference server slot. The zero value
// serves nothing.
type localServer struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	cell   string
	cancel context.CancelFunc
	done   chan struct{}
}

// start starts argv as the server for cell, leading its own process group,
// unless that cell's server already runs. A server running for another cell
// refuses it: one local server serves at a time, and it is stopped when its
// cells end. A server that ended by itself frees the slot.
func (s *localServer) start(cell string, argv []string) error {
	if len(argv) == 0 || argv[0] == "" {
		return errors.New("a local server needs the command that serves the cell's model")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runningLocked() {
		if s.cell == cell {
			return nil
		}
		return fmt.Errorf("the local server for cell %s is still running, and one local server serves at a time; "+
			"it is stopped when that cell's copies end, before cell %s's starts", s.cell, cell)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	// The group is killed through the handle when the context ends, and what
	// it left once its leader exits is killed too.
	procgroup.Bound(cmd, serverGrace)
	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("the local server for cell %s did not start: %w", cell, err)
	}
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		procgroup.KillLeft(cmd)
		close(done)
	}()
	s.cmd, s.cell, s.cancel, s.done = cmd, cell, cancel, done
	return nil
}

// serving names the cell the running server serves, "" when none runs.
func (s *localServer) serving() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.runningLocked() {
		return ""
	}
	return s.cell
}

// stop stops the running server's whole process group and waits for it to be
// reaped; with none running it does nothing.
func (s *localServer) stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd == nil {
		return nil
	}
	s.cancel()
	select {
	case <-s.done:
	case <-time.After(2 * serverGrace):
		return fmt.Errorf("the local server for cell %s did not end after its group was killed", s.cell)
	}
	s.clearLocked()
	return nil
}

// runningLocked reports whether a server runs, freeing the slot of one that
// ended by itself. The caller holds s.mu.
func (s *localServer) runningLocked() bool {
	if s.cmd == nil {
		return false
	}
	select {
	case <-s.done:
		s.cancel()
		s.clearLocked()
		return false
	default:
		return true
	}
}

func (s *localServer) clearLocked() { s.cmd, s.cell, s.cancel, s.done = nil, "", nil, nil }
