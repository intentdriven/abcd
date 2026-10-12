package gitutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestNetRunsTheRepositorysHooksWithNoPrompt holds the networked runner to its
// contract: the repository's hooks run (the pre-push gate is the point), under
// no terminal prompt, and a failure carries git's own reason.
func TestNetRunsTheRepositorysHooksWithNoPrompt(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Commit("seed")
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	r.Git("remote", "add", "origin", bare)
	hooks := t.TempDir()
	marker := filepath.Join(t.TempDir(), "pre-push-ran")
	hook := "#!/bin/sh\nprintf '%s' \"$GIT_TERMINAL_PROMPT\" > '" + marker + "'\n"
	if err := os.WriteFile(filepath.Join(hooks, "pre-push"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("config", "core.hooksPath", hooks)

	if _, err := gitutil.Net(r.Root(), "push", "--porcelain", "origin", "HEAD:refs/heads/main"); err != nil {
		t.Fatalf("Net push: %v", err)
	}
	got, err := os.ReadFile(marker)
	if err != nil || string(got) != "0" {
		t.Fatalf("the pre-push hook saw GIT_TERMINAL_PROMPT=%q (%v); want it run, with prompts off", got, err)
	}

	r.Git("remote", "set-url", "origin", filepath.Join(t.TempDir(), "absent.git"))
	_, err = gitutil.Net(r.Root(), "fetch", "origin")
	if err == nil || !strings.Contains(err.Error(), "git fetch") || !strings.Contains(err.Error(), "absent.git") {
		t.Fatalf("a fetch from a missing origin = %v; want git's reason naming the path", err)
	}
}
