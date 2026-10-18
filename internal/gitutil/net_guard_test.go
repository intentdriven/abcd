package gitutil_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// guardedRepo is a checkout whose own config names a credential helper that
// leaves a marker when git calls it, standing in for the helper a system
// config names (osxkeychain). The test's own environment reads no system or
// global config, so even a regressed guard cannot raise a dialog: the marker
// is the only helper there is to call.
func guardedRepo(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	gittest.Env(t)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))
	r := gittest.NewRepo(t)
	r.Commit("seed")
	dir := t.TempDir()
	marker := filepath.Join(dir, "helper-ran")
	helper := filepath.Join(dir, "helper")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\necho \"$1\" >> '"+marker+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("config", "credential.helper", helper)
	return r, marker
}

// TestNetUnderTestReachesNoForgeAndCallsNoHelper: under a test binary every
// networked git command is held to local transports with no credential
// helper (iss-2610170718356567). A fetch from a server that answers 401, as
// GitHub does for a repository that does not exist, is refused before any
// request, so no helper is ever asked; https, git and ssh are refused the
// same way; and a local remote, including one a repository's insteadOf
// rewrites a github.com URL to, still answers.
func TestNetUnderTestReachesNoForgeAndCallsNoHelper(t *testing.T) {
	r, marker := guardedRepo(t)
	// An allow list inherited from the developer's shell outranks
	// protocol.*.allow, so the guard has to hold against one.
	t.Setenv("GIT_ALLOW_PROTOCOL", "http:https:git:ssh:file")
	for _, k := range []string{"GIT_SSH_COMMAND", "GIT_SSH"} {
		t.Setenv(k, "")
		if err := os.Unsetenv(k); err != nil {
			t.Fatal(err)
		}
	}
	sshRecord := fakeSSH(t)

	var requests atomic.Int64
	forge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("WWW-Authenticate", `Basic realm="forge"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer forge.Close()

	for _, tc := range []struct{ url, transport string }{
		{forge.URL + "/abcd-test-nonexistent/x.git", "http"},
		{"https://abcd-test.invalid/x.git", "https"},
		{"git://abcd-test.invalid/x.git", "git"},
		{"ssh://git@abcd-test.invalid/x.git", "ssh"},
	} {
		_, err := gitutil.Net(r.Root(), "fetch", tc.url, "main")
		if err == nil || !strings.Contains(err.Error(), "transport '"+tc.transport+"' not allowed") {
			t.Errorf("a fetch from %s = %v; want git's refusal of the %s transport", tc.url, err, tc.transport)
		}
	}
	if n := requests.Load(); n != 0 {
		t.Errorf("the 401 server was asked %d times; want no request at all", n)
	}
	if got, err := os.ReadFile(marker); err == nil {
		t.Errorf("a credential helper ran (%q); want none called", got)
	}
	if got, err := os.ReadFile(sshRecord); err == nil {
		t.Errorf("git ran ssh (%q); want the ssh transport refused before it starts", got)
	}

	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	r.Git("remote", "add", "origin", bare)
	if _, err := gitutil.Net(r.Root(), "push", "origin", "HEAD:refs/heads/main"); err != nil {
		t.Fatalf("a push to a bare repository on disk: %v", err)
	}
	r.Git("config", "url."+bare+".insteadOf", "https://github.com/acme/widgets.git")
	r.Git("remote", "set-url", "origin", "https://github.com/acme/widgets.git")
	if _, err := gitutil.Net(r.Root(), "fetch", "origin"); err != nil {
		t.Fatalf("a fetch from a github.com URL the repository rewrites to disk: %v", err)
	}
}

// TestRunLimitedNetContextIsGuardedToo: the isolated networked runner (the
// implement loop's outage probe) takes the same environment, so its ls-remote
// against a server answering 401 is refused before any request, and a local
// origin still answers.
func TestRunLimitedNetContextIsGuardedToo(t *testing.T) {
	r, marker := guardedRepo(t)
	var requests atomic.Int64
	forge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer forge.Close()
	_, err := gitutil.RunLimitedNetContext(context.Background(), r.Root(), 4096, "ls-remote", forge.URL+"/x.git")
	if err == nil || !strings.Contains(err.Error(), "transport 'http' not allowed") || requests.Load() != 0 {
		t.Fatalf("ls-remote against a 401 server = %v after %d requests; want the http transport refused before any", err, requests.Load())
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("a credential helper ran; want none called")
	}
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	if _, err := gitutil.RunLimitedNetContext(context.Background(), r.Root(), 4096, "ls-remote", bare); err != nil {
		t.Fatalf("ls-remote of a bare repository on disk: %v", err)
	}
}

// TestNetEnvOutsideATestIsUnchanged: outside a test binary the networked
// environment is exactly the one the caller built, so the person's own
// configuration and credential helpers apply as their own git applies them.
// Under one, the guard follows any GIT_CONFIG_COUNT entries already present,
// and its empty credential.helper clears a helper every lower scope named:
// `git credential fill` calls none.
func TestNetEnvOutsideATestIsUnchanged(t *testing.T) {
	base := []string{"HOME=/h", "GIT_CONFIG_COUNT=2", "GIT_CONFIG_KEY_0=a.b", "GIT_CONFIG_VALUE_0=1", "GIT_CONFIG_KEY_1=c.d", "GIT_CONFIG_VALUE_1=2"}
	if got := gitutil.NetEnvFor(slices.Clone(base), false); !slices.Equal(got, base) {
		t.Fatalf("outside a test the environment changed:\n%q\nwant\n%q", got, base)
	}
	guarded := gitutil.NetEnvFor(slices.Clone(base), true)
	if !slices.Equal(guarded[:len(base)], base) || guarded[len(guarded)-1] != "GIT_CONFIG_COUNT=9" ||
		!slices.Contains(guarded, "GIT_CONFIG_KEY_2=credential.helper") || !slices.Contains(guarded, "GIT_CONFIG_VALUE_2=") {
		t.Fatalf("the guard does not follow the entries already present: %q", guarded)
	}

	r, marker := guardedRepo(t)
	cmd := exec.Command("git", "-C", r.Root(), "credential", "fill")
	// An askpass program that fails, so a fill no helper answers ends there
	// rather than at an askpass the environment names.
	cmd.Env = gitutil.NetEnvFor(append(gitutil.ScrubbedEnv(), "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=false"), true)
	cmd.Stdin = strings.NewReader("protocol=https\nhost=github.com\npath=abcd-test-nonexistent/x.git\n\n")
	if out, err := cmd.CombinedOutput(); err == nil {
		t.Fatalf("credential fill under the guard answered:\n%s", out)
	}
	if got, err := os.ReadFile(marker); err == nil {
		t.Fatalf("the repository's credential helper ran under the guard (%q)", got)
	}
}
