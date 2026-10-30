package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
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
		"--visibility", "private", "--docs-target", "agents_md",
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
	// An abort keeps exit 1, where scripts already read it; a refusal is exit 2
	// (TestAhoyRemoteApplyExitCodes, iss-2610031915386832).
	var ee *exitError
	if !errors.As(err, &ee) || ee.Code != 1 {
		t.Errorf("an unconfirmed apply: %v, want exit 1", err)
	}
	if !strings.Contains(string(out), "aborted") {
		t.Errorf("the abort is not rendered:\n%s", out)
	}
}

// TestAhoyRemoteApplyRefusalAfterAToggleDoesNotSayNothingChanged
// (iss-2610072347230409): secret scanning lands, push protection is refused by
// GitHub. The verb still refuses (exit 2), but its closing line must not claim
// nothing was changed, and must name the toggle that did land.
func TestAhoyRemoteApplyRefusalAfterAToggleDoesNotSayNothingChanged(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	hermeticEnv(t)
	repo := gittest.NewRepo(t)
	repo.Git("remote", "add", "origin", "https://github.com/example-org/example-repo.git")
	t.Chdir(repo.Root())
	if _, err := runCLIErr(t, "ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "agents_md",
		"--oracle-backend", "host-delegated", "--scan-deep", "false"); err != nil {
		t.Fatalf("install: %v", err)
	}

	// A stand-in `gh`: the GET reports both toggles disabled, the secret-scanning
	// PATCH succeeds, the push-protection PATCH fails as GitHub would.
	dir := t.TempDir()
	script := "#!/usr/bin/env bash\nstdin=$(cat)\n" +
		`case "$*" in *PATCH*)` + "\n" +
		`  case "$stdin" in *secret_scanning_push_protection*) echo "gh: HTTP 422" >&2; exit 1;; esac` + "\n" +
		"  exit 0;;\nesac\n" +
		`printf '%s' '{"security_and_analysis":{"secret_scanning":{"status":"disabled"},` +
		`"secret_scanning_push_protection":{"status":"disabled"}}}'` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	out, err := runCLIErr(t, "ahoy", "remote", "apply", "--yes")
	var ee *exitError
	if !errors.As(err, &ee) || ee.Code != 2 {
		t.Fatalf("a refusal after a toggle landed: %v, want exit 2\n%s", err, out)
	}
	if strings.Contains(ee.Msg, "nothing was changed") {
		t.Errorf("the closing line says nothing was changed after secret scanning landed: %q", ee.Msg)
	}
	if !strings.Contains(ee.Msg, "secret_scanning: disabled -> enabled") {
		t.Errorf("the closing line does not name the toggle that landed: %q", ee.Msg)
	}
	if !strings.Contains(string(out), "change:   secret_scanning: disabled -> enabled") {
		t.Errorf("the landed change is not rendered:\n%s", out)
	}
}

// TestRenderRemoteResultPrintsOneLinePerSettingThatIsNotOK pins the audit's
// text form (itd-2610290746077716): one line per setting that is not at its
// target, naming the setting, its current value and its target, its state and
// why; the settings already at their target as one count; the visibility and
// the plan; and every value sanitised, since GitHub's answers are input.
func TestRenderRemoteResultPrintsOneLinePerSettingThatIsNotOK(t *testing.T) {
	res := ahoy.RemoteResult{
		Repo:   "example-org/example-repo",
		Status: "pending",
		Audit: &ahoy.SettingsAudit{
			Visibility: "public", DeclaredVisibility: "public", Plan: "free",
			RequiredChecks: []string{"build", "lint"},
			Findings: []ahoy.Finding{
				{Setting: "has_wiki", Current: "true", Target: "false", State: "differs"},
				{Setting: "release_tags", Current: "unknown", Target: "protected", State: "owed", Note: "owed until public"},
				{Setting: "org_actions_policy", Current: "unknown", Target: "pinned", State: "unknown", Note: "needs admin:org\x1b[31m"},
				{Setting: "default_branch_deletion", Current: "blocked", Target: "blocked", State: "ok"},
				{Setting: "default_branch_force_push", Current: "blocked", Target: "blocked", State: "ok"},
			},
		},
	}
	var b strings.Builder
	renderRemoteResult(&b, "abcd ahoy --remote", res)
	out := b.String()
	for _, want := range []string{
		"has_wiki: true -> false",
		"[differs]",
		"release_tags: unknown -> protected",
		"[owed] owed until public",
		"needs admin:org",
		"2 settings at their target",
		"visibility: public (declared public), plan free",
		"required checks: build, lint",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("render lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "default_branch_deletion") {
		t.Errorf("a setting at its target got a line of its own:\n%s", out)
	}
	if strings.Contains(out, "\x1b") {
		t.Errorf("an escape sequence from GitHub's answer reached the terminal:\n%q", out)
	}
}
