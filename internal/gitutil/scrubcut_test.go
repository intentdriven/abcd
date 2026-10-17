package gitutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// failingHookStderr is what a failing hook writes: padding, then a credential
// URL placed so the runners' 2048-byte stderr cap falls inside its token, 40
// bytes in. Cutting before scrubbing leaves a URL the scrubber no longer
// recognises (it has no host), and the token's start goes out with it.
func failingHookStderr(token string) string {
	const cap, into = 2048, 40
	lead := "https://x-access-token:"
	return strings.Repeat("x", cap-len(lead)-into) + " " + lead + token + "@example.com/o/r.git\n"
}

func writeFailingHook(t *testing.T, r *gittest.Repo, name, stderr string) {
	t.Helper()
	hooks := t.TempDir()
	msg := filepath.Join(t.TempDir(), "stderr")
	if err := os.WriteFile(msg, []byte(stderr), 0o600); err != nil {
		t.Fatal(err)
	}
	hook := "#!/bin/sh\ncat '" + msg + "' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(hooks, name), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("config", "core.hooksPath", hooks)
}

// R3-F4: both runners scrub a credential URL from git's stderr BEFORE cutting
// it to the cap, so no part of a token reaches the error.
func TestTheRunnersScrubStderrBeforeCuttingIt(t *testing.T) {
	token := testsecret.Synthetic(4, 64)
	for _, c := range []struct {
		name string
		run  func(r *gittest.Repo) error
	}{
		{"LocalGit", func(r *gittest.Repo) error {
			writeFailingHook(t, r, "pre-commit", failingHookStderr(token))
			_, err := gitutil.LocalGit(r.Root(), "commit", "-q", "--allow-empty", "-m", "local")
			return err
		}},
		{"Net", func(r *gittest.Repo) error {
			bare := filepath.Join(t.TempDir(), "origin.git")
			gittest.InitBare(t, bare)
			r.Git("remote", "add", "origin", bare)
			writeFailingHook(t, r, "pre-push", failingHookStderr(token))
			_, err := gitutil.Net(r.Root(), "push", "--porcelain", "origin", "HEAD:refs/heads/main")
			return err
		}},
	} {
		r := gittest.NewRepo(t)
		r.Commit("seed")
		err := c.run(r)
		if err == nil {
			t.Fatalf("%s: the failing hook did not fail the command", c.name)
		}
		if strings.Contains(err.Error(), token[:8]) {
			t.Errorf("%s: the error carries the token's start:\n%s", c.name, err)
		}
		if len(err.Error()) > 2048+256 {
			t.Fatalf("%s: the error is %d bytes, want stderr cut to the cap", c.name, len(err.Error()))
		}
	}
}

// R3-F4: a token carried in a URL's query (access_token=, token=) is dropped
// as one in its userinfo is.
func TestScrubURLCredentialsStripsQueryTokens(t *testing.T) {
	tok := testsecret.Synthetic(5, 40)
	in := "see https://example.com/o/r.git?access_token=" + tok + "&x=1 and 'https://example.com/a?Token=" + tok + "#f' and https://example.com/b?mytoken=keep"
	want := "see https://example.com/o/r.git?access_token=&x=1 and 'https://example.com/a?Token=#f' and https://example.com/b?mytoken=keep"
	if got := gitutil.ScrubURLCredentials(in); got != want {
		t.Fatalf("ScrubURLCredentials =\n%q\nwant\n%q", got, want)
	}
}
