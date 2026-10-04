package cli

import (
	"github.com/intentdriven/abcd/internal/abcdhome"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/surface/cli/ask"
	"github.com/intentdriven/abcd/internal/term/ptytest"
)

// connectPtyChildEnv marks a child of this test binary that runs `ahoy
// connect` on a pseudo-terminal's terminal end instead of the tests.
const connectPtyChildEnv = "ABCD_CLI_CONNECT_PTY_CHILD"

// actAsConnectChild reports whether this test binary was started as the
// pseudo-terminal child: the marker AND an `ahoy connect` argument list, so
// a marker left in the environment never stands the suite down.
func actAsConnectChild(marker string, args []string) bool {
	return marker == "1" && len(args) > 2 && args[0] == "ahoy" && args[1] == "connect"
}

// ptyConnect is one `ahoy connect` run as a child on a fresh
// pseudo-terminal, every stream the terminal end, so the key is read hidden.
type ptyConnect struct {
	pty    *ptytest.Pty
	cmd    *exec.Cmd
	before syscall.Termios
	home   string
}

const ptyWait = 20 * time.Second

// ptySettle is how long the terminal must stay quiet before a check that the
// key never reached it reads the output (ptytest.Settled).
const ptySettle = 200 * time.Millisecond

func startPtyConnect(t *testing.T, args ...string) *ptyConnect {
	t.Helper()
	p := ptytest.Open(t, 80, 24)
	before, err := ptytest.Attrs(p.Terminal)
	if err != nil {
		t.Fatal(err)
	}
	if before.Lflag&syscall.ICANON == 0 || before.Lflag&syscall.ECHO == 0 {
		t.Fatalf("a fresh pseudo-terminal is not in cooked mode: lflag %#x", before.Lflag)
	}
	home := t.TempDir()
	cmd := exec.Command(os.Args[0], append([]string{"ahoy", "connect"}, args...)...)
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), connectPtyChildEnv+"=1", "HOME="+home, "TERM=xterm", "NO_COLOR=1",
		"ABCD_ACCESSIBLE=", "ACCESSIBLE=")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = p.Terminal, p.Terminal, p.Terminal
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	c := &ptyConnect{pty: p, cmd: cmd, before: before, home: home}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	return c
}

