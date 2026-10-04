package question_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
)

// The typed part (spc-2610031241482088, "The typed part on the question
// type"; open question 1, decided (a)): a question may take a typed answer
// besides its options. It counts as one option toward the asking limits'
// floor, so a question whose only listed answer is the way to decide later is
// still a whole question, and the host's free-text row is where it is typed.

// typedQuestion is the guided connect's address question: a typed part and
// decide later, no listed option.
func typedQuestion() question.Question {
	return question.Question{
		ID:   "address",
		Chip: "Setup Q1",
		Material: []question.Block{{Kind: question.KindParagraph,
			Text: "abcd connects to a model service at the address the service gives for its OpenAI-compatible list of models."}},
		Ask:         "What is the service's address?",
		Typed:       "Type the address, starting https://, or http:// for a server on this machine",
		Later:       question.Option{Value: "later", Label: "Decide later", Meaning: "Nothing is set up. Run the guide again to pick up here."},
		Now:         "no service is connected under this name",
		ChangeLater: "run the guide again",
	}
}

// TestTypedPartIsAnAnswer: the structural check admits a question whose
// answers are a typed part and decide later, the typed part round-trips
// under the key "typed", and the field view carries it so the limits count
// it as one option toward their floor of two.
func TestTypedPartIsAnAnswer(t *testing.T) {
	a := one(typedQuestion())
	if fs := question.Check(a); len(fs) != 0 {
		t.Fatalf("a question with a typed part and no options is refused:\n%s", listFindings(fs))
	}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"typed":"Type the address`) {
		t.Fatalf("the typed part is not under the key typed: %s", b)
	}
	f := a.Fields()
	if got := f.Tabs[0].Typed; got != typedQuestion().Typed {
		t.Fatalf("the field view's typed part = %q", got)
	}
	if fs := question.CheckLimits(f, question.Default, question.Addressee{}); len(fs) != 0 {
		t.Fatalf("the typed part is not counted toward the floor:\n%s", listFindings(fs))
	}

	// Without the typed part the same question is one option short, and
	// both checks say so: the typed part is what makes it whole.
	bare := typedQuestion()
	bare.Typed = ""
	if fs := question.Check(one(bare)); !findingAt(fs, 1, "options") {
		t.Errorf("a question with no options, no list and no typed part is admitted:\n%s", listFindings(fs))
	}
	if fs := question.CheckLimits(one(bare).Fields(), question.Default, question.Addressee{}); !hasRule(fs, question.RuleOptions) {
		t.Errorf("one option without a typed part is admitted by the limits:\n%s", listFindings(fs))
	}
}

// TestTypedPartCountsOnlyTowardTheFloor: the typed part is the host's
// free-text row, never one of the four options it lists, so a question with
// four options and a typed part is within the limits, and five are not.
func TestTypedPartCountsOnlyTowardTheFloor(t *testing.T) {
	q := typedQuestion()
	for _, v := range []string{"a", "b", "c"} {
		q.Options = append(q.Options, question.Option{Value: v, Label: "Model " + v, Meaning: "This model."})
	}
	if fs := question.CheckLimits(one(q).Fields(), question.Default, question.Addressee{}); len(fs) != 0 {
		t.Fatalf("three options, decide later and a typed part are refused:\n%s", listFindings(fs))
	}
	q.Options = append(q.Options, question.Option{Value: "d", Label: "Model d", Meaning: "This model."})
	if fs := question.CheckLimits(one(q).Fields(), question.Default, question.Addressee{}); !hasRule(fs, question.RuleOptions) {
		t.Fatalf("four options and decide later are admitted because of a typed part:\n%s", listFindings(fs))
	}
}
