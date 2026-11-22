package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The delivery chapter's index once gave 01-build-sequence.md a delivery order
// of MVP, then the companion harness, then a named host, while the page itself
// is a dependency DAG of numbered build milestones (iss-2610262040327780). The
// index row describes the page as it is: a DAG naming each milestone, and none
// of the retired linear order.
func TestBriefDeliveryIndexDescribesTheBuildSequenceAsADAG(t *testing.T) {
	const indexRel = ".abcd/development/brief/06-delivery/README.md"
	root := filepath.Join("..", "..", "..")
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(indexRel)))
	if err != nil {
		t.Fatalf("read %s: %v", indexRel, err)
	}

	var row string
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "|") && strings.Contains(line, "(01-build-sequence.md)") {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatalf("%s has no index row linking 01-build-sequence.md", indexRel)
	}
	lower := strings.ToLower(row)

	for _, stale := range []string{"mvp", "companion harness", "claude code", "→"} {
		if strings.Contains(lower, stale) {
			t.Errorf("the 01-build-sequence.md row of %s still gives a linear delivery order (%q): %s", indexRel, stale, row)
		}
	}
	for _, want := range []string{
		"dependency dag",
		"scaffold",
		"install and launch",
		"history",
		"memory",
		"intent and review",
		"spec engine",
		"run seam",
		"lifeboat",
	} {
		if !strings.Contains(lower, want) {
			t.Errorf("the 01-build-sequence.md row of %s does not name %q: %s", indexRel, want, row)
		}
	}
}
