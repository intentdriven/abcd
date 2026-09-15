//go:build smoke || coldreading

// Package evals holds abcd's repository evals. Two lanes share one harness and
// one built binary:
//
//   - The self-discovering smoke harness (smoke_test.go, build tag `smoke`),
//     which walks the Cobra command tree in-process and exercises every command
//     against the built binary.
//   - The cold-reading evals (coldreading_*_test.go, build tags `smoke` or
//     `coldreading`), which falsify the cold-reading assembler's read-block by
//     planting sentinel warm content in a fixture repository state and asserting
//     its absence from what the assembler passes.
//
// Run them with:
//
//	go test -tags smoke ./evals/...        # both lanes  -> make smoke
//	go test -tags coldreading ./evals/...  # the cold-reading lane alone
//	                                       # -> make evals-cold-reading
//
// The second spelling exists because CI stands the smoke job down on a change
// confined to the record and the docs, and those are precisely the paths the
// cold-reading evals read. The cold-reading lane carries no such condition, so
// it needs to be selectable without the command-tree smokes attached.
//
// A file that must be visible to BOTH lanes carries `//go:build smoke ||
// coldreading`; that is also how a later cold-reading eval joins the lane, with
// no Makefile or workflow edit.
package evals

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// abcdBin is the freshly-built binary under test, set once by TestMain.
var abcdBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "abcd-smoke")
	if err != nil {
		panic("smoke: mktemp: " + err.Error())
	}
	defer os.RemoveAll(dir)

	abcdBin = filepath.Join(dir, "abcd")
	// Build from the module root (this package lives at <root>/evals).
	build := exec.Command("go", "build", "-o", abcdBin, "./cmd/abcd")
	build.Dir = ".."
	build.Stdout, build.Stderr = os.Stderr, os.Stderr
	if err := build.Run(); err != nil {
		panic("smoke: build abcd: " + err.Error())
	}
	os.Exit(m.Run())
}

// run executes the built binary and returns combined output + exit code. A
// non-zero exit is returned, not fataled — callers decide whether it is expected.
// Any failure to launch the process at all is fatal.
func run(t *testing.T, args ...string) (string, int) {
	t.Helper()
	return runIn(t, "", nil, args...)
}

// runIn is run with a working directory and environment overrides: the same
// binary, invoked where a fixture repository is and with the HOME a fixture
// home occupies. It is one runner, not a second harness — every eval in this
// package goes through it, so there is one definition of how the binary under
// test is launched.
//
// Overrides are `KEY=value` strings applied over the test process's own
// environment, last one winning, so PATH and the Go toolchain's variables
// survive. PWD is dropped unconditionally: it is the TEST process's directory,
// and a child that inherits it can report a working directory it is not in.
func runIn(t *testing.T, dir string, overrides []string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(abcdBin, args...)
	cmd.Dir = dir
	if len(overrides) > 0 || dir != "" {
		env := make([]string, 0, len(os.Environ())+len(overrides))
		for _, kv := range os.Environ() {
			if strings.HasPrefix(kv, "PWD=") {
				continue
			}
			env = append(env, kv)
		}
		cmd.Env = append(env, overrides...)
	}
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return string(out), ee.ExitCode()
	}
	t.Fatalf("could not launch `abcd %s`: %v", strings.Join(args, " "), err)
	return "", -1
}
