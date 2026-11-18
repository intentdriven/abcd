package banlist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The banlist chapter once said bare `abcd ahoy` lays down the guard
// artefacts (iss-2611040554189675). Bare `abcd ahoy` mutates nothing (brief
// 04-surfaces/01-ahoy.md); the install sub-verb is what writes
// .githooks/pre-commit, .githooks/pre-merge-commit and .gitattributes. The
// chapter prose may not spell a sub-verb path (TestSurfaceChapterProseStatesNoShape
// in internal/surface/cli), so it names the installer the way the other surface
// chapters do, "the ahoy installer". The chapter is held to that here.
func TestBriefBanlistChapterNamesAhoyInstallerAsTheScaffolder(t *testing.T) {
	const rel = ".abcd/development/brief/04-surfaces/20-banlist.md"
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	// Fold line wrapping so a phrase split across lines still matches.
	body := strings.Join(strings.Fields(string(raw)), " ")
	if strings.Contains(body, "`abcd ahoy` lays down the guard artefacts") {
		t.Errorf("%s still says bare `abcd ahoy` lays down the guard artefacts; bare ahoy mutates nothing", rel)
	}
	for _, want := range []string{
		"The [ahoy installer](01-ahoy.md) lays down the guard artefacts",
		"bare `abcd ahoy` only reports, and writes none of them",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("%s does not say %q", rel, want)
		}
	}
}
