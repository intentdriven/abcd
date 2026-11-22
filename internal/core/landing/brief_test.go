package landing

import (
	"strings"
	"testing"
)

// fillBrief fills a review brief as a reviewer would: its findings, and the
// verdict word in place of the choice.
func fillBrief(t *testing.T, brief, findings, verdict string) string {
	t.Helper()
	if !strings.Contains(brief, briefFindings) || !strings.Contains(brief, briefVerdict) {
		t.Fatalf("the brief leaves no findings or verdict to fill:\n%s", brief)
	}
	return strings.Replace(strings.Replace(brief, briefFindings, findings, 1), briefVerdict, verdict, 1)
}

// A brief filled in with a passing verdict is a return --record-review records: the
// brief is the shape the reading holds a return to, for a full pair and a delta.
func TestAReviewBriefFilledInWithAShipVerdictIsRecorded(t *testing.T) {
	t.Run("a full pair", func(t *testing.T) {
		w := newDeltaWorld(t)
		head := w.fix()
		brief, err := ReviewBrief(w.repo.Root(), head, "")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(brief, reviewedHeadLabel+head+"\n") {
			t.Fatalf("the brief does not open naming the head:\n%s", brief)
		}
		returns := []ReviewReturn{
			{Name: "ruthless.md", Text: fillBrief(t, brief, "Nothing survived.", "SHIP")},
			{Name: "security.md", Text: fillBrief(t, brief, "No finding.", "APPROVE")},
		}
		if _, err := RecordReview(w.repo.Root(), head, returns); err != nil {
			t.Fatalf("a filled brief was refused: %v\n%s", err, returns[0].Text)
		}
	})
	t.Run("a delta", func(t *testing.T) {
		w := newDeltaWorld(t)
		head := w.fix()
		brief, err := ReviewBrief(w.repo.Root(), head, w.base)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(brief, reviewedDeltaLabel+w.base+".."+head+"\n") || strings.Contains(brief, reviewedHeadLabel) {
			t.Fatalf("the brief does not open naming the delta alone:\n%s", brief)
		}
		returns := []ReviewReturn{
			{Name: "ruthless.md", Text: fillBrief(t, brief, "Nothing survived.", "SHIP")},
			{Name: "security.md", Text: fillBrief(t, brief, "No finding.", "APPROVE")},
		}
		if _, err := RecordDeltaReview(w.repo.Root(), head, w.base, returns); err != nil {
			t.Fatalf("a filled delta brief was refused: %v\n%s", err, returns[0].Text)
		}
	})
}

// A brief left unfilled states no verdict, so it records nothing, and a brief
// filled with a verdict that does not pass is refused naming it.
func TestAReviewBriefLeftUnfilledOrFailingIsRefused(t *testing.T) {
	w := newDeltaWorld(t)
	head := w.fix()
	brief, err := ReviewBrief(w.repo.Root(), head, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReadVerdict(brief, head); err == nil {
		t.Fatal("an unfilled brief states a verdict")
	}
	for _, word := range knownVerdicts {
		if !strings.Contains(brief, word) {
			t.Errorf("the brief does not offer the verdict %s", word)
		}
	}
	returns := []ReviewReturn{
		{Name: "ruthless.md", Text: fillBrief(t, brief, "A major finding.", "FIX FIRST")},
		{Name: "security.md", Text: fillBrief(t, brief, "No finding.", "APPROVE")},
	}
	if _, err := RecordReview(w.repo.Root(), head, returns); err == nil || !strings.Contains(err.Error(), "FIX FIRST") {
		t.Fatalf("err = %v; want a refusal naming FIX FIRST", err)
	}
}

// The verdict heading a brief writes is one the reading reads.
func TestAReviewBriefsHeadingIsTheOneRead(t *testing.T) {
	if !verdictHeading.MatchString(verdictHeadingLine) {
		t.Fatalf("%q is not a heading the reading takes", verdictHeadingLine)
	}
}

// A brief names commits the repository holds, by their full names, and a head
// resting on itself is refused.
func TestAReviewBriefRefusesWhatItCannotName(t *testing.T) {
	w := newDeltaWorld(t)
	head := w.fix()
	for name, c := range map[string]struct{ head, base, want string }{
		"an abbreviated head":     {head[:12], "", "full object name"},
		"a head it does not hold": {strings.Repeat("ab", 20), "", "not a commit"},
		"an abbreviated base":     {head, w.base[:12], "full object name"},
		"a base it does not hold": {head, strings.Repeat("ab", 20), "not a commit"},
		"a head on itself":        {head, head, restsOnItself},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ReviewBrief(w.repo.Root(), c.head, c.base); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v; want %q", err, c.want)
			}
		})
	}
}
