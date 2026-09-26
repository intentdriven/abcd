package main

import "testing"

// TestPrunedNoteNamesWhatWasSkipped: record-lint prunes gitignored paths under
// its roots like the docs lint, and names them on stderr so a smaller tree is
// never read as a clean one (iss-2609151952353626).
func TestPrunedNoteNamesWhatWasSkipped(t *testing.T) {
	if got := prunedNote(nil); got != "" {
		t.Errorf("nothing pruned must say nothing, got %q", got)
	}
	want := "record-lint: skipped 2 gitignored path(s) under the roots: a/, b.md"
	if got := prunedNote([]string{"a/", "b.md"}); got != want {
		t.Errorf("prunedNote = %q, want %q", got, want)
	}
}