// waitHidden waits for the prompt line and then for echo off: the hidden
// read has begun, so what is typed next is the paste.
func (c *ptyConnect) waitHidden(t *testing.T) {
	t.Helper()
	c.pty.WaitFor(t, 0, "It is not shown.", ptyWait)
	deadline := time.Now().Add(ptyWait)
	for {
		a, err := ptytest.Attrs(c.pty.Terminal)
		if err != nil {
			t.Fatal(err)
		}
		if a.Lflag&syscall.ECHO == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("echo was never turned off; the terminal shows:\n%q", c.pty.Output())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (c *ptyConnect) exit(t *testing.T) syscall.WaitStatus {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- c.cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(ptyWait):
		_ = c.cmd.Process.Kill()
		t.Fatalf("the child did not exit; the terminal shows:\n%q", c.pty.Output())
	}
	return c.cmd.ProcessState.Sys().(syscall.WaitStatus)
}

func (c *ptyConnect) restored(t *testing.T, when string) {
	t.Helper()
	after, err := ptytest.Attrs(c.pty.Terminal)
	if err != nil {
		t.Fatal(err)
	}
	if after.Lflag&syscall.ECHO == 0 || after.Lflag&syscall.ICANON == 0 {
		t.Errorf("%s the terminal's echo is left off: lflag %#x", when, after.Lflag)
	}
	if !ptytest.Same(after, c.before) {
		t.Errorf("%s the terminal is %+v, want %+v", when, after, c.before)
	}
}

// TestConnectReadsTheKeyHiddenAtARealTerminal: on a real terminal the key
// pasted is read with echo off, never drawn, sent as the verification's key,
// stored, and the terminal is left as it was found.
func TestConnectReadsTheKeyHiddenAtARealTerminal(t *testing.T) {
	base, calls, auth := fakeProvider(t, 200, completionReply("typesafe/jev-1.13"))
	c := startPtyConnect(t, "openrouter", "--base-url", base, "--model", "typesafe/jev-1.13", "--home", "abcd")
	c.waitHidden(t)
	c.pty.Type(t, connectKey+"\r")
	ws := c.exit(t)
	if !ws.Exited() || ws.ExitStatus() != 0 {
		t.Fatalf("wait status %v\n%q", ws, c.pty.Output())
	}
	if calls.Load() != 1 || auth.Load() != "Bearer "+connectKey {
		t.Fatalf("verification: %d call(s), auth %v", calls.Load(), auth.Load())
	}
	if out := c.pty.Settled(t, ptySettle); strings.Contains(out, connectKey) {
		t.Fatalf("the key was drawn on the terminal:\n%q", out)
	}
	c.restored(t, "after the setup")
	raw, err := os.ReadFile(abcdhome.Path(c.home, "credentials.json"))
	if err != nil || !strings.Contains(string(raw), connectKey) {
		t.Fatalf("the key was not stored: %v", err)
	}
}

// TestHiddenKeyRestoresTerminalOnInterrupt (G5): Ctrl-C part-way through the
// paste exits 130 with ECHO set again and the terminal as it was found, no
// call made and nothing written; a SIGINT, SIGTERM or SIGHUP from outside
// ends the run by that signal with the terminal restored too.
func TestHiddenKeyRestoresTerminalOnInterrupt(t *testing.T) {
	t.Run("ctrl-c exits 130", func(t *testing.T) {
		base, calls, _ := fakeProvider(t, 200, completionReply("m"))
		c := startPtyConnect(t, "openrouter", "--base-url", base, "--model", "m", "--home", "abcd")
		c.waitHidden(t)
		c.pty.Type(t, "sk-half-a-pas")
		time.Sleep(50 * time.Millisecond)
		c.pty.Type(t, "\x03")
		ws := c.exit(t)
		if !ws.Exited() || ws.ExitStatus() != ask.ExitInterrupted {
			t.Errorf("Ctrl-C: wait status %v, want exit 130\n%q", ws, c.pty.Output())
		}
		c.restored(t, "after Ctrl-C")
		if calls.Load() != 0 {
			t.Errorf("an interrupted setup made %d call(s)", calls.Load())
		}
		if _, err := os.Lstat(c.home + "/.abcd"); err == nil {
			t.Error("an interrupted setup wrote under ~/.abcd")
		}
		if out := c.pty.Settled(t, ptySettle); strings.Contains(out, "half-a-pas") {
			t.Errorf("the paste was drawn:\n%q", out)
		}
	})
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP} {
		t.Run(sig.String()+" from outside", func(t *testing.T) {
			base, _, _ := fakeProvider(t, 200, completionReply("m"))
			c := startPtyConnect(t, "openrouter", "--base-url", base, "--model", "m", "--home", "abcd")
			c.waitHidden(t)
			c.pty.Type(t, "sk-half")
			time.Sleep(50 * time.Millisecond)
			if err := c.cmd.Process.Signal(sig); err != nil {
				t.Fatal(err)
			}
			ws := c.exit(t)
			if !ws.Signaled() || ws.Signal() != sig {
				t.Errorf("wait status %v, want ended by %v\n%q", ws, sig, c.pty.Output())
			}
			c.restored(t, "after "+sig.String())
		})
	}
}

// TestActAsConnectChildOnlyForAConnectRun: the marker alone never turns the
// test binary into abcd; only an `ahoy connect` argument list does.
func TestActAsConnectChildOnlyForAConnectRun(t *testing.T) {
	if !actAsConnectChild("1", []string{"ahoy", "connect", "x"}) {
		t.Error("the pseudo-terminal child must act as abcd")
	}
	for _, args := range [][]string{{"-test.run=TestX"}, {}, {"ahoy", "connect"}, {"ahoy", "install", "x"}} {
		if actAsConnectChild("1", args) {
			t.Errorf("the marker with %q must run the tests", args)
		}
	}
	if actAsConnectChild("", []string{"ahoy", "connect", "x"}) {
		t.Error("without the marker the binary always runs the tests")
	}
}

