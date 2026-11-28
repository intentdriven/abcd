package findings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
)

// setUserNameNotUTF8 makes the identity probe read a git user.name that is
// not valid UTF-8 and nothing else.
func setUserNameNotUTF8(t *testing.T) {
	t.Helper()
	for _, kv := range os.Environ() {
		if k, _, _ := strings.Cut(kv, "="); strings.HasPrefix(k, "GIT_") {
			t.Setenv(k, "")
			os.Unsetenv(k)
		}
	}
	global := filepath.Join(t.TempDir(), "global.gitconfig")
	if err := os.WriteFile(global, []byte("[user]\n\tname = Zorv\xe9n Quill\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv(abcdhome.StageHomeEnv, "")
}

// A text withheld because an identity value is not valid UTF-8 names the
// setting to re-save, as the stored reason does, and not the scanner's
// configuration, which is not what is wrong; the line is the same once the
// entry is read back.
func TestAWithheldTextNamesTheIdentitySettingToReSave(t *testing.T) {
	s, _ := newTestStore(t)
	repo := gittest.NewRepo(t).Root()
	setUserNameNotUTF8(t)
	e := entry(KindUntested, "s1", "seen by Zorvn Quill")
	e.Checkout = repo
	got := mustPut(t, s, e)
	v := mustRead(t, s)
	if len(v.Open) != 1 {
		t.Fatalf("the list holds %d entries; want 1", len(v.Open))
	}
	for label, line := range map[string]string{"put": got.WithheldLine(), "read": v.Open[0].WithheldLine()} {
		if !strings.Contains(line, "re-save git config user.name as UTF-8") || strings.Contains(line, "pii.json") {
			t.Errorf("%s: the withheld line = %q; want it to name git config user.name to re-save as UTF-8, not pii.json", label, line)
		}
	}
}

// A text withheld because the pattern set is degraded still names the
// scanner's configuration.
func TestAWithheldTextUnderADegradedPatternSetNamesPIIJSON(t *testing.T) {
	s, _ := newTestStore(t)
	repo := gittest.NewRepo(t).Root()
	writePII(t, repo, `{`)
	e := entry(KindUntested, "s1", "a finding the scan cannot clear")
	e.Checkout = repo
	line := mustPut(t, s, e).WithheldLine()
	if !strings.Contains(line, ".abcd/config/pii.json") || strings.Contains(line, "re-save") {
		t.Errorf("the withheld line = %q; want the scanner's configuration named", line)
	}
}
