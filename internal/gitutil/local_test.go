package gitutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestLocalGitRunsHooksUnderTheRepositorysOwnSettings: a local git command
// (a commit) runs the repository's hooks with no prompt and none of Net's
// network settings: no injected ssh command, no low-speed bounds in
// GIT_CONFIG_PARAMETERS, so a hook sees the repository as the person's own
// git runs it.
func TestLocalGitRunsHooksUnderTheRepositorysOwnSettings(t *testing.T) {
	t.Setenv("GIT_SSH_COMMAND", "")
	os.Unsetenv("GIT_SSH_COMMAND")
	r := gittest.NewRepo(t)
	r.Commit("seed")
	// LocalGit runs with the scrubbed environment, so the commit takes its
	// identity from the repository, as the person's own git would: a CI
	// runner's account has no name for git to fall back on.
	r.Git("config", "user.name", "t")
	r.Git("config", "user.email", "t@example.invalid")
	hooks := t.TempDir()
	env := filepath.Join(t.TempDir(), "env")
	hook := "#!/bin/sh\n{ printf 'prompt=%s\\n' \"$GIT_TERMINAL_PROMPT\"; printf 'ssh=%s\\n' \"$GIT_SSH_COMMAND\"; printf 'params=%s\\n' \"$GIT_CONFIG_PARAMETERS\"; } > '" + env + "'\n"
	if err := os.WriteFile(filepath.Join(hooks, "post-commit"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("config", "core.hooksPath", hooks)

	if _, err := gitutil.LocalGit(r.Root(), "commit", "-q", "--allow-empty", "-m", "local"); err != nil {
		t.Fatalf("LocalGit commit: %v", err)
	}
	b, err := os.ReadFile(env)
	if err != nil {
		t.Fatalf("the post-commit hook did not run: %v", err)
	}
	got := string(b)
	if !strings.Contains(got, "prompt=0\n") || !strings.Contains(got, "ssh=\n") || strings.Contains(strings.ToLower(got), "lowspeed") {
		t.Fatalf("the hook saw %q; want prompts off, no ssh command and no low-speed bounds", got)
	}
	if !strings.Contains(got, "core.quotePath") {
		t.Fatalf("the hook saw %q; want core.quotePath=false kept", got)
	}
}

// TestScrubURLCredentialsStripsTokensFromText: a token in a URL git or the
// forge quotes is dropped, as the user of an http(s) URL or the password of
// any; an ssh login is a route and stays.
func TestScrubURLCredentialsStripsTokensFromText(t *testing.T) {
	in := "fatal: unable to access 'https://user:ghp_FAKE0123@example.com/o/r.git/' (see https://ghp_FAKE0123@example.com/o/r) via ssh://git@example.com/o/r and ssh://git:pw@example.com/o/r"
	want := "fatal: unable to access 'https://example.com/o/r.git/' (see https://example.com/o/r) via ssh://git@example.com/o/r and ssh://example.com/o/r"
	if got := gitutil.ScrubURLCredentials(in); got != want {
		t.Fatalf("ScrubURLCredentials =\n%q\nwant\n%q", got, want)
	}
}
