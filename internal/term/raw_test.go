package term

import (
	"os"
	"syscall"
	"testing"

	"github.com/intentdriven/abcd/internal/term/ptytest"
)

func attrs(t *testing.T, f *os.File) syscall.Termios {
	t.Helper()
	a, err := ptytest.Attrs(f)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// TestRawSessionRestoresOnce holds RawSession on a pseudo-terminal: raw mode
// clears ICANON, ECHO and ISIG, Restore puts back exactly the attributes it
// found, and a second Restore is a no-op that changes nothing a person set in
// between.
func TestRawSessionRestoresOnce(t *testing.T) {
	p := ptytest.Open(t, 80, 24)
	before := attrs(t, p.Terminal)
	if before.Lflag&syscall.ICANON == 0 || before.Lflag&syscall.ECHO == 0 {
		t.Fatalf("a fresh pseudo-terminal is not in cooked mode: lflag %#x", before.Lflag)
	}
	s, err := StartRaw(p.Terminal, Hooks{})
	if err != nil {
		t.Fatal(err)
	}
	raw := attrs(t, p.Terminal)
	if raw.Lflag&(syscall.ICANON|syscall.ECHO|syscall.ISIG) != 0 {
		t.Errorf("raw mode left lflag %#x", raw.Lflag)
	}
	if err := s.Restore(); err != nil {
		t.Fatal(err)
	}
	if got := attrs(t, p.Terminal); !ptytest.Same(got, before) {
		t.Errorf("Restore left %+v, want %+v", got, before)
	}
	// A change made after the session ended is not undone by a late Restore.
	changed := before
	changed.Lflag &^= syscall.ECHO
	if err := ptytest.SetAttrs(p.Terminal, changed); err != nil {
		t.Fatal(err)
	}
	if err := s.Restore(); err != nil {
		t.Fatal(err)
	}
	if got := attrs(t, p.Terminal); !ptytest.Same(got, changed) {
		t.Errorf("a second Restore wrote the terminal: %+v, want %+v", got, changed)
	}
}

// TestRawSessionGuardRestoresAndRepanics holds the deferred guard: a panic
// inside the session restores the terminal, then panics on with the same
// value.
func TestRawSessionGuardRestoresAndRepanics(t *testing.T) {
	p := ptytest.Open(t, 80, 24)
	before := attrs(t, p.Terminal)
	var got any
	func() {
		defer func() { got = recover() }()
		s, err := StartRaw(p.Terminal, Hooks{})
		if err != nil {
			t.Fatal(err)
		}
		defer s.Guard()
		panic("inside the session")
	}()
	if got != "inside the session" {
		t.Errorf("the guard re-panicked with %v, want the original value", got)
	}
	if after := attrs(t, p.Terminal); !ptytest.Same(after, before) {
		t.Errorf("after the panic the terminal is %+v, want %+v", after, before)
	}
}

// TestRawSessionRefusedOffATerminal holds that raw mode on a descriptor that
// is not a terminal is an error the caller falls back on, never a session.
func TestRawSessionRefusedOffATerminal(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	if s, err := StartRaw(r, Hooks{}); err == nil {
		s.Restore()
		t.Fatal("raw mode on a pipe was granted")
	}
}
