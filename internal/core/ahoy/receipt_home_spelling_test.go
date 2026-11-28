package ahoy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// A receipt note that is a sentence rather than a bare path reaches the
// embedded pass, which must redact the home in the cleaned spelling a
// filepath.Join path carries as well as in the one HOME is written in
// (iss-2611270714040589).
func TestReceiptPathRedactsAnUncleanHomeEmbeddedInANote(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "alex"), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := base + "//alex"
	t.Setenv("HOME", raw)
	t.Setenv(abcdhome.StageHomeEnv, "")
	cwd := t.TempDir()
	for in, want := range map[string]string{
		"run the fix in " + filepath.Join(raw, "bin"): "run the fix in ~/bin",
		"run the fix in " + raw + "/bin":              "run the fix in ~/bin",
	} {
		if got := receiptPath(cwd, in); got != want {
			t.Errorf("receiptPath(%q) = %q, want %q", in, got, want)
		}
	}
}
