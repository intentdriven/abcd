package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// countRootCommitSpawns puts a git on PATH that logs every root-commit lookup
// (`rev-list … --max-parents=0`) and then runs the real git, and returns a
// function that reads how many lookups were spawned so far. It counts the
// processes themselves, so it sees a lookup whichever code path makes it.
func countRootCommitSpawns(t *testing.T) func() int {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "root-lookups.log")
	shim := "#!/bin/sh\ncase \" $* \" in *\" --max-parents=0 \"*) echo x >> '" + log + "';; esac\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(shim), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return func() int {
		data, err := os.ReadFile(log)
		if os.IsNotExist(err) {
			return 0
		}
		if err != nil {
			t.Fatal(err)
		}
		return strings.Count(string(data), "x\n")
	}
}

// TestPromptRouterLooksUpTheRootOncePerPrompt (F1): with a personal note tied
// to this repository on the machine, one prompt spawns at most one
// root-commit lookup — the one the live drain already makes — whether the
// prompt matches nothing or matches the tied note. The recall reuses that
// answer and never spawns its own.
func TestPromptRouterLooksUpTheRootOncePerPrompt(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	_, repo := memoryFixture(t)
	id := addRaceNote(t, "session-one") // tied to this repository
	spawned := countRootCommitSpawns(t)

	before := spawned()
	other, _ := recallRouterJSON(t, "session-two", repo, "paint a landscape in oils")
	if other.Text != "" {
		t.Fatalf("an unrelated prompt injected text:\n%s", other.Text)
	}
	if n := spawned() - before; n > 1 {
		t.Fatalf("a prompt matching nothing spawned %d root-commit lookups, want at most the drain's one", n)
	}

	before = spawned()
	env, _ := recallRouterJSON(t, "session-two", repo, "why are the tests failing?")
	if env.Recalled == nil || !slices.Equal(*env.Recalled, []string{id}) {
		t.Fatalf("the tied note was not recalled: %+v", env)
	}
	if n := spawned() - before; n > 1 {
		t.Fatalf("a prompt matching a tied note spawned %d root-commit lookups, want at most one", n)
	}
}

// TestARootLookupTimeoutIsNamedAndRetried (F2): when git does not answer the
// root-commit lookup in time, the router says so on stderr in the recall's
// diagnostic shape, the rules still inject, and the tied note is left
// unrecorded — so the next turn, with git answering, brings it back.
func TestARootLookupTimeoutIsNamedAndRetried(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	_, repo := memoryFixture(t)
	id := addRaceNote(t, "session-one") // tied to this repository

	prev := routerRootCommit
	t.Cleanup(func() { routerRootCommit = prev })
	routerRootCommit = func(context.Context, string) (string, error) {
		return "", fmt.Errorf("git rev-list -n 1 --max-parents=0 HEAD: %w", gitutil.ErrGitTimedOut)
	}
	env, errlog := recallRouterJSON(t, "session-two", repo, "commit and push the tests")
	if env.Error != "" || !slices.Contains(env.Injected, "COMMITTING") {
		t.Fatalf("a root lookup timeout stopped the rules: %+v\nstderr:\n%s", env, errlog)
	}
	if env.Recalled == nil || len(*env.Recalled) != 0 || strings.Contains(env.Text, "## MEMORY") {
		t.Fatalf("a tied note came back with no root: %+v", env)
	}
	want := "abcd memory: this repository's root commit: "
	if !strings.Contains(errlog, want) || !strings.Contains(errlog, "did not answer in time") ||
		!strings.Contains(errlog, "recalling no notes tied to it this turn") {
		t.Fatalf("stderr does not name the timeout:\n%s", errlog)
	}
	if n := strings.Count(errlog, want); n != 1 {
		t.Fatalf("the timeout was named %d times, want once:\n%s", n, errlog)
	}

	routerRootCommit = prev
	next, _ := recallRouterJSON(t, "session-two", repo, "the tests again")
	if next.Recalled == nil || !slices.Equal(*next.Recalled, []string{id}) {
		t.Fatalf("the tied note was recorded on the turn it was left out: %+v", next)
	}
}
