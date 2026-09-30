package changelog

import (
	"testing"

	"github.com/intentdriven/abcd/internal/core/launch"
)

// TestTargetMoveNoteNamesEveryMove is criterion 3's changelog half
// (itd-2609212103572513, ruling BS1 of 2026-09-29): the note names each
// targeted intent the cut passed and the target it carried, says each targets
// `next`, and is the one line the note predicate recognises, so a reader that
// stamps a release onto the records a section credits can pass over it.
func TestTargetMoveNoteNamesEveryMove(t *testing.T) {
	if got := TargetMoveNote(nil); got != "" {
		t.Errorf("no move, no note: %q", got)
	}
	got := TargetMoveNote([]launch.TargetMove{
		{ID: "itd-7", Path: "p/itd-7.md", From: "v0.11.0"},
		{ID: "itd-9", Path: "p/itd-9.md", From: "next"},
	})
	want := "Targeted and not shipped in this release, so each targets the next release (`next`): " +
		"itd-7 (targeted v0.11.0), itd-9 (targeted `next`)."
	if got != want {
		t.Errorf("TargetMoveNote =\n%q\nwant\n%q", got, want)
	}
	if !IsTargetMoveNote(got) {
		t.Error("the predicate does not recognise the note the renderer writes")
	}
	for _, line := range []string{"", "- **A version is a fact.** Derived. (itd-7)", "### Added", "Targeted"} {
		if IsTargetMoveNote(line) {
			t.Errorf("IsTargetMoveNote(%q) = true", line)
		}
	}
}
