package guard

import "testing"

// TestGuardRefusesClaudeFlagsThatOfferTheBypassOrWidenItsReach:
// `--allow-dangerously-skip-permissions` lets the agent turn its own
// permission checks off, so it is refused under the bypass flag's entry, and
// `--add-dir` grants the agent directories outside the one it starts in, as
// `permissions.additionalDirectories` does in settings, so it is refused
// under its own entry. A plain launch and the flags after `--` are not.
func TestGuardRefusesClaudeFlagsThatOfferTheBypassOrWidenItsReach(t *testing.T) {
	r := Defaults()
	for line, id := range map[string]string{
		`claude -p x --allow-dangerously-skip-permissions`:                        "agent-launch-claude-unchecked",
		`claude --allow-dangerously-skip-permissions -p x --permission-mode plan`: "agent-launch-claude-unchecked",
		`npx @anthropic-ai/claude-code --allow-dangerously-skip-permissions -p x`: "agent-launch-claude-unchecked",
		`claude -p x --add-dir /`:                                                 "agent-launch-claude-add-dir",
		`claude -p x --add-dir=../other`:                                          "agent-launch-claude-add-dir",
		`claude --add-dir "$D" -p x`:                                              "agent-launch-claude-add-dir",
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
		`claude -p "fix the tests"`,
		`claude --model x -p x`,
		`claude -p x -- --add-dir /`,
		`abcd capture "an agent ran claude --add-dir /"`,
	} {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}
}
