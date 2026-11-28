package tools

import (
	"path/filepath"
	"testing"
)

// displayIn is handed the home as the environment spells it and a path that
// filepath.Join built from it, which is the cleaned spelling. A home spelled
// with a doubled slash or a dot segment must still be shown as "~", or the
// message pasted into an issue carries the username (iss-2611270714040589).
func TestDisplayInMatchesTheCleanedSpellingOfAnUncleanHome(t *testing.T) {
	base := t.TempDir()
	for _, spell := range []string{"//alex", "/./alex"} {
		home := base + spell
		for p, want := range map[string]string{
			filepath.Join(home, ".local", "bin"): "~/.local/bin",
			home + "/.local/bin":                 "~/.local/bin",
			filepath.Clean(home):                 "~",
			home:                                 "~",
		} {
			if got := displayIn(home, p); got != want {
				t.Errorf("displayIn(%q, %q) = %q, want %q", home, p, got, want)
			}
		}
	}
}
