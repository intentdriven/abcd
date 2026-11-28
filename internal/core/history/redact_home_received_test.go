package history

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// A home whose path is not valid UTF-8 above the account name reaches a
// transcript with U+FFFD in place of each invalid byte, and the history store
// sweeps it in that form. (An account name that is not valid UTF-8 is refused
// instead: TestRedactRefusesWhileAnIdentityValueIsNotUTF8.)
func TestRedactSweepsAHomeThatIsNotUTF8AsATranscriptCarriesIt(t *testing.T) {
	repoRoot, _ := setupStore(t)
	base := t.TempDir()
	t.Setenv("HOME", filepath.Join(base, "donn\xe9es", "maya"))
	t.Setenv(abcdhome.StageHomeEnv, "")
	received := filepath.Join(base, "donn�es", "maya")
	got, err := Redact(repoRoot, "user: open "+received+"/journal.md\n", "a text")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got.Text, received) || !strings.Contains(got.Text, "~/journal.md") {
		t.Errorf("the home as a transcript carries it was not swept: %+q", got.Text)
	}
}
