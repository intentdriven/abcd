package scanner

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

func mustGitIdentity(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// TestProbeIdentityIgnoresInjectedConfig proves an injected GIT_CONFIG_* cannot
// forge the caller's identity: ProbeIdentity feeds the hard_fail identity gate, so
// if a hostile sandbox/CI exports a fake user.email, the probe must still resolve
// the repo's real identity — otherwise the caller's genuine identity in scanned
// content escapes redaction. ScrubbedEnv strips the injection while keeping global
// config (so a real global identity is still probed).
func TestProbeIdentityIgnoresInjectedConfig(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	dir := t.TempDir()
	mustGitIdentity(t, dir, "init")
	mustGitIdentity(t, dir, "config", "user.email", "real@example.com")
	mustGitIdentity(t, dir, "config", "user.name", "Real Name")

	// A hostile/injected identity that would displace the real one, as GIT_CONFIG_*
	// parameters outrank repo-local config.
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "user.email")
	t.Setenv("GIT_CONFIG_VALUE_0", "evil@example.com")

	id := ProbeIdentity(dir)
	if id.GitUserEmail != "real@example.com" {
		t.Errorf("ProbeIdentity honoured an injected GIT_CONFIG_* identity: got %q, want real@example.com", id.GitUserEmail)
	}
}

// TestProbeIdentityRemoteHostCaseInsensitive proves the GitHub handle is derived
// from the remote URL regardless of the host's letter case. git stores the remote
// verbatim, so a hand-typed mixed-case host (GitHub.com) must still yield the
// username — otherwise GitRemoteUsername stays empty, the github_username matcher
// is never compiled, and the caller's handle escapes the redaction gate.
func TestProbeIdentityRemoteHostCaseInsensitive(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	for _, remote := range []string{
		"git@GitHub.com:alice/repo.git",
		"https://GITHUB.COM/alice/repo.git",
		"ssh://git@GitHub.com/alice/repo.git",
	} {
		dir := t.TempDir()
		mustGitIdentity(t, dir, "init")
		mustGitIdentity(t, dir, "remote", "add", "origin", remote)
		if id := ProbeIdentity(dir); id.GitRemoteUsername != "alice" {
			t.Errorf("remote %q: GitRemoteUsername = %q, want alice", remote, id.GitRemoteUsername)
		}
	}
}

// TestProbeIdentityUnionsTheEnvironmentIdentity: GIT_AUTHOR_NAME/EMAIL and
// GIT_COMMITTER_NAME/EMAIL are an identity scope `git config --get-all` never
// reports, and they outrank every config file when a commit is written. A CI
// runner, a direnv profile or a rebase wrapper sets them routinely, so the
// persona that AUTHORS the caller's commits was absent from the matcher set
// and every write-time redactor stored it in clear.
func TestProbeIdentityUnionsTheEnvironmentIdentity(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	dir := t.TempDir()
	mustGitIdentity(t, dir, "init")
	mustGitIdentity(t, dir, "config", "user.email", "config@example.com")
	mustGitIdentity(t, dir, "config", "user.name", "Config Name")

	t.Setenv("GIT_AUTHOR_NAME", "Author Persona")
	t.Setenv("GIT_AUTHOR_EMAIL", "author@ci.example")
	t.Setenv("GIT_COMMITTER_NAME", "Committer Persona")
	t.Setenv("GIT_COMMITTER_EMAIL", "committer@ci.example")

	id := ProbeIdentity(dir)
	if id.GitUserEmail != "config@example.com" || id.GitUserName != "Config Name" {
		t.Fatalf("the environment displaced the configured identity: %q <%s>", id.GitUserName, id.GitUserEmail)
	}
	for _, want := range []string{"author@ci.example", "committer@ci.example"} {
		if !containsFold(id.OtherGitUserEmails, want) {
			t.Errorf("OtherGitUserEmails %v lacks the environment address %q", id.OtherGitUserEmails, want)
		}
	}
	for _, want := range []string{"Author Persona", "Committer Persona"} {
		if !containsFold(id.OtherGitUserNames, want) {
			t.Errorf("OtherGitUserNames %v lacks the environment name %q", id.OtherGitUserNames, want)
		}
	}

	text := "signed off by Author Persona <author@ci.example> and committer@ci.example"
	findings := ScanText(text, id, DefaultPatterns(), DefaultIdentitySeverities(), "t")
	out, _ := Redact(text, findings)
	for _, leak := range []string{"author@ci.example", "committer@ci.example", "Author Persona"} {
		if strings.Contains(out, leak) {
			t.Errorf("the environment identity %q survives redaction: %q", leak, out)
		}
	}
}
