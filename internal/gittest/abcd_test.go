package gittest

import (
	"os"
	"os/exec"
	"slices"
	"testing"
)

// TestAbcdBuildsOnceAndRuns: every call returns the one build, the binary
// runs, and Remove deletes it. Env is called first, so the build is asked for
// under a test's temp HOME, as a fixture asks for it.
func TestAbcdBuildsOnceAndRuns(t *testing.T) {
	_ = Env(t)
	var a Abcd
	t.Cleanup(a.Remove)
	first := a.Path(t)
	if second := a.Path(t); second != first {
		t.Fatalf("a second call returned %s, want the first build %s", second, first)
	}
	if out, err := exec.Command(first, "--help").CombinedOutput(); err != nil {
		t.Fatalf("the built abcd does not run: %v\n%s", err, out)
	}
	a.Remove()
	if _, err := os.Stat(first); !os.IsNotExist(err) {
		t.Fatalf("Remove left the build at %s (%v)", first, err)
	}
}

// TestAbcdBuildsUnderTheLaunchHome: the build runs under the HOME and
// XDG_CONFIG_HOME the test binary started with, never the temp ones a test
// pinned, so it reads the machine's warm build cache.
func TestAbcdBuildsUnderTheLaunchHome(t *testing.T) {
	saved := []any{launchHome, launchHomeSet, launchConfig, launchConfigSet}
	t.Cleanup(func() {
		launchHome, launchHomeSet = saved[0].(string), saved[1].(bool)
		launchConfig, launchConfigSet = saved[2].(string), saved[3].(bool)
	})
	launchHome, launchHomeSet = "/home/person", true
	launchConfig, launchConfigSet = "", false
	got := launchEnv([]string{"PATH=/bin", "HOME=/tmp/test-home", "XDG_CONFIG_HOME=/tmp/test-home/.config"})
	want := []string{"PATH=/bin", "HOME=/home/person"}
	if !slices.Equal(got, want) {
		t.Fatalf("launchEnv = %q, want %q", got, want)
	}
}
