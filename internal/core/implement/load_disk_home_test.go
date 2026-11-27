package implement

import (
	"path/filepath"
	"testing"
)

// fsRedact is handed the home the disk check was given, which may be written
// uncleanly, and an error that names it either as written (a path
// concatenated onto it) or cleaned (one filepath.Join built, or the kernel
// reported); both are redacted (iss-2611270714040589). An empty home
// redacts nothing.
func TestFsRedactRedactsTheRawAndTheCleanedSpellingOfHome(t *testing.T) {
	base := t.TempDir()
	for _, spell := range []string{"//alex", "/./alex"} {
		raw := base + spell
		for in, want := range map[string]string{
			"statfs " + raw + "/Library/Caches: denied":                      "statfs ~/Library/Caches: denied",
			"statfs " + filepath.Join(raw, "Library", "Caches") + ": denied": "statfs ~/Library/Caches: denied",
		} {
			if got := fsRedact(in, raw); got != want {
				t.Errorf("fsRedact(%q, %q) = %q, want %q", in, raw, got, want)
			}
		}
	}
	if in := "statfs /x: denied"; fsRedact(in, "") != in {
		t.Errorf("fsRedact with no home changed %q", in)
	}
}
