package ask

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/term"
)

// longList is a question offering n model names (n a multiple of six): six
// vendors in turn, the third of them claude, each label in title case and each
// value the vendor's lower-case route, so a filter can match a value its
// label does not hold.
func longList(n int) question.Ask {
	vendors := []struct{ route, label string }{
		{"openai/gpt", "GPT"}, {"google/gemini", "Gemini"}, {"anthropic/claude", "Claude"},
		{"meta/llama", "Llama"}, {"mistral/mistral", "Mistral"}, {"qwen/qwen", "Qwen"},
	}
	per := n / len(vendors)
	var choices []question.Option
	for _, v := range vendors {
		for i := 0; i < per; i++ {
			choices = append(choices, question.Option{
				Value: fmt.Sprintf("%s-%d", v.route, i),
				Label: fmt.Sprintf("%s Model %d", v.label, i),
			})
		}
	}
	return question.Ask{Questions: []question.Question{{
		ID:   "model",
		Chip: "Setup Q4",
		Material: []question.Block{{Kind: question.KindParagraph,
			Text: "The service lists the models it offers."}},
		Ask:   "Which model should abcd use?",
		List:  &question.List{Choices: choices},
		Later: question.Option{Value: "later", Label: "Decide later", Meaning: "Leave the model unset."},
	}}}
}

func typeText(p *Picker, s string) {
	for _, r := range s {
		p.Apply(Key{Kind: KeyRune, Rune: r})
	}
}

func press(p *Picker, kinds ...KeyKind) Outcome {
	out := Pending
	for _, k := range kinds {
		out = p.Apply(Key{Kind: k})
	}
	return out
}

// current is the current entry's value: a choice's, or Later's.
func current(p *Picker, a question.Ask) string {
	q := a.Questions[p.Tab()]
	if i := p.Current(); i < len(q.List.Choices) {
		return q.List.Choices[i].Value
	}
	return q.Later.Value
}

