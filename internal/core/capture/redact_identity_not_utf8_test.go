package capture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// identityNotUTF8Cases is each git identity value and the account name, invalid
// at the start, in the middle and at the end, with the setting a refusal names.
var identityNotUTF8Cases = []struct{ setting, value string }{
	{"user.name", "\xe9orvan Quill"}, {"user.name", "Zorv\xe9n Quill"}, {"user.name", "Zorvan Quill\xe9"},
	{"user.email", "\xe9orvan@example.com"}, {"user.email", "zorv\xe9n@example.com"}, {"user.email", "zorvan@example.co\xe9"},
	{"remote.origin.url", "\xe9orvan"}, {"remote.origin.url", "zorv\xe9n"}, {"remote.origin.url", "zorvan\xe9"},
	{"the home folder's name", "\xe9orvan"}, {"the home folder's name", "zorv\xe9n"}, {"the home folder's name", "zorvan\xe9"},
}

// setIdentityValue makes the identity probe read value for setting and
// nothing else: every git variable cleared, the global config a fresh file,
// HOME named after the account for the home folder's name.
func setIdentityValue(t *testing.T, setting, value string) {
	t.Helper()
	for _, kv := range os.Environ() {
		if k, _, _ := strings.Cut(kv, "="); strings.HasPrefix(k, "GIT_") {
			t.Setenv(k, "")
			os.Unsetenv(k)
		}
	}
	dir := t.TempDir()
	var text string
	home := filepath.Join(dir, "maya")
	switch setting {
	case "user.name":
		text = "[user]\n\tname = " + value + "\n"
	case "user.email":
		text = "[user]\n\temail = " + value + "\n"
	case "remote.origin.url":
		text = "[remote \"origin\"]\n\turl = git@github.com:" + value + "/notes.git\n"
	case "the home folder's name":
		home = filepath.Join(dir, value)
	}
	global := filepath.Join(dir, "global.gitconfig")
	if err := os.WriteFile(global, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv(abcdhome.StageHomeEnv, "")
	t.Setenv("HOME", home)
}

// checkIdentityRefusal holds err to a refusal that names the setting to
// re-save as UTF-8 and never prints the value.
func checkIdentityRefusal(t *testing.T, setting, value string, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("%s %+q: written, want a refusal", setting, value)
		return
	}
	msg := err.Error()
	if !strings.Contains(msg, setting) || !strings.Contains(msg, "UTF-8") || !strings.Contains(msg, "nothing written") {
		t.Errorf("%s %+q: the refusal does not name the setting to re-save: %q", setting, value, msg)
	}
	if low := strings.ToLower(msg); strings.Contains(low, "orv") || strings.Contains(low, "quill") ||
		!utf8.ValidString(msg) || strings.ContainsRune(msg, utf8.RuneError) {
		t.Errorf("%s %+q: the refusal prints the value: %+q", setting, value, msg)
	}
}

// The issue ledger refuses a capture while an identity value is not valid
// UTF-8, writing nothing, where it writes and notes every other degraded
// scanner: no pattern can match the value, so it would reach the ledger in
// clear.
func TestCaptureRefusesWhileAnIdentityValueIsNotUTF8(t *testing.T) {
	for _, c := range identityNotUTF8Cases {
		setIdentityValue(t, c.setting, c.value)
		repo, ir := ledger(t)
		text := "seen by " + strings.ToValidUTF8(c.value, "�") + " and " + strings.ToValidUTF8(c.value, "")
		_, err := Capture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: text, Severity: SeverityMinor,
			Category: "process", Source: "user-observation", Slug: "identity-refusal", FoundDuring: "unit-test",
			Remedy: "keep the name out of the ledger"})
		checkIdentityRefusal(t, c.setting, c.value, err)
		if entries, _ := os.ReadDir(filepath.Join(ir, "open")); len(entries) != 0 {
			t.Errorf("%s %+q: the capture wrote %d record(s)", c.setting, c.value, len(entries))
		}
	}
}

// The batch redactor refuses the same way, before a reading or a disposition
// writes anything.
func TestLedgerRedactorRefusesWhileAnIdentityValueIsNotUTF8(t *testing.T) {
	for _, c := range identityNotUTF8Cases {
		setIdentityValue(t, c.setting, c.value)
		_, err := newLedgerRedactor(t.TempDir())
		checkIdentityRefusal(t, c.setting, c.value, err)
	}
}

// A UTF-8 identity is redacted from a capture as before.
func TestCaptureRedactsAUTF8IdentityAsBefore(t *testing.T) {
	setIdentityValue(t, "user.name", "Zorvan Quillé")
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "seen by Zorvan Quillé today",
		Severity: SeverityMinor, Category: "process", Source: "user-observation", Slug: "identity-kept-out",
		FoundDuring: "unit-test", Remedy: "keep the name out of the ledger"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(repo, res.Path))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "Zorvan Quillé") || res.Degraded != "" {
		t.Errorf("the name reached the ledger, or the write was noted degraded (%q):\n%s", res.Degraded, data)
	}
}
