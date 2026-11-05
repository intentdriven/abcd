package gitutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestAHookRunsInTheNarrowedEnvironment: the commands that run a repository's
// hooks on a drive give them only what the drive allows.
func TestAHookRunsInTheNarrowedEnvironment(t *testing.T) {
	repo := gittest.NewRepo(t)
	log := filepath.Join(t.TempDir(), "env")
	hook := filepath.Join(repo.Root(), ".git", "hooks", "pre-commit")
	if err := os.MkdirAll(filepath.Dir(hook), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hook, []byte("#!/bin/sh\n/usr/bin/env >> '"+log+"'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SSH_AUTH_SOCK", "a-value-never-passed")
	gitutil.SetHookedEnv(func(env []string) []string {
		var out []string
		for _, kv := range env {
			if !strings.HasPrefix(kv, "SSH_AUTH_SOCK=") {
				out = append(out, kv)
			}
		}
		return out
	})
	t.Cleanup(func() { gitutil.SetHookedEnv(nil) })
	repo.Write("a.txt", "x\n")
	repo.Git("add", "a.txt")
	if _, err := gitutil.LocalGit(repo.Root(), "commit", "-q", "--allow-empty", "-m", "one"); err != nil {
		t.Fatal(err)
	}
	if out, err := gitutil.RunTreeCommand(gitutil.WorkingTreeCommitCommand(t.Context(), repo.Root(), "commit", "-q", "--allow-empty", "-m", "two"), nil); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(raw), "PATH=") < 2 {
		t.Fatalf("the hook ran for both commands:\n%s", raw)
	}
	if strings.Contains(string(raw), "a-value-never-passed") {
		t.Fatalf("a hook held the person's environment:\n%s", raw)
	}
}
