// Package gittest is the shared hermetic-git environment for tests that spawn
// git as a subprocess (iss-28).
//
// The problem it closes: a test that builds `exec.Command("git", …)` and lets it
// inherit os.Environ() also inherits any ambient GIT_DIR/GIT_WORK_TREE/
// GIT_INDEX_FILE/GIT_CONFIG_* the process was launched with. A pre-commit or
// prompt hook, for instance, exports GIT_DIR — which OVERRIDES `-C dir`/cmd.Dir
// and silently redirects the test's `git init`/`commit` onto the real repository.
// The developer's ~/.gitconfig identity/aliases leak in the same way. Both make a
// test non-hermetic, and the redirect can mutate the ambient repo.
//
// Env(t) is the single fix: it reuses the SAME production scrub as
// gitutil.IsolatedEnv() (so a test git command runs under exactly the isolation
// production git runs under) and additionally pins HOME/XDG to a per-test temp
// dir. Assign its result to cmd.Env for every git command a test spawns.
package gittest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// isolatedSentinel marks, for the duration of a test, that Env has already
// pinned HOME/XDG. It lets Env be called from a per-command git wrapper any
// number of times while pinning the home dirs exactly ONCE, so the test's HOME is
// stable across its git calls rather than hopping to a fresh temp on each.
const isolatedSentinel = "ABCD_GITTEST_ISOLATED"

// Env pins HOME and the XDG config/data dirs to a per-test temp location and
// returns the isolated environment a git subprocess must run under. The returned
// slice is gitutil.IsolatedEnv(): the parent environment with every repo-selection
// and config-injection variable stripped (GIT_DIR, GIT_WORK_TREE, GIT_INDEX_FILE,
// GIT_CONFIG_*, …) and the global/system config-file neutralisers appended.
//
// Assign it to cmd.Env for each git command. It is safe to call once per test or
// once per git call — the HOME/XDG pin happens only on the first call within a
// test (see isolatedSentinel). A caller that needs a commit identity appends its
// own GIT_AUTHOR_*/GIT_COMMITTER_* (or sets git config) on top of the result; Env
// deliberately does NOT pin an identity, so config-based identity tests keep
// resolving the repo's own user.name/user.email.
//
// If the test has ALREADY pointed HOME at a temp dir it owns (the common case for
// tests that stand up a hermetic ~/.abcd store), Env reuses that HOME rather than
// replacing it — replacing it would leave the process HOME and the test's captured
// home var pointing at different directories, so a store the test wrote under its
// own HOME would be invisible to the in-process production code under test. Env
// only mints a fresh temp HOME when HOME still points outside the test temp area.
func Env(t *testing.T) []string {
	t.Helper()
	if os.Getenv(isolatedSentinel) != "1" {
		home := os.Getenv("HOME")
		if !testOwnedHome(home) {
			home = t.TempDir()
			t.Setenv("HOME", home)
		}
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
		t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
		turnGoTelemetryOff(t)
		// Never block on a credential/terminal prompt in a test.
		t.Setenv("GIT_TERMINAL_PROMPT", "0")
		t.Setenv(isolatedSentinel, "1")
	}
	return gitutil.IsolatedEnv()
}

// testOwnedHome reports whether home already points inside the OS temp area — the
// signature of a HOME the test itself redirected with t.TempDir()/t.Setenv. Such a
// HOME is already hermetic and belongs to the test, so Env leaves it in place.
func testOwnedHome(home string) bool {
	if home == "" {
		return false
	}
	tmp := filepath.Clean(os.TempDir()) + string(os.PathSeparator)
	return strings.HasPrefix(filepath.Clean(home)+string(os.PathSeparator), tmp)
}

// turnGoTelemetryOff writes Go's telemetry mode file under the test HOME's user
// config directory. With the default mode the go command spawns a detached child
// that keeps writing counters there after the command exits, so a test that runs
// go (the commit hooks build abcd) loses its TempDir cleanup to "directory not
// empty". The mode file is the setting `go telemetry off` writes.
func turnGoTelemetryOff(t *testing.T) {
	t.Helper()
	cfg, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("gittest: user config dir: %v", err)
	}
	dir := filepath.Join(cfg, "go", "telemetry")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("gittest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mode"), []byte("off"), 0o644); err != nil {
		t.Fatalf("gittest: %v", err)
	}
}
