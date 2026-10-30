package gittest

import (
	"os"
	"os/exec"
	"testing"
)

// After HideGHLogin no token variable is set, gh's configuration directory is
// an empty one of its own, and gh itself, where it is installed, reports no
// login: a machine's own login stays out of reach of the tests.
func TestHideGHLoginLeavesGHWithNoLogin(t *testing.T) {
	t.Setenv("GH_CONFIG_DIR", t.TempDir())
	for _, k := range ghTokenVars {
		t.Setenv(k, "a-machine-credential")
	}
	cleanup, err := HideGHLogin()
	if err != nil {
		t.Fatal(err)
	}
	dir := os.Getenv("GH_CONFIG_DIR")
	for _, k := range ghTokenVars {
		if v, ok := os.LookupEnv(k); ok {
			t.Errorf("%s is still set (%q) after HideGHLogin", k, v)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Errorf("GH_CONFIG_DIR %s is not an empty directory: %v entries, %v", dir, len(entries), err)
	}
	if gh, err := exec.LookPath("gh"); err == nil {
		out, err := exec.Command(gh, "auth", "status", "--hostname", "github.com").CombinedOutput()
		if err == nil {
			t.Errorf("gh reports a login after HideGHLogin:\n%s", out)
		}
	}
	cleanup()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("cleanup left %s behind (%v)", dir, err)
	}
}
