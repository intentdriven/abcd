package term

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestIsTerminalRefusesWhatIsNotATerminal pins that the one canonical check
// answers true only for a terminal. /dev/null is a character device, so a
// device-mode test called it a terminal and every consent gate asked a
// question no one could answer; a closed fd 0 is reopened on /dev/null by the
// Go runtime, so it is the same case.
func TestIsTerminalRefusesWhatIsNotATerminal(t *testing.T) {
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	reg, err := os.Create(filepath.Join(t.TempDir(), "regular"))
	if err != nil {
		t.Fatal(err)
	}
	defer reg.Close()
	closed, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	closed.Close()

	for name, f := range map[string]*os.File{
		"the null device":    null,
		"a pipe's read end":  r,
		"a pipe's write end": w,
		"a regular file":     reg,
		"a closed file":      closed,
		"no file at all":     nil,
	} {
		if IsTerminal(f) {
			t.Errorf("IsTerminal(%s) = true, want false: it is not a terminal", name)
		}
	}
}

// ptyChildEnv marks the re-executed test binary running under script(1).
const ptyChildEnv = "ABCD_TERM_PTY_CHILD"

// TestIsTerminalAnswersTrueOnAPty runs this test binary again under script(1),
// which hands it a pseudo-terminal, so the true side of the check is proved
// against a real terminal and not only asserted. It runs on macOS, whose
// script(1) takes the command as trailing arguments; util-linux script takes a
// shell string with -c and behaves differently on a non-terminal stdin across
// versions, so on Linux the true side rests on the same termios get
// (TCGETS) the platform's isatty(3) makes.
func TestIsTerminalAnswersTrueOnAPty(t *testing.T) {
	if os.Getenv(ptyChildEnv) == "1" {
		if IsTerminal(os.Stdin) && IsTerminal(os.Stdout) {
			os.Stdout.WriteString("PTY-IS-TERMINAL\n")
		} else {
			os.Stdout.WriteString("PTY-NOT-TERMINAL\n")
		}
		return
	}
	if runtime.GOOS != "darwin" {
		t.Skip("the pty probe runs on macOS's script(1); see the test comment")
	}
	script, err := exec.LookPath("script")
	if err != nil {
		t.Skip("script(1) is not available")
	}
	cmd := exec.Command(script, "-q", os.DevNull, os.Args[0], "-test.run=^TestIsTerminalAnswersTrueOnAPty$", "-test.count=1")
	cmd.Env = append(os.Environ(), ptyChildEnv+"=1")
	stdin, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	cmd.Stdin = stdin
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script(1) failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "PTY-IS-TERMINAL") {
		t.Fatalf("under a pseudo-terminal IsTerminal answered false:\n%s", out)
	}
}
