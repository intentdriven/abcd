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

	"github.com/intentdriven/abcd/internal/abcdrepo"
)

// scripts/abcd, the source checkout's own abcd (iss-2610241240133009), driven
// against a stub `go` so no compile runs. The stub records how it was called
// and, for `build -buildvcs=false -o <path>`, writes a fake binary there that
// records its own arguments and working directory and exits with FAKE_EXIT.

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
[ "$1" = build ] && [ "$2" = -buildvcs=false ] && [ "$3" = -o ] || { echo "stub go: unexpected arguments: $*" >&2; exit 9; }
cat > "$4" <<'BIN'
#!/bin/sh
{
	printf 'bin-cwd=%s\n' "$(pwd -P)"
	for a in "$@"; do printf 'bin-arg=%s\n' "$a"; done
} >> "$STUB_LOG"
exit "${FAKE_EXIT:-0}"
BIN
chmod +x "$4"
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
	return runDevBinaryAs(t, filepath.Join(root, "sub"), filepath.Join(root, "scripts", "abcd"), env, args...)
}

// runDevBinaryAs runs the script as path from dir; a relative path is
// resolved from dir, as a shell resolves it.
func runDevBinaryAs(t *testing.T, dir, path string, env []string, args ...string) devBinaryRun {
	t.Helper()
	stubBin := filepath.Join(t.TempDir(), "stub")
	if err := os.MkdirAll(stubBin, 0o755); err != nil {
		t.Fatal(err)
	}
	writeExec(t, filepath.Join(stubBin, "go"), devBinaryStubGo)
	logPath := filepath.Join(t.TempDir(), "calls.log")

	cmd := exec.Command(path, args...)
	cmd.Dir = dir
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

// The tree is built at the checkout root, unstamped as `go run` built it, to
// abcdrepo.CheckoutBuildRel, the path vintage.GoRunModule holds current; no
// GOTMPDIR is set inside the tree, whose transient go-build directories
// `gofmt -l .` would walk. The verb then runs where the caller stands, with its
// arguments as given.
func TestTheCheckoutsAbcdBuildsToTheLocalTierAndRunsInPlace(t *testing.T) {
	root := devBinaryCheckout(t)
	got := runDevBinary(t, root, nil, "capture", "two words", "--severity", "minor")
	if got.code != 0 {
		t.Fatalf("scripts/abcd exited %d\nstderr:\n%s", got.code, got.stderr)
	}
	for _, want := range []string{
		"cwd=" + root + "\n",
		"gotmpdir=\n",
		"args=build -buildvcs=false -o " + filepath.Join(root, filepath.FromSlash(abcdrepo.CheckoutBuildRel)) + " ./cmd/abcd\n",
		"bin-cwd=" + filepath.Join(root, "sub") + "\n",
		"bin-arg=capture\nbin-arg=two words\nbin-arg=--severity\nbin-arg=minor\n",
	} {
		if !strings.Contains(got.log, want) {
			t.Errorf("the calls do not show %q:\n%s", want, got.log)
		}
	}
}

// Called by its relative path from the root, the script finds the root the
// same way under an exported CDPATH: bash's cd consults CDPATH for a relative
// directory, so a CDPATH entry holding a scripts/ directory would otherwise
// take the cd there, and its printed name into the root.
func TestTheCheckoutsAbcdIgnoresCDPATH(t *testing.T) {
	root := devBinaryCheckout(t)
	decoy := t.TempDir()
	if err := os.MkdirAll(filepath.Join(decoy, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := runDevBinaryAs(t, root, filepath.Join("scripts", "abcd"), []string{"CDPATH=" + decoy}, "status")
	if got.code != 0 {
		t.Fatalf("scripts/abcd exited %d\nstderr:\n%s", got.code, got.stderr)
	}
	if want := "cwd=" + root + "\n"; !strings.Contains(got.log, want) {
		t.Errorf("with CDPATH=%s the build did not run at the root %s:\n%s", decoy, root, got.log)
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
