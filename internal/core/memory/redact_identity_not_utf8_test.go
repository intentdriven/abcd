package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// The memory store refuses a write while a git identity value or the account
// name is not valid UTF-8, naming the setting to re-save and never the value.
func TestStoreRedactorRefusesWhileAnIdentityValueIsNotUTF8(t *testing.T) {
	for _, c := range []struct{ setting, value string }{
		{"user.name", "\xe9orvan Quill"}, {"user.name", "Zorv\xe9n Quill"}, {"user.name", "Zorvan Quill\xe9"},
		{"user.email", "\xe9orvan@example.com"}, {"user.email", "zorv\xe9n@example.com"}, {"user.email", "zorvan@example.co\xe9"},
		{"remote.origin.url", "\xe9orvan"}, {"remote.origin.url", "zorv\xe9n"}, {"remote.origin.url", "zorvan\xe9"},
		{"the home folder's name", "\xe9orvan"}, {"the home folder's name", "zorv\xe9n"}, {"the home folder's name", "zorvan\xe9"},
	} {
		for _, kv := range os.Environ() {
			if k, _, _ := strings.Cut(kv, "="); strings.HasPrefix(k, "GIT_") {
				t.Setenv(k, "")
				os.Unsetenv(k)
			}
		}
		dir := t.TempDir()
		global := filepath.Join(dir, "global.gitconfig")
		var text string
		switch c.setting {
		case "user.name":
			text = "[user]\n\tname = " + c.value + "\n"
		case "user.email":
			text = "[user]\n\temail = " + c.value + "\n"
		case "remote.origin.url":
			text = "[remote \"origin\"]\n\turl = git@github.com:" + c.value + "/notes.git\n"
		}
		if err := os.WriteFile(global, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GIT_CONFIG_GLOBAL", global)
		t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
		t.Setenv(abcdhome.StageHomeEnv, "")
		home := filepath.Join(dir, "maya")
		if c.setting == "the home folder's name" {
			home = filepath.Join(dir, c.value)
		}
		t.Setenv("HOME", home)
		_, err := newStoreRedactor(t.TempDir())
		if err == nil {
			t.Errorf("%s %+q: the store redactor was built, want a refusal", c.setting, c.value)
			continue
		}
		if msg := err.Error(); !strings.Contains(msg, c.setting) || strings.Contains(strings.ToLower(msg), "orv") {
			t.Errorf("%s %+q: the refusal = %q, want the setting named and the value left out", c.setting, c.value, msg)
		}
	}
}
