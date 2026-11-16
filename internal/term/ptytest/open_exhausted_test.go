package ptytest

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"testing"
)

// recTB is a testing.TB that records a skip or a failure instead of ending
// the test it stands in for.
type recTB struct {
	testing.TB
	skipped, failed string
}

func (r *recTB) Helper() {}

func (r *recTB) Fatalf(format string, args ...any) {
	r.failed = fmt.Sprintf(format, args...)
	runtime.Goexit()
}

func (r *recTB) Skipf(format string, args ...any) {
	r.skipped = fmt.Sprintf(format, args...)
	runtime.Goexit()
}

// openWith runs openHeld on a recTB over t with openPty standing in as fn,
// and reports what it recorded and the pseudo-terminal it opened, if any.
func openWith(t *testing.T, fn func() (*os.File, *os.File, error)) (*recTB, *Pty) {
	t.Helper()
	old, oldWait := openPty, exhaustedWait
	openPty, exhaustedWait = fn, 0
	t.Cleanup(func() { openPty, exhaustedWait = old, oldWait })
	r := &recTB{TB: t}
	var p *Pty
	done := make(chan struct{})
	go func() {
		defer close(done)
		p = openHeld(r, 80, 24, nil)
	}()
	<-done
	return r, p
}

// TestOpenSkipsWhenTheSystemIsOutOfPseudoTerminals (iss-2610151050413162): a
// system out of pseudo-terminals, which a loaded machine's open of /dev/ptmx
// reports as "device not configured" (ENXIO) or as no space or try again,
// skips the test with the reason once a few tries have failed, rather than
// failing a whole preflight on a transient shortage; a try that succeeds
// opens the terminal; and any other error still fails the test.
func TestOpenSkipsWhenTheSystemIsOutOfPseudoTerminals(t *testing.T) {
	for _, errno := range []syscall.Errno{syscall.ENXIO, syscall.ENOSPC, syscall.EAGAIN} {
		t.Run(errno.Error(), func(t *testing.T) {
			tries := 0
			r, _ := openWith(t, func() (*os.File, *os.File, error) {
				tries++
				return nil, nil, fmt.Errorf("/dev/ptmx: %w", errno)
			})
			if r.failed != "" || r.skipped == "" {
				t.Fatalf("an exhausted system must skip, not fail: skipped %q, failed %q", r.skipped, r.failed)
			}
			if tries != exhaustedTries {
				t.Errorf("tried %d times before skipping, want %d", tries, exhaustedTries)
			}
		})
	}
	t.Run("a later try opens it", func(t *testing.T) {
		tries := 0
		var real error
		r, p := openWith(t, func() (*os.File, *os.File, error) {
			if tries++; tries < exhaustedTries {
				return nil, nil, fmt.Errorf("/dev/ptmx: %w", syscall.ENXIO)
			}
			m, s, err := open()
			real = err
			return m, s, err
		})
		// The last try is a real open: a machine that refuses it a
		// pseudo-terminal for the moment skips this case too.
		if r.skipped != "" && outOfPtys(real) {
			t.Skipf("the real open was refused a pseudo-terminal: %v", real)
		}
		if r.failed != "" || r.skipped != "" || p == nil {
			t.Fatalf("a try that opens the terminal must be used: skipped %q, failed %q", r.skipped, r.failed)
		}
	})
	t.Run("another error fails", func(t *testing.T) {
		tries := 0
		r, _ := openWith(t, func() (*os.File, *os.File, error) {
			tries++
			return nil, nil, fmt.Errorf("/dev/ptmx: %w", syscall.EACCES)
		})
		if r.failed == "" || r.skipped != "" || tries != 1 {
			t.Fatalf("a refusal that is not a shortage must fail at once: tries %d, skipped %q, failed %q", tries, r.skipped, r.failed)
		}
	})
}
