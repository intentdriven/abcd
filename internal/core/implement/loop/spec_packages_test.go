package loop

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The packages line's entries (iss-2610241549334990): a build reserves each
// entry's leading path alone, a trailing parenthetical note dropped and a
// comma inside one not splitting the entry, and refuses an entry that is
// still prose, naming the spec and the entry, rather than reserve a path
// that shares nothing with the package it means.

// notedSpec is spc-1 with a packages line carrying a new-package note, a
// parenthetical with a comma inside it, and a plain path.
var notedSpec = specWithSteps("") + "\n## Footprint\n\n" +
	"- packages: internal/core/x (new), scripts/cloud/connectors/claude (new, the host connector), commands/\n" +
	"- tests: the widget\n"

// TestBuildReservesEachFootprintEntryAsItsLeadingPath: the reservation names
// the three paths, with no note and no half of a split parenthetical.
func TestBuildReservesEachFootprintEntryAsItsLeadingPath(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), notedSpec)
	bare := onOrigin(t, repo)
	if _, err := Start(repo.Root(), "itd-10", Options{}); err != nil {
		t.Fatal(err)
	}
	want := []string{"internal/core/x", "scripts/cloud/connectors/claude", "commands"}
	if fp := footprintOf(t, bare, "itd-10"); !slices.Equal(fp, want) {
		t.Fatalf("the reservation's footprint = %q, want %q", fp, want)
	}
}

// TestBuildRefusesAProseFootprintEntry: an entry that still holds whitespace
// once its note is dropped, and names nothing in the tree, refuses the start
// at the reservation, naming the spec and the entry; nothing is started or
// pushed.
func TestBuildRefusesAProseFootprintEntry(t *testing.T) {
	prose := specWithSteps("") + "\n## Footprint\n\n" +
		"- packages: internal/widget, the OpenAI-compatible client adapter (new)\n- tests: the widget\n"
	repo := loopRepo(t, readyIntent("", settledQuestions), prose)
	bare := onOrigin(t, repo)
	_, err := Start(repo.Root(), "itd-10", Options{})
	r := mustRefusal(t, err)
	if r.Stage != StageReservation || r.Contention {
		t.Fatalf("want a reservation refusal, got %+v", r)
	}
	for _, want := range []string{"spc-1", `"the OpenAI-compatible client adapter (new)"`, "nothing was started"} {
		if !strings.Contains(r.Reason, want) {
			t.Fatalf("the refusal names %s: %q", want, r.Reason)
		}
	}
	runTierAbsent(t, repo.Root())
	if tip := originReservation(t, bare, reservedRef); tip.commit != "" {
		t.Fatalf("nothing is pushed: %+v", tip)
	}
}

// TestAFootprintEntryWithASpaceThatNamesAFileIsKept: whitespace alone is not
// prose; an entry naming a path that exists in the tree is reserved as it
// stands.
func TestAFootprintEntryWithASpaceThatNamesAFileIsKept(t *testing.T) {
	spaced := specWithSteps("") + "\n## Footprint\n\n- packages: docs/a note.md, internal/widget\n- tests: the widget\n"
	repo := loopRepo(t, readyIntent("", settledQuestions), spaced)
	if err := os.MkdirAll(filepath.Join(repo.Root(), "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo.Root(), "docs", "a note.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := specPackages(repo.Root(), "spc-1")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"docs/a note.md", "internal/widget"}; !slices.Equal(got, want) {
		t.Fatalf("specPackages = %q, want %q", got, want)
	}
}

// TestBuildNextExcludesAnIntentWhoseFootprintIsProse: the pick does not
// abort on one spec's prose entry; it excludes that intent at the check
// `footprint`, naming the entry, and moves on.
func TestBuildNextExcludesAnIntentWhoseFootprintIsProse(t *testing.T) {
	repo := pickRepo(t, map[string][2]string{
		"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", "- packages: callers across capture and lint\n- tests: the widget\n")},
		"11": {pickIntent("11", "", settledQuestions, "- it works\n"), pickSpec("11", "- packages: a, b\n- tests: the widget\n")},
	})
	onOrigin(t, repo)
	set, err := candidates(repo.Root(), "")
	if err != nil {
		t.Fatalf("the pick does not abort on a prose entry: %v", err)
	}
	i := slices.IndexFunc(set.Excluded, func(e Excluded) bool { return e.ID == "itd-10" })
	if i < 0 || set.Excluded[i].Check != CheckFootprint || !strings.Contains(set.Excluded[i].Reason, `"callers across capture and lint"`) {
		t.Fatalf("the prose footprint is excluded at the footprint check, naming the entry: %+v", set.Excluded)
	}
	if len(set.Candidates) != 1 || set.Candidates[0].ID != "itd-11" {
		t.Fatalf("only the footprinted intent is a candidate: %+v", set.Candidates)
	}
}
