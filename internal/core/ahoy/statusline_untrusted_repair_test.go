package ahoy

import (
	"bytes"
	"github.com/intentdriven/abcd/internal/shellquote"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/statusline"
)

// untrustedGap returns the statusline.untrusted gap among gaps, or nil.
func untrustedGap(gaps []Gap) *Gap {
	for i, g := range gaps {
		if g.ID == StatusLineUntrustedGapID {
			return &gaps[i]
		}
	}
	return nil
}

// TestInstallRepairsAnUntrustedStatusLine: detection's remedy for an abcd
// status line that runs a binary failing the trust checks is "re-run `abcd
// ahoy install`", so install must make that true. Under config-change
// approval, the way the dangling repair runs, the line is repointed at the
// recorded, trusted PATH entry: a copy of the file is kept first, every other
// key survives, and the stale command is never recorded as the previous one.
func TestInstallRepairsAnUntrustedStatusLine(t *testing.T) {
	cases := []struct {
		name    string
		command func(home, pluginRoot string) string
	}{
		{"the plugin-root binary", func(_, root string) string { return shellquote.Single(filepath.Join(root, "abcd")) + " statusline" }},
		{"a stale local build", func(home, _ string) string {
			return filepath.Join(home, "ABCDevelopment", "abcd", "bin", "abcd-darwin-arm64") + " statusline"
		}},
		{"a bare name", func(string, string) string { return "abcd statusline" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home, pluginRoot := setupHermetic(t)
			repo := installedRepo(t)
			writeTrustBinary(t, filepath.Join(home, "ABCDevelopment", "abcd", "bin", "abcd-darwin-arm64"))
			settings := homeHarnessSettings(t, `{"padding": 1, "statusLine": {"type": "command", "command": "`+
				tc.command(home, pluginRoot)+`", "padding": 2}}`)
			det, err := Detect(repo)
			if err != nil {
				t.Fatal(err)
			}
			g := untrustedGap(det.Gaps)
			if g == nil || !g.Required || !g.Resolvable {
				t.Fatalf("precondition: a required, resolvable %s gap; got %+v", StatusLineUntrustedGapID, g)
			}

			res, err := Install(repo, installOpts(), RefusingPrompter{})
			if err != nil {
				t.Fatal(err)
			}
			line, ok := statusLineOf(t, settings)
			if want := shellquote.Single(os.Getenv("ABCD_BIN_TARGET")) + " statusline"; !ok || line["command"] != want {
				t.Fatalf("statusLine = %v, want it repointed at %q; notes = %v", line, want, res.Notes)
			}
			if line["padding"] != float64(2) {
				t.Errorf("the line's own keys were not kept: %v", line)
			}
			if doc := readJSONFile(t, settings); doc["padding"] != float64(1) {
				t.Errorf("the rest of the file was not kept: %v", doc)
			}
			if b := backupsIn(t, home); len(b) != 1 {
				t.Errorf("backups = %v, want one copy kept before the repair", b)
			}
			if raw, err := os.ReadFile(abcdhome.Path(home, statusline.SettingsFileName)); err == nil && bytes.Contains(raw, []byte("statusline")) {
				t.Errorf("the untrusted command was recorded as the previous one:\n%s", raw)
			}
			after, _ := Detect(repo)
			if g := untrustedGap(after.Gaps); g != nil {
				t.Errorf("still untrusted after install: %+v", *g)
			}
			if after.Signals["statusline"] != "installed" {
				t.Errorf("statusline signal = %q, want installed", after.Signals["statusline"])
			}
		})
	}
}

// TestUntrustedStatusLineWithNoTrustedEntryRestoresThePrevious: when abcd has
// no trusted entry to offer, the untrusted line is handed back to the command
// recorded before abcd took the row — never repointed at the plugin's copy.
func TestUntrustedStatusLineWithNoTrustedEntryRestoresThePrevious(t *testing.T) {
	home, pluginRoot := setupHermetic(t)
	settings := homeHarnessSettings(t, `{"statusLine": {"type": "command", "command": "`+
		shellquote.Single(filepath.Join(pluginRoot, "abcd"))+` statusline"}}`)
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abcdhome.Path(home, "statusline.json"),
		[]byte(`{"schema_version": 1, "previous_command": "`+previousStatusCommand+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	det, _ := Detect(repo)
	if untrustedGap(det.Gaps) == nil {
		t.Fatalf("precondition: an untrusted gap; gaps = %v", gapIDs(det.Gaps))
	}
	a := &applyCtx{cwd: repo, det: det, approved: map[GapCategory]bool{ConfigChange: true},
		gapPresent: gapIDSet(det.Gaps), prompter: RefusingPrompter{}}
	a.stepStatusLine()
	line, ok := statusLineOf(t, settings)
	if !ok || line["command"] != previousStatusCommand {
		t.Errorf("statusLine = %v, want the previous command restored; notes = %v", line, a.notes)
	}
}

// TestUntrustedLineThatIsNotAbcdsStatusLineIsReportOnly: a status line that
// runs some abcd for another purpose is the person's own command, not abcd's
// status line, so install never rewrites it, and the remedy says to edit it by
// hand rather than promising a repair install will not make.
func TestUntrustedLineThatIsNotAbcdsStatusLineIsReportOnly(t *testing.T) {
	setupHermetic(t)
	repo := installedRepo(t)
	settings := homeHarnessSettings(t, `{"statusLine": {"type": "command", "command": "/opt/nowhere/abcd-linux-amd64 --version; echo hi"}}`)
	det, _ := Detect(repo)
	g := untrustedGap(det.Gaps)
	if g == nil {
		t.Fatalf("no untrusted gap; gaps = %v", gapIDs(det.Gaps))
	}
	if g.Resolvable || strings.Contains(g.FixHint, "abcd ahoy install") || !strings.Contains(g.FixHint, "edit") {
		t.Errorf("want a report-only gap telling the person to edit it: %+v", *g)
	}
	before, _ := os.ReadFile(settings)
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.ReadFile(settings); !bytes.Equal(after, before) {
		t.Errorf("install rewrote a status line that is not abcd's:\n%s", after)
	}
}

// TestReachesStatusVerbKnowsEveryBuildName: the recursion guard recognises a
// built binary by every name a build gives it — the bare `abcd` and the
// `abcd-<goos>-<arch>` names `make build` and a release publish — so a stale
// local build's status line is never recorded as the previous command.
func TestReachesStatusVerbKnowsEveryBuildName(t *testing.T) {
	for _, cmd := range []string{
		"abcd statusline",
		"/Users/x/ABCDevelopment/abcd/bin/abcd-darwin-arm64 statusline",
		"'/opt/abcd-linux-amd64' statusline | head",
		`"C:/bin/abcd-windows-amd64.exe" statusline`,
	} {
		if !reachesStatusVerb(cmd) {
			t.Errorf("reachesStatusVerb(%q) = false, want true", cmd)
		}
	}
	for _, cmd := range []string{previousStatusCommand, "abcd-notes statusline", "abcd version"} {
		if reachesStatusVerb(cmd) {
			t.Errorf("reachesStatusVerb(%q) = true, want false", cmd)
		}
	}
}
