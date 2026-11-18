package banlist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The banlist chapter once said bare `abcd ahoy` lays down the guard
// artefacts (iss-2611040554189675). Bare `abcd ahoy` mutates nothing (brief
// 04-surfaces/01-ahoy.md); `abcd ahoy install` is what writes
// .githooks/pre-commit, .githooks/pre-merge-commit and .gitattributes. The
// chapter is held to that here.
func TestBriefBanlistChapterNamesAhoyInstallAsTheScaffolder(t *testing.T) {
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
	if want := "`abcd ahoy install` lays down the guard artefacts"; !strings.Contains(body, want) {
		t.Errorf("%s does not say %q", rel, want)
	}
}