// TestLongListNarrowsAsThePersonTypes is B6's state machine
// (spc-2610030911534855): over 300 model names, a pure function of key events
// with no terminal, typing narrows, the arrows move within what is left,
// Backspace widens, Escape clears, a page moves a window, Enter chooses by the
// arrows or by a typed number, and a filter nothing matches says so and keeps
// Later visible.
func TestLongListNarrowsAsThePersonTypes(t *testing.T) {
	a := longList(300)
	choices := a.Questions[0].List.Choices

	t.Run("typing narrows and the arrows move within what is left", func(t *testing.T) {
		p := NewPicker(a)
		p.SetWindow(10)
		if got := len(p.Shown()); got != 300 {
			t.Fatalf("before typing %d shown, want 300", got)
		}
		typeText(p, "claude")
		shown := p.Shown()
		if len(shown) != 50 {
			t.Fatalf("typing claude leaves %d, want the 50 claude names", len(shown))
		}
		for _, i := range shown {
			if !strings.Contains(strings.ToLower(choices[i].Label+choices[i].Value), "claude") {
				t.Errorf("%q is shown under the filter claude", choices[i].Label)
			}
		}
		if p.Current() != shown[0] {
			t.Errorf("narrowing makes the first match current; current is %d", p.Current())
		}
		press(p, KeyDown, KeyDown)
		if p.Current() != shown[2] {
			t.Errorf("Down twice: current %d, want %d", p.Current(), shown[2])
		}
		press(p, KeyUp)
		if p.Current() != shown[1] {
			t.Errorf("Up: current %d, want %d", p.Current(), shown[1])
		}
		press(p, KeyUp, KeyUp)
		if p.Current() != shown[0] {
			t.Errorf("Up past the top holds at the first match; current %d", p.Current())
		}
		_, live := p.Lines(Frame{Width: 80, Mode: term.Mono})
		joined := strings.Join(live, "\n")
		if !strings.Contains(joined, "filter: claude") {
			t.Errorf("no filter: line:\n%s", joined)
		}
		// The numbers are the full list's, so a number read off a narrowed
		// list still chooses what it names: claude's first name is 101.
		if !strings.Contains(joined, "101. Claude Model 0") {
			t.Errorf("the narrowed list does not keep the full list's numbers:\n%s", joined)
		}
		if !strings.Contains(joined, "301. Decide later") {
			t.Errorf("Later is not drawn last under a filter:\n%s", joined)
		}
		if strings.Contains(joined, "GPT") {
			t.Errorf("a name the filter excludes is drawn:\n%s", joined)
		}
	})

	t.Run("a value narrows as a label does", func(t *testing.T) {
		p := NewPicker(a)
		typeText(p, "OpenAI/")
		if got := len(p.Shown()); got != 50 {
			t.Errorf("typing openai/ (in the values alone, any case) leaves %d, want 50", got)
		}
	})

	t.Run("backspace widens and escape clears", func(t *testing.T) {
		p := NewPicker(a)
		typeText(p, "claude model 1")
		if got := len(p.Shown()); got != 11 { // 1 and 10 to 19
			t.Fatalf("claude model 1 leaves %d, want 11", got)
		}
		press(p, KeyDown)
		kept := p.Current()
		press(p, KeyBackspace)
		if got, typed := len(p.Shown()), p.Typed(); got != 50 || typed != "claude model " {
			t.Errorf("Backspace: %d shown under %q, want 50 under %q", got, typed, "claude model ")
		}
		if p.Current() != kept {
			t.Errorf("widening moved the current name from %d to %d", kept, p.Current())
		}
		press(p, KeyEscape)
		if got, typed := len(p.Shown()), p.Typed(); got != 300 || typed != "" {
			t.Errorf("Escape: %d shown under %q, want 300 and no filter", got, typed)
		}
		_, live := p.Lines(Frame{Width: 80, Mode: term.Mono})
		if strings.Contains(strings.Join(live, "\n"), "filter:") {
			t.Error("a cleared filter still draws its line")
		}
	})

	t.Run("a page moves a window and the ends are reached", func(t *testing.T) {
		p := NewPicker(a)
		p.SetWindow(10)
		press(p, KeyPageDown)
		if p.Current() != 10 {
			t.Errorf("Page Down: current %d, want 10", p.Current())
		}
		if p.Above() == 0 || p.Below() == 0 {
			t.Errorf("mid-list: %d above and %d below, want both", p.Above(), p.Below())
		}
		_, live := p.Lines(Frame{Width: 80, Mode: term.Mono})
		joined := strings.Join(live, "\n")
		if !strings.Contains(joined, "more above") || !strings.Contains(joined, "more below") {
			t.Errorf("no more above / more below lines:\n%s", joined)
		}
		press(p, KeyPageUp)
		if p.Current() != 0 {
			t.Errorf("Page Up: current %d, want 0", p.Current())
		}
		press(p, KeyEnd)
		if got := current(p, a); got != "later" {
			t.Errorf("End: current %q, want Later", got)
		}
		press(p, KeyHome)
		if p.Current() != 0 {
			t.Errorf("Home: current %d, want 0", p.Current())
		}
	})

	t.Run("enter chooses by the arrows", func(t *testing.T) {
		p := NewPicker(a)
		typeText(p, "claude")
		want := choices[p.Shown()[1]].Value
		if out := press(p, KeyDown, KeyEnter); out != Done {
			t.Fatalf("Enter: outcome %v, want Done", out)
		}
		got := p.Answers()
		if len(got) != 1 || got[0].ID != "model" || got[0].Value != want || got[0].Later {
			t.Errorf("answers %+v, want model = %q", got, want)
		}
	})

	t.Run("enter chooses by a typed number", func(t *testing.T) {
		p := NewPicker(a)
		typeText(p, "200")
		if got := len(p.Shown()); got != 300 {
			t.Errorf("a number does not narrow: %d shown, want 300", got)
		}
		if out := press(p, KeyEnter); out != Done {
			t.Fatalf("Enter: outcome %v, want Done", out)
		}
		if got := p.Answers(); len(got) != 1 || got[0].Value != choices[199].Value {
			t.Errorf("answers %+v, want number 200 (%q)", got, choices[199].Value)
		}
	})

	t.Run("a number out of range chooses nothing and says so", func(t *testing.T) {
		p := NewPicker(a)
		typeText(p, "999")
		if out := press(p, KeyEnter); out != Pending {
			t.Fatalf("Enter on 999: outcome %v, want Pending", out)
		}
		_, live := p.Lines(Frame{Width: 80, Mode: term.Mono})
		if joined := strings.Join(live, "\n"); !strings.Contains(joined, "no number 999") {
			t.Errorf("the out-of-range number is not named:\n%s", joined)
		}
		press(p, KeyEscape)
		typeText(p, "301")
		if out := press(p, KeyEnter); out != Done {
			t.Fatalf("Enter on 301: outcome %v, want Done", out)
		}
		if got := p.Answers(); len(got) != 1 || got[0].Value != "later" || !got[0].Later {
			t.Errorf("answers %+v, want Later by its number", got)
		}
	})

	t.Run("nothing matching says so and keeps Later", func(t *testing.T) {
		p := NewPicker(a)
		typeText(p, "zzz")
		if got := len(p.Shown()); got != 0 {
			t.Fatalf("zzz leaves %d, want none", got)
		}
		if got := current(p, a); got != "later" {
			t.Errorf("with nothing matching the current entry is %q, want Later", got)
		}
		_, live := p.Lines(Frame{Width: 80, Mode: term.Mono})
		joined := strings.Join(live, "\n")
		if !strings.Contains(joined, `nothing matches "zzz"`) || !strings.Contains(joined, "Decide later") {
			t.Errorf("nothing-matching does not say so and keep Later:\n%s", joined)
		}
		press(p, KeyDown, KeyUp)
		if out := press(p, KeyEnter); out != Done {
			t.Fatalf("Enter: outcome %v, want Done", out)
		}
		if got := p.Answers(); len(got) != 1 || !got[0].Later {
			t.Errorf("answers %+v, want Later", got)
		}
	})

	t.Run("ctrl-c and ctrl-z are outcomes, not choices", func(t *testing.T) {
		p := NewPicker(a)
		typeText(p, "claude")
		if out := press(p, KeyInterrupt); out != Interrupted {
			t.Errorf("Ctrl-C: outcome %v, want Interrupted", out)
		}
		if out := press(p, KeySuspend); out != Suspended {
			t.Errorf("Ctrl-Z: outcome %v, want Suspended", out)
		}
		if got := p.Answers(); len(got) != 0 {
			t.Errorf("an interrupt recorded %+v", got)
		}
	})
}

