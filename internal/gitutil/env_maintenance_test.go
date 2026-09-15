package gitutil_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestIsolatedEnvDisablesBackgroundMaintenance pins the class remedy for the
// ".git/objects: directory not empty" cleanup race: every git command run under
// the isolated environment — abcd's own subprocesses and every test helper that
// inits its own repository from IsolatedEnv — sees gc.auto=0, gc.autodetach=false,
// maintenance.auto=false and core.fsmonitor=false, so nothing detaches and
// outlives the command. iss-252 closed this for gittest.NewRepo alone; this test
// proves the guarantee at the environment, where every caller inherits it.
func TestIsolatedEnvDisablesBackgroundMaintenance(t *testing.T) {
	// A parent-injected config must be displaced, not merged: the only
	// environment config in effect is the async-disable set.
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "user.email")
	t.Setenv("GIT_CONFIG_VALUE_0", "evil@example.com")

	env := gittest.Env(t)
	for _, kv := range env {
		if strings.Contains(kv, "evil@example.com") {
			t.Fatalf("parent config injection leaked into the isolated env: %s", kv)
		}
	}

	root := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init", "-q")
	for key, want := range map[string]string{
		"gc.auto":          "0",
		"gc.autodetach":    "false",
		"maintenance.auto": "false",
		"core.fsmonitor":   "false",
	} {
		if got := run("config", "--get", key); got != want {
			t.Errorf("%s = %q under IsolatedEnv, want %q", key, got, want)
		}
	}
	// --get exits 1 when the key is unset, which is the outcome wanted here.
	probe := exec.Command("git", "-C", root, "config", "--get", "user.email")
	probe.Env = env
	if out, err := probe.CombinedOutput(); err == nil || strings.TrimSpace(string(out)) != "" {
		t.Errorf("user.email = %q under IsolatedEnv (err=%v); the parent's injection must not survive", strings.TrimSpace(string(out)), err)
	}
	_ = gitutil.IsolatedEnv
}
