package ptytest

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestSettledWaitsForBytesTheDrainHasNotRead: a child writes and exits before
// Settled starts, and its bytes stay in the kernel's terminal buffer while the
// drain is starved (held here, as a loaded machine can hold it), so the drain
// reads nothing for longer than the quiet window. Settled must not take that
// for silence: it returns only once the drain has read the bytes the kernel
// holds, so a check that something did NOT reach the terminal never passes on
// output it has not seen yet.
func TestSettledWaitsForBytesTheDrainHasNotRead(t *testing.T) {
	const quiet = 200 * time.Millisecond
	hold := make(chan struct{})
	p := openHeld(t, 80, 24, hold)
	cmd := exec.Command("/bin/sh", "-c", "printf settled-marker")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = p.Terminal, p.Terminal, p.Terminal
	if err := cmd.Run(); err != nil {
		t.Fatalf("the child: %v", err)
	}
	released := make(chan struct{})
	go func() {
		time.Sleep(3 * quiet)
		close(hold)
		close(released)
	}()
	out := p.Settled(t, quiet)
	<-released
	if !strings.Contains(out, "settled-marker") {
		t.Fatalf("Settled returned %q before the drain had read the child's bytes", out)
	}
}
