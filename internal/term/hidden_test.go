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
		if out := p.Settled(t, 200*time.Millisecond); strings.Contains(out, "hidden-0123") {
			t.Errorf("enter %q: the key was drawn on the terminal: %q", enter, out)
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

// TestReadHiddenRefusesALineAtTheReadersCap (security review finding 4):
// golang.org/x/term's line reader keeps at most 4096 runes and drops every key
// past them, so a line that reaches the cap may have been cut there. It is
// refused, never returned short, whether it is exactly the cap or longer, and
// the terminal is restored; a line one rune under the cap is read whole.
func TestReadHiddenRefusesALineAtTheReadersCap(t *testing.T) {
	for _, n := range []int{maxHiddenRunes - 1, maxHiddenRunes, maxHiddenRunes + 100} {
		p := ptytest.Open(t, 80, 24)
		before := attrs(t, p.Terminal)
		got := hiddenRead(p)
		waitEchoOff(t, p)
		typed := strings.Repeat("a", n)
		for i := 0; i < len(typed); i += 256 {
			p.Type(t, typed[i:min(i+256, len(typed))])
		}
		p.Type(t, "\r")
		v, err := result(t, got)
		switch {
		case n < maxHiddenRunes && (err != nil || v != typed):
			t.Errorf("%d runes: ReadHidden = %d runes, %v; want the line whole", n, len(v), err)
		case n >= maxHiddenRunes && (err == nil || v != "" || !strings.Contains(err.Error(), "4096")):
			t.Errorf("%d runes: ReadHidden = %d runes, %v; want nothing and the cap's refusal", n, len(v), err)
		}
		if after := attrs(t, p.Terminal); !ptytest.Same(after, before) {
			t.Errorf("%d runes: the read left %+v, want %+v", n, after, before)
		}
	}
}
