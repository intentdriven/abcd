package history

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// The history store's literal home backstop sweeps a non-ASCII home in the
// other Unicode normal form: macOS hands a name back decomposed (NFD) when it
// was typed composed (NFC), and neither the detector nor the sweep read the
// other form (iss-2609290055092630).
func TestRedactSweepsTheOtherNormalFormOfTheHome(t *testing.T) {
	repoRoot, _ := setupStore(t)
	base := t.TempDir()
	t.Setenv("HOME", filepath.Join(base, "café"))
	t.Setenv(abcdhome.StageHomeEnv, "")
	nfd := filepath.Join(base, "café")
	got, err := Redact(repoRoot, "user: open "+nfd+"/journal.md\n", "a text")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got.Text, nfd) || !strings.Contains(got.Text, "~/journal.md") {
		t.Errorf("the home in its other normal form was not swept: %q", got.Text)
	}
}
