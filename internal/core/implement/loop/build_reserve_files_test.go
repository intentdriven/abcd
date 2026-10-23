package loop

import (
	"slices"
	"strings"
	"testing"
)

// TestBuildReservesTheFilesItsSpecNamesAndWarnsOfADirectory
// (iss-2610230016544142): a build reserves its intent with exactly the paths
// its spec's footprint names, so a file named there is reserved as that file;
// a path naming a directory git tracks is reserved as it stands, and the
// start warns of it, naming the directory and the tracked files it covers.
// A footprint of files alone draws no warning, on the start or the resume.
func TestBuildReservesTheFilesItsSpecNamesAndWarnsOfADirectory(t *testing.T) {
	spec := specWithSteps("") + "\n## Footprint\n\n- packages: internal/widget/a.go, commands\n- tests: the widget\n"
	repo := loopRepo(t, readyIntent("", settledQuestions), spec)
	repo.Write("internal/widget/a.go", "package widget\n")
	repo.Write("internal/widget/b.go", "package widget\n")
	repo.Write("commands/one.md", "one\n")
	repo.Write("commands/two.md", "two\n")
	repo.Commit("the widget and the commands")
	bare := onOrigin(t, repo)

	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if fp := footprintOf(t, bare, "itd-10"); !slices.Equal(fp, []string{"internal/widget/a.go", "commands"}) {
		t.Fatalf("the reservation names the spec's file as that file, and its directory as it stands: %v", fp)
	}
	if len(res.ReservationWarnings) != 1 || !strings.Contains(res.ReservationWarnings[0], `"commands" is a directory covering 2 tracked files on origin/main`) ||
		!strings.Contains(res.ReservationWarnings[0], "name the files the work changes instead") {
		t.Fatalf("the start warns of the directory with its file count: %q", res.ReservationWarnings)
	}
}

// TestBuildOfAFileFootprintDrawsNoWarning: a spec whose footprint names files
// alone is reserved with them, and the start warns of nothing.
func TestBuildOfAFileFootprintDrawsNoWarning(t *testing.T) {
	spec := specWithSteps("") + "\n## Footprint\n\n- packages: internal/widget/a.go, internal/widget/new.go\n- tests: the widget\n"
	repo := loopRepo(t, readyIntent("", settledQuestions), spec)
	repo.Write("internal/widget/a.go", "package widget\n")
	repo.Commit("the widget")
	bare := onOrigin(t, repo)

	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if fp := footprintOf(t, bare, "itd-10"); !slices.Equal(fp, []string{"internal/widget/a.go", "internal/widget/new.go"}) {
		t.Fatalf("the reservation names the spec's files: %v", fp)
	}
	if len(res.ReservationWarnings) != 0 {
		t.Fatalf("a file footprint draws no warning: %q", res.ReservationWarnings)
	}
}

// TestAResumeWarnsOnlyWhenItReservesADirectoryAfresh: a resume that renews
// this copy's live reservation keeps its footprint and warns of nothing; one
// that reserves afresh (after the reservation was released) names the
// directory again, as a start does.
func TestAResumeWarnsOnlyWhenItReservesADirectoryAfresh(t *testing.T) {
	spec := specWithSteps("") + "\n## Footprint\n\n- packages: internal/widget/a.go, commands\n- tests: the widget\n"
	repo := loopRepo(t, readyIntent("", settledQuestions), spec)
	repo.Write("internal/widget/a.go", "package widget\n")
	repo.Write("commands/one.md", "one\n")
	repo.Write("commands/two.md", "two\n")
	repo.Commit("the widget and the commands")
	onOrigin(t, repo)

	if _, err := Start(repo.Root(), "itd-10", Options{}); err != nil {
		t.Fatal(err)
	}
	renewed, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil || !renewed.Resumed || !strings.Contains(renewed.Reservation, "renewed the reservation on itd-10") {
		t.Fatalf("the resume renews: %+v %v", renewed, err)
	}
	if len(renewed.ReservationWarnings) != 0 {
		t.Fatalf("a resume that renews warns again: %q", renewed.ReservationWarnings)
	}
	if _, err := ReleaseReservation(repo.Root(), "itd-10", Options{}); err != nil {
		t.Fatal(err)
	}
	again, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil || !again.Resumed || !strings.HasPrefix(again.Reservation, "reserved itd-10 on origin") {
		t.Fatalf("the resume reserves afresh: %+v %v", again, err)
	}
	if len(again.ReservationWarnings) != 1 || !strings.Contains(again.ReservationWarnings[0], `"commands" is a directory covering 2 tracked files`) {
		t.Fatalf("a resume that reserves afresh names the directory: %q", again.ReservationWarnings)
	}
}
