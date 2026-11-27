package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// A forge text the guard cannot judge because a git identity value is not
// valid UTF-8 names that setting to re-save, not the scanner's configuration.
func TestForgeTextUnjudgedForAnIdentityValueNamesTheSettingToReSave(t *testing.T) {
	_, reg := forgeRepo(t, forgeOptedIn)
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
	d := forgeCheck(t, reg, "gh pr create --body \"Summary\n\nAssisted-by: Claude:claude-opus-5-5\"")
	if d.Verdict != VerdictWarn || !contains(d.Matches, forgeUnreadEntryID) {
		t.Fatalf("Check = %q via %q (matches %v), want warn via %q", d.Verdict, d.EntryID, d.Matches, forgeUnreadEntryID)
	}
	said := d.Message + "\n" + d.Reason + "\n" + d.Successor
	if !strings.Contains(said, "re-save git config user.name as UTF-8") || strings.Contains(said, "pii.json") {
		t.Errorf("the warning = %q; want git config user.name named to re-save as UTF-8, not pii.json", said)
	}
	if strings.Contains(strings.ToLower(said), "orv") {
		t.Errorf("the warning prints the value: %q", said)
	}
}
