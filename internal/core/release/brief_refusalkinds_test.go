package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The launch chapter's refusal-kinds table describes each kind Emit raises. Emit
// raises unfixed-finding for two cases: a consequential finding this cycle
// captured that is still open and unanswered, and an open issue record that
// differs from HEAD, at any severity (iss-2610090642371836). The table's row
// once named the first case only while the guardrail list in the same chapter
// named both (iss-2610101803326081), so the row is held to both here.
func TestBriefRefusalKindsRowForUnfixedFindingNamesBothCases(t *testing.T) {
	const rel = ".abcd/development/brief/04-surfaces/04-launch.md"
	body, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	prefix := "| `" + string(RefusalUnfixedFinding) + "` |"
	var row string
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, prefix) {
			if row != "" {
				t.Fatalf("%s carries more than one %s row", rel, RefusalUnfixedFinding)
			}
			row = line
		}
	}
	if row == "" {
		t.Fatalf("%s has no refusal-kinds row for %s", rel, RefusalUnfixedFinding)
	}
	for _, want := range []string{
		"consequential finding this cycle captured", // the graded half
		"open issue record that differs from HEAD",  // the uncommitted half
		"any severity",
	} {
		if !strings.Contains(row, want) {
			t.Errorf("%s's %s row does not say %q:\n%s", rel, RefusalUnfixedFinding, want, row)
		}
	}
}
