package ahoy

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeExecutable writes body to path with the executable bit set.
func writeExecutable(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

// TestDevShimFailsLoudlyOnBuildFailure is coverage (d): it exercises the
// production shim content against a fake `go` that fails the build, and asserts
// the shim exits non-zero with a loud message and NEVER execs the stale binary.
func TestDevShimFailsLoudlyOnBuildFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shim; no /bin/sh on windows")
	}
	repo := t.TempDir()
	binDir := t.TempDir()
	freshBin := filepath.Join(binDir, "abcd-fresh")
	// A stale binary that shouts if it is ever exec'd.
	writeExecutable(t, freshBin, "#!/bin/sh\necho STALE-EXEC\n")

	// A fake `go` on PATH that fails every build (exit 3).
	fakeGoDir := t.TempDir()
	writeExecutable(t, filepath.Join(fakeGoDir, "go"), "#!/bin/sh\nexit 3\n")

	shimPath := filepath.Join(binDir, "abcd-shim")
	writeExecutable(t, shimPath, renderDevShim(repo, freshBin))

	cmd := exec.Command(shimPath, "--version")
	cmd.Env = append(os.Environ(), "PATH="+fakeGoDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err == nil {
		t.Fatalf("shim exited 0 on a failed build; want non-zero")
	}
	if ec, ok := err.(*exec.ExitError); ok && ec.ExitCode() == 0 {
		t.Fatalf("shim exit code 0; want non-zero")
	}
	if !strings.Contains(stderr.String(), "build failed") {
		t.Errorf("stderr missing the loud build-failure message: %q", stderr.String())
	}
	if strings.Contains(stdout.String(), "STALE-EXEC") {
		t.Errorf("shim exec'd the stale binary after a failed build (loud-staging violated)")
	}
}

// TestDevShimExecsFreshBinaryOnBuildSuccess proves the success path: when the
// build succeeds the shim execs the (freshly built) binary and passes args
// through.
func TestDevShimExecsFreshBinaryOnBuildSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shim; no /bin/sh on windows")
	}
	repo := t.TempDir()
	binDir := t.TempDir()
	freshBin := filepath.Join(binDir, "abcd-fresh")
	// The "built" binary echoes a sentinel plus its args.
	writeExecutable(t, freshBin, "#!/bin/sh\necho FRESH-EXEC:\"$@\"\n")

	// A fake `go` that "succeeds" without touching the pre-built binary.
	fakeGoDir := t.TempDir()
	writeExecutable(t, filepath.Join(fakeGoDir, "go"), "#!/bin/sh\nexit 0\n")

	shimPath := filepath.Join(binDir, "abcd-shim")
	writeExecutable(t, shimPath, renderDevShim(repo, freshBin))

	cmd := exec.Command(shimPath, "--version", "--json")
	cmd.Env = append(os.Environ(), "PATH="+fakeGoDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("shim failed on a successful build: %v (stderr=%q)", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "FRESH-EXEC:--version --json") {
		t.Errorf("shim did not exec the fresh binary with args: %q", stdout.String())
	}
}
