package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/gittest"
)

// commitGuardConfig makes dir a git repository whose HEAD carries cfg as
// .abcd/guard.json. A weakening override — the kill switch, a retiered blocker
// — takes effect only once it is committed (iss-147), so a test of what such
// an override does has to commit it.
func commitGuardConfig(t *testing.T, dir, cfg string) {
	t.Helper()
	gitInitAt(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".abcd", "guard.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"add", ".abcd/guard.json"},
		{"-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture", "-c", "commit.gpgsign=false", "commit", "-m", "guard config"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
}

// TestGuardHookAndCheckAgreeOnAnUncommittedKillSwitch pins both front doors to
// one posture, decided in core (iss-2608291814576261), for the cheapest way to
// switch the guard off: an uncommitted write of `"disabled": true` (iss-147).
// The hook keeps the session guarded by the committed registry and says the
// edit was refused; the check refuses to answer and names the edit. Neither
// reads the working-tree file as the registry in force.
func TestGuardHookAndCheckAgreeOnAnUncommittedKillSwitch(t *testing.T) {
	dir := guardRepo(t)
	gitInitAt(t, dir)
	if err := os.WriteFile(filepath.Join(dir, ".abcd", "guard.json"), []byte(`{"schema_version":1,"disabled":true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, code := runGuard(preToolUse(t, "Bash", "cd scratch && rm -rf *", dir), "guard", "hook")
	if code != 2 || !strings.Contains(stderr, "rm-rf-after-cd-chain") {
		t.Errorf("an uncommitted kill switch disarmed the hook: exit %d, stderr %q", code, stderr)
	}
	if !strings.Contains(stderr, "REFUSED") || !strings.Contains(stderr, guard.RepoRelPath) {
		t.Errorf("the hook must say the uncommitted edit was refused; stderr %q", stderr)
	}

	stdout, stderr, code := runGuard("", "guard", "check", "--command", "cd scratch && rm -rf *")
	if code != 2 {
		t.Errorf("the check must refuse to answer from a registry that is not the one in force: exit %d, stdout %q", code, stdout)
	}
	if !strings.Contains(stderr, "not carry that edit") {
		t.Errorf("the check must name the refused edit; stderr %q", stderr)
	}
}
