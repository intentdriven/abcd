package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/gittest"
)

// identityGateRepo is a hermetic, real git repo whose user.* config is the
// given identity and whose committed pin is Alex Reppel, with every identity
// environment override cleared so a case sets only what it means to.
func identityGateRepo(t *testing.T, name, email string) string {
	t.Helper()
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(k, "")
	}
	repo := hermeticRepo(t)
	gitRepoWithIdentity(t, repo, name, email)
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config", "identity.json"),
		[]byte(`{"name":"Alex Reppel","email":"alex@example.com"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return repo
}

// TestAhoyIdentityRefusesADivergentCommitter: the gate's canonical entrypoint
// fails closed on a committer that differs from the pin, and names it, even
// though the author matches (itd-131 criterion 2 at the front door).
func TestAhoyIdentityRefusesADivergentCommitter(t *testing.T) {
	identityGateRepo(t, "Alex Reppel", "alex@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test User")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	out, err := runCLIErr(t, "ahoy", "--identity")
	if err == nil {
		t.Fatalf("a divergent committer passed the identity gate:\n%s", out)
	}
	if !strings.Contains(err.Error()+string(out), `committer "Test User" <test@example.com>`) {
		t.Fatalf("the refusal does not name the committer: %v\n%s", err, out)
	}
}

// TestAhoyInstallPipedAnswersNeverRewriteTheIdentity is criterion 3 at the
// front door: piped answers drive an install (iss-167), but a divergent
// identity is never rewritten without a terminal — even when every answer is
// yes — and the run says so rather than blocking or writing.
func TestAhoyInstallPipedAnswersNeverRewriteTheIdentity(t *testing.T) {
	repo := identityGateRepo(t, "Test User", "test@example.com")
	before, err := os.ReadFile(filepath.Join(repo, ".git", "config"))
	if err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := runCLIPipedStdinSplit(t, strings.Repeat("y\n", 16), "ahoy", "install", "--adopt",
		"--visibility", "private", "--docs-target", "both",
		"--oracle-backend", "host-delegated", "--scan-deep", "false")
	if err != nil {
		t.Fatalf("piped install exited non-zero: %v\n%s\n%s", err, stdout, stderr)
	}
	after, err := os.ReadFile(filepath.Join(repo, ".git", "config"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("a run with no terminal rewrote git config:\n%s", after)
	}
	all := string(stdout) + string(stderr)
	if !strings.Contains(all, "no terminal to confirm at") {
		t.Fatalf("the fail-closed refusal was not reported:\n%s", all)
	}
	if !strings.Contains(all, "git_identity.mismatch") {
		t.Fatalf("the divergence must stay reported as remaining work:\n%s", all)
	}
	cmd := exec.Command("git", "-C", repo, "config", "--local", "user.name")
	cmd.Env = gittest.Env(t)
	if got, _ := cmd.Output(); strings.TrimSpace(string(got)) != "Test User" {
		t.Fatalf("user.name changed without a terminal: %q", got)
	}
}

// TestStdinPrompterSaysWhetherAPersonIsAtATerminal: the establish step asks
// only a TerminalPrompter that reports a terminal, so the front door's prompter
// must carry what it already knows — a person typing, or a pipe.
func TestStdinPrompterSaysWhetherAPersonIsAtATerminal(t *testing.T) {
	for _, tty := range []bool{true, false} {
		var p ahoy.Prompter = &stdinPrompter{tty: tty}
		tp, ok := p.(ahoy.TerminalPrompter)
		if !ok || tp.AtTerminal() != tty {
			t.Fatalf("stdinPrompter{tty: %v}: TerminalPrompter=%v", tty, ok)
		}
	}
}
