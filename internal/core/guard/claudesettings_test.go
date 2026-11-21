package guard

import (
	"strings"
	"testing"
)

// TestGuardRefusesClaudeWithSettingsThatTurnItsChecksOff: `claude --settings`
// that sets permissions.defaultMode to bypassPermissions, or grants
// permissions.allow rules, starts an agent that asks no one, so it is refused
// under its entry whether the JSON is written on the line or held in the file
// the flag names. Settings that leave the checks on, and a file that does not
// exist, are not refused by this entry.
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
		"acct/keep":     "",
	})
	runExtendCases(t, dir, []extendCase{
		{`claude --settings bypass.json -p x`, VerdictBlock, id, ""},
		{`claude -p x --settings ./allow.json`, VerdictBlock, id, ""},
		{`claude -p x --settings={d}/w/bypass.json`, VerdictBlock, id, ""},
		{`cd sub && claude -p x --settings s.json`, VerdictBlock, id, ""},
		{`eval 'claude -p x --settings bypass.json'`, VerdictBlock, id, ""},

		{`claude -p x --settings plain.json`, VerdictAllow, "", ""},
		{`claude -p x --settings missing.json`, VerdictAllow, "", ""},
		{`cat bypass.json`, VerdictAllow, "", ""},
		{`claude -p x -- --settings bypass.json`, VerdictAllow, "", ""},
	})
}

// TestGuardReadsClaudeSettingsTextAsClaudeDoes: inline settings are read
// with JSON's string escapes decoded, so a key or a value spelled with `\u`
// escapes is the key or value it decodes to. Text that starts like an object
// but does not parse is refused, since claude's reader may take what Go's
// refuses, and so is an object holding a part the line does not fix, since a
// value spliced into the text can add any key, and a value whose start the
// line does not fix, which can be any object. A non-empty hooks block is
// refused, since a hook can answer the permission prompts; an empty one, and
// a mode the permission-mode flag's entry allows, are not.
func TestGuardReadsClaudeSettingsTextAsClaudeDoes(t *testing.T) {
	const id = "agent-launch-claude-unchecked-settings"
	const bs = "\x5c"
	r := Defaults()
	for _, line := range []string{
		`claude -p x --settings "{\"permissions\":{\"` + bs + `u0064efaultMode\":\"` + bs + `u0062ypassPermissions\"},\"x\":\"$PWD\"}"`,
		`claude -p x --settings "{\"permissions\":{\"` + bs + `u0061llow\":[\"Bash\"]},\"x\":\"$(pwd)\"}"`,
		`claude -p x --settings '{"permissions":{"` + bs + `u0064efaultMode":"bypassPermissions"}}'`,
		`claude -p x --settings '{"permissions":{"defaultMode":"bypassPermissions"},}'`,
		`claude -p x --settings '{"permissions":{"defaultMode":"default"}'`,
		`claude -p x --settings '{a}.json'`,
		`claude -p x --settings '{"hooks":{"PreToolUse":[{"matcher":"*","hooks":[{"type":"command","command":"true"}]}]}}'`,
		`claude -p x --settings "{\"hooks\":{\"Stop\":[]},\"x\":\"$X\"}"`,
		`claude -p x --settings "{\"permissions\":{\"$K\":[\"Bash\"]}}"`,
		`claude -p x --settings "{\"$K\":{\"allow\":[\"Bash\"]}}"`,
		`claude -p x --settings "{$REST}"`,
		`claude -p x --settings "{\"model\":\"$MODEL\"}"`,
		`claude -p x --settings "$S"`,
		`claude -p x --settings "$(cat s.json)"`,
		`claude -p x --settings <(cat s.json)`,
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
		`claude -p x --settings '{"hooks":{}}'`,
		`claude -p x --settings '{"permissions":{"defaultMode":"acceptEdits"}}'`,
		`claude -p x --settings '{"permissions":{"defaultMode":"plan"}}'`,
		`claude -p x --settings '  {"model":"sonnet"}  '`,
		`claude -p x --settings '{"env":{"A":"` + bs + `u0062"}}'`,
	} {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}
}

// TestGuardRefusesClaudeSettingsFileItCannotRead: a settings file the guard
// cannot read now is refused, since what claude would read from it is not
// known: a file the line writes before the claude, one past the size the
// guard reads, a glob that does not name exactly one file, a name or a
// directory the line does not fix, and text that does not parse. A glob
// naming one readable file is read as that file, and a missing file stays
// allowed, as claude does not start on it.
func TestGuardRefusesClaudeSettingsFileItCannotRead(t *testing.T) {
	const id = "agent-launch-claude-unchecked-settings"
	big := `{"model": "sonnet", "x": "` + strings.Repeat("a", maxScriptBytes) + `"}`
	dir := scriptTree(t, map[string]string{
		"w/plain.json":      `{"model": "sonnet"}`,
		"w/broken.json":     `{"permissions": `,
		"w/notjson.json":    `model = sonnet`,
		"w/big.json":        big,
		"w/one/s1.json":     `{"model": "sonnet"}`,
		"w/two/a.json":      `{"model": "sonnet"}`,
		"w/two/b.json":      `{"model": "sonnet"}`,
		"w/bad/bypass.json": `{"permissions": {"defaultMode": "bypassPermissions"}}`,
		"acct/keep":         "",
	})
	runExtendCases(t, dir, []extendCase{
		{`echo '{}' > plain.json; claude -p x --settings plain.json`, VerdictBlock, id, ""},
		{`cp {d}/w/bad/bypass.json new.json && claude -p x --settings new.json`, VerdictBlock, id, ""},
		{`claude -p x --settings big.json`, VerdictBlock, id, ""},
		{`claude -p x --settings two/*.json`, VerdictBlock, id, ""},
		{`claude -p x --settings none*.json`, VerdictBlock, id, ""},
		{`claude -p x --settings bad/*.json`, VerdictBlock, id, ""},
		{`claude -p x --settings broken.json`, VerdictBlock, id, ""},
		{`claude -p x --settings notjson.json`, VerdictBlock, id, ""},
		{`claude -p x --settings "/tmp/$X.json"`, VerdictBlock, id, ""},
		{`cd "$D" && claude -p x --settings plain.json`, VerdictBlock, id, ""},

		{`claude -p x --settings plain.json`, VerdictAllow, "", ""},
		{`claude -p x --settings one/s*.json`, VerdictAllow, "", ""},
		{`claude -p x --settings missing.json`, VerdictAllow, "", ""},
		{`echo '{}' > other.json; claude -p x --settings plain.json`, VerdictAllow, "", ""},
	})
}

