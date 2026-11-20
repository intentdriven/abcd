package repolint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTheLibraryEntrySanctionsTheSourcesCorpus holds the library glossary entry
// to the ruling in dcn-2610271112330536 (iss-2610262038322994): "sources
// corpus" is the sanctioned name of the store beneath the library, as the
// --corpus flag, commands/library.md and the library, banlist and surfaces
// chapters have it, so the entry must not forbid "corpus" and must name the
// sources corpus as that store.
func TestTheLibraryEntrySanctionsTheSourcesCorpus(t *testing.T) {
	const entry = ".abcd/development/brief/glossary/core/library.md"
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the test working directory")
		}
		dir = parent
	}
	raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(entry)))
	if err != nil {
		t.Fatalf("read %s: %v", entry, err)
	}

	var forbidden string
	found := false
	for _, line := range strings.Split(string(raw), "\n") {
		if v, ok := strings.CutPrefix(line, "forbidden_synonyms:"); ok {
			forbidden, found = v, true
			break
		}
	}
	if !found {
		t.Fatalf("%s has no forbidden_synonyms line", entry)
	}
	if strings.Contains(strings.ToLower(forbidden), `"corpus"`) {
		t.Errorf("%s forbids \"corpus\" (%s); dcn-2610271112330536 sanctions "+
			"\"sources corpus\" as the store beneath the library", entry, strings.TrimSpace(forbidden))
	}

	text := strings.Join(strings.Fields(string(raw)), " ")
	if !strings.Contains(text, "sources corpus") {
		t.Errorf("%s does not name the sources corpus as the store beneath the library", entry)
	}
}
