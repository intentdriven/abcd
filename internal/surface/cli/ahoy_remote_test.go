package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestAhoyRemoteRefusesAnUnmanagedFolderFromTheCLI is the front-door half of
// itd-153's trust boundary. The verb is the one thing in abcd that mutates state
// outside the machine, so the CLI must carry the refusal all the way out: a
// non-zero exit for the apply, and the reason on stdout rather than only in a
// result nobody rendered.
func TestAhoyRemoteRefusesAnUnmanagedFolderFromTheCLI(t *testing.T) {
	hermeticEnv(t)
	dir := t.TempDir()
	t.Chdir(dir)

	// The read is reachable and reports the refusal without exiting non-zero: it is
	// abcd's bare-render convention, and looking is never an error.
	out := string(runCLI(t, "ahoy", "--remote"))
	if !strings.Contains(out, "refused") {
		t.Fatalf("`ahoy --remote` did not refuse a folder abcd does not manage:\n%s", out)
	}
	if !strings.Contains(out, "note:") {
		t.Errorf("the refusal carries no reason on stdout:\n%s", out)
	}

	applied, err := runCLIErr(t, "ahoy", "remote", "apply")
	if err == nil {
		t.Fatalf("`ahoy remote apply` exited zero on a refusal; a write that did not happen must not look like one that did:\n%s", applied)
	}
	if !strings.Contains(string(applied), "refused") {
		t.Errorf("the apply's refusal is not rendered:\n%s", applied)
	}
	// Nothing was written, least of all the desired-state mirror.
	if _, statErr := os.Stat(filepath.Join(dir, ".abcd", "work", "rulesets", "repo-settings.json")); statErr == nil {
		t.Error("a refused apply wrote the desired-state mirror")
	}
}

// TestAhoyRemoteApplyExitsNonZeroWhenItChangesNothing pins the exit-code contract
// on the abort path, which is the one a script actually hits: without --yes a
// non-interactive run reads EOF from the prompter, DECLINES, and would otherwise
// print "aborted" and exit 0 — indistinguishable, to the caller, from a write that
// landed. The confirmation is adr-44's fourth gate; an exit code that hides it
// would hand the caller back the ambiguity the gate exists to remove.
func TestAhoyRemoteApplyExitsNonZeroWhenItChangesNothing(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	hermeticEnv(t)
	repo := gittest.NewRepo(t)
	repo.Git("remote", "add", "origin", "https://github.com/example-org/example-repo.git")
	t.Chdir(repo.Root())
	if _, err := runCLIErr(t, "ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "both",
		"--oracle-backend", "host-delegated", "--scan-deep", "false"); err != nil {
		t.Fatalf("install: %v", err)
	}

	// A stand-in `gh` reporting both toggles disabled, so the verb reaches the
	// confirmation with a real change to offer.
	dir := t.TempDir()
	script := "#!/usr/bin/env bash\ncat >/dev/null\n" +
		`printf '%s' '{"security_and_analysis":{"secret_scanning":{"status":"disabled"},` +
		`"secret_scanning_push_protection":{"status":"disabled"}}}'` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	out, err := runCLIErr(t, "ahoy", "remote", "apply")
	if err == nil {
		t.Fatalf("an unconfirmed apply exited zero; a change that did not happen must not look like one that did:\n%s", out)
	}
	if !strings.Contains(string(out), "aborted") {
		t.Errorf("the abort is not rendered:\n%s", out)
	}
}