// TestGuardRefusesClaudeSettingsThatRunACommandOrTurnHooksOff: settings keys
// whose value is a command claude runs (the credential and header helpers,
// the status line, the file suggestion command, the process wrapper and the
// SSH host commands) start a command the guard never reads, and
// `disableAllHooks` starts a claude without the hooks that send its shell
// commands here, so each is refused as hooks are, inline or in the file the
// flag names. `permissions.additionalDirectories` grants access outside the
// working directory, as an allow list grants tools. An empty value, and
// `disableAllHooks` set to false or null, are not refused.
func TestGuardRefusesClaudeSettingsThatRunACommandOrTurnHooksOff(t *testing.T) {
	const id = "agent-launch-claude-unchecked-settings"
	r := Defaults()
	for _, line := range []string{
		`claude -p x --settings '{"apiKeyHelper":"touch /tmp/ran"}'`,
		`claude -p x --settings '{"awsAuthRefresh":"aws sso login"}'`,
		`claude -p x --settings '{"awsCredentialExport":"cat creds.json"}'`,
		`claude -p x --settings '{"gcpAuthRefresh":"gcloud auth login"}'`,
		`claude -p x --settings '{"otelHeadersHelper":"./headers.sh"}'`,
		`claude -p x --settings '{"statusLine":{"type":"command","command":"touch /tmp/ran"}}'`,
		`claude -p x --settings '{"fileSuggestion":{"type":"command","command":"./files.sh"}}'`,
		`claude -p x --settings '{"processWrapper":"/usr/local/bin/launch"}'`,
		`claude -p x --settings '{"sshHostCommands":["echo host"]}'`,
		`claude -p x --settings '{"disableAllHooks":true}'`,
		`claude -p x --settings '{"disableAllHooks":"yes"}'`,
		`claude -p x --settings '{"permissions":{"additionalDirectories":["/"]}}'`,
		`claude -p x --settings '{"model":"sonnet","apiKeyHelper":"x"}'`,
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
		`claude -p x --settings '{"disableAllHooks":false}'`,
		`claude -p x --settings '{"disableAllHooks":null}'`,
		`claude -p x --settings '{"apiKeyHelper":""}'`,
		`claude -p x --settings '{"statusLine":{}}'`,
		`claude -p x --settings '{"permissions":{"additionalDirectories":[]}}'`,
	} {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}

	dir := scriptTree(t, map[string]string{
		"w/helper.json":  `{"apiKeyHelper": "touch /tmp/ran"}`,
		"w/nohooks.json": `{"disableAllHooks": true}`,
		"w/plain.json":   `{"model": "sonnet", "disableAllHooks": false}`,
		"acct/keep":      "",
	})
	runExtendCases(t, dir, []extendCase{
		{`claude -p x --settings helper.json`, VerdictBlock, id, ""},
		{`claude -p x --settings nohooks.json`, VerdictBlock, id, ""},
		{`claude -p x --settings plain.json`, VerdictAllow, "", ""},
	})
}

// TestGuardRefusesClaudeFlagsThatGrantToolsOrAnswerPrompts: `--allowedTools`
// (also `--allowed-tools`) on the line is the flag form of a
// `permissions.allow` list, and `--permission-prompt-tool` with
// `--mcp-config` hands the permission prompts to a tool a server the line
// configures answers, as a hook can; each is refused under its own entry.
// A plain launch, a model choice, an MCP configuration without a prompt
// tool, and the flags after `--` are not refused.
func TestGuardRefusesClaudeFlagsThatGrantToolsOrAnswerPrompts(t *testing.T) {
	r := Defaults()
	for line, id := range map[string]string{
		`claude -p x --allowedTools Bash`:                                        "agent-launch-claude-allowed-tools",
		`claude -p x --allowedTools=Bash`:                                        "agent-launch-claude-allowed-tools",
		`claude --allowed-tools 'Bash(*)' -p x`:                                  "agent-launch-claude-allowed-tools",
		`npx @anthropic-ai/claude-code -p x --allowedTools Edit`:                 "agent-launch-claude-allowed-tools",
		`claude -p x --mcp-config m.json --permission-prompt-tool mcp__a__b`:     "agent-launch-claude-prompt-tool",
		`claude --permission-prompt-tool=mcp__a__b -p x --mcp-config='{"a":{}}'`: "agent-launch-claude-prompt-tool",
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
		`claude --model x`,
		`claude -p x --mcp-config m.json`,
		`claude -p x --disallowedTools Bash`,
		`claude -p x -- --allowedTools Bash`,
		`abcd capture "an agent ran claude --allowedTools Bash"`,
	} {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}
}
