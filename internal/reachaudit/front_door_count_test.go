package reachaudit

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// frontDoorOrdinal matches a front door counted by position: "second front
// door", "third thin front door", "adr-23's third".
var frontDoorOrdinal = regexp.MustCompile(`(?i)\b(first|second|third|fourth|fifth)\s+(thin\s+)?(front\s+)?door\b|adr-23's\s+(first|second|third|fourth|fifth)\b`)

// TestTheMCPDoorIsNotCountedByPosition (iss-2610262039030685): adr-23 names
// the CLI as the first front door and the MCP server and the markdown plugin
// surface as later ones, in no order, so a page that counts the MCP door
// ("second", "third") contradicts another that counts it differently. The MCP
// intent calls it the MCP front door, and neither it nor the configuration
// chapter's layout gives the door a position.
func TestTheMCPDoorIsNotCountedByPosition(t *testing.T) {
	root := filepath.Join("..", "..")
	pages := []string{
		".abcd/development/intents/drafts/itd-113-abcd-s-core-verbs-are-reachable-over-mcp.md",
		".abcd/development/brief/05-internals/03-configuration.md",
	}
	for _, page := range pages {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(page)))
		if err != nil {
			t.Fatal(err)
		}
		// Strip the tree-drawing columns so a phrase wrapped across a
		// layout comment still reads as one.
		text := strings.Join(strings.Fields(strings.NewReplacer("│", " ", "#", " ").Replace(string(b))), " ")
		for _, m := range frontDoorOrdinal.FindAllString(text, -1) {
			t.Errorf("%s: counts a front door by position (%q); adr-23 names the MCP door as one of its later doors, in no order", page, m)
		}
		if strings.Contains(page, "itd-113") && !strings.Contains(text, "the MCP front door") {
			t.Errorf("%s: does not call the MCP door the MCP front door", page)
		}
	}
}
