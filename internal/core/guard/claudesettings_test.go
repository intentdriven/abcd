package guard

import "testing"

// TestGuardRefusesClaudeWithSettingsThatTurnItsChecksOff: `claude --settings`
// that sets permissions.defaultMode to bypassPermissions, or grants
// permissions.allow rules, starts an agent that asks no one, so it is refused
// under its entry whether the JSON is written on the line or held in the file
// the flag names, where the guard can read that file. Settings that leave the
// checks on, a file the guard cannot find, and a file the line writes first
// are not refused by this entry.
func TestGuardRefusesClaudeWithSettingsThatTurnItsChecksOff(t *testing.T) {
	const id = "agent-launch-claude-unchecked-settings"
	r := Defaults()
	for _, line := range []string{
		`claude -p x --settings '{"permissions":{"defaultMode":"bypassPermissions"}}'`,
		`claude --settings '{"permissions":{"allow":["Bash(*)"]}}' -p x`,
		`claude --settings='{"permissions":{"defaultMode":"bypassPermissions"}}' -p x`,
		`npx @anthropic-ai/claude-code --settings '{"permissions":{"allow":["Bash"]}}' -p x`,
		`sh -c 'claude -p x --settings "{\"permissions\":{\"defaultMode\":\"bypassPermissions\"}}"'`,
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
	// Settings the line does not wholly fix are read by the text it fixes.
	if d, err := r.Check(`claude -p x --settings "{\"permissions\":{\"defaultMode\":\"$(cat mode)\"}}"`); err != nil || d.Verdict != VerdictBlock {
		t.Errorf("settings with a substitution in them: verdict %q under %q (%v), want a block", d.Verdict, d.EntryID, err)
	}
	for _, line := range []string{
		`claude -p x --settings '{"permissions":{"defaultMode":"acceptEdits"}}'`,
		`claude -p x --settings '{"permissions":{"allow":[]}}'`,
		`claude -p x --settings '{"permissions":{"deny":["Bash(rm:*)"]}}'`,
		`claude -p x --settings '{"model":"sonnet"}'`,
		`claude -p x --settings settings.json`,
		`echo '{"permissions":{"defaultMode":"bypassPermissions"}}'`,
	} {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}

	dir := scriptTree(t, map[string]string{
		"w/bypass.json": `{"permissions": {"defaultMode": "bypassPermissions"}}`,
		"w/allow.json":  `{"permissions": {"allow": ["Bash(*)"]}}`,
		"w/sub/s.json":  `{"permissions": {"allow": ["Edit"]}}`,
		"w/plain.json":  `{"model": "sonnet", "permissions": {"deny": ["WebFetch"]}}`,
		"w/broken.json": `{"permissions": `,
		"acct/keep":     "",
	})
	runExtendCases(t, dir, []extendCase{
		{`claude --settings bypass.json -p x`, VerdictBlock, id, ""},
		{`claude -p x --settings ./allow.json`, VerdictBlock, id, ""},
		{`claude -p x --settings={d}/w/bypass.json`, VerdictBlock, id, ""},
		{`cd sub && claude -p x --settings s.json`, VerdictBlock, id, ""},
		{`eval 'claude -p x --settings bypass.json'`, VerdictBlock, id, ""},

		{`claude -p x --settings plain.json`, VerdictAllow, "", ""},
		{`claude -p x --settings broken.json`, VerdictAllow, "", ""},
		{`claude -p x --settings missing.json`, VerdictAllow, "", ""},
		{`cat bypass.json`, VerdictAllow, "", ""},
		{`claude -p x -- --settings bypass.json`, VerdictAllow, "", ""},
	})
}
