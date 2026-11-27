package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// identityWorld clears every git variable the identity probe reads, points the
// global git config at a fresh file and returns it, so a case adds only the
// one value it names.
func identityWorld(t *testing.T) string {
	t.Helper()
	for _, kv := range os.Environ() {
		if k, _, _ := strings.Cut(kv, "="); strings.HasPrefix(k, "GIT_") {
			t.Setenv(k, "")
			os.Unsetenv(k)
		}
	}
	global := filepath.Join(t.TempDir(), "global.gitconfig")
	if err := os.WriteFile(global, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv(abcdhome.StageHomeEnv, "")
	return global
}

// setIdentityValue writes value where the probe reads setting: the global git
// config, or HOME for the account name.
func setIdentityValue(t *testing.T, global, home, setting, value string) {
	t.Helper()
	var text string
	switch setting {
	case "user.name":
		text = "[user]\n\tname = " + value + "\n"
	case "user.email":
		text = "[user]\n\temail = " + value + "\n"
	case "remote.origin.url":
		text = "[remote \"origin\"]\n\turl = git@github.com:" + value + "/notes.git\n"
	case "the home folder's name":
		t.Setenv("HOME", filepath.Join(home, value))
		return
	}
	if err := os.WriteFile(global, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

// identityValuesNotUTF8 is each git identity value and the account name,
// invalid at the start, in the middle and at the end.
var identityValuesNotUTF8 = []struct{ setting, value string }{
	{"user.name", "\xe9orvan Quill"}, {"user.name", "Zorv\xe9n Quill"}, {"user.name", "Zorvan Quill\xe9"},
	{"user.email", "\xe9orvan@example.com"}, {"user.email", "zorv\xe9n@example.com"}, {"user.email", "zorvan@example.co\xe9"},
	{"remote.origin.url", "\xe9orvan"}, {"remote.origin.url", "zorv\xe9n"}, {"remote.origin.url", "zorvan\xe9"},
	{"the home folder's name", "\xe9orvan"}, {"the home folder's name", "zorv\xe9n"}, {"the home folder's name", "zorvan\xe9"},
}

// The history store refuses to redact while an identity value is not valid
// UTF-8, naming the setting to re-save and never the value.
func TestRedactRefusesWhileAnIdentityValueIsNotUTF8(t *testing.T) {
	for _, c := range identityValuesNotUTF8 {
		repoRoot, home := setupStore(t)
		global := identityWorld(t)
		setIdentityValue(t, global, home, c.setting, c.value)
		text := "user: by " + strings.ToValidUTF8(c.value, "�") + " and " + strings.ToValidUTF8(c.value, "") + "\n"
		got, err := Redact(repoRoot, text, "a text")
		if err == nil {
			t.Errorf("%s %+q: Redact wrote %+q, want a refusal", c.setting, c.value, got.Text)
			continue
		}
		msg := err.Error()
		if !strings.Contains(msg, c.setting) || !strings.Contains(msg, "UTF-8") {
			t.Errorf("%s %+q: the refusal does not name the setting to re-save: %q", c.setting, c.value, msg)
		}
		if low := strings.ToLower(msg); strings.Contains(low, "orv") || strings.Contains(low, "quill") ||
			!utf8.ValidString(msg) || strings.ContainsRune(msg, utf8.RuneError) {
			t.Errorf("%s %+q: the refusal prints the value: %+q", c.setting, c.value, msg)
		}
	}
}

// An identity that is valid UTF-8 is redacted as before.
func TestRedactRedactsAnIdentityThatIsUTF8AsBefore(t *testing.T) {
	repoRoot, home := setupStore(t)
	global := identityWorld(t)
	setIdentityValue(t, global, home, "user.name", "Zorvan Quillé")
	got, err := Redact(repoRoot, "user: written by Zorvan Quillé today\n", "a text")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got.Text, "Zorvan Quillé") {
		t.Errorf("the name was stored: %q", got.Text)
	}
}
