package lab

import (
	"errors"
	"path/filepath"
	"testing"
)

// The store's paths are built with filepath.Join, which cleans the home they
// start from, while redact is handed the home as the environment spells it. A
// home spelled with a doubled slash or a dot segment must still be redacted
// out of an error that names a store path (iss-2611270714040589).
func TestRedactRedactsTheCleanedSpellingOfAnUncleanHome(t *testing.T) {
	base := t.TempDir()
	for _, spell := range []string{"//alex", "/./alex"} {
		home := base + spell
		err := errors.New("mkdir " + filepath.Join(home, "store", "lab") + ": permission denied")
		if got, want := redact(err, home), "mkdir ~/store/lab: permission denied"; got != want {
			t.Errorf("redact(%q, %q) = %q, want %q", err, home, got, want)
		}
	}
}
