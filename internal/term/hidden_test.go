package term

import (
	"errors"
	"io"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/term/ptytest"
)

// hiddenRead runs ReadHidden on p's terminal end in the background, and
// returns what it read once it ends.
func hiddenRead(p *ptytest.Pty) <-chan [2]any {
	got := make(chan [2]any, 1)
	go func() {
		v, err := ReadHidden(p.Terminal, p.Terminal)
		got <- [2]any{v, err}
	}()
	return got
}

// waitEchoOff waits until the terminal's echo is off: the read has begun.
func waitEchoOff(t *testing.T, p *ptytest.Pty) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if a := attrs(t, p.Terminal); a.Lflag&syscall.ECHO == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the hidden read never turned echo off")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func result(t *testing.T, got <-chan [2]any) (string, error) {
	t.Helper()
	select {
	case r := <-got:
		err, _ := r[1].(error)
		return r[0].(string), err
	case <-time.After(10 * time.Second):
		t.Fatal("the hidden read did not end")
	}
	return "", nil
}

// TestReadHiddenReadsWithoutEcho (spc-2610031241482088, "The key on hidden
// input"): a key pasted at the terminal is read whole, never drawn back, and
// the terminal is left exactly as the read found it.
func TestReadHiddenReadsWithoutEcho(t *testing.T) {
	const pasted = "hidden-0123456789-pasted-value"
	for _, enter := range []string{"\r", "\n", "\r\n"} {
		p := ptytest.Open(t, 80, 24)
		before := attrs(t, p.Terminal)
		got := hiddenRead(p)
		waitEchoOff(t, p)
		p.Type(t, pasted+enter)
		v, err := result(t, got)
		if err != nil || v != pasted {
			t.Fatalf("enter %q: ReadHidden = %q, %v", enter, v, err)
		}
		if after := attrs(t, p.Terminal); !ptytest.Same(after, before) {
			t.Errorf("enter %q: the read left %+v, want %+v", enter, after, before)
		}
		time.Sleep(50 * time.Millisecond)
		if strings.Contains(p.Output(), "hidden-0123") {
			t.Errorf("enter %q: the key was drawn on the terminal: %q", enter, p.Output())
		}
	}
}

// TestReadHiddenRestoresOnInterrupt: Ctrl-C part-way through a paste ends the
// read with ErrInterrupted, nothing read, and the terminal's echo back on.
func TestReadHiddenRestoresOnInterrupt(t *testing.T) {
	p := ptytest.Open(t, 80, 24)
	before := attrs(t, p.Terminal)
	got := hiddenRead(p)
	waitEchoOff(t, p)
	p.Type(t, "half-a-ke")
	p.Type(t, "\x03")
	v, err := result(t, got)
	if !errors.Is(err, ErrInterrupted) || v != "" {
		t.Fatalf("Ctrl-C: ReadHidden = %q, %v; want nothing and ErrInterrupted", v, err)
	}
	after := attrs(t, p.Terminal)
	if after.Lflag&syscall.ECHO == 0 || !ptytest.Same(after, before) {
		t.Errorf("after Ctrl-C the terminal is %+v, want %+v", after, before)
	}
}

// TestReadHiddenEndsOnAnEmptyLine: Enter on nothing, or Ctrl-D, reads
// nothing; the caller refuses the empty answer.
func TestReadHiddenEndsOnAnEmptyLine(t *testing.T) {
	for _, keys := range []string{"\r", "\x04"} {
		p := ptytest.Open(t, 80, 24)
		got := hiddenRead(p)
		waitEchoOff(t, p)
		p.Type(t, keys)
		if v, err := result(t, got); v != "" || err != nil {
			t.Errorf("%q: ReadHidden = %q, %v; want nothing read and no error", keys, v, err)
		}
	}
}

// TestReadHiddenRefusesOffATerminal: a file that is not a terminal is
// refused, and nothing is read from it.
func TestReadHiddenRefusesOffATerminal(t *testing.T) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := ReadHidden(f, io.Discard); err == nil {
		t.Fatal("ReadHidden read from /dev/null")
	}
}
