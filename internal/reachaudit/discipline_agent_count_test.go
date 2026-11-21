package reachaudit

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// agentCount matches a claim of how many agents abcd ships: "ships all 14
// (now 15) agents", "ships eighteen agents". A dated line ("the five agents
// shipped as of 2026-07-12") is history, not a roster claim, and is left alone.
var agentCount = regexp.MustCompile(`(?i)\bships\s+(all\s+)?(\d+|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|thirteen|fourteen|fifteen|sixteen|seventeen|eighteen|nineteen|twenty)\b(\s*\(now\s+\d+\))?\s+agents\b`)

// TestPromptQualityDisciplineDoesNotCountTheAgents (iss-2610262038443634):
// itd-5 said abcd ships "all 14 (now 15) agents" under its rule while the
// agents chapter counts eighteen prompts in agents/. The roster grows, so the
// discipline names what it governs, every agent prompt in agents/, with no
// count to go stale.
func TestPromptQualityDisciplineDoesNotCountTheAgents(t *testing.T) {
	page := ".abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md"
	b, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(page)))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(strings.Fields(string(b)), " ")
	for _, m := range agentCount.FindAllString(text, -1) {
		t.Errorf("%s: counts the agent roster (%q); name every agent prompt in agents/ instead", page, m)
	}
	if !strings.Contains(text, "every agent prompt in `agents/`") && !strings.Contains(text, "every agent prompt in agents/") {
		t.Errorf("%s: does not say the rule governs every agent prompt in agents/", page)
	}
}
