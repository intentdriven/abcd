package ask

import (
	"bytes"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/term"
)

// The typed part in the Terminal (spc-2610031241482088, "The typed part on
// the question type"): drawn as one line after the options, "or type:
// <prompt>"; the numbered reader takes a line that is not a number as the
// typed answer, and the arrow-key list takes the typed text on Enter when no
// option matches it.

// typedAsk is a question with two options, decide later and a typed part.
func typedAsk() question.Ask {
	return question.Ask{Questions: []question.Question{{
		ID:   "model",
		Chip: "Setup Q3",
		Material: []question.Block{{Kind: question.KindParagraph,
			Text: "api.example.com lists 300 models."}},
		Ask: "Which model should abcd set up?",
		Options: []question.Option{
			{Value: "vendor/coder-large", Label: "vendor/coder-large", Meaning: "A model you already use."},
			{Value: "vendor/coder-small", Label: "vendor/coder-small", Meaning: "A model you already use."},
		},
		Typed:       "Type part of a model's name",
		Later:       question.Option{Value: "later", Label: "Decide later", Meaning: "Nothing is set up."},
		Now:         "no model is chosen",
		ChangeLater: "run the guide again",
	}}}
}

// TestTypedPartDrawsAsALine: the typed part is one line after the options,
// decide later included, and before the Now: line; a question without one
// draws no such line; the prompt is sanitised as every part is.
func TestTypedPartDrawsAsALine(t *testing.T) {
	a := typedAsk()
	a.Questions[0].Typed = "Type part of a\x1b[31m name"
	lines := Layout(a, 80, term.Mono)
	later, typed, now := -1, -1, -1
	for i, l := range lines {
		switch {
		case strings.Contains(l, "Decide later"):
			later = i
		case strings.HasPrefix(l, "  or type: "):
			typed = i
		case strings.Contains(l, "Now:"):
			now = i
		}
	}
	if typed < 0 || later < 0 || now < 0 || !(later < typed && typed < now) {
		t.Fatalf("the typed line is not drawn after the options and before Now: (later %d, typed %d, now %d):\n%s",
			later, typed, now, strings.Join(lines, "\n"))
	}
	if strings.ContainsRune(lines[typed], 0x1b) || !strings.Contains(lines[typed], "Type part of a?[31m name") {
		t.Errorf("the typed prompt is not sanitised: %q", lines[typed])
	}
	b := typedAsk()
	b.Questions[0].Typed = ""
	for _, l := range Layout(b, 80, term.Mono) {
		if strings.Contains(l, "or type:") {
			t.Errorf("a question without a typed part draws %q", l)
		}
	}
}

// TestNumberedReaderTakesTypedText: in the numbered reader a number still
// chooses, and a line that is not a number is the typed answer, kept as
// typed, never a filter.
func TestNumberedReaderTakesTypedText(t *testing.T) {
	for _, tc := range []struct {
		in, value string
		typed     bool
	}{
		{"2\n", "vendor/coder-small", false},
		{"  coder \n", "coder", true},
		{"https://api.example.com/v1\n", "https://api.example.com/v1", true},
	} {
		var out bytes.Buffer
		tm := Terminal{In: pipeIn(t, tc.in), Out: &out, Getenv: env("TERM", "xterm"), Mode: term.Mono, Roots: roots(t, numberedSetting, "")}
		got, err := tm.Put(typedAsk())
		if err != nil {
			t.Fatalf("%q: %v\n%s", tc.in, err, out.String())
		}
		if len(got) != 1 || got[0].Value != tc.value || got[0].Typed != tc.typed || got[0].Later {
			t.Errorf("%q: answers %+v, want %q (typed %v)", tc.in, got, tc.value, tc.typed)
		}
		if !strings.Contains(out.String(), "or type: Type part of a model's name") {
			t.Errorf("%q: the typed line is not drawn:\n%s", tc.in, out.String())
		}
	}
}

// TestPickerTakesTypedTextWhenNothingMatches: in the arrow-key list typing
// narrows as ever, Enter on a match chooses it, and Enter on text no option
// matches is the typed answer; a question without a typed part keeps decide
// later current there instead.
func TestPickerTakesTypedTextWhenNothingMatches(t *testing.T) {
	p := NewPicker(typedAsk())
	typeText(p, "small")
	if out := press(p, KeyEnter); out != Done {
		t.Fatalf("Enter on a match = %v", out)
	}
	if got := p.Answers(); len(got) != 1 || got[0].Value != "vendor/coder-small" || got[0].Typed {
		t.Fatalf("Enter on a match answered %+v", got)
	}

	p = NewPicker(typedAsk())
	typeText(p, "mistral-7b")
	if out := press(p, KeyEnter); out != Done {
		t.Fatalf("Enter on unmatched text = %v", out)
	}
	if got := p.Answers(); len(got) != 1 || got[0].Value != "mistral-7b" || !got[0].Typed || got[0].Later {
		t.Fatalf("Enter on unmatched text answered %+v, want the typed text", got)
	}
	if c := p.Collapsed(Frame{Mode: term.Mono}); len(c) != 1 || !strings.HasSuffix(c[0], ": mistral-7b") {
		t.Errorf("collapsed = %q", c)
	}

	plain := typedAsk()
	plain.Questions[0].Typed = ""
	p = NewPicker(plain)
	typeText(p, "mistral-7b")
	press(p, KeyEnter)
	if got := p.Answers(); len(got) != 1 || !got[0].Later || got[0].Typed {
		t.Fatalf("without a typed part, Enter on unmatched text answered %+v, want decide later", got)
	}
}