// TestConnectPicksAtARealTerminal (security review finding 2): the picker
// the setup draws at a terminal (terminalPick) on a real one. With no
// --model, after the key is pasted hidden, the service's models are listed
// with it and drawn; typing part of a name narrows the list and Enter picks,
// so the completion asks for the model picked and the block holds it. Ctrl-C
// at the list exits 130 with no completion and nothing written. Either way
// the terminal is left as it was found and the key is never drawn.
func TestConnectPicksAtARealTerminal(t *testing.T) {
	t.Run("a fragment and Enter pick", func(t *testing.T) {
		svc := newListingService(t, []string{"vendor/coder-large", "vendor/coder-small"}, 200)
		c := startPtyConnect(t, "example", "--base-url", svc.base(), "--home", "abcd")
		c.waitHidden(t)
		c.pty.Type(t, connectKey+"\r")
		mark := c.pty.WaitFor(t, 0, "lists 2 models", ptyWait)
		c.pty.Type(t, "small")
		c.pty.WaitFor(t, mark, "filter: small", ptyWait)
		c.pty.Type(t, "\r")
		ws := c.exit(t)
		if !ws.Exited() || ws.ExitStatus() != 0 {
			t.Fatalf("wait status %v\n%q", ws, c.pty.Output())
		}
		listAuth, _, chatModel := svc.seen()
		if len(listAuth) != 1 || listAuth[0] != "Bearer "+connectKey || len(chatModel) != 1 || chatModel[0] != "vendor/coder-small" {
			t.Fatalf("listed with %d request(s), completion asked for %q; want one keyed listing and the model picked", len(listAuth), chatModel)
		}
		raw, err := os.ReadFile(abcdhome.Path(c.home, "config.json"))
		if err != nil || !strings.Contains(string(raw), `"vendor/coder-small"`) || strings.Contains(string(raw), "coder-large") {
			t.Fatalf("the provider block does not hold the model picked: %v\n%s", err, raw)
		}
		if out := c.pty.Settled(t, ptySettle); strings.Contains(out, connectKey) {
			t.Fatalf("the key was drawn on the terminal:\n%q", out)
		}
		c.restored(t, "after the pick")
	})
	t.Run("ctrl-c at the list exits 130", func(t *testing.T) {
		svc := newListingService(t, []string{"vendor/coder-large", "vendor/coder-small"}, 200)
		c := startPtyConnect(t, "example", "--base-url", svc.base(), "--home", "abcd")
		c.waitHidden(t)
		c.pty.Type(t, connectKey+"\r")
		mark := c.pty.WaitFor(t, 0, "lists 2 models", ptyWait)
		c.pty.Type(t, "coder")
		c.pty.WaitFor(t, mark, "filter: coder", ptyWait)
		c.pty.Type(t, "\x03")
		ws := c.exit(t)
		if !ws.Exited() || ws.ExitStatus() != ask.ExitInterrupted {
			t.Fatalf("Ctrl-C at the list: wait status %v, want exit 130\n%q", ws, c.pty.Output())
		}
		if _, _, chatModel := svc.seen(); len(chatModel) != 0 {
			t.Errorf("an interrupted pick made a completion: %q", chatModel)
		}
		for _, name := range []string{"config.json", "credentials.json", "credential-homes.json"} {
			if _, err := os.Lstat(abcdhome.Path(c.home, name)); err == nil {
				t.Errorf("an interrupted pick wrote %s", abcdhome.Display(name))
			}
		}
		if out := c.pty.Settled(t, ptySettle); strings.Contains(out, connectKey) {
			t.Fatalf("the key was drawn on the terminal:\n%q", out)
		}
		c.restored(t, "after Ctrl-C at the list")
	})
}

// TestReadKeyRefusesATerminalSayingWhatIsTrue (security review finding 6):
// the piped-key reader still refuses a terminal (ahoy credential reads only a
// piped key), but no longer says the key "would be echoed": abcd reads a key
// hidden at a terminal elsewhere (ahoy connect), so the refusal names what
// this read takes, a pipe, and how to give it one.
func TestReadKeyRefusesATerminalSayingWhatIsTrue(t *testing.T) {
	p := ptytest.Open(t, 80, 24)
	_, err := readKey(p.Terminal)
	if err == nil {
		t.Fatal("readKey read from a terminal")
	}
	if strings.Contains(err.Error(), "echoed") {
		t.Errorf("the refusal says the key would be echoed: %v", err)
	}
	for _, want := range []string{"only when it is piped in", "stdin is a terminal", "pipe it in from a file or a variable"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not say %q: %v", want, err)
		}
	}
}
