package ahoy

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// homeHarnessSettings writes body as the harness's user settings at the
// default location under HOME (~/.claude/settings.json), so every rendering of
// its path is home-relative the way a person's own file is. It returns the
// file's path.
func homeHarnessSettings(t *testing.T, body string) string {
	t.Helper()
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	dir := filepath.Join(os.Getenv("HOME"), ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// strayGaps returns the stray-hook gaps by id, sorted.
func strayGaps(gaps []Gap) map[string]Gap {
	out := map[string]Gap{}
	for _, g := range gaps {
		if strings.HasPrefix(g.ID, harnessStrayHookGapPrefix) {
			out[g.ID] = g
		}
	}
	return out
}

func sortedKeys(m map[string]Gap) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TestDetectReportsAbcdHooksInTheHarnessSettings is iss-2610050556323779: a
// hook in the person's harness settings that runs abcd — the stale
// `abcd-darwin-arm64 hook subagent-stop` that recreated ~/.abcd is the case —
// is reported, naming the file, the event, the command and the remedy, and
// nothing ever removes it: the gap is report-only, and install leaves the file
// byte-identical. A hook that merely mentions an abcd directory is not one.
func TestDetectReportsAbcdHooksInTheHarnessSettings(t *testing.T) {
	home, _ := setupHermetic(t)
	repo := installedRepo(t)
	stale := filepath.Join(home, "ABCDevelopment", "abcd", "bin", "abcd-darwin-arm64")
	settings := homeHarnessSettings(t, `{
  "permissions": {"allow": ["Bash(git log:*)"]},
  "hooks": {
    "SubagentStop": [{"hooks": [{"type": "command", "command": "`+stale+` hook subagent-stop"}]}],
    "PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "abcd guard hook"}]}],
    "UserPromptSubmit": [{"hooks": [{"type": "command", "command": "bash -c 'abcd hook prompt-router'"}]}],
    "Stop": [{"hooks": [{"type": "command", "command": "echo a && echo b"}]}],
    "Notification": [{"hooks": [{"type": "command", "command": "cd /src/abcd && make"}]}]
  }
}
`)
	before, err := os.ReadFile(settings)
	if err != nil {
		t.Fatal(err)
	}

	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	got := strayGaps(det.Gaps)
	want := []string{
		harnessStrayHookGapPrefix + ".PreToolUse",
		harnessStrayHookGapPrefix + ".SubagentStop",
		harnessStrayHookGapPrefix + ".UserPromptSubmit",
	}
	if strings.Join(sortedKeys(got), ",") != strings.Join(want, ",") {
		t.Fatalf("stray-hook gaps = %v, want %v", sortedKeys(got), want)
	}
	for id, g := range got {
		if !g.Required || g.Resolvable || g.Category != ConfigChange || g.Scope != "machine" {
			t.Errorf("%s is not a report-only machine gap: %+v", id, g)
		}
		for _, field := range []string{g.Title, g.Detail, g.FixHint} {
			if strings.Contains(field, home) {
				t.Errorf("%s carries the home path: %q", id, field)
			}
		}
		if !strings.Contains(g.Detail, "~/.claude/settings.json") || !strings.Contains(g.FixHint, "remove this entry from ~/.claude/settings.json") {
			t.Errorf("%s does not name the file and the remedy: %+v", id, g)
		}
	}
	sub := got[harnessStrayHookGapPrefix+".SubagentStop"]
	if !strings.Contains(sub.Detail, "SubagentStop") || !strings.Contains(sub.Detail, "~/ABCDevelopment/abcd/bin/abcd-darwin-arm64 hook subagent-stop") {
		t.Errorf("the SubagentStop gap does not name the event and the command: %q", sub.Detail)
	}
	// The plugin's manifest (validHooksJSON) registers UserPromptSubmit but not
	// SubagentStop, and the remedy says only what is true of each.
	if ups := got[harnessStrayHookGapPrefix+".UserPromptSubmit"]; !strings.Contains(ups.FixHint, "the abcd plugin already runs UserPromptSubmit itself") {
		t.Errorf("UserPromptSubmit remedy = %q", ups.FixHint)
	}
	if strings.Contains(sub.FixHint, "already runs") || !strings.Contains(sub.FixHint, "abcd's hooks belong in its plugin") {
		t.Errorf("SubagentStop remedy = %q", sub.FixHint)
	}

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if after, _ := os.ReadFile(settings); string(after) != string(before) {
		t.Errorf("install edited the harness settings:\n%s", after)
	}
	for _, id := range res.Remaining {
		if strings.HasPrefix(id, harnessStrayHookGapPrefix) {
			t.Errorf("a report-only gap reached install's remaining list: %v", res.Remaining)
		}
	}
	if det, _ := Detect(repo); len(strayGaps(det.Gaps)) != 3 {
		t.Errorf("the stray hooks are no longer reported after install: %v", sortedKeys(strayGaps(det.Gaps)))
	}

	// A machine-scope fact: reported from a folder abcd does not manage too.
	if det, _ := Detect(t.TempDir()); len(strayGaps(det.Gaps)) != 3 {
		t.Errorf("unmanaged folder: stray-hook gaps = %v, want 3", sortedKeys(strayGaps(det.Gaps)))
	}
}

// TestDetectReportsTwoAbcdHooksOnOneEventApart: each entry is its own finding
// with its own id, so neither hides the other.
func TestDetectReportsTwoAbcdHooksOnOneEventApart(t *testing.T) {
	setupHermetic(t)
	homeHarnessSettings(t, `{"hooks": {"SessionEnd": [
  {"hooks": [{"type": "command", "command": "abcd hook session-end"}]},
  {"hooks": [{"type": "command", "command": "/opt/abcd/abcd hook session-end"}]}
]}}`)
	det, err := Detect(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	got := sortedKeys(strayGaps(det.Gaps))
	want := []string{harnessStrayHookGapPrefix + ".SessionEnd", harnessStrayHookGapPrefix + ".SessionEnd.2"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("gaps = %v, want %v", got, want)
	}
}

// TestDetectReportsAnUntrustedStatusLine is the status-line half: a line that
// runs an abcd failing statusLineEntryTrust is reported with the reason and the
// remedy, report-only; the recorded install raises nothing, and a dangling line
// keeps its own repair gap rather than gaining a second one.
func TestDetectReportsAnUntrustedStatusLine(t *testing.T) {
	cases := []struct {
		name    string
		command func(home, pluginRoot string) string
		reason  string // "" => no untrusted gap
	}{
		{"the recorded install", func(string, string) string { return shSingleQuote(os.Getenv("ABCD_BIN_TARGET")) + " statusline" }, ""},
		{"the plugin-root binary", func(_, root string) string { return shSingleQuote(filepath.Join(root, "abcd")) + " statusline" }, "does not record it"},
		{"a stale local build", func(home, _ string) string {
			return filepath.Join(home, "ABCDevelopment", "abcd", "bin", "abcd-darwin-arm64") + " statusline"
		}, "does not record it"},
		{"a bare name", func(string, string) string { return "abcd statusline" }, "not an absolute path"},
		{"a foreign command", func(string, string) string { return previousStatusCommand }, ""},
		{"a dangling line", func(string, string) string { return "'/nowhere/at/all/abcd' statusline" }, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home, pluginRoot := setupHermetic(t)
			repo := installedRepo(t)
			stale := filepath.Join(home, "ABCDevelopment", "abcd", "bin", "abcd-darwin-arm64")
			writeTrustBinary(t, stale)
			homeHarnessSettings(t, `{"statusLine": {"type": "command", "command": "`+tc.command(home, pluginRoot)+`"}}`)
			det, err := Detect(repo)
			if err != nil {
				t.Fatal(err)
			}
			var found *Gap
			for i, g := range det.Gaps {
				if g.ID == statusLineUntrustedGapID {
					found = &det.Gaps[i]
				}
			}
			if tc.reason == "" {
				if found != nil {
					t.Errorf("unexpected %s: %+v", statusLineUntrustedGapID, *found)
				}
				return
			}
			if found == nil {
				t.Fatalf("no %s gap; gaps = %v", statusLineUntrustedGapID, gapIDs(det.Gaps))
			}
			g := *found
			if !g.Required || g.Resolvable || g.Category != ConfigChange || g.Scope != "machine" {
				t.Errorf("not a report-only machine gap: %+v", g)
			}
			if !strings.Contains(g.Detail, tc.reason) || !strings.Contains(g.Detail, "~/.claude/settings.json") {
				t.Errorf("detail does not name the file and the reason %q: %q", tc.reason, g.Detail)
			}
			if !strings.Contains(g.FixHint, "re-run `abcd ahoy install` to repoint the status line") {
				t.Errorf("fix hint = %q", g.FixHint)
			}
			for _, field := range []string{g.Title, g.Detail, g.FixHint} {
				if strings.Contains(field, home) {
					t.Errorf("gap carries the home path: %q", field)
				}
			}
		})
	}
}

// TestAbcdInvocations pins the recogniser: a command word whose name is an
// abcd binary — bare, a path, quoted, behind an assignment or exec, inside a
// `bash -c` string — and never an abcd directory in argument position.
func TestAbcdInvocations(t *testing.T) {
	cases := []struct {
		cmd  string
		want string // "" => none
	}{
		{"abcd hook session-end", "abcd"},
		{"/Users/x/ABCDevelopment/abcd/bin/abcd-darwin-arm64 hook subagent-stop", "/Users/x/ABCDevelopment/abcd/bin/abcd-darwin-arm64"},
		{`"$CLAUDE_PLUGIN_ROOT/abcd" hook prompt-router`, "$CLAUDE_PLUGIN_ROOT/abcd"},
		{"'/my dir/abcd' statusline", "/my dir/abcd"},
		{"FOO=1 exec ~/.local/bin/abcd hook x", "~/.local/bin/abcd"},
		{"bash -c 'abcd hook prompt-router'", "abcd"},
		{"true && abcd guard hook", "abcd"},
		{"x=$(abcd version)", "abcd"},
		{"cd /src/abcd && make", ""},
		{"echo a && echo b", ""},
		{"ls ~/abcd-notes", ""},
		{"/usr/local/bin/abcdef run", ""},
	}
	for _, tc := range cases {
		got := abcdInvocations(tc.cmd)
		first := ""
		if len(got) > 0 {
			first = got[0]
		}
		if first != tc.want {
			t.Errorf("abcdInvocations(%q) = %q, want first %q", tc.cmd, got, tc.want)
		}
	}
}
