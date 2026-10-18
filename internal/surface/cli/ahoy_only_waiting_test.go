package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAhoyInstallOnlyWaitingWithYesLeavesTheValueQuestions: the flag
// reaches the install: an --only-waiting --yes run leaves the artefact kind
// a plain --yes run would declare by default, and says what still waits.
func TestAhoyInstallOnlyWaitingWithYesLeavesTheValueQuestions(t *testing.T) {
	repo, _, _ := afterUpdateRepo(t)
	t.Setenv("ABCD_BIN_TARGET", filepath.Join(t.TempDir(), "bin", "abcd"))
	t.Setenv("PATH", "/usr/bin:/bin")
	stdout, stderr, code := runSessionStart("", "ahoy", "install", "--only-waiting", "--yes", "--adopt", "--allow-stale-binary")
	if code != 0 && code != 2 {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".abcd", "config", "artefact.json")); !os.IsNotExist(err) {
		t.Errorf("--only-waiting --yes declared the artefact kind (err %v):\n%s", err, stdout)
	}
	if !strings.Contains(stdout, "abcd ahoy install --only-waiting") {
		t.Errorf("no note names the run that asks what was left:\n%s", stdout)
	}
}
