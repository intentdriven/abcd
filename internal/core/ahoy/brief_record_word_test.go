package ahoy

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestTheAhoyChapterCallsItsUnidentifiedArtefactsFiles holds the ahoy chapter
// to the glossary's reservation of "record" (iss-2610262032441828).
//
// glossary/core/record.md keeps the word for a filed document with an id that a
// command mints; an artefact with no id and no minting command is not one. The
// setup answers written under interviews/, the `path-entry` file and the
// provenance it carries have no id, so the chapter calls them files.
func TestTheAhoyChapterCallsItsUnidentifiedArtefactsFiles(t *testing.T) {
	const chapter = ".abcd/development/brief/04-surfaces/01-ahoy.md"
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(chapter)))
	if err != nil {
		t.Fatalf("read %s: %v", chapter, err)
	}
	text := strings.Join(strings.Fields(string(raw)), " ")
	// The `path-entry` phrase is singular and ends on a word boundary, so the
	// verb ("the copy `path-entry` records") is not read as the noun.
	for _, phrase := range []string{
		`answers records?\b`,
		`provenance records?\b`,
		"path-entry`? record\\b",
		`records it writes there`,
		`with its record\b`,
	} {
		if loc := regexp.MustCompile(phrase).FindStringIndex(text); loc != nil {
			t.Errorf("%s writes %q; the artefact carries no id and no minting command, "+
				"so glossary/core/record.md calls it a file, not a record", chapter, text[loc[0]:loc[1]])
		}
	}
}
