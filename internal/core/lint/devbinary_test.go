package lint_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// scripts/abcd, the source checkout's own abcd (iss-2610241240133009), driven
// against a stub `go` so no compile runs. The stub records how it was called
// and, for `build -o <path>`, writes a fake binary there that records its own
// arguments and working directory and exits with FAKE_EXIT.

const devBinaryStubGo = `#!/bin/sh
{
	printf 'cwd=%s\n' "$(pwd -P)"
	printf 'gotmpdir=%s\n' "$GOTMPDIR"
	printf 'args=%s\n' "$*"
} >> "$STUB_LOG"
if [ -n "$STUB_BUILD_FAIL" ]; then
	echo "stub go: $STUB_BUILD_FAIL" >&2
	exit 1
fi
[ "$1" = build ] && [ "$2" = -o ] || { echo "stub go: unexpected arguments: $*" >&2; exit 9; }
cat > "$3" <<'BIN'
#!/bin/sh
{
	printf 'bin-cwd=%s\n' "$(pwd -P)"
	for a in "$@"; do printf 'bin-arg=%s\n' "$a"; done
} >> "$STUB_LOG"
exit "${FAKE_EXIT:-0}"
BIN
chmod +x "$3"
`

type devBinaryRun struct {
	code   int
	stderr string
	log    string
}

// devBinaryCheckout lays out a checkout holding only scripts/abcd, the real
// script, and a subdirectory to call it from; it returns the checkout root
// with symlinks resolved, as the script's `pwd -P` reports it.
func devBinaryCheckout(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("scripts/abcd is a POSIX shell script; the checkout runs on macOS and Linux")
	}
	body, err := os.ReadFile(filepath.Join("..", "..", "..", "scripts", "abcd"))
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"scripts", "sub"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeExec(t, filepath.Join(root, "scripts", "abcd"), string(body))
	return root
}

func runDevBinary(t *testing.T, root string, env []string, args ...string) devBinaryRun {
	t.Helper()
	stubBin := filepath.Join(t.TempDir(), "stub")
	if err := os.MkdirAll(stubBin, 0o755); err != nil {
		t.Fatal(err)
	}
	writeExec(t, filepath.Join(stubBin, "go"), devBinaryStubGo)
	logPath := filepath.Join(t.TempDir(), "calls.log")

	cmd := exec.Command(filepath.Join(root, "scripts", "abcd"), args...)
	cmd.Dir = filepath.Join(root, "sub")
	cmd.Env = append([]string{
		"PATH=" + stubBin + string(os.PathListSeparator) + "/usr/bin" + string(os.PathListSeparator) + "/bin",
		"HOME=" + t.TempDir(),
		"STUB_LOG=" + logPath,
	}, env...)
	var errb bytes.Buffer
	cmd.Stderr = &errb
	code := 0
	if err := cmd.Run(); err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("running scripts/abcd: %v", err)
		}
		code = ee.ExitCode()
	}
	log, _ := os.ReadFile(logPath)
	return devBinaryRun{code, errb.String(), string(log)}
}

// TestTheCheckoutsAbcdKeepsTheBinarysExitCode is iss-2610241240133009's
// detector: `go run ./cmd/abcd` turned every nonzero exit into 1, so exit 2
// and exit 3 were unreadable to the agents the dogfooding rule sends to it.
// scripts/abcd builds the tree to the local tier and runs the binary itself,
// so each code arrives as the binary exited with it.
func TestTheCheckoutsAbcdKeepsTheBinarysExitCode(t *testing.T) {
	root := devBinaryCheckout(t)
	for _, want := range []int{0, 1, 2, 3} {
		got := runDevBinary(t, root, []string{"FAKE_EXIT=" + strconv.Itoa(want)}, "reserve", "iss-1")
		if got.code != want {
			t.Errorf("the binary exited %d; scripts/abcd exited %d, want the same code\nstderr:\n%s", want, got.code, got.stderr)
		}
	}
}

// The tree is built at the checkout root to the local tier, under a GOTMPDIR
// beside the binary so the new one is renamed into place rather than copied
// over a running one, and the verb then runs where the caller stands, with its
// arguments as given.
func TestTheCheckoutsAbcdBuildsToTheLocalTierAndRunsInPlace(t *testing.T) {
	root := devBinaryCheckout(t)
	got := runDevBinary(t, root, nil, "capture", "two words", "--severity", "minor")
	if got.code != 0 {
		t.Fatalf("scripts/abcd exited %d\nstderr:\n%s", got.code, got.stderr)
	}
	bindir := filepath.Join(root, ".abcd", ".work.local", "bin")
	for _, want := range []string{
		"cwd=" + root + "\n",
		"gotmpdir=" + filepath.Join(bindir, ".tmp") + "\n",
		"args=build -o " + filepath.Join(bindir, "abcd") + " ./cmd/abcd\n",
		"bin-cwd=" + filepath.Join(root, "sub") + "\n",
		"bin-arg=capture\nbin-arg=two words\nbin-arg=--severity\nbin-arg=minor\n",
	} {
		if !strings.Contains(got.log, want) {
			t.Errorf("the calls do not show %q:\n%s", want, got.log)
		}
	}
	if fi, err := os.Stat(filepath.Join(bindir, ".tmp")); err != nil || !fi.IsDir() {
		t.Errorf("GOTMPDIR %s does not exist as a directory (%v); go refuses a GOTMPDIR that does not", filepath.Join(bindir, ".tmp"), err)
	}
}

// A tree that does not build runs nothing: never the binary an earlier build
// left, which is the stale answer the dogfooding rule exists to rule out.
func TestTheCheckoutsAbcdRunsNoStaleBinaryWhenTheBuildFails(t *testing.T) {
	root := devBinaryCheckout(t)
	bindir := filepath.Join(root, ".abcd", ".work.local", "bin")
	if err := os.MkdirAll(bindir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeExec(t, filepath.Join(bindir, "abcd"), "#!/bin/sh\necho stale-binary-ran >> \"$STUB_LOG\"\nexit 0\n")
	got := runDevBinary(t, root, []string{"STUB_BUILD_FAIL=cmd/abcd does not compile"}, "status")
	if got.code == 0 {
		t.Errorf("a failed build exited 0; it must fail")
	}
	if strings.Contains(got.log, "stale-binary-ran") {
		t.Errorf("a failed build ran the binary an earlier build left:\n%s", got.log)
	}
	if !strings.Contains(got.stderr, "cmd/abcd does not compile") {
		t.Errorf("the build's own error is not on stderr:\n%s", got.stderr)
	}
}
