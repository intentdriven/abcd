package statusline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBoardChapterNamesTheMergingElement holds the board chapter's list of the
// row to the row itself: the presence line's paragraph names the merging
// element, by its label and its key, and points at the mode chapter that
// describes it (iss-2611040554051652).
func TestBoardChapterNamesTheMergingElement(t *testing.T) {
	path := filepath.Join("..", "..", "..", ".abcd", "development", "brief", "04-surfaces", "08-abcd.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the board chapter: %v", err)
	}
	const opener = "**The presence line**"
	_, rest, ok := strings.Cut(string(raw), opener)
	if !ok {
		t.Fatalf("%s has no %q paragraph", path, opener)
	}
	para, _, _ := strings.Cut(rest, "\n\n")
	para = strings.Join(strings.Fields(para), " ")
	for _, want := range []string{
		"`" + MergingLabel,
		"`" + string(KeyMerging) + "`",
		"26-mode.md",
	} {
		if !strings.Contains(para, want) {
			t.Errorf("the presence line's paragraph does not name %s", want)
		}
	}
	branch := strings.Index(para, "the branch")
	merging := strings.Index(para, "`"+MergingLabel)
	counts := strings.Index(para, "the record's counts")
	if branch < 0 || counts < 0 || (merging >= 0 && (merging <= branch || merging >= counts)) {
		t.Errorf("the paragraph does not list the merging element between the branch and the record's counts")
	}
}
