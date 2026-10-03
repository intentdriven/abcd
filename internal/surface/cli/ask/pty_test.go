package ask

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/term"
	"github.com/intentdriven/abcd/internal/term/ptytest"
)

// childEnv names the case a re-executed test binary runs as the child on a
// pseudo-terminal's terminal end, instead of running the tests.
const childEnv = "ABCD_ASK_PTY_CHILD"

func TestMain(m *testing.M) {
	if c := os.Getenv(childEnv); c != "" {
		os.Exit(ptyChild(c))
	}
	os.Exit(m.Run())
}

// ptyChild is the child: it puts the 300-name list on its stdin and stderr
// through the arrow-key loop, as an interview's front door would, and exits
// 130 on an interrupt, as the front door does. The "panic" case panics inside
// the loop on the key '!'.
func ptyChild(c string) int {
	if c == "panic" {
		keyHook = func(k Key) {
			if k.Kind == KeyRune && k.Rune == '!' {
				panic("forced inside the answer loop")
			}
		}
	}
	t := Terminal{In: os.Stdin, Out: os.Stderr, Getenv: os.Getenv, Mode: term.Mono, List: layered.InterviewListArrows}
	got, err := t.Put(longList(300))
	switch {
	case errors.Is(err, ErrInterrupted):
		return 130
	case err != nil:
		fmt.Fprintf(os.Stderr, "ERROR %v\n", err)
		return 3
	}
	fmt.Fprintf(os.Stderr, "CHOSE %s\n", got[0].Value)
	return 0
}

// child is one run of the test binary as the child, on a fresh
// pseudo-terminal.
type child struct {
	pty    *ptytest.Pty
	cmd    *exec.Cmd
	before syscall.Termios
}

const wait = 20 * time.Second

func startChild(t *testing.T, c string) *child {
	t.Helper()
	p := ptytest.Open(t, 80, 24)
	before, err := ptytest.Attrs(p.Terminal)
	if err != nil {
		t.Fatal(err)
	}
	if before.Lflag&syscall.ICANON == 0 || before.Lflag&syscall.ECHO == 0 {
		t.Fatalf("a fresh pseudo-terminal is not in cooked mode: lflag %#x", before.Lflag)
	}
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), childEnv+"="+c, "TERM=xterm", "LANG=en_US.UTF-8",
		"ABCD_ACCESSIBLE=", "ACCESSIBLE=")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = p.Terminal, p.Terminal, p.Terminal
	// Its own process group in this session: a signal sent to it reaches it
	// alone, and Ctrl-Z's stop is not discarded as an orphaned group's would be.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	ch := &child{pty: p, cmd: cmd, before: before}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	// The list is drawn after raw mode is entered, so Later on screen means
	// the keys typed next reach the loop as keys.
	p.WaitFor(t, 0, "301. Decide later", wait)
	return ch
}

// exit waits for the child and returns its wait status.
func (c *child) exit(t *testing.T) syscall.WaitStatus {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- c.cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(wait):
		_ = c.cmd.Process.Kill()
		t.Fatalf("the child did not exit; the terminal shows:\n%q", c.pty.Output())
	}
	return c.cmd.ProcessState.Sys().(syscall.WaitStatus)
}

// restored asserts the terminal's attributes equal those before the child.
func (c *child) restored(t *testing.T, when string) {
	t.Helper()
	after, err := ptytest.Attrs(c.pty.Terminal)
	if err != nil {
		t.Fatal(err)
	}
	if after.Lflag&syscall.ICANON == 0 || after.Lflag&syscall.ECHO == 0 {
		t.Errorf("%s the terminal is left raw: lflag %#x", when, after.Lflag)
	}
	if !ptytest.Same(after, c.before) {
		t.Errorf("%s the terminal is %+v, want %+v", when, after, c.before)
	}
}

