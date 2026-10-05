package ahoy

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/core/statusline"
)

// The guards around the one write abcd makes into the harness's user settings
// (iss-2610050556383525): the change is shown before consent, the file is
// backed up before it is replaced, the write is read back after, and only the
// recorded, trusted PATH entry is ever wired.

// offerQuestionAsked returns the status-line offer among the confirms p was
// asked, or "" when it was never asked.
func offerQuestionAsked(p *scriptedPrompter) string {
	for _, q := range p.asked {
		if strings.Contains(q, "Install abcd's status line?") {
			return q
		}
	}
	return ""
}

// backupsIn lists the backups abcd keeps in home, by name.
func backupsIn(t *testing.T, home string) []string {
	t.Helper()
	ents, err := os.ReadDir(abcdhome.Path(home, "backups"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range ents {
		out = append(out, e.Name())
	}
	return out
}

var backupNameRe = regexp.MustCompile(`^settings\.json\.\d{8}T\d{6}Z(-\d+)?$`)

// swapEntryTrust makes the entry trust check answer as the test says.
func swapEntryTrust(t *testing.T, ok bool, reason string) {
	t.Helper()
	prev := entryTrust
	entryTrust = func(string) (bool, string) { return ok, reason }
	t.Cleanup(func() { entryTrust = prev })
}

// swapHarnessWrite makes the harness write land body instead of what was asked.
func swapHarnessWrite(t *testing.T, body string) {
	t.Helper()
	prev := writeHarnessSettings
	writeHarnessSettings = func(path string, _ []byte) error {
		return os.WriteFile(path, []byte(body), 0o644)
	}
	t.Cleanup(func() { writeHarnessSettings = prev })
}

// TestStatusLineOfferShowsTheOneChange is risk 4: the confirm the person
// answers names the file, the one entry's value now and after, that the
// current command is recorded and put back by uninstall, and that a copy of
// the file is kept first — and it is still recognised as the offer.
func TestStatusLineOfferShowsTheOneChange(t *testing.T) {
	cases := []struct {
		name, line, now string
	}{
		{"a command is set", `{"type": "command", "command": "` + previousStatusCommand + `"}`, `now: "` + previousStatusCommand + `"`},
		{"nothing is set", "", "now: none"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home, _ := setupHermetic(t)
			settings := harnessFixture(t, harnessSettingsWith(tc.line))
			repo := installedRepo(t)
			entry := os.Getenv("ABCD_BIN_TARGET")
			p := offerPrompter(false, nil)
			if _, err := Install(repo, InstallOptions{}, p); err != nil {
				t.Fatal(err)
			}
			q := offerQuestionAsked(p)
			if q == "" {
				t.Fatalf("the offer was never asked: %q", p.asked)
			}
			for _, want := range []string{
				displayPath(settings),
				harnessStatusKey,
				tc.now,
				"after: " + displayPath(entry) + " " + statusVerb,
				"abcd ahoy uninstall",
				abcdhome.Display("backups") + "/",
			} {
				if !strings.Contains(q, want) {
					t.Errorf("the offer does not show %q:\n%s", want, q)
				}
			}
			if strings.Contains(q, home) || strings.Contains(q, "\n\n") {
				t.Errorf("the offer carries the home path or a blank line:\n%s", q)
			}
			if id := setupConfirmID(q); id != StatusLineOfferGapID {
				t.Errorf("setupConfirmID = %q, want %q", id, StatusLineOfferGapID)
			}
			sq := SetupConfirmQuestion(1, q)
			if sq.Ask != "Install abcd's status line?" {
				t.Errorf("ask = %q", sq.Ask)
			}
			if fs := question.Check(question.Ask{Questions: []question.Question{sq}}); len(fs) > 0 {
				t.Errorf("structural findings: %v", fs)
			}
		})
	}
}

// TestStatusLineWritesKeepABackup is risk 4/5: every replacement of the
// harness file — the wiring, the dangling repair and the uninstall restore —
// first keeps the file's exact bytes in ~/.abcd.noindex/backups, private to
// the account, and names the copy where the person will read it.
func TestStatusLineWritesKeepABackup(t *testing.T) {
	t.Run("wire then uninstall", func(t *testing.T) {
		home, _ := setupHermetic(t)
		settings := harnessFixture(t, harnessSettingsWith(
			`{"type": "command", "command": "`+previousStatusCommand+`"}`))
		repo := installedRepo(t)
		before, _ := os.ReadFile(settings)
		res, err := Install(repo, InstallOptions{}, offerPrompter(true, nil))
		if err != nil {
			t.Fatal(err)
		}
		if n := statusLineNotes(res.Notes); len(n) != 0 {
			t.Fatalf("unexpected status-line notes: %v", n)
		}
		names := backupsIn(t, home)
		if len(names) != 1 || !backupNameRe.MatchString(names[0]) {
			t.Fatalf("backups = %v, want one settings.json.<UTC stamp>", names)
		}
		bp := abcdhome.Path(home, "backups", names[0])
		if got, _ := os.ReadFile(bp); !bytes.Equal(got, before) {
			t.Errorf("the backup is not the file as it was:\n%s", got)
		}
		if fi, err := os.Stat(bp); err != nil || fi.Mode().Perm() != 0o600 {
			t.Errorf("backup mode = %v, want 0600", fi.Mode())
		}
		if fi, err := os.Stat(abcdhome.Path(home, "backups")); err != nil || fi.Mode().Perm() != 0o700 {
			t.Errorf("backups folder mode = %v, want 0700", fi.Mode())
		}
		if !containsString(res.Writes, abcdhome.Display("backups", names[0])) {
			t.Errorf("writes do not name the backup: %v", res.Writes)
		}

		wired, _ := os.ReadFile(settings)
		receipt, err := Uninstall(repo, "")
		if err != nil {
			t.Fatal(err)
		}
		names = backupsIn(t, home)
		if len(names) != 2 {
			t.Fatalf("backups after uninstall = %v, want two", names)
		}
		var found string
		for _, n := range names {
			if got, _ := os.ReadFile(abcdhome.Path(home, "backups", n)); bytes.Equal(got, wired) {
				found = n
			}
		}
		if found == "" {
			t.Fatal("no backup holds the file as uninstall found it")
		}
		if !receipt.StatusLine.Restored || !strings.Contains(receipt.StatusLine.Note, abcdhome.Display("backups", found)) ||
			strings.Contains(receipt.StatusLine.Note, home) {
			t.Errorf("receipt = %+v, want restored with the backup named in tilde form", receipt.StatusLine)
		}
	})
	t.Run("dangling repair", func(t *testing.T) {
		home, _ := setupHermetic(t)
		settings := harnessFixture(t, harnessSettingsWith(
			`{"type": "command", "command": "'/nowhere/at/all/abcd' statusline"}`))
		before, _ := os.ReadFile(settings)
		repo := t.TempDir()
		if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		res, err := Install(repo, installOpts(), RefusingPrompter{})
		if err != nil {
			t.Fatal(err)
		}
		names := backupsIn(t, home)
		if len(names) != 1 {
			t.Fatalf("backups = %v, want one", names)
		}
		if got, _ := os.ReadFile(abcdhome.Path(home, "backups", names[0])); !bytes.Equal(got, before) {
			t.Errorf("the backup is not the file as it was:\n%s", got)
		}
		if !containsString(res.Writes, abcdhome.Display("backups", names[0])) {
			t.Errorf("writes do not name the backup: %v", res.Writes)
		}
	})
}

// TestStatusLineBackupFailureWritesNothing: a copy that cannot be kept
// refuses the write; neither the harness file nor the user-level setting is
// touched.
func TestStatusLineBackupFailureWritesNothing(t *testing.T) {
	blockBackups := func(t *testing.T, home string) {
		t.Helper()
		if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abcdhome.Path(home, "backups"), []byte("not a folder"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("install", func(t *testing.T) {
		home, _ := setupHermetic(t)
		settings := harnessFixture(t, harnessSettingsWith(""))
		repo := installedRepo(t)
		blockBackups(t, home)
		before, _ := os.ReadFile(settings)
		res, err := Install(repo, InstallOptions{}, offerPrompter(true, nil))
		if err != nil {
			t.Fatal(err)
		}
		n := statusLineNotes(res.Notes)
		if len(n) != 1 || !strings.Contains(n[0], abcdhome.Display("backups")) || !strings.Contains(n[0], "nothing was written") ||
			strings.Contains(n[0], home) {
			t.Errorf("notes = %v, want one refusal naming the backups folder", res.Notes)
		}
		if after, _ := os.ReadFile(settings); !bytes.Equal(after, before) {
			t.Error("the harness file was written without a backup")
		}
		if _, err := os.Stat(abcdhome.Path(home, "statusline.json")); err == nil {
			t.Error("the user-level setting was written without a backup")
		}
	})
	t.Run("uninstall", func(t *testing.T) {
		home, _ := setupHermetic(t)
		repo := installedRepo(t)
		settings := harnessFixture(t, harnessSettingsWith(
			`{"type": "command", "command": "'`+os.Getenv("ABCD_BIN_TARGET")+`' statusline"}`))
		blockBackups(t, home)
		before, _ := os.ReadFile(settings)
		receipt, err := Uninstall(repo, "")
		if err != nil {
			t.Fatal(err)
		}
		if receipt.StatusLine.Restored || !strings.Contains(receipt.StatusLine.Note, abcdhome.Display("backups")) {
			t.Errorf("receipt = %+v, want the restore refused naming the backups folder", receipt.StatusLine)
		}
		if after, _ := os.ReadFile(settings); !bytes.Equal(after, before) {
			t.Error("uninstall wrote the harness file without a backup")
		}
	})
}

// TestStatusLineWriteIsReadBack is risk 5: after the replace the file is read
// back, and one that does not parse, lost or changed another key, or carries
// a statusLine other than the one intended is put back from the backup, with
// the setting this run created taken back out and the backup named.
func TestStatusLineWriteIsReadBack(t *testing.T) {
	corruptions := map[string]string{
		"not JSON":          "{\"statusLine\": ",
		"another key lost":  `{"statusLine": {"type": "command", "command": "x"}}`,
		"wrong status line": harnessSettingsWith(`{"type": "command", "command": "bash /tmp/someone-else.sh"}`),
	}
	for name, body := range corruptions {
		t.Run("install: "+name, func(t *testing.T) {
			home, _ := setupHermetic(t)
			settings := harnessFixture(t, harnessSettingsWith(""))
			repo := installedRepo(t)
			before, _ := os.ReadFile(settings)
			swapHarnessWrite(t, body)
			res, err := Install(repo, InstallOptions{}, offerPrompter(true, nil))
			if err != nil {
				t.Fatal(err)
			}
			n := statusLineNotes(res.Notes)
			if len(n) != 1 || !strings.Contains(n[0], "did not read back") || !strings.Contains(n[0], abcdhome.Display("backups")+"/settings.json.") ||
				strings.Contains(n[0], home) {
				t.Errorf("notes = %v, want one refusal naming the backup", res.Notes)
			}
			if after, _ := os.ReadFile(settings); !bytes.Equal(after, before) {
				t.Errorf("the harness file was not put back:\n%s", after)
			}
			if _, err := os.Stat(abcdhome.Path(home, "statusline.json")); err == nil {
				t.Error("the user-level setting this run created was left behind")
			}
		})
	}
	t.Run("uninstall", func(t *testing.T) {
		setupHermetic(t)
		repo := installedRepo(t)
		settings := harnessFixture(t, harnessSettingsWith(
			`{"type": "command", "command": "'`+os.Getenv("ABCD_BIN_TARGET")+`' statusline"}`))
		before, _ := os.ReadFile(settings)
		swapHarnessWrite(t, "{")
		receipt, err := Uninstall(repo, "")
		if err != nil {
			t.Fatal(err)
		}
		if receipt.StatusLine.Restored || !strings.Contains(receipt.StatusLine.Note, "did not read back") {
			t.Errorf("receipt = %+v, want the restore refused", receipt.StatusLine)
		}
		if after, _ := os.ReadFile(settings); !bytes.Equal(after, before) {
			t.Errorf("the harness file was not put back:\n%s", after)
		}
	})
}

// TestStatusLineWiresOnlyTheRecordedTrustedEntry is risk 6 and the versioned
// half of risk 1: the harness is pointed only at the PATH entry
// ~/.abcd.noindex/path-entry records, and only when that entry passes the
// trust check. The plugin's own binary lives in a versioned cache directory a
// plugin update deletes, so it is never wired, by the offer or the repair.
func TestStatusLineWiresOnlyTheRecordedTrustedEntry(t *testing.T) {
	t.Run("no recorded entry refuses before asking", func(t *testing.T) {
		home, _ := setupHermetic(t)
		settings := harnessFixture(t, harnessSettingsWith(""))
		repo := installedRepo(t)
		if err := os.Remove(abcdhome.Path(home, "path-entry")); err != nil {
			t.Fatal(err)
		}
		det, _ := Detect(repo)
		before, _ := os.ReadFile(settings)
		p := offerPrompter(true, nil)
		a := &applyCtx{cwd: repo, det: det, approved: map[GapCategory]bool{StatusLine: true},
			gapPresent: map[string]bool{StatusLineOfferGapID: true}, prompter: p,
			binTarget: os.Getenv("ABCD_BIN_TARGET")}
		a.stepStatusLine()
		if len(a.notes) != 1 || !strings.Contains(a.notes[0], "`abcd ahoy install`") || !strings.Contains(a.notes[0], "PATH") ||
			strings.Contains(a.notes[0], home) {
			t.Errorf("notes = %v, want one refusal giving the remedy", a.notes)
		}
		if q := offerQuestionAsked(p); q != "" {
			t.Errorf("the offer was asked although nothing can be wired:\n%s", q)
		}
		if after, _ := os.ReadFile(settings); !bytes.Equal(after, before) {
			t.Errorf("the harness was wired with no recorded entry:\n%s", after)
		}
		if _, err := os.Stat(abcdhome.Path(home, statusline.SettingsFileName)); err == nil {
			t.Error("the user-level setting was written")
		}
	})
	t.Run("an untrusted entry refuses before asking", func(t *testing.T) {
		home, _ := setupHermetic(t)
		settings := harnessFixture(t, harnessSettingsWith(""))
		repo := installedRepo(t)
		swapEntryTrust(t, false, "its folder can be written by other accounts")
		before, _ := os.ReadFile(settings)
		p := offerPrompter(true, nil)
		res, err := Install(repo, InstallOptions{}, p)
		if err != nil {
			t.Fatal(err)
		}
		n := statusLineNotes(res.Notes)
		if len(n) != 1 || !strings.Contains(n[0], "its folder can be written by other accounts") || strings.Contains(n[0], home) {
			t.Errorf("notes = %v, want one refusal naming the reason", res.Notes)
		}
		if q := offerQuestionAsked(p); q != "" {
			t.Errorf("the offer was asked for an untrusted entry:\n%s", q)
		}
		if after, _ := os.ReadFile(settings); !bytes.Equal(after, before) {
			t.Errorf("the harness was wired to an untrusted entry:\n%s", after)
		}
	})
	t.Run("repair never repoints at the plugin binary", func(t *testing.T) {
		_, pluginRoot := setupHermetic(t)
		settings := harnessFixture(t, harnessSettingsWith(
			`{"type": "command", "command": "'/nowhere/at/all/abcd' statusline"}`))
		repo := t.TempDir()
		det, _ := Detect(repo)
		a := &applyCtx{cwd: repo, det: det, approved: map[GapCategory]bool{ConfigChange: true},
			gapPresent: map[string]bool{statusLineDanglingGapID: true}, prompter: RefusingPrompter{}}
		a.det.pluginRoot = pluginRoot
		a.stepStatusLine()
		if line, ok := statusLineOf(t, settings); ok {
			t.Errorf("statusLine = %v, want it removed: nothing was recorded and abcd has no PATH entry", line)
		}
	})
	t.Run("repair never repoints at an untrusted entry", func(t *testing.T) {
		home, _ := setupHermetic(t)
		settings := harnessFixture(t, harnessSettingsWith(
			`{"type": "command", "command": "'/nowhere/at/all/abcd' statusline"}`))
		if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abcdhome.Path(home, "statusline.json"),
			[]byte(`{"schema_version": 1, "previous_command": "`+previousStatusCommand+`"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		swapEntryTrust(t, false, "it is not this account's")
		repo := t.TempDir()
		if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
			t.Fatal(err)
		}
		line, ok := statusLineOf(t, settings)
		if !ok || line["command"] != previousStatusCommand {
			t.Errorf("statusLine = %v, want the previous command restored rather than the untrusted entry", line)
		}
	})
}
