package interview

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/question"
)

// TestRepairMendsWhatTheRefusingRulesAllow: the repair a question gets when
// it is shown at the redrafts bound clears every finding of the rules it may
// mend mechanically (the decide-later option, bold and recommendation marks,
// over-long labels), keeps the parts those rules do not reach, and leaves the
// question it was given unchanged.
func TestRepairMendsWhatTheRefusingRulesAllow(t *testing.T) {
	mended := []question.Rule{question.RuleDecideLater, question.RuleNoBold, question.RuleNeverRecommended, question.RuleLabelWords}
	q := sampleQuestion(1)
	q.Material[0].Text = "The **thing** being decided, quoted in full, with one __concrete__ example."
	q.Material[1].Items = []string{"a **bold** list item"}
	q.Ask = "The one **plain** question?"
	q.Options = []question.Option{
		{Value: "a", Label: "★ A few words (Recommended)", Meaning: "What choosing it **means**, with its gain and its cost."},
		{Value: "b", Label: "Another answer that runs on for far too many words", Meaning: "What choosing it means, with its gain and its cost."},
		{Value: "c", Label: "Decide later", Meaning: "A decide-later option in the wrong place."},
	}
	q.Later = question.Option{Value: "a", Label: "Maybe", Meaning: ""}
	a := question.Ask{Questions: []question.Question{q}}
	before := question.Ask{Questions: []question.Question{q}}
	before.Questions[0].Options = slices.Clone(q.Options)

	broken := map[question.Rule]bool{}
	for _, f := range question.CheckLimits(a.Fields(), question.Default, question.Addressee{}) {
		broken[f.Rule] = true
	}
	for _, r := range mended {
		if !broken[r] {
			t.Fatalf("the fixture does not break %s", r)
		}
	}
	if len(question.Check(a)) == 0 {
		t.Fatal("the fixture passes the structural check; its decide-later value should collide")
	}

	got := repair(a, question.Default)
	if fs := question.Check(got); len(fs) != 0 {
		t.Fatalf("the repaired question fails the structural check: %v", fs)
	}
	for _, f := range question.CheckLimits(got.Fields(), question.Default, question.Addressee{}) {
		if slices.Contains(mended, f.Rule) {
			t.Errorf("still broken after the repair: %s", f)
		}
	}
	r := got.Questions[0]
	if len(r.Options) != 2 || r.Options[0].Label != "A few words" || r.Options[1].Label != "Another answer that runs on" {
		t.Errorf("options %+v", r.Options)
	}
	if r.Later.Value != "later" || r.Later.Label != question.Default.LaterLabels[0] || r.Later.Meaning == "" {
		t.Errorf("later %+v", r.Later)
	}
	if r.Ask != "The one plain question?" || strings.Contains(r.Material[0].Text, "*") || strings.Contains(r.Material[0].Text, "__") ||
		r.Material[1].Items[0] != "a bold list item" || r.Chip != q.Chip || r.Now != q.Now {
		t.Errorf("repaired %+v", r)
	}
	if !reflect.DeepEqual(a.Questions[0].Options, before.Questions[0].Options) || !strings.Contains(a.Questions[0].Ask, "**") ||
		!strings.Contains(a.Questions[0].Material[1].Items[0], "**") {
		t.Error("the repair changed the question it was given")
	}
}

// TestTheRepairedQuestionIsSanitised: the repair's result passes
// question.Safe, so a mended part can never carry what the screen must not.
func TestTheRepairedQuestionIsSanitised(t *testing.T) {
	q := sampleQuestion(1)
	q.Options[0].Label = "**\x1b[2J**Wipe"
	got := repair(question.Ask{Questions: []question.Question{q}}, question.Default)
	if l := got.Questions[0].Options[0].Label; strings.ContainsRune(l, 0x1b) || l != "?[2JWipe" {
		t.Fatalf("label %q", l)
	}
	if !reflect.DeepEqual(got, question.Safe(got)) {
		t.Fatal("the repaired question is not what question.Safe leaves")
	}
}

// TestRepairLabelCutsTheMarkFromTheLabelItself: the recommendation mark is
// found and cut in the label it is cut from, so a rune whose lowercase has a
// different length in bytes (U+023A grows, U+0130 shrinks) neither panics the
// repair nor shifts the cut onto the wrong bytes.
func TestRepairLabelCutsTheMarkFromTheLabelItself(t *testing.T) {
	for label, want := range map[string]string{
		"Ⱥ (Recommended)":              "Ⱥ",
		"Ⱥ(Recommended)":               "Ⱥ",
		"İzmir (Recommended)":          "İzmir",
		"ẞ Ω K (RECOMMENDED) Ⱦ":        "ẞ Ω K Ⱦ",
		"Ⱥ (recommended)(Recommended)": "Ⱥ",
		"(Recommended":                 "(Recommended",
	} {
		got := repairLabel(label, "", question.Default)
		if got != want {
			t.Errorf("repairLabel(%q) = %q, want %q", label, got, want)
		}
		if f := question.CheckLimits(question.Ask{Questions: []question.Question{{Options: []question.Option{{Value: "a", Label: got}}}}}.Fields(), question.Default, question.Addressee{}); slices.ContainsFunc(f, func(f question.Finding) bool { return f.Rule == question.RuleNeverRecommended }) {
			t.Errorf("repairLabel(%q) = %q still reads as recommended", label, got)
		}
	}
}

// TestOnlyASCIILowercasesToTheMark: no rune outside ASCII lowercases to a
// letter of the recommendation mark, so the ASCII match cutRecommended makes
// on the label finds every mark the check's strings.ToLower search finds.
func TestOnlyASCIILowercasesToTheMark(t *testing.T) {
	for r := rune(utf8.RuneSelf); r <= unicode.MaxRune; r++ {
		if l := unicode.ToLower(r); l < utf8.RuneSelf && strings.ContainsRune(recommendedMark, l) {
			t.Errorf("U+%04X lowercases to %q", r, l)
		}
	}
}

// TestTheEndAtTheBoundIsPlain: the message an interview ends with at the
// bound says what happened and the command to run next, in the person's
// words, and counts what it names.
func TestTheEndAtTheBoundIsPlain(t *testing.T) {
	w := &Written{Name: Planning, Verb: "abcd intent interview", Target: "itd-1\x1b[2J"}
	for _, c := range []struct {
		err  *RedraftsSpentError
		want []string
	}{
		{w.spent(true, 3, 2), []string{"the planning interview has ended", "closing summary", "even after 3 more attempts", "The 2 answers you gave are kept.", "`abcd intent interview itd-1?[2J`"}},
		{w.spent(false, 1, 1), []string{"next question", "even after 1 more attempt.", "The 1 answer you gave is kept."}},
		{w.spent(false, 0, 0), []string{"next question in a form abcd can show.", "No answer had been given yet."}},
	} {
		msg := c.err.Error()
		for _, want := range c.want {
			if !strings.Contains(msg, want) {
				t.Errorf("%q does not say %q", msg, want)
			}
		}
		if strings.ContainsRune(msg, 0x1b) {
			t.Errorf("%q carries an escape", msg)
		}
	}
}
