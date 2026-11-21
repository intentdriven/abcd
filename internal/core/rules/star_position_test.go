package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNamingRowStatesWhereAStarCommandMayStand pins iss-2611040554024081: the
// naming table's `*<DOMAIN>` row called the star command a leading-anchored
// prompt prefix, while parseStarCommands accepts one anywhere it starts the
// prompt or follows whitespace. The test runs Match on a prompt whose star
// command stands mid-prompt, then holds the row to what Match did.
func TestNamingRowStatesWhereAStarCommandMayStand(t *testing.T) {
	const prompt = "hello *SCRATCH there"
	var got []string
	for _, d := range Defaults().Match(prompt) {
		got = append(got, d.Name)
	}
	found := false
	for _, name := range got {
		if name == "SCRATCH" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Match(%q) = %v, want SCRATCH among them: a star command after whitespace activates", prompt, got)
	}
	if stars := parseStarCommands("hello*SCRATCH there"); stars["SCRATCH"] {
		t.Fatalf("parseStarCommands(%q) activated SCRATCH: a star command glued to a word must not", "hello*SCRATCH there")
	}

	path := filepath.Join("..", "..", "..", ".abcd", "development", "brief", "02-constraints", "04-naming.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the naming table: %v", err)
	}
	var row string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "| `*<DOMAIN>` |") {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatalf("%s has no `*<DOMAIN>` row", path)
	}
	if strings.Contains(strings.ToLower(row), "leading-anchored") {
		t.Errorf("the `*<DOMAIN>` row still says leading-anchored, which parseStarCommands is not:\n%s", row)
	}
	if !strings.Contains(row, "start the prompt or follow whitespace") {
		t.Errorf("the `*<DOMAIN>` row does not say the star command must %q:\n%s", "start the prompt or follow whitespace", row)
	}
}
