package gittest

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// Abcd is abcd built from this checkout, once for the whole test binary of
// one package. The git-hook and cloud-script fixtures run the real binary: as
// the attribution gate's outbound half, which a fixture clone cannot `go run`,
// and as the ABCD_BIN a script calls. Built afresh for every test, each a link
// of the whole module, the build cost those packages minutes of serial time.
//
// A package declares one, and its TestMain removes the build after the tests:
//
//	var builtAbcd gittest.Abcd
//
//	func TestMain(m *testing.M) {
//		code := m.Run()
//		builtAbcd.Remove()
//		os.Exit(code)
//	}
//
// The binary is shared, so a test reads it and never changes it.
type Abcd struct {
	once sync.Once
	dir  string
	bin  string
	err  error
}

// The home and config directory the test binary started with, read before
// any test runs. Env pins both to a test's temp directory, and a go build
// under that HOME would read a cold build cache; the build runs under these
// instead, whichever test asks first.
var (
	launchHome, launchHomeSet     = os.LookupEnv("HOME")
	launchConfig, launchConfigSet = os.LookupEnv("XDG_CONFIG_HOME")
)

// Path returns the built binary, building it on the first call. A machine
// with no go skips the test; a failed build fails every test that asks.
func (a *Abcd) Path(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go unavailable")
	}
	a.once.Do(a.build)
	if a.err != nil {
		t.Fatalf("gittest: build abcd: %v", a.err)
	}
	return a.bin
}

// Remove deletes the build, if one was made. TestMain calls it after m.Run.
func (a *Abcd) Remove() {
	if a.dir != "" {
		_ = os.RemoveAll(a.dir)
	}
}

func (a *Abcd) build() {
	root, err := moduleRoot()
	if err != nil {
		a.err = err
		return
	}
	a.dir, err = os.MkdirTemp("", "abcd-test-bin-")
	if err != nil {
		a.err = err
		return
	}
	bin := filepath.Join(a.dir, "abcd")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/abcd")
	cmd.Dir = root
	cmd.Env = launchEnv(os.Environ())
	if out, err := cmd.CombinedOutput(); err != nil {
		a.err = fmt.Errorf("%v\n%s", err, out)
		return
	}
	a.bin = bin
}

// launchEnv is env with HOME and XDG_CONFIG_HOME as the test binary started
// with them, set or unset.
func launchEnv(env []string) []string {
	out := make([]string, 0, len(env)+2)
	for _, kv := range env {
		if !strings.HasPrefix(kv, "HOME=") && !strings.HasPrefix(kv, "XDG_CONFIG_HOME=") {
			out = append(out, kv)
		}
	}
	if launchHomeSet {
		out = append(out, "HOME="+launchHome)
	}
	if launchConfigSet {
		out = append(out, "XDG_CONFIG_HOME="+launchConfig)
	}
	return out
}

// moduleRoot is the directory holding go.mod, found from the test's working
// directory, which go test sets to the package's own.
func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no go.mod above the test's directory")
		}
		dir = parent
	}
}
