package repolint_test

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

// The privacy lint reads no identity, so an identity value that is not UTF-8
// is no error of its: it is not reported as a broken per-repo pattern file.
func TestAC_PrivacyLintIsNotDegradedByAnIdentityValueNotUTF8(t *testing.T) {
	for _, c := range []struct{ setting, value string }{
		{"user.name", "Zorv\xe9n Quill"}, {"the home folder's name", "zorvan\xe9"},
	} {
		b := newFixtureRepo(t).conforming().commit()
		setIdentityValue(t, c.setting, c.value)
		if f := findingFor(b.run(), "privacy-hygiene"); f != nil {
			t.Errorf("%s: the privacy lint reported %q", c.setting, f.Message)
		}
	}
}
