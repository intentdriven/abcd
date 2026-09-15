package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// The three front doors that share rules.ResolveRoot, held to the same bound
// when git itself will not answer for a repository that plainly is one.
//
// abcd runs every git command under an isolated environment
// (GIT_CONFIG_GLOBAL=/dev/null, GIT_CONFIG_NOSYSTEM=1) so an inherited GIT_DIR
// or an injected config cannot redirect it. That isolation also discards the
// developer's `safe.directory` exceptions, so `rev-parse --show-toplevel` fails
// inside a checkout owned by another uid — and it fails outright when the host
// launches the hook with git off its PATH. Resolving those to cwd silently
// dropped the repository's own configuration layer: the guard registry became
// the bundled defaults, the rules kill switch stopped applying, and the private
// banlist store moved to whatever subdirectory the session was in. Nothing on
// any of those paths says "the repo layer was dropped", because none of them
// saw an error.

// mustMkdirAll creates dir (and its parents) or fails the test.
func mustMkdirAll(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// gitRefusesOwnership stages the ownership refusal deterministically: git's own
// test hook makes the check fire, and the isolated environment abcd runs git
// under has already discarded the `safe.directory` line that would rescue it.
func gitRefusesOwnership(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("GIT_TEST_ASSUME_DIFFERENT_OWNER", "1")
	if _, err := gitutil.Run(dir, "rev-parse", "--show-toplevel"); err == nil {
		t.Skip("this git ignores GIT_TEST_ASSUME_DIFFERENT_OWNER; the refusal could not be staged")
	}
}

// TestGuardHookHonoursTheRepoRegistryWhenGitRefuses is the hazard half. A repo
// that added its own blocker to .abcd/guard.json must keep it from a nested
// working directory even when git will not name the toplevel — otherwise the
// registry silently degrades to the bundled defaults and the repo's own hazard
// becomes an allow, with exit 0 and no diagnostic at all.
func TestGuardHookHonoursTheRepoRegistryWhenGitRefuses(t *testing.T) {
	repo := t.TempDir()
	gitInitAt(t, repo)
	mustMkdirAll(t, filepath.Join(repo, ".abcd"))
	cfg := `{"schema_version":1,"entries":{"drop-database":{
		"tier":"blocker",
		"pattern":{"command":"dropdb"},
		"why":"It deletes the whole database.",
		"successor":"Take a dump first."}}}`
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "guard.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := mustMkdirAll(t, filepath.Join(repo, "internal", "deep"))

	gitRefusesOwnership(t, sub)

	_, stderr, code := runGuard(preToolUse(t, "Bash", "dropdb production", sub), "guard", "hook")
	if code != 2 {
		t.Errorf("the repo's own blocker did not fire from a subdirectory git would not answer for: exit %d, stderr %q", code, stderr)
	}
	if !strings.Contains(stderr, "deletes the whole database") {
		t.Errorf("the block message must come from the repo registry; stderr = %q", stderr)
	}
}

// TestGuardHookHonoursTheRepoKillSwitchWhenGitRefuses is the same resolution on
// the other side of the switch: a repo that deliberately turned its guard off
// must be reported as UNGUARDED from a nested directory, not quietly re-armed
// with the bundled defaults. Reading the wrong file is wrong in both
// directions; the point is that the repo's file is the one that governs.
func TestGuardHookHonoursTheRepoKillSwitchWhenGitRefuses(t *testing.T) {
	repo := t.TempDir()
	gitInitAt(t, repo)
	mustMkdirAll(t, filepath.Join(repo, ".abcd"))
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "guard.json"), []byte(`{"schema_version":1,"disabled":true,"entries":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := mustMkdirAll(t, filepath.Join(repo, "pkg"))

	gitRefusesOwnership(t, sub)

	_, stderr, code := runGuard(preToolUse(t, "Bash", "cd scratch && rm -rf *", sub), "guard", "hook")
	if code == 2 {
		t.Errorf("the repo's kill switch was not seen from a subdirectory: the session was told it is guarded when the repo switched the guard off (stderr %q)", stderr)
	}
	if !strings.Contains(stderr, "UNGUARDED") {
		t.Errorf("a disabled repo registry must announce the unguarded session; stderr = %q", stderr)
	}
}

// TestHookPromptRouterHonoursTheRepoKillSwitchWhenGitIsAbsent is the rules half,
// with git absent from the hook's PATH rather than refusing: a repo that set
// "disabled": true must inject nothing into a session in a subdirectory. Under
// the collapsed resolution the root became the subdirectory, rules.Load found no
// file there, returned the bundled defaults with no error, and the kill switch
// the repo committed was simply not read.
func TestHookPromptRouterHonoursTheRepoKillSwitchWhenGitIsAbsent(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	repo := t.TempDir()
	gitInitAt(t, repo) // needs git, so it happens before the PATH is emptied
	mustMkdirAll(t, filepath.Join(repo, ".abcd"))
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "rules.json"), []byte(`{"schema_version":1,"disabled":true,"domains":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := mustMkdirAll(t, filepath.Join(repo, "internal", "deep"))

	t.Setenv("PATH", "/nonexistent")
	if out, err := gitutil.Run(sub, "rev-parse", "--show-toplevel"); err == nil {
		t.Fatalf("git answered %q with an emptied PATH; the fixture did not stage the failure", out)
	}

	out, errlog := runHook(t, hookInputJSON(t, "s1", sub, "commit and push"), "hook", "prompt-router")
	if out != "" {
		t.Fatalf("the repo kill switch was ignored from a subdirectory git could not be asked about; injected:\n%s\nstderr:\n%s", out, errlog)
	}
}

// TestBanlistFindsTheRepoStoreWhenGitRefuses is the third front door.
// banlistRoot asks git first and falls through to the shared resolver exactly
// when git will not answer, so the collapsed resolution made the verb read the
// SUBDIRECTORY: a repo with a populated private store reported INACTIVE, which
// on this layer reads as "this machine has not opted in" — the operator is told
// their banned names are not being matched when they are, or vice versa. (The
// write path is separately fail-closed here: requireIgnoredStore refuses to
// create a store it cannot prove is gitignored. The read path had no such
// backstop.)
func TestBanlistFindsTheRepoStoreWhenGitRefuses(t *testing.T) {
	repo := blRepo(t, "widget-partner   widgetworks\n")
	gitInitAt(t, repo)
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(".abcd/.work.local/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := mustMkdirAll(t, filepath.Join(repo, "internal", "deep"))
	t.Chdir(sub)

	gitRefusesOwnership(t, sub)

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"banlist"}, &stdout, &stderr); code != 0 {
		t.Fatalf("banlist exit = %d, stderr = %s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "widget-partner") {
		t.Errorf("the repo's private store was not found from a subdirectory git would not answer for:\n%s", out)
	}
	if !strings.Contains(out, "harness/gemini") {
		t.Errorf("the repo's public config was not found either:\n%s", out)
	}
	if strings.Contains(out, "INACTIVE") {
		t.Errorf("a populated private store was reported inactive:\n%s", out)
	}
	if strings.Contains(out, "widgetworks") {
		t.Errorf("the render leaks the private pattern:\n%s", out)
	}
}

// TestHookPromptRouterNamesASkippedDomain is the front-door half of the
// ruleless-domain skip: the drop is per domain, so the rest of the ruleset must
// still inject — and the skipped domain must be named out of band, because a
// domain that silently stops existing is the same "suppression nobody sees"
// the drop exists to prevent. The diagnostic goes to stderr; stdout is the
// model-facing context and carries nothing about it.
func TestHookPromptRouterNamesASkippedDomain(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	repo := t.TempDir()
	mustMkdirAll(t, filepath.Join(repo, ".abcd"))
	body := `{"schema_version":1,"domains":{"COMMITTING":{"rules":[]},"MINE":{"recall":["widget"],"rules":["mind the widget"]}}}`
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "rules.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	out, errlog := runHook(t, hookInputJSON(t, "s1", repo, "commit the widget"), "hook", "prompt-router")
	if !strings.Contains(out, "mind the widget") {
		t.Errorf("one ruleless domain must not stop the rest of the ruleset injecting; stdout:\n%s\nstderr:\n%s", out, errlog)
	}
	if strings.Contains(out, "## COMMITTING") {
		t.Errorf("the ruleless domain must not inject a heading-only block:\n%s", out)
	}
	if !strings.Contains(errlog, "COMMITTING") || !strings.Contains(errlog, "SKIPPED") {
		t.Errorf("the skipped domain must be named out of band; stderr = %q", errlog)
	}
	if strings.Contains(out, "SKIPPED") {
		t.Errorf("the diagnostic must not reach the model-facing context:\n%s", out)
	}
}

// plantGitMarker creates a `.git`-named entry at dir that is NOT a repository —
// the one-command plant (`: > .git`, `mkdir .git`) any unprivileged process can
// make in a directory it can write, a shared temp directory included.
func plantGitMarker(t *testing.T, dir, shape string) {
	t.Helper()
	marker := filepath.Join(dir, ".git")
	if shape == "dir" {
		mustMkdirAll(t, marker)
		return
	}
	if err := os.WriteFile(marker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestHookPromptRouterIgnoresAPlantedGitMarker is the front-door half of the
// bound on the git-refused fallback. The fallback asks for the nearest
// `.git`-NAMED ancestor, so before the marker was required to look like a
// repository, an empty `/tmp/.git` beside a planted `/tmp/.abcd/rules.json`
// governed every session whose working directory was a plain directory
// underneath: the planted domain injected into the model-facing context.
func TestHookPromptRouterIgnoresAPlantedGitMarker(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	outer := t.TempDir()
	mustMkdirAll(t, filepath.Join(outer, ".abcd"))
	planted := `{"schema_version":1,"domains":{"ANCESTOR":{"state":"active","recall":["ancestor"],"rules":["rules planted beside a fake .git marker"]}}}`
	if err := os.WriteFile(filepath.Join(outer, ".abcd", "rules.json"), []byte(planted), 0o644); err != nil {
		t.Fatal(err)
	}
	plantGitMarker(t, outer, "file")
	plain := mustMkdirAll(t, filepath.Join(outer, "work"))

	out, errlog := runHook(t, hookInputJSON(t, "planted", plain, "ancestor"), "hook", "prompt-router")
	if strings.Contains(out, "## ANCESTOR") {
		t.Fatalf("a .abcd beside a planted .git marker governed the session:\n%s", out)
	}
	if out != "" {
		t.Fatalf("expected a no-match (zero model-facing stdout), got:\n%s\nstderr:\n%s", out, errlog)
	}
}

// TestGuardHookIgnoresAPlantedGitMarkerKillSwitch is the same plant on the
// guard plane, where the consequence is the loud one: a planted
// `.abcd/guard.json` with "disabled": true answered a hazardous command with
// "UNGUARDED" and exit 1, so a session in a plain directory beneath the plant
// ran `cd scratch && rm -rf *` unblocked. The bundled hazards must stay armed.
func TestGuardHookIgnoresAPlantedGitMarkerKillSwitch(t *testing.T) {
	outer := t.TempDir()
	mustMkdirAll(t, filepath.Join(outer, ".abcd"))
	if err := os.WriteFile(filepath.Join(outer, ".abcd", "guard.json"), []byte(`{"schema_version":1,"disabled":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	plantGitMarker(t, outer, "dir")
	plain := mustMkdirAll(t, filepath.Join(outer, "work"))

	_, stderr, code := runGuard(preToolUse(t, "Bash", "cd scratch && rm -rf *", plain), "guard", "hook")
	if code != 2 {
		t.Errorf("a guard.json beside a planted .git marker disarmed the guard: exit %d, stderr %q", code, stderr)
	}
	if !strings.Contains(stderr, "rm-rf-after-cd-chain") {
		t.Errorf("the bundled blocker must still fire; stderr = %q", stderr)
	}
	if strings.Contains(stderr, "UNGUARDED") {
		t.Errorf("the planted kill switch was honoured; stderr = %q", stderr)
	}
}
