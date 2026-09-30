package reflect

import (
	"os"
	"strings"
	"testing"
)

// The lessons a retrospective carries are read back out of the file the writer
// produced: one lesson per bullet, each carrying its release.
func TestReadLessonsReadsTheWrittenRetrospective(t *testing.T) {
	r := releaseRepo(t)
	res, err := Write(r.Root(), WriteRequest{Tag: "v0.2.0", Answers: fullAnswers(), ProceedDespiteUnshipped: true, Now: fixedNow})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	data, err := os.ReadFile(abs(r, res.Path))
	if err != nil {
		t.Fatal(err)
	}
	got := ReadLessons(data)
	want := []Lesson{
		{Release: "v0.2.0", Text: "Cut the release before the audit backlog grows"},
		{Release: "v0.2.0", Text: "Audit every intent in the window it ships"},
	}
	if len(got) != len(want) {
		t.Fatalf("lessons = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("lesson %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// A lessons section written as paragraphs yields one lesson per paragraph, and
// a fenced example inside it is not a lesson.
func TestReadLessonsTakesParagraphsAndSkipsFences(t *testing.T) {
	doc := "---\nrelease: v1.0.0\n---\n\n# Retrospective\n\n## Lessons learned\n\nKeep the seed small.\nIt reads faster.\n\n```\n- not a lesson\n```\n\nName the owner.\n\n## Decisions made\n\n- not a lesson either\n"
	got := ReadLessons([]byte(doc))
	if len(got) != 2 || got[0].Text != "Keep the seed small. It reads faster." || got[1].Text != "Name the owner." || got[0].Release != "v1.0.0" {
		t.Fatalf("lessons = %+v", got)
	}
}

// Criterion 6's ranking (spec scope 7, decision 2): the lessons most like the
// new voyage's brief come first, three of them, and the rest follow as a list.
// The ranking says it is a heuristic.
func TestRankLessonsShowsTheFewMostLikeTheBriefAndListsTheRest(t *testing.T) {
	framing := "A command-line tool that packs a repository's release history into a portable archive, " +
		"verifies the archive against its manifest, and unpacks it into a new repository."
	lessons := []Lesson{
		{Release: "v0.1.0", Text: "Choose font colours for the website early"},
		{Release: "v0.1.0", Text: "Verify every archive against its manifest before unpacking"},
		{Release: "v0.2.0", Text: "Keep the release history portable between repositories"},
		{Release: "v0.2.0", Text: "Hire a designer for the logo"},
		{Release: "v0.3.0", Text: "The command-line tool should pack the archive in one pass"},
	}
	rk := RankLessons(framing, lessons)
	if rk.Heuristic == "" || !strings.Contains(rk.Heuristic, "heuristic") {
		t.Errorf("heuristic = %q, want the method named as a heuristic", rk.Heuristic)
	}
	if len(rk.Top) != TopLessons || len(rk.Rest) != len(lessons)-TopLessons {
		t.Fatalf("top %d / rest %d, want %d / %d", len(rk.Top), len(rk.Rest), TopLessons, len(lessons)-TopLessons)
	}
	top := map[string]bool{}
	for _, l := range rk.Top {
		top[l.Text] = true
		if l.Score <= 0 || len(l.Shared) == 0 {
			t.Errorf("top lesson %q has score %v and shared %v, want both", l.Text, l.Score, l.Shared)
		}
	}
	for _, off := range []string{"Choose font colours for the website early", "Hire a designer for the logo"} {
		if top[off] {
			t.Errorf("an unrelated lesson ranked in the top few: %q", off)
		}
	}
	for i := 1; i < len(rk.Top); i++ {
		if rk.Top[i].Score > rk.Top[i-1].Score {
			t.Errorf("top is not best first: %+v", rk.Top)
		}
	}
}

// Fewer lessons than the few: all of them are shown, none listed.
func TestRankLessonsWithFewerThanTheFew(t *testing.T) {
	rk := RankLessons("anything at all", []Lesson{{Release: "v1.0.0", Text: "one lesson"}})
	if len(rk.Top) != 1 || len(rk.Rest) != 0 {
		t.Fatalf("top %d / rest %d", len(rk.Top), len(rk.Rest))
	}
}
