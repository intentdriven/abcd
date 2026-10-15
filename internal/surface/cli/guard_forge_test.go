package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// The forge-text check at the hook (iss-2610142210511831): in a repository
// that opted in with `ahoy install --outbound-forge`, a posting gh command
// whose text carries a live agent-session URL or a tool attribution footer is
// the host's deny, whose reason names what was found and never the text.

// forgeHookFooter is a tool attribution footer, assembled at run time so no
// committed file carries one.
func forgeHookFooter() string {
	return "🤖 Generated " + "with [Some Tool](https://sometool.dev)"
}

func forgeHookCommand() string {
	return `gh pr create --title "fix: the walk" --body "Summary` + "\n\n" + forgeHookFooter() + `"`
}

func TestGuardHookRefusesForgeTextInAnOptedInRepository(t *testing.T) {
	dir := guardRepo(t)
	writeFile(t, filepath.Join(dir, ".abcd", "config.json"), `{"outbound": {"forge": true}}`)

	stdout, stderr, code := runGuard(preToolUse(t, "Bash", forgeHookCommand(), dir), "guard", "hook")
	reason := mustDeny(t, stdout, stderr, code)
	if !strings.Contains(reason, "forge-text-outbound") || !strings.Contains(reason, "tool attribution footer") {
		t.Errorf("reason = %q, want the forge-text block naming the footer", reason)
	}
	if strings.Contains(reason, "Some Tool") || strings.Contains(reason, "sometool.dev") {
		t.Errorf("the deny republished the text it refused: %q", reason)
	}

	clean := `gh pr create --title "fix: the walk" --body "Assisted-by: Claude:claude-opus-5-5"`
	if stdout, stderr, code := runGuard(preToolUse(t, "Bash", clean, dir), "guard", "hook"); code != 0 || stdout != "" {
		t.Errorf("a clean posting command: exit %d stdout %q stderr %q, want a silent allow", code, stdout, stderr)
	}
}

func TestGuardHookLeavesForgeTextAloneWhereTheRepositoryDidNotOptIn(t *testing.T) {
	dir := guardRepo(t)
	if stdout, stderr, code := runGuard(preToolUse(t, "Bash", forgeHookCommand(), dir), "guard", "hook"); code != 0 || stdout != "" {
		t.Errorf("exit %d stdout %q stderr %q, want a silent allow in a repository that did not opt in", code, stdout, stderr)
	}
}

// TestGuardHookRefusesForgeTextInThisRepository: abcd's own repository opts
// in (.abcd/config.json), so a session here that hands the forge a tool footer
// is refused before the command runs.
func TestGuardHookRefusesForgeTextInThisRepository(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	stdout, stderr, code := runGuard(preToolUse(t, "Bash", forgeHookCommand(), root), "guard", "hook")
	reason := mustDeny(t, stdout, stderr, code)
	if !strings.Contains(reason, "forge-text-outbound") {
		t.Errorf("reason = %q, want the forge-text block", reason)
	}
}
