package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// committedGuardRepo is sessionEndRepo with abcd's guard hook committed in
// .githooks, the state of a clone of abcd's own repository before git is
// pointed at the hooks.
func committedGuardRepo(t *testing.T) string {
	t.Helper()
	repo, _ := sessionEndRepo(t)
	// What git runs is read with the person's global and system config in
	// force, so neither may name a hooks path here.
	global := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(global, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	if err := os.MkdirAll(filepath.Join(repo, ".githooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	hook := "#!/bin/sh\n# abcd-name-guard: v1\nexit 0\n"
	if err := os.WriteFile(filepath.Join(repo, ".githooks", "pre-commit"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	return repo
}

func setHooksPath(t *testing.T, repo, path string) {
	t.Helper()
	cmd := exec.Command("git", "config", "--local", "core.hooksPath", path)
	cmd.Dir = repo
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git config: %v: %s", err, out)
	}
}

// TestSessionStartSaysTheCommitChecksAreOff is iss-2610241133284600: on
// 2026-10-24 a clone ran none of its committed hooks, commits and a
// receipt-less push went through, and nothing said so until a gate failed
// afterwards. The session start now names it, on stderr in full and on stdout
// as a constant the session reads.
func TestSessionStartSaysTheCommitChecksAreOff(t *testing.T) {
	repo := committedGuardRepo(t)
	noAmbientPluginRoot(t)

	stdout, stderr, code := runSessionStart(startPayload("s-hooks", repo), "hook", "session-start")

	if code != 0 {
		t.Fatalf("a notice is not a hook failure; got exit %d (stderr %q)", code, stderr)
	}
	if !strings.Contains(stderr, "git does not run this repository's commit checks (.githooks)") ||
		strings.Contains(stderr, "preflight receipt") ||
		!strings.Contains(stderr, "git config --local core.hooksPath .githooks") {
		t.Errorf("stderr must name the unrun checks and the fix; got %q", stderr)
	}
	if !strings.Contains(stdout, "does not run this repository's committed commit checks in this clone") {
		t.Errorf("stdout must carry the constant the session reads; got %q", stdout)
	}
}

// TestSessionStartSaysNothingOnceTheChecksRun holds the other side: pointed at
// .githooks, the same clone starts with no word about the hooks.
func TestSessionStartSaysNothingOnceTheChecksRun(t *testing.T) {
	repo := committedGuardRepo(t)
	setHooksPath(t, repo, ".githooks")
	noAmbientPluginRoot(t)

	stdout, stderr, code := runSessionStart(startPayload("s-armed", repo), "hook", "session-start")

	if code != 0 {
		t.Fatalf("exit %d (stderr %q)", code, stderr)
	}
	if strings.Contains(stderr+stdout, "commit checks") {
		t.Errorf("armed hooks must not be reported off; stdout=%q stderr=%q", stdout, stderr)
	}
}

// TestSessionStartNamesAHooksPathSetElsewhere: a clone whose own config names
// another directory runs the committed checks only if that directory's hooks
// call them, and is told so, and that abcd does not replace the value.
func TestSessionStartNamesAHooksPathSetElsewhere(t *testing.T) {
	repo := committedGuardRepo(t)
	setHooksPath(t, repo, "other-hooks")
	noAmbientPluginRoot(t)

	stdout, stderr, _ := runSessionStart(startPayload("s-elsewhere", repo), "hook", "session-start")

	if !strings.Contains(stderr, "core.hooksPath names another directory") ||
		!strings.Contains(stderr, "run only if that directory's hooks call them") {
		t.Errorf("stderr must say the checks may not run; got %q", stderr)
	}
	// That directory's hooks may call the checks, so the session is not told
	// as a fact that they are off.
	if strings.Contains(stdout, "committed commit checks") {
		t.Errorf("stdout must not claim the checks are off when that is not known; got %q", stdout)
	}
}
