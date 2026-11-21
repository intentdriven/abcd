package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTheMemoryBriefCitesTheLibraryExitContract pins the memory chapter's
// `/abcd:library lint` row to the exit contract the library surface chapter
// states: only blockers exit non-zero. The row once called warn-only exit 0
// "the recorded divergence from the standalone grammar" and cited the lint
// contract's severity model, which covers the two tree gates, not the memory
// lint, and records no such divergence (iss-2610262038368760).
func TestTheMemoryBriefCitesTheLibraryExitContract(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	data, err := os.ReadFile(filepath.Join(root, ".abcd", "development", "brief", "05-internals", "07-memory.md"))
	if err != nil {
		t.Fatal(err)
	}
	var row string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "| `/abcd:library lint`") {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatal("the memory chapter has no `/abcd:library lint` row")
	}
	for _, wrong := range []string{
		"recorded divergence",
		"06-lint.md",
	} {
		if strings.Contains(row, wrong) {
			t.Errorf("the library lint row still says %q: the lint contract covers the two tree gates and records no divergence for the memory lint", wrong)
		}
	}
	const cite = "../04-surfaces/07-library.md"
	if !strings.Contains(row, cite) {
		t.Errorf("the library lint row does not cite the library chapter's exit contract (%s)", cite)
	}
	lib, err := os.ReadFile(filepath.Join(root, ".abcd", "development", "brief", "04-surfaces", "07-library.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(strings.Fields(string(lib)), " "), "blockers exit non-zero, warnings alone exit 0") {
		t.Error("the library chapter no longer states the store lint's exit contract the memory chapter cites")
	}
}
