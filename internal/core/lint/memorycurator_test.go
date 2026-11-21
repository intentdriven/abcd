package lint

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestMemoryCuratorSectionCarriesNoStaleAgentCount (iss-2610262038359555):
// section 6 of the memory chapter, the curator, makes the point that
// `principle-distiller` gains roles rather than peers. It once said "The agent
// count stays at 15" while agents/ shipped eighteen prompts and 01-agents.md
// said so. The section may carry no number for the roster, or one that matches
// the prompts directly under agents/, and it keeps the roles-not-peers point.
func TestMemoryCuratorSectionCarriesNoStaleAgentCount(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRootFromPackage, ".abcd", "development", "brief", "05-internals", "07-memory.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	at := strings.Index(text, "\n## 6. The curator")
	if at < 0 {
		t.Fatal("the memory chapter carries no `## 6. The curator` section")
	}
	section := text[at+1:]
	if end := strings.Index(section, "\n## "); end >= 0 {
		section = section[:end+1]
	}

	pages, err := filepath.Glob(filepath.Join(repoRootFromPackage, "agents", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) == 0 {
		t.Fatal("no agent prompts found under agents/")
	}

	words := map[string]int{}
	for i, w := range []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine",
		"ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen",
		"twenty"} {
		words[w] = i
	}
	count := regexp.MustCompile(`(?i)agent count (?:stays|is|remains|holds) (?:at )?(\d+|[a-z]+)|\b(\d+|[a-z]+) agents? (?:prompts?|in the catalog|ship)`)
	for _, m := range count.FindAllStringSubmatch(section, -1) {
		tok := strings.ToLower(m[1] + m[2])
		n, err := strconv.Atoi(tok)
		if err != nil {
			v, ok := words[tok]
			if !ok {
				continue
			}
			n = v
		}
		if n != len(pages) {
			t.Errorf("the curator section states an agent count of %d (%q), but agents/ ships %d prompts", n, m[0], len(pages))
		}
	}

	if !strings.Contains(section, "gains role responsibilities, not agent peers") {
		t.Error("the curator section no longer says `principle-distiller` gains role responsibilities, not agent peers")
	}
}
