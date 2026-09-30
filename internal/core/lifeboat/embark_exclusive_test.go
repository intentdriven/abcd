package lifeboat

import (
	"os"
	"path/filepath"
	"testing"
)

// A file that lands at a create target after the rejudge — a retrospective a
// `reflect write` made in that window, say — refuses the write loudly instead
// of being replaced by the lifeboat's copy: every planned write is a create,
// and a create never renames over what it did not plan for.
func TestWriteEmbarkRefusesToReplaceAFileThatLandedAtACreateTarget(t *testing.T) {
	target := t.TempDir()
	const rel = ".abcd/development/retrospectives/v0.1.0/README.md"
	const theirs = "the retrospective written in the window\n"
	mustWrite(t, filepath.Join(target, rel), []byte(theirs))

	planned := []PlannedEmbark{{
		LifeboatPath: "retrospectives/v0.1.0/README.md",
		TargetPath:   rel,
		Family:       "retrospectives",
		Action:       ActionCreate,
		Content:      []byte("the lifeboat's copy\n"),
	}}
	written, _, _, _, err := writeEmbark(target, planned)
	if err == nil {
		t.Fatalf("writeEmbark replaced a file that landed at a create target (written %d)", written)
	}
	if got, _ := os.ReadFile(filepath.Join(target, rel)); string(got) != theirs {
		t.Errorf("the file at the create target was replaced: %q", got)
	}
}
