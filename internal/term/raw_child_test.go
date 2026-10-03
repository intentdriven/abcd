package term

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/term/ptytest"
)

// rawChildEnv names the case a re-executed test binary runs as a child on a
// pseudo-terminal's terminal end, instead of running the tests: the cases
// that end the process by a signal cannot run inside the test binary.
const rawChildEnv = "ABCD_TERM_RAW_CHILD"

func TestMain(m *testing.M) {
	if c := os.Getenv(rawChildEnv); c != "" {
		os.Exit(rawChild(c))
	}
	os.Exit(m.Run())
}

// rawChild runs one child case and returns its exit code; a case that
// expects to be ended by a signal returns 0 only when it survived.
func rawChild(c string) int {
	switch c {
	case "signal-in-hook":
		// A SIGTERM arrives while the signal goroutine is in a hook, and the
		// session ends (Restore) before the hook returns, so the goroutine
		// finds the session's end and the signal ready at once.
		inHook, release := make(chan struct{}), make(chan struct{})
		s, err := StartRaw(os.Stdin, Hooks{Resized: func() {
			close(inHook)
			<-release
		}})
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERROR", err)
			return 3
		}
		_ = syscall.Kill(os.Getpid(), syscall.SIGWINCH)
		<-inHook
		_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
		time.Sleep(20 * time.Millisecond)
		_ = s.Restore()
		close(release)
		time.Sleep(500 * time.Millisecond)
		fmt.Fprintln(os.Stderr, "SURVIVED")
		return 0
	}
	fmt.Fprintln(os.Stderr, "ERROR unknown case", c)
	return 3
}

// runRawChild runs case c on a fresh pseudo-terminal and returns its wait
// status and the terminal's attributes before it ran.
func runRawChild(t *testing.T, c string) (syscall.WaitStatus, *ptytest.Pty, syscall.Termios) {
	t.Helper()
	p := ptytest.Open(t, 80, 24)
	before := attrs(t, p.Terminal)
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), rawChildEnv+"="+c)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = p.Terminal, p.Terminal, p.Terminal
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		t.Fatalf("%s: the child did not exit; the terminal shows:\n%q", c, p.Output())
	}
	return cmd.ProcessState.Sys().(syscall.WaitStatus), p, before
}

// TestRawSessionRelaysASignalQueuedAsItEnds holds the relay against the
// session's end: a SIGTERM delivered while the signal goroutine is in a hook,
// with Restore in the same instant, still ends the process by SIGTERM, every
// time, and the terminal is restored. A goroutine that took the session's end
// first used to return with the signal unread, and the process lived on.
func TestRawSessionRelaysASignalQueuedAsItEnds(t *testing.T) {
	for i := range 20 {
		ws, p, before := runRawChild(t, "signal-in-hook")
		if !ws.Signaled() || ws.Signal() != syscall.SIGTERM {
			t.Fatalf("run %d: wait status %v, want killed by SIGTERM; the terminal shows:\n%q", i+1, ws, p.Output())
		}
		if after := attrs(t, p.Terminal); !ptytest.Same(after, before) {
			t.Fatalf("run %d: the terminal is %+v, want %+v", i+1, after, before)
		}
	}
}
