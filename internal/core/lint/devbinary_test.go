package lint_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdrepo"
)

// scripts/abcd.sh, the source checkout's own abcd (iss-2610241240133009), driven
// against a stub `go` so no compile runs. The stub records how it was called
// and, for `build -buildvcs=false -o <path>`, removes <path> and writes a fake
// binary there, as go does, unless STUB_UP_TO_DATE says the tree is unchanged;
// then it records what the binary's own path holds at that moment. The fake
// records its own arguments, working directory and the script's mark, and exits
// with FAKE_EXIT.

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
[ -n "$STUB_UP_TO_DATE" ] && exit 0
rm -f "$4"
cat > "$4" <<'BIN'
#!/bin/sh
{
	printf 'bin-cwd=%s\n' "$(pwd -P)"
	printf 'bin-mark=%s\n' "$@MARK@"
	for a in "$@"; do printf 'bin-arg=%s\n' "$a"; done
} >> "$STUB_LOG"
exit "${FAKE_EXIT:-0}"
BIN
chmod +x "$4"
printf 'bin-at-link-end=%s\n' "$(cat "$STUB_BIN" 2>/dev/null)" >> "$STUB_LOG"
`

// devBinaryFake is the binary the stub writes, for a test that lays one down
// as an earlier build's.
func devBinaryFake(stubGo string) string {
	start := strings.Index(stubGo, "<<'BIN'\n") + len("<<'BIN'\n")
	return stubGo[start:strings.Index(stubGo, "BIN\nchmod")]
}

type devBinaryRun struct {
	code   int
	stderr string
	log    string
}

// devBinaryCheckout lays out a checkout holding the real scripts/abcd.sh, a
// cmd/abcd/main.go to mark it a source checkout, and a subdirectory to call it
// from; it returns the checkout root with symlinks resolved, as the script's
// `pwd -P` reports it.
func devBinaryCheckout(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("scripts/abcd.sh is a POSIX shell script; the checkout runs on macOS and Linux")
	}
	body, err := os.ReadFile(filepath.Join("..", "..", "..", "scripts", "abcd.sh"))
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"scripts", "sub", filepath.Join("cmd", "abcd")} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "abcd", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeExec(t, filepath.Join(root, "scripts", "abcd.sh"), string(body))
	return root
}

// devBinaryPath is where the script builds root's abcd.
func devBinaryPath(root string) string {
	return filepath.Join(root, filepath.FromSlash(abcdrepo.CheckoutBuildRel))
}

func runDevBinary(t *testing.T, root string, env []string, args ...string) devBinaryRun {
	t.Helper()
	return runDevBinaryAs(t, filepath.Join(root, "sub"), filepath.Join(root, "scripts", "abcd.sh"), root, env, args...)
}

// runDevBinaryAs runs the script as path from dir, for the checkout at root; a
// relative path is resolved from dir, as a shell resolves it.
func runDevBinaryAs(t *testing.T, dir, path, root string, env []string, args ...string) devBinaryRun {
	t.Helper()
	stubBin := filepath.Join(t.TempDir(), "stub")
	if err := os.MkdirAll(stubBin, 0o755); err != nil {
		t.Fatal(err)
	}
	writeExec(t, filepath.Join(stubBin, "go"), strings.ReplaceAll(devBinaryStubGo, "@MARK@", abcdrepo.CheckoutBuildEnv))
	logPath := filepath.Join(t.TempDir(), "calls.log")

	cmd := exec.Command(path, args...)
	cmd.Dir = dir
	cmd.Env = append([]string{
		"PATH=" + stubBin + string(os.PathListSeparator) + "/usr/bin" + string(os.PathListSeparator) + "/bin",
		"HOME=" + t.TempDir(),
		"TMPDIR=" + t.TempDir(),
		"STUB_LOG=" + logPath,
		"STUB_BIN=" + devBinaryPath(root),
	}, env...)
	var errb bytes.Buffer
	cmd.Stderr = &errb
	code := 0
	if err := cmd.Run(); err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("running scripts/abcd.sh: %v", err)
		}
		code = ee.ExitCode()
	}
	log, _ := os.ReadFile(logPath)
	return devBinaryRun{code, errb.String(), string(log)}
}

// devBinaryLeftovers lists what the script left beside the binary: the private
// names it builds to must not outlive the run.
func devBinaryLeftovers(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(devBinaryPath(root)))
	if err != nil {
		return nil
	}
	var left []string
	for _, e := range entries {
		if e.Name() != filepath.Base(devBinaryPath(root)) {
			left = append(left, e.Name())
		}
	}
	return left
}

// TestTheCheckoutsAbcdKeepsTheBinarysExitCode is iss-2610241240133009's
// detector: `go run ./cmd/abcd` turned every nonzero exit into 1, so exit 2
// and exit 3 were unreadable to the agents the dogfooding rule sends to it.
// scripts/abcd.sh builds the tree to the local tier and runs the binary itself,
// so each code arrives as the binary exited with it.
func TestTheCheckoutsAbcdKeepsTheBinarysExitCode(t *testing.T) {
	root := devBinaryCheckout(t)
	for _, want := range []int{0, 1, 2, 3} {
		got := runDevBinary(t, root, []string{"FAKE_EXIT=" + strconv.Itoa(want)}, "reserve", "iss-1")
		if got.code != want {
			t.Errorf("the binary exited %d; scripts/abcd.sh exited %d, want the same code\nstderr:\n%s", want, got.code, got.stderr)
		}
	}
}

// The tree is built at the checkout root, unstamped as `go run` built it, to a
// private name beside abcdrepo.CheckoutBuildRel that is then renamed into place;
// no GOTMPDIR is set inside the tree, whose transient go-build directories
// `gofmt -l .` would walk. The verb then runs where the caller stands, with its
// arguments as given, and with abcdrepo.CheckoutBuildEnv naming the binary, the
// mark vintage.CheckoutBuild holds a fresh build by.
func TestTheCheckoutsAbcdBuildsToTheLocalTierAndRunsInPlace(t *testing.T) {
	root := devBinaryCheckout(t)
	got := runDevBinary(t, root, nil, "capture", "two words", "--severity", "minor")
	if got.code != 0 {
		t.Fatalf("scripts/abcd.sh exited %d\nstderr:\n%s", got.code, got.stderr)
	}
	for _, want := range []string{
		"cwd=" + root + "\n",
		"gotmpdir=\n",
		"bin-cwd=" + filepath.Join(root, "sub") + "\n",
		"bin-mark=" + devBinaryPath(root) + "\n",
		"bin-arg=capture\nbin-arg=two words\nbin-arg=--severity\nbin-arg=minor\n",
	} {
		if !strings.Contains(got.log, want) {
			t.Errorf("the calls do not show %q:\n%s", want, got.log)
		}
	}
	build := regexp.MustCompile(`(?m)^args=build -buildvcs=false -o ` + regexp.QuoteMeta(devBinaryPath(root)) + `\.[0-9]+ \./cmd/abcd$`)
	if !build.MatchString(got.log) {
		t.Errorf("the build did not link to a private name beside %s:\n%s", devBinaryPath(root), got.log)
	}
	if left := devBinaryLeftovers(t, root); len(left) != 0 {
		t.Errorf("the run left %v beside the binary", left)
	}
}

// A relink never touches the binary another call may be running or about to
// run: the link writes to a private name, so the old binary is still whole at
// its path when the link ends, and the new one replaces it by rename. An
// unchanged tree relinks nothing and runs the binary already there.
func TestTheCheckoutsAbcdReplacesTheBinaryWhole(t *testing.T) {
	root := devBinaryCheckout(t)
	bin := devBinaryPath(root)
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	old := "#!/bin/sh\n# old build\necho old-binary-ran >> \"$STUB_LOG\"\n"
	writeExec(t, bin, old)
	held := filepath.Join(t.TempDir(), "held")
	if err := os.Link(bin, held); err != nil {
		t.Skipf("no hard links here: %v", err)
	}

	got := runDevBinary(t, root, nil, "status")
	if got.code != 0 {
		t.Fatalf("scripts/abcd.sh exited %d\nstderr:\n%s", got.code, got.stderr)
	}
	if !strings.Contains(got.log, "bin-at-link-end=#!/bin/sh\n# old build\n") {
		t.Errorf("the link wrote to the binary's own path, so a call starting then could run it half written:\n%s", got.log)
	}
	if regexp.MustCompile(`(?m)^old-binary-ran$`).MatchString(got.log) || !strings.Contains(got.log, "bin-arg=status\n") {
		t.Errorf("the relinked binary did not run:\n%s", got.log)
	}
	if body, _ := os.ReadFile(held); string(body) != old {
		t.Errorf("the old binary's file was written over in place; it now holds:\n%s", body)
	}

	writeExec(t, bin, devBinaryFake(strings.ReplaceAll(devBinaryStubGo, "@MARK@", abcdrepo.CheckoutBuildEnv)))
	got = runDevBinary(t, root, []string{"STUB_UP_TO_DATE=1"}, "peers")
	if got.code != 0 || !strings.Contains(got.log, "bin-arg=peers\n") {
		t.Errorf("with the tree unchanged the binary already there did not run (exit %d):\n%s\nstderr:\n%s", got.code, got.log, got.stderr)
	}
	if left := devBinaryLeftovers(t, root); len(left) != 0 {
		t.Errorf("the run left %v beside the binary", left)
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
	got := runDevBinaryAs(t, root, filepath.Join("scripts", "abcd.sh"), root, []string{"CDPATH=" + decoy}, "status")
	if got.code != 0 {
		t.Fatalf("scripts/abcd.sh exited %d\nstderr:\n%s", got.code, got.stderr)
	}
	if want := "cwd=" + root + "\n"; !strings.Contains(got.log, want) {
		t.Errorf("with CDPATH=%s the build did not run at the root %s:\n%s", decoy, root, got.log)
	}
}

// Run through a symlink to it, from a directory on PATH say, the script still
// builds the checkout it belongs to, not the link's.
func TestTheCheckoutsAbcdFollowsASymlinkToItself(t *testing.T) {
	root := devBinaryCheckout(t)
	// Canonical like root: the kernel resolves the relative link from the
	// link's real directory, and on macOS the temporary directory sits behind
	// /var -> /private/var, so a link computed from the spelled path lands one
	// level off.
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	links := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(links, 0o755); err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(links, filepath.Join(root, "scripts", "abcd.sh"))
	if err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(links, "abcd")
	if err := os.Symlink(rel, link); err != nil {
		t.Fatal(err)
	}
	got := runDevBinaryAs(t, filepath.Join(root, "sub"), link, root, nil, "status")
	if got.code != 0 {
		t.Fatalf("scripts/abcd.sh through %s exited %d\nstderr:\n%s", link, got.code, got.stderr)
	}
	if want := "cwd=" + root + "\n"; !strings.Contains(got.log, want) {
		t.Errorf("through a symlink the build did not run at the root %s:\n%s", root, got.log)
	}
}

// A copy of the script outside a source checkout of abcd builds nothing and
// says why, exiting 2: there is no tree for it to build.
func TestTheCheckoutsAbcdRefusesOutsideASourceCheckout(t *testing.T) {
	root := devBinaryCheckout(t)
	if err := os.RemoveAll(filepath.Join(root, "cmd")); err != nil {
		t.Fatal(err)
	}
	got := runDevBinary(t, root, nil, "status")
	if got.code != 2 {
		t.Errorf("outside a source checkout scripts/abcd.sh exited %d, want 2", got.code)
	}
	if got.log != "" {
		t.Errorf("outside a source checkout it still ran something:\n%s", got.log)
	}
	if !strings.Contains(got.stderr, "cmd/abcd/main.go") {
		t.Errorf("the refusal does not say what is missing:\n%s", got.stderr)
	}
}

// The refusal names the folder under the home directory in its ~ form, as
// every abcd surface prints a home path, never with the account's home spelt
// out.
func TestTheCheckoutsAbcdNamesARefusedFolderInItsHomeForm(t *testing.T) {
	root := devBinaryCheckout(t)
	if err := os.RemoveAll(filepath.Join(root, "cmd")); err != nil {
		t.Fatal(err)
	}
	got := runDevBinary(t, root, []string{"HOME=" + filepath.Dir(root)}, "status")
	if want := "~/" + filepath.Base(root) + " holds no cmd/abcd/main.go"; !strings.Contains(got.stderr, want) {
		t.Errorf("the refusal does not name the folder as %q:\n%s", want, got.stderr)
	}
	if strings.Contains(got.stderr, root) {
		t.Errorf("the refusal spells out the home directory:\n%s", got.stderr)
	}
}

// A tree that does not build runs nothing: never the binary an earlier build
// left, which is the stale answer the dogfooding rule exists to rule out. It
// exits 2, so a broken build never reads as the 1 of a guard's block, and it
// leaves no private build name behind.
func TestTheCheckoutsAbcdRunsNoStaleBinaryWhenTheBuildFails(t *testing.T) {
	root := devBinaryCheckout(t)
	bin := devBinaryPath(root)
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	writeExec(t, bin, "#!/bin/sh\necho stale-binary-ran >> \"$STUB_LOG\"\nexit 0\n")
	got := runDevBinary(t, root, []string{"STUB_BUILD_FAIL=cmd/abcd does not compile"}, "status")
	if got.code != 2 {
		t.Errorf("a failed build exited %d, want 2", got.code)
	}
	if regexp.MustCompile(`(?m)^stale-binary-ran$`).MatchString(got.log) {
		t.Errorf("a failed build ran the binary an earlier build left:\n%s", got.log)
	}
	if !strings.Contains(got.stderr, "cmd/abcd does not compile") {
		t.Errorf("the build's own error is not on stderr:\n%s", got.stderr)
	}
	if left := devBinaryLeftovers(t, root); len(left) != 0 {
		t.Errorf("a failed build left %v beside the binary", left)
	}
}

// In a checkout this account cannot write to, the script builds to a private
// temporary directory instead, runs the binary as a child so the exit code
// still arrives as it was, unmarked because it is no build the local tier
// holds, and removes the directory afterwards.
func TestTheCheckoutsAbcdBuildsElsewhereWhenTheTierIsReadOnly(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes through any mode, so a read-only tier cannot be laid out")
	}
	root := devBinaryCheckout(t)
	local := filepath.Join(root, ".abcd", ".work.local")
	if err := os.MkdirAll(local, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(local, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(local, 0o755) })

	// A mark an earlier run left in the environment must not reach this build.
	got := runDevBinary(t, root, []string{"FAKE_EXIT=3", abcdrepo.CheckoutBuildEnv + "=/inherited/" + abcdrepo.CheckoutBuildRel}, "reserve", "iss-1")
	if got.code != 3 {
		t.Errorf("with a read-only tier scripts/abcd.sh exited %d, want the binary's 3\nstderr:\n%s", got.code, got.stderr)
	}
	m := regexp.MustCompile(`(?m)^args=build -buildvcs=false -o (\S+) \./cmd/abcd$`).FindStringSubmatch(got.log)
	if m == nil {
		t.Fatalf("no build ran:\n%s", got.log)
	}
	if strings.HasPrefix(m[1], root+string(filepath.Separator)) {
		t.Errorf("the build went into the read-only checkout, to %s", m[1])
	}
	if _, err := os.Stat(filepath.Dir(m[1])); !os.IsNotExist(err) {
		t.Errorf("the temporary build directory %s outlived the run (stat: %v)", filepath.Dir(m[1]), err)
	}
	if !strings.Contains(got.log, "bin-mark=\n") || !strings.Contains(got.log, "bin-arg=iss-1\n") {
		t.Errorf("the temporary build did not run unmarked with its arguments:\n%s", got.log)
	}
}
