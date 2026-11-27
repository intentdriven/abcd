package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// The forge check judges the disclosure class, which reads no identity, so a
// git identity value that is not valid UTF-8 leaves it judging: a session URL
// or a tool footer is still blocked, and a clean, disclosing text is neither
// blocked nor warned as unjudged.
func TestForgeTextIsJudgedWhileAnIdentityValueIsNotUTF8(t *testing.T) {
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
	for _, cmd := range []string{
		`gh pr create --body "Summary` + "\n\nSee " + forgeSessionURL() + "\n\nAssisted-by: Claude:claude-opus-5-5\"",
		`gh pr create --body "Summary` + "\n\n" + forgeFooter() + "\n\nAssisted-by: Claude:claude-opus-5-5\"",
	} {
		if d := forgeCheck(t, reg, cmd); d.Verdict != VerdictBlock || d.EntryID != forgeTextEntryID {
			t.Errorf("Check = %q via %q (%s), want block via %q", d.Verdict, d.EntryID, d.Message, forgeTextEntryID)
		}
	}
	d := forgeCheck(t, reg, "gh pr create --body \"Summary\n\nAssisted-by: Claude:claude-opus-5-5\"")
	if contains(d.Matches, forgeTextEntryID) || contains(d.Matches, forgeUnreadEntryID) {
		t.Errorf("a clean body = %q via %q (matches %v), want neither blocked nor unjudged", d.Verdict, d.EntryID, d.Matches)
	}
}
