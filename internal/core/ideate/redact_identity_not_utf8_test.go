package ideate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

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

// A verdict write refused for an identity value that is not UTF-8 names the
// setting to re-save, not the per-repo pattern file.
func TestVerdictRefusalNamesTheIdentitySettingNotUTF8(t *testing.T) {
	for _, c := range []struct{ setting, value string }{
		{"user.name", "Zorv\xe9n Quill"}, {"the home folder's name", "zorvan\xe9"},
	} {
		setIdentityValue(t, c.setting, c.value)
		_, err := newRecordRedactor(t.TempDir())
		if err == nil {
			t.Fatalf("%s: the verdict redactor was built", c.setting)
		}
		msg := err.Error()
		if !strings.Contains(msg, c.setting) || !strings.Contains(msg, "re-save it as UTF-8") || strings.Contains(msg, "pii.json") ||
			!strings.Contains(msg, "nothing was written") || strings.Contains(strings.ToLower(msg), "orv") {
			t.Errorf("%s: the refusal = %q, want the setting to re-save and no pattern file", c.setting, msg)
		}
	}
}
