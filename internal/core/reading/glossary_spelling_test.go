package reading

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestTheGlossaryDescribesTheReadingChaptersSpelling holds the spelling section
// of the reading-position glossary entry to what the reading chapter does
// (iss-2610262035432356).
//
// The entry once said the record writes the full phrase "reading position" on
// first use and drops to bare "position" inside the chapter, but the chapter
// opens with bare "position" ("Assembly takes a position and a target state")
// and never writes the full phrase. So the test checks both ends: the chapter
// still never writes "reading position" on its own, and the glossary no longer
// claims a first use in full and says the chapter is bare throughout.
func TestTheGlossaryDescribesTheReadingChaptersSpelling(t *testing.T) {
	const (
		chapter  = ".abcd/development/brief/04-surfaces/23-reading.md"
		glossary = ".abcd/development/brief/glossary/core/reading-position.md"
	)
	read := func(rel string) string {
		raw, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		return strings.Join(strings.Fields(string(raw)), " ")
	}

	// "cold-reading position" names the agent's route key, not the term, so a
	// hyphen before "reading" does not count as the full phrase.
	full := regexp.MustCompile(`(?i)(^|[^-\w])reading position`)
	if loc := full.FindStringIndex(read(chapter)); loc != nil {
		t.Errorf("%s writes the full phrase \"reading position\"; the glossary records the "+
			"chapter as bare \"position\" throughout, so one of the two has drifted", chapter)
	}

	section := read(glossary)
	start := strings.Index(section, "## The spelling")
	if start < 0 {
		t.Fatalf("%s has no \"## The spelling\" section", glossary)
	}
	section = section[start:]
	if end := strings.Index(section[len("## The spelling"):], "## "); end >= 0 {
		section = section[:len("## The spelling")+end]
	}
	if strings.Contains(section, "full phrase on first use") {
		t.Errorf("%s still says the record writes the full phrase on first use; the reading "+
			"chapter's first use is bare \"position\"", glossary)
	}
	if !strings.Contains(section, `bare "position" throughout`) {
		t.Errorf("%s's spelling section does not say the reading chapter writes bare "+
			"\"position\" throughout", glossary)
	}
	if !strings.Contains(section, "--position") {
		t.Errorf("%s's spelling section no longer ties bare \"position\" to the `--position` flag", glossary)
	}
}