// TestLongListRestoresTerminalOnInterrupt is B6's restore half
// (spc-2610030911534855): a child on a pseudo-terminal types "claude" into
// the 300-name list and is then interrupted, terminated from outside, made to
// panic inside the loop, or suspended; on every exit the terminal's attributes
// equal those before it, ICANON and ECHO set.
func TestLongListRestoresTerminalOnInterrupt(t *testing.T) {
	t.Run("ctrl-c exits 130", func(t *testing.T) {
		c := startChild(t, "interrupt")
		c.pty.Type(t, "claude")
		c.pty.WaitFor(t, 0, "filter: claude", wait)
		c.pty.Type(t, "\x03")
		ws := c.exit(t)
		if !ws.Exited() || ws.ExitStatus() != 130 {
			t.Errorf("Ctrl-C: wait status %v, want exit 130\n%q", ws, c.pty.Output())
		}
		c.restored(t, "after Ctrl-C")
	})
	for _, sig := range []syscall.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP} {
		t.Run(sig.String()+" from outside", func(t *testing.T) {
			c := startChild(t, "signal")
			c.pty.Type(t, "claude")
			c.pty.WaitFor(t, 0, "filter: claude", wait)
			if err := c.cmd.Process.Signal(sig); err != nil {
				t.Fatal(err)
			}
			ws := c.exit(t)
			if !ws.Signaled() || ws.Signal() != sig {
				t.Errorf("wait status %v, want killed by %v\n%q", ws, sig, c.pty.Output())
			}
			c.restored(t, "after "+sig.String())
		})
	}
	t.Run("a panic inside the loop", func(t *testing.T) {
		c := startChild(t, "panic")
		c.pty.Type(t, "claude")
		c.pty.WaitFor(t, 0, "filter: claude", wait)
		c.pty.Type(t, "!")
		ws := c.exit(t)
		if !ws.Exited() || ws.ExitStatus() == 0 {
			t.Errorf("wait status %v, want a failed exit", ws)
		}
		if out := c.pty.Output(); !strings.Contains(out, "forced inside the answer loop") {
			t.Errorf("the panic did not reach the terminal:\n%q", out)
		}
		c.restored(t, "after the panic")
	})
	t.Run("ctrl-z restores, and SIGCONT re-enters and redraws", func(t *testing.T) {
		c := startChild(t, "suspend")
		c.pty.Type(t, "claude")
		mark := c.pty.WaitFor(t, 0, "filter: claude", wait)
		c.pty.Type(t, "\x1a")
		var ws syscall.WaitStatus
		pid, err := syscall.Wait4(c.cmd.Process.Pid, &ws, syscall.WUNTRACED, nil)
		if err != nil || pid != c.cmd.Process.Pid || !ws.Stopped() {
			t.Fatalf("Ctrl-Z: wait4 = %d %v %v, want the child stopped", pid, ws, err)
		}
		c.restored(t, "while stopped")
		if err := c.cmd.Process.Signal(syscall.SIGCONT); err != nil {
			t.Fatal(err)
		}
		mark = c.pty.WaitFor(t, mark, "filter: claude", wait)
		raw, err := ptytest.Attrs(c.pty.Terminal)
		if err != nil {
			t.Fatal(err)
		}
		if raw.Lflag&syscall.ICANON != 0 {
			t.Errorf("after SIGCONT the terminal is not raw again: lflag %#x", raw.Lflag)
		}
		c.pty.Type(t, "\r")
		c.pty.WaitFor(t, mark, "CHOSE anthropic/claude-0", wait)
		if ws := c.exit(t); !ws.Exited() || ws.ExitStatus() != 0 {
			t.Errorf("wait status %v, want exit 0", ws)
		}
		c.restored(t, "after the choice")
	})
}

// TestArrowListAnswersOnATerminal drives the arrow-key loop on a
// pseudo-terminal end to end: Down then Enter chooses the second name, the
// list collapses to the one plain line, the drawing moves the cursor with
// cursor-up and erase-line alone and never takes the alternate screen, and a
// resize redraws at the new width.
func TestArrowListAnswersOnATerminal(t *testing.T) {
	c := startChild(t, "answer")
	c.pty.Type(t, "\x1b[B")
	mark := c.pty.WaitFor(t, 0, "\x1b[", wait)
	if err := ptytest.SetSize(c.pty.Terminal, 120, 30); err != nil {
		t.Fatal(err)
	}
	if err := c.cmd.Process.Signal(syscall.SIGWINCH); err != nil {
		t.Fatal(err)
	}
	mark = c.pty.WaitFor(t, mark, "Which model should abcd use?", wait)
	c.pty.Type(t, "\r")
	c.pty.WaitFor(t, mark, "CHOSE openai/gpt-1", wait)
	if ws := c.exit(t); !ws.Exited() || ws.ExitStatus() != 0 {
		t.Errorf("wait status %v, want exit 0", ws)
	}
	c.restored(t, "after the choice")
	out := c.pty.Output()
	if !strings.Contains(out, "› Setup Q4: GPT Model 1\r\n") {
		t.Errorf("no collapsed line for the choice:\n%q", out)
	}
	for _, seq := range escapeSequences(out) {
		switch {
		case seq == "\x1b[2K":
		case strings.HasSuffix(seq, "A") && strings.TrimLeft(seq[2:len(seq)-1], "0123456789") == "":
		default:
			t.Errorf("the loop wrote %q; only cursor-up and erase-line are drawn in Mono", seq)
		}
	}
}

// escapeSequences lists every CSI sequence in s.
func escapeSequences(s string) []string {
	var out []string
	for i := 0; i < len(s); i++ {
		if s[i] != 0x1b {
			continue
		}
		j := i + 1
		if j < len(s) && s[j] == '[' {
			j++
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
				j++
			}
		}
		if j < len(s) {
			j++
		}
		out = append(out, s[i:j])
		i = j - 1
	}
	return out
}
