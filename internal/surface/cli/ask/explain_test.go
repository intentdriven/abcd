package ask

import (
	"bytes"
	"errors"
	"testing"

	"github.com/intentdriven/abcd/internal/term"
)

// TestQuestionMarkAsksForTheExplanation is itd-2610240823055889 criterion 6:
// a setup question is shown on one line, and answering ? asks for the full
// explanation, which the front door then shows before asking again. In both
// modes ? alone is the request; in the numbered reader it is a whole line, in
// the arrow-key list the first key typed.
func TestQuestionMarkAsksForTheExplanation(t *testing.T) {
	t.Run("numbered", func(t *testing.T) {
		var out bytes.Buffer
		tm := Terminal{In: pipeIn(t, "?\n"), Out: &out, Getenv: env("TERM", "xterm"), Mode: term.Mono,
			Roots: roots(t, numberedSetting, ""), Explain: true}
		if _, err := tm.Put(fixture(t, "key-home")); !errors.Is(err, ErrExplain) {
			t.Fatalf("a ? line returned %v, want ErrExplain", err)
		}
	})
	t.Run("numbered, not offered", func(t *testing.T) {
		var out bytes.Buffer
		tm := Terminal{In: pipeIn(t, "?\n2\n"), Out: &out, Getenv: env("TERM", "xterm"), Mode: term.Mono,
			Roots: roots(t, numberedSetting, "")}
		got, err := tm.Put(fixture(t, "key-home"))
		if err != nil || len(got) != 1 || got[0].Value != "file" {
			t.Fatalf("without Explain a ? narrows as before and the next line answers: %+v, %v", got, err)
		}
	})
	t.Run("arrows", func(t *testing.T) {
		p := NewPicker(fixture(t, "key-home"))
		p.AllowExplain()
		if got := p.Apply(Key{Kind: KeyRune, Rune: '?'}); got != Explain {
			t.Fatalf("? as the first key = %v, want Explain", got)
		}
		p = NewPicker(fixture(t, "key-home"))
		p.AllowExplain()
		p.Apply(Key{Kind: KeyRune, Rune: 'f'})
		if got := p.Apply(Key{Kind: KeyRune, Rune: '?'}); got != Pending {
			t.Fatalf("? after other typing = %v, want Pending (it narrows)", got)
		}
		p = NewPicker(fixture(t, "key-home"))
		if got := p.Apply(Key{Kind: KeyRune, Rune: '?'}); got != Pending {
			t.Fatalf("? on a picker that offers no explanation = %v, want Pending", got)
		}
	})
}
