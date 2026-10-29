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

// TestTheGuardHookReadsTheFacilitatorMode is spc-2610161609350616 step 5 at
// the hook: with no facilitator.mode (human) the hook refuses an agent's
// `gh pr merge --auto`, naming the setting that allows it; with auto on the
// default branch it lets it through. The approval act is refused in both.
func TestTheGuardHookReadsTheFacilitatorMode(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	merge := strings.Join([]string{"gh", "pr", "merge", "7", "--auto", "--merge"}, " ")
	dir := guardRepo(t)
	commitConfig(t, dir, "")
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
	stdout, _, code = runGuard(preToolUse(t, "Bash", merge, auto), "guard", "hook")
	if code != 0 || strings.Contains(stdout, "deny") {
		t.Fatalf("auto lets gh pr merge --auto through: exit %d, stdout %q", code, stdout)
	}
	stdout, stderr, code = runGuard(preToolUse(t, "Bash", "abcd land --approve", auto), "guard", "hook")
	if reason := mustDeny(t, stdout, stderr, code); !strings.Contains(reason, "abcd-land-approve") {
		t.Fatalf("the approval act is refused in auto too: %q", reason)
	}
}
