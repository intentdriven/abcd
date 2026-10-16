package guard

import (
	"strings"
	"testing"
)

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

// TestGuardRefusesOpencodeWithItsPermissionFromTheEnvironment: opencode
// reads its permission setting from OPENCODE_PERMISSION as well as from its
// configuration, so a line that sets the variable for the opencode it
// starts (an assignment before it, an operand of env, or an export earlier
// in the line) is refused under the opencode entry, with that entry's
// reason and what to run instead. A line that sets it for another program,
// or starts opencode without setting it, is not.
func TestGuardRefusesOpencodeWithItsPermissionFromTheEnvironment(t *testing.T) {
	const id = "agent-launch-opencode-permission-env"
	r := Defaults()
	for _, line := range []string{
		`OPENCODE_PERMISSION='{"*":"allow"}' opencode run x`,
		`FOO=1 OPENCODE_PERMISSION='{"bash":"allow"}' opencode run x`,
		`env OPENCODE_PERMISSION='{"*":"allow"}' opencode run x`,
		`env -i PATH=/usr/bin OPENCODE_PERMISSION=x opencode`,
		`OPENCODE_PERMISSION=x npx opencode-ai run x`,
		`export OPENCODE_PERMISSION='{"*":"allow"}'; opencode run x`,
		`export OPENCODE_PERMISSION='{"*":"allow"}' && opencode run x`,
		"export OPENCODE_PERMISSION=x\nopencode run x",
		`export FOO=1 OPENCODE_PERMISSION=x; cd sub && opencode run x`,
		`OPENCODE_PERMISSION=x; export OPENCODE_PERMISSION; opencode run x`,
		`cd sub && OPENCODE_PERMISSION=x opencode run x`,
	} {
		d, err := r.Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock || d.EntryID != id {
			t.Errorf("%q: verdict %q under %q, want a block under %q", line, d.Verdict, d.EntryID, id)
		}
		a := approveShell(clean(), NewFiles(t.TempDir()), line)
		if a.Allow || a.EntryID != id {
			t.Errorf("%q: the approver answered %+v, want a refusal under %q", line, a, id)
			continue
		}
		if !strings.Contains(a.Reason, "OPENCODE_PERMISSION") || !strings.Contains(a.Reason, "Run instead: ") {
			t.Errorf("%q: the refusal does not carry the entry's reason and successor: %s", line, a.Reason)
		}
	}
	for _, line := range []string{
		`opencode run "fix the tests"`,
		`OPENCODE_PERMISSION=x echo hi`,
		`export OPENCODE_PERMISSION=x`,
		`env -u OPENCODE_PERMISSION opencode run x`,
		`opencode run "OPENCODE_PERMISSION=x"`,
		`FOO=OPENCODE_PERMISSION opencode run x`,
		`abcd capture "an agent ran OPENCODE_PERMISSION=x opencode run"`,
	} {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}
}

// TestAnEnvPatternIsValidatedAndOverridden: a pattern's env names are
// variable names a line can set, so any other word is refused at load, and
// a repository's override replaces the list as it replaces every other
// field, so its own entry can name a variable for its own program.
func TestAnEnvPatternIsValidatedAndOverridden(t *testing.T) {
	bad := Registry{SchemaVersion: 1, Entries: map[string]Entry{"x": {
		Tier: TierBlocker, Why: "w", Successor: "s",
		Pattern: Pattern{Command: "tool", Env: []string{"NOT-A-NAME"}},
	}}}
	if err := Validate(bad); err == nil || !strings.Contains(err.Error(), "env name") {
		t.Errorf("an env name no line can set was accepted: %v", err)
	}
	root := writeOverride(t, `{"schema_version":1,"entries":{
		"tool-env":{"tier":"blocker","pattern":{"command":"tool","env":["TOOL_MODE"]},"why":"tool reads its mode from TOOL_MODE","successor":"run tool without it"}}}`)
	r, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for line, want := range map[string]Verdict{
		`TOOL_MODE=open tool run`:         VerdictBlock,
		`export TOOL_MODE=open; tool run`: VerdictBlock,
		`tool run`:                        VerdictAllow,
		`TOOL_MODE=open other run`:        VerdictAllow,
		`OTHER=open tool run`:             VerdictAllow,
	} {
		if d, err := r.Check(line); err != nil || d.Verdict != want {
			t.Errorf("%q: verdict %q under %q (%v), want %q", line, d.Verdict, d.EntryID, err, want)
		}
	}
	if got := r.Entries["tool-env"].Pattern.Describe(); !strings.Contains(got, "`TOOL_MODE` set in its environment") {
		t.Errorf("the lesson does not name the variable: %s", got)
	}
}
