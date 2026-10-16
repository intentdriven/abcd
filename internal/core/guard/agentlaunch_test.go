package guard

import "testing"

// TestGuardRefusesAnAgentLaunchWithItsChecksOff: a line that starts another
// agent harness with that harness's own permission checks turned off is a
// registry blocker, so the hook blocks it and the headless approver refuses
// it under the entry's id. The same harness started with its checks on is
// not touched.
func TestGuardRefusesAnAgentLaunchWithItsChecksOff(t *testing.T) {
	r := Defaults()
	for line, id := range map[string]string{
		`claude -p "fix the tests" --dangerously-skip-permissions`:    "agent-launch-claude-unchecked",
		`claude --dangerously-skip-permissions`:                       "agent-launch-claude-unchecked",
		`claude -p x --permission-mode bypassPermissions`:             "agent-launch-claude-unchecked-mode",
		`claude -p x --permission-mode=bypassPermissions`:             "agent-launch-claude-unchecked-mode",
		`codex exec --dangerously-bypass-approvals-and-sandbox "fix"`: "agent-launch-codex-unchecked",
		`codex exec --yolo "fix"`:                                     "agent-launch-codex-unchecked",
		`codex -a never exec "fix"`:                                   "agent-launch-codex-never-asks",
		`codex exec --ask-for-approval=never "fix"`:                   "agent-launch-codex-never-asks",
		`codex exec -s danger-full-access "fix"`:                      "agent-launch-codex-sandbox-off",
		`codex exec --sandbox danger-full-access "fix"`:               "agent-launch-codex-sandbox-off",
		`codex exec -c approval_policy=never "fix"`:                   "agent-launch-codex-config-unchecked",
		`codex exec -c 'sandbox_mode="danger-full-access"' "fix"`:     "agent-launch-codex-config-unchecked",
		`opencode run --auto "fix the tests"`:                         "agent-launch-opencode-auto",
		`cd sub && opencode run --auto x`:                             "agent-launch-opencode-auto",
	} {
		d, err := r.Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock || d.EntryID != id {
			t.Errorf("%q: verdict %q under %q, want a block under %q", line, d.Verdict, d.EntryID, id)
		}
		if a := approveShell(clean(), NewFiles(t.TempDir()), line); a.Allow || a.EntryID != id {
			t.Errorf("%q: the approver answered %+v, want a refusal under %q", line, a, id)
		}
	}
	for _, line := range []string{
		`claude -p "fix the tests" --permission-mode default`,
		`claude -p x --permission-mode acceptEdits`,
		`codex exec "fix the tests"`,
		`codex exec -s workspace-write -a on-request "fix"`,
		`opencode run "fix the tests"`,
		`abcd capture "an agent ran claude --dangerously-skip-permissions"`,
	} {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}
}

// TestGuardRefusesAnAgentLaunchThroughARunner: the harness is the program
// whether the line names it or a launcher starts it: a package runner
// (npx, bunx, bun x, pnpm dlx, pnpm exec, yarn dlx, npm exec, with or
// without `--`), by its program name or by the package that ships it, and
// env or command in front of it.
func TestGuardRefusesAnAgentLaunchThroughARunner(t *testing.T) {
	r := Defaults()
	const claudeID, codexID, opencodeID = "agent-launch-claude-unchecked", "agent-launch-codex-unchecked", "agent-launch-opencode-auto"
	cases := map[string]string{}
	for _, runner := range []string{"npx", "npx --yes", "npx --", "bunx", "bunx --bun", "bun x", "pnpm dlx", "pnpm exec", "pnpm exec --", "yarn dlx", "npm exec", "npm exec --", "npm x --", "env", "env FOO=1", "command", "env -- command"} {
		cases[runner+` claude -p x --dangerously-skip-permissions`] = claudeID
		cases[runner+` codex exec --yolo "fix"`] = codexID
		cases[runner+` opencode run --auto x`] = opencodeID
	}
	for _, runner := range []string{"npx", "npx -y", "npx --", "bunx", "bun x", "pnpm dlx", "yarn dlx", "npm exec --"} {
		cases[runner+` @anthropic-ai/claude-code --dangerously-skip-permissions`] = claudeID
		cases[runner+` @anthropic-ai/claude-code@latest -p x --dangerously-skip-permissions`] = claudeID
		cases[runner+` @openai/codex exec --yolo "fix"`] = codexID
		cases[runner+` opencode-ai@1 run --auto x`] = opencodeID
	}
	for line, id := range cases {
		d, err := r.Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock || d.EntryID != id {
			t.Errorf("%q: verdict %q under %q, want a block under %q", line, d.Verdict, d.EntryID, id)
		}
		if a := approveShell(clean(), NewFiles(t.TempDir()), line); a.Allow || a.EntryID != id {
			t.Errorf("%q: the approver answered %+v, want a refusal under %q", line, a, id)
		}
	}
	for _, line := range []string{
		`npx @anthropic-ai/claude-code -p "fix the tests"`,
		`bunx @openai/codex exec "fix the tests"`,
		`pnpm exec opencode run "fix the tests"`,
		`npx claude-code-helper --dangerously-skip-permissions`,
		`npx prettier --write .`,
	} {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}
}