// TestPickerMovesBetweenTabs holds Left, Right and Tab between the parts of a
// tabbed Ask: Enter answers the part shown and moves to the next unanswered
// one, and the Ask is done when every part is answered.
func TestPickerMovesBetweenTabs(t *testing.T) {
	a := fixture(t, "tabs")
	if len(a.Questions) < 2 {
		t.Fatal("the tabs fixture has one part")
	}
	p := NewPicker(a)
	press(p, KeyRight)
	if p.Tab() != 1 {
		t.Errorf("Right: tab %d, want 1", p.Tab())
	}
	press(p, KeyLeft)
	if p.Tab() != 0 {
		t.Errorf("Left: tab %d, want 0", p.Tab())
	}
	press(p, KeyLeft)
	if p.Tab() != len(a.Questions)-1 {
		t.Errorf("Left from the first part: tab %d, want the last", p.Tab())
	}
	press(p, KeyTab)
	if p.Tab() != 0 {
		t.Errorf("Tab from the last part: tab %d, want 0", p.Tab())
	}
	var out Outcome
	for i := range a.Questions {
		if p.Tab() != i {
			t.Fatalf("after answering %d parts the tab is %d", i, p.Tab())
		}
		out = press(p, KeyEnter)
	}
	if out != Done {
		t.Fatalf("every part answered: outcome %v, want Done", out)
	}
	var ids []string
	for _, ans := range p.Answers() {
		ids = append(ids, ans.ID)
	}
	var want []string
	for _, q := range a.Questions {
		want = append(want, q.ID)
	}
	if !reflect.DeepEqual(ids, want) {
		t.Errorf("answers for %v, want %v", ids, want)
	}
}

// TestPickerDrawsWhatTheLayoutDraws holds the answer loop's frame to step 1's
// drawing: with nothing typed and a window that holds every option, the head
// and the live region are the layout's lines, so the arrow-key list and the
// golden layouts cannot drift apart.
func TestPickerDrawsWhatTheLayoutDraws(t *testing.T) {
	for _, name := range fixtures(t) {
		a := fixture(t, name)
		for _, w := range []int{80, 160} {
			p := NewPicker(a)
			p.SetWindow(1000)
			head, live := p.Lines(Frame{Width: w, Mode: term.Mono})
			got := append(append([]string(nil), head...), live...)
			want := Layout(a, w, term.Mono)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s at %d: the picker draws\n%s\nthe layout draws\n%s",
					name, w, strings.Join(got, "\n"), strings.Join(want, "\n"))
			}
		}
	}
}

// TestDecodeKeys holds the key decoder the raw loop reads through: the arrow,
// page and end sequences in both cursor modes, Enter as CR or LF, Backspace as
// DEL or BS, a lone ESC at the end of a read as Escape, the control bytes, a
// rune split across reads kept for the next, and an unknown sequence dropped
// whole.
func TestDecodeKeys(t *testing.T) {
	keys, rest := DecodeKeys([]byte("\x1b[A\x1bOB\x1b[5~\x1b[6~\x1b[H\x1b[F\x1b[1~\x1b[4~\r\n\x7f\x08\x1b[C\x1b[D\t\x1b[Z\x03\x1a\x1b[99;5uaé"))
	want := []Key{
		{Kind: KeyUp}, {Kind: KeyDown}, {Kind: KeyPageUp}, {Kind: KeyPageDown},
		{Kind: KeyHome}, {Kind: KeyEnd}, {Kind: KeyHome}, {Kind: KeyEnd},
		{Kind: KeyEnter}, {Kind: KeyEnter}, {Kind: KeyBackspace}, {Kind: KeyBackspace},
		{Kind: KeyRight}, {Kind: KeyLeft}, {Kind: KeyTab}, {Kind: KeyLeft},
		{Kind: KeyInterrupt}, {Kind: KeySuspend},
		{Kind: KeyRune, Rune: 'a'}, {Kind: KeyRune, Rune: 'é'},
	}
	if !reflect.DeepEqual(keys, want) || len(rest) != 0 {
		t.Errorf("decoded %v rest %q, want %v", keys, rest, want)
	}
	keys, rest = DecodeKeys([]byte("x\x1b"))
	if !reflect.DeepEqual(keys, []Key{{Kind: KeyRune, Rune: 'x'}, {Kind: KeyEscape}}) || len(rest) != 0 {
		t.Errorf("a lone ESC: %v rest %q", keys, rest)
	}
	keys, rest = DecodeKeys([]byte("y\xc3"))
	if !reflect.DeepEqual(keys, []Key{{Kind: KeyRune, Rune: 'y'}}) || string(rest) != "\xc3" {
		t.Errorf("a split rune: %v rest %q, want the lead byte kept", keys, rest)
	}
}
