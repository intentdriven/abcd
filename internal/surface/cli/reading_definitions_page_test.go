package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The bare verb's `definitions` field is what the definition LOCATOR resolved,
// not an inventory of what sits under `agents/`. The page said "present", which
// stopped being true at itd-184: a `cold-reading-<name>.md` naming no closed
// position is invisible to the locator, and a definition silent about its
// position or its regime — or stating another position's — refuses the whole
// verb with exit 2 rather than being listed (iss-2608311039586922).
//
// framework 8.4: four agent definitions, one per position, each holding "its
// regime value". The regime is the definition's property, which is exactly why a
// definition that does not state one cannot be reported as an instrument that is
// there — so the page has to describe a refusal, not a listing.
func TestReadingPageDescribesDefinitionsAsResolvedNotPresent(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRootFromTest(t), "commands", "reading.md"))
	if err != nil {
		t.Fatalf("read the plugin surface: %v", err)
	}
	// Bounded to the bare verb's own section: the page says "resolve", "refuses"
	// and "exit 2" elsewhere about the assemble and ingest verbs, and an
	// unbounded search would pass on those and assert nothing here.
	section := statusSectionOf(t, string(raw))

	if strings.Contains(section, "present under") {
		t.Error("the status section still describes `definitions` as what is PRESENT under agents/; " +
			"the field is what the locator resolves")
	}
	for _, want := range []string{"resolve", "refus", "exit 2"} {
		if !strings.Contains(section, want) {
			t.Errorf("the bare verb's `definitions` account does not mention %q:\n%s", want, section)
		}
	}
}

// statusSectionOf returns the page's `## Status (bare)` section — the account of
// the bare verb, up to the next heading.
func statusSectionOf(t *testing.T, body string) string {
	t.Helper()
	const head = "\n## Status (bare)\n"
	i := strings.Index(body, head)
	if i < 0 {
		t.Fatal("commands/reading.md carries no `## Status (bare)` section")
	}
	rest := body[i+len(head):]
	if j := strings.Index(rest, "\n## "); j >= 0 {
		return rest[:j]
	}
	return rest
}
