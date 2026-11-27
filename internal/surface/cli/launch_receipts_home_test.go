package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// scrubMessage takes the home out of a receipts-check message in every
// spelling it can carry: the one HOME is written in, the cleaned one a
// filepath.Join path carries, and the symlink-resolved one the kernel reports
// (iss-2611270714040589).
func TestScrubMessageRedactsEverySpellingOfTheHome(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "alex")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(home)
	if err != nil {
		t.Fatal(err)
	}
	raw := base + "//alex"
	t.Setenv("HOME", raw)
	t.Setenv(abcdhome.StageHomeEnv, "")
	for in, want := range map[string]string{
		"receipt missing under " + raw + "/r":      "receipt missing under ~/r",
		"receipt missing under " + home + "/r":     "receipt missing under ~/r",
		"receipt missing under " + resolved + "/r": "receipt missing under ~/r",
	} {
		if got := scrubMessage(in); got != want {
			t.Errorf("scrubMessage(%q) = %q, want %q", in, got, want)
		}
	}
}
