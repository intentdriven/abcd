package source

import (
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
)

// TestClassForTakesOnlyThePersonsTwoAnswers: private is stored as the
// confidential class and public as public; anything else, deciding later
// included, resolves to no class, so nothing here files an item by default.
func TestClassForTakesOnlyThePersonsTwoAnswers(t *testing.T) {
	for answer, want := range map[string]string{AnswerPrivate: ClassConfidential, AnswerPublic: ClassPublic} {
		if got, ok := ClassFor(answer); !ok || got != want {
			t.Errorf("ClassFor(%q) = %q, %v; want %q", answer, got, ok, want)
		}
	}
	for _, answer := range []string{"", AnswerLater, ClassConfidential, "Private", "yes"} {
		if got, ok := ClassFor(answer); ok {
			t.Errorf("ClassFor(%q) = %q; want no class", answer, got)
		}
	}
}

// TestClassQuestionIsWellFormed: the class question passes the structural
// check every drawn question passes, and its answers are exactly the two
// ClassFor takes, with deciding later beside them.
func TestClassQuestionIsWellFormed(t *testing.T) {
	q := ClassQuestion()
	if fs := question.Check(question.Ask{Questions: []question.Question{q}}); len(fs) > 0 {
		t.Fatalf("the class question fails the structural check: %+v", fs)
	}
	if len(q.Options) != 2 || q.Options[0].Value != AnswerPrivate || q.Options[1].Value != AnswerPublic || q.Later.Value != AnswerLater {
		t.Errorf("the class question's answers = %+v, later %+v", q.Options, q.Later)
	}
}
