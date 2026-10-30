package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// commitConfig makes dir a git repository whose main branch carries cfg as
// .abcd/config.json ("" commits none).
func commitConfig(t *testing.T, dir, cfg string) {
	t.Helper()
	gitInitAt(t, dir)
	if cfg != "" {
		if err := os.WriteFile(filepath.Join(dir, ".abcd", "config.json"), []byte(cfg), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"add", "-A"},
		{"-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "config"},
		{"branch", "-M", "main"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
}

// withOrigin gives the repository at dir an origin: a bare clone of it as it
// stands, fetched, so origin/main and origin/HEAD name its main.
func withOrigin(t *testing.T, dir string) {
	t.Helper()
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.CloneBare(t, dir, bare)
	for _, args := range [][]string{
		{"remote", "add", "origin", bare},
		{"fetch", "-q", "origin"},
		{"remote", "set-head", "origin", "main"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
}

// TestTheGuardHookReadsTheFacilitatorMode is spc-2610161609350616 step 5 at
// the hook: with no facilitator.mode (human) the hook refuses an agent's
// `gh pr merge --auto`, naming the setting that allows it; with auto on
// origin's default branch it lets it through. The approval act is refused in
// both.
func TestTheGuardHookReadsTheFacilitatorMode(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	merge := strings.Join([]string{"gh", "pr", "merge", "7", "--auto", "--merge"}, " ")
	dir := guardRepo(t)
	commitConfig(t, dir, "")
	withOrigin(t, dir)
	stdout, stderr, code := runGuard(preToolUse(t, "Bash", merge, dir), "guard", "hook")
	if reason := mustDeny(t, stdout, stderr, code); !strings.Contains(reason, "merge-auto-human") || !strings.Contains(reason, "facilitator.mode: auto") {
		t.Fatalf("human refuses gh pr merge --auto, naming the setting: %q", reason)
	}
	stdout, stderr, code = runGuard(preToolUse(t, "Bash", "go run ./cmd/abcd land --approve", dir), "guard", "hook")
	if reason := mustDeny(t, stdout, stderr, code); !strings.Contains(reason, "abcd land --approve") {
		t.Fatalf("the approval act is refused to an agent: %q", reason)
	}

	auto := guardRepo(t)
	commitConfig(t, auto, `{"facilitator":{"mode":"auto"}}`+"\n")
	withOrigin(t, auto)
	stdout, _, code = runGuard(preToolUse(t, "Bash", merge, auto), "guard", "hook")
	if code != 0 || strings.Contains(stdout, "deny") {
		t.Fatalf("auto lets gh pr merge --auto through: exit %d, stdout %q", code, stdout)
	}
	stdout, stderr, code = runGuard(preToolUse(t, "Bash", "abcd land --approve", auto), "guard", "hook")
	if reason := mustDeny(t, stdout, stderr, code); !strings.Contains(reason, "abcd-land-approve") {
		t.Fatalf("the approval act is refused in auto too: %q", reason)
	}
}

// TestTheGuardReadsTheModeOnlyAsJustFetched: the mode is origin's, read after
// a fetch, never the refs this checkout holds. A checkout whose origin/main
// and origin/HEAD were moved onto a commit that says auto, while origin itself
// says nothing (human), is still refused; so is a checkout whose origin cannot
// be reached, though its last fetch said auto, and the refusal says the mode
// could not be read.
func TestTheGuardReadsTheModeOnlyAsJustFetched(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	merge := strings.Join([]string{"gh", "pr", "merge", "7", "--auto", "--merge"}, " ")
	git := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = gittest.Env(t)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}

	moved := guardRepo(t)
	commitConfig(t, moved, "")
	withOrigin(t, moved)
	git(moved, "checkout", "-q", "-b", "side")
	if err := os.WriteFile(filepath.Join(moved, ".abcd", "config.json"), []byte(`{"facilitator":{"mode":"auto"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(moved, "add", "-A")
	git(moved, "-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "auto")
	git(moved, "update-ref", "refs/remotes/origin/main", git(moved, "rev-parse", "HEAD"))
	git(moved, "remote", "set-head", "origin", "main")
	stdout, stderr, code := runGuard(preToolUse(t, "Bash", merge, moved), "guard", "hook")
	if reason := mustDeny(t, stdout, stderr, code); !strings.Contains(reason, "merge-auto-human") {
		t.Fatalf("a moved origin/main that says auto does not decide the mode: %q", reason)
	}

	gone := guardRepo(t)
	commitConfig(t, gone, `{"facilitator":{"mode":"auto"}}`+"\n")
	withOrigin(t, gone)
	git(gone, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "no-such.git"))
	stdout, stderr, code = runGuard(preToolUse(t, "Bash", merge, gone), "guard", "hook")
	if reason := mustDeny(t, stdout, stderr, code); !strings.Contains(reason, "merge-auto-human") || !strings.Contains(reason, "cannot be read") {
		t.Fatalf("an origin that cannot be reached counts as human, saying the mode cannot be read: %q", reason)
	}
}
