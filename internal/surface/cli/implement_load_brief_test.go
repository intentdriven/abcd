package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement"
)

// The brief's load section says what the bare check cannot run without: the
// --site flag, and each value it takes (iss-2611040555309558). Without one of
// them the check exits 2, so "exits 0 on every status" alone reads as if a bare
// `abcd implement load` answered. The prose names the flag in words, as it does
// the wait flag, and the generated appendix spells it:
// TestSurfaceChapterProseStatesNoShape keeps a flag's spelling out of the prose.
func TestTheBriefsLoadSectionNamesSiteAndItsValues(t *testing.T) {
	rel := ".abcd/development/brief/04-surfaces/27-implement.md"
	raw, err := os.ReadFile(filepath.Join(testRepoRoot(), filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	_, section, found := strings.Cut(string(raw), "\n## The load check\n")
	if !found {
		t.Fatalf("%s has no load section", rel)
	}
	if i := strings.Index(section, "\n## "); i >= 0 {
		section = section[:i]
	}
	if !strings.Contains(strings.Join(strings.Fields(section), " "), "the check needs its site flag") {
		t.Errorf("%s: the load section never says the check without a wait flag needs its site flag", rel)
	}
	_, entry, found := strings.Cut(string(raw), "\n### `abcd implement load`\n")
	if i := strings.Index(entry, "\n### "); i >= 0 {
		entry = entry[:i]
	}
	if !found || !strings.Contains(entry, "| `--site` |") {
		t.Errorf("%s: the appendix entry for the load check does not spell the site flag `--site`", rel)
	}
	for _, site := range implement.LoadSites() {
		if !strings.Contains(section, "`"+site+"`") {
			t.Errorf("%s: the load section does not name the --site value `%s`", rel, site)
		}
	}
}
