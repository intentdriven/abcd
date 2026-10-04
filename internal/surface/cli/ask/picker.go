package ask

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Outcome is what a key leaves the list waiting on.
type Outcome int

const (
	// Pending: the list waits for more keys.
	Pending Outcome = iota
	// Done: every part of the Ask is answered.
	Done
	// Interrupted: Ctrl-C. Nothing is recorded for the Ask.
	Interrupted
	// Suspended: Ctrl-Z. The loop stops the process and redraws on return.
	Suspended
)

func (o Outcome) String() string {
	switch o {
	case Pending:
		return "Pending"
	case Done:
		return "Done"
	case Interrupted:
		return "Interrupted"
	case Suspended:
		return "Suspended"
	}
	return fmt.Sprintf("Outcome(%d)", int(o))
}

// Answer is one part's answer: the question's id and the value chosen, as the
// question carried them, with the label drawn for it.
type Answer struct {
	ID    string
	Value string
	Label string
	// Later is true when the way to decide later was chosen.
	Later bool
}

// defaultWindow is the list's window before the loop sizes it to the rows.
const defaultWindow = 10

// minWindow is the fewest entries the arrow-key list shows at once
// (spc-2610030911534855, "at least five").
const minWindow = 5

// Picker is the arrow-key list's state (spc-2610030911534855, "The answer
// loop: Arrow keys first, typing to narrow"): a pure function of key events,
// with no terminal, so every behaviour is a test of keys in and state out.
//
// Up and Down move the current entry; Page Up and Page Down move a window;
// Home and End go to the ends, Later being the last. A printable character
// narrows: the typed text is matched by question.Matches against each
// answer's label and value, and Backspace widens, Escape clears. Text that is
// all digits is a number instead: it narrows nothing, and Enter chooses the
// entry with that number, which is the entry's place in the full list, so a
// number read off a narrowed list chooses what it names. Enter otherwise
// chooses the current entry. Left, Right and Tab move between the parts of a
// tabbed Ask, and the Ask is done when every part is answered. Later is never
// narrowed away: when nothing matches, it is the current entry.
type Picker struct {
	orig   question.Ask // as the caller gave it: what an answer records
	ask    question.Ask // sanitised: what is drawn and narrowed
	parts  []*part
	tab    int
	window int
}

// part is one question's list state.
type part struct {
	all    []question.Option // the answers, Later excluded
	typed  string
	shown  []int // indices into all the filter keeps
	cur    int   // index into shown; len(shown) is Later
	top    int   // the first shown entry in the window
	chosen int   // -1 unanswered; an index into all, len(all) for Later
	note   string
}

// NewPicker is the list for a, its first part shown, each part's first entry
// current, nothing typed.
func NewPicker(a question.Ask) *Picker {
	p := &Picker{orig: a, ask: Safe(a), window: defaultWindow}
	for _, q := range p.ask.Questions {
		all := entries(q)
		all = all[:len(all)-1]
		pt := &part{all: all, chosen: -1}
		pt.shown = span(len(all))
		p.parts = append(p.parts, pt)
	}
	return p
}

// SetWindow sets how many entries the list shows at once, at least five.
func (p *Picker) SetWindow(n int) {
	p.window = max(n, minWindow)
	for _, pt := range p.parts {
		pt.scroll(p.window)
	}
}

// Tab is the part shown, counted from zero.
func (p *Picker) Tab() int { return p.tab }

// Typed is the text typed into the part shown.
func (p *Picker) Typed() string { return p.cur().typed }

// Shown is the answers the filter keeps, as indices into the part's options
// (a long list's choices); Later is never among them and always offered.
func (p *Picker) Shown() []int { return append([]int(nil), p.cur().shown...) }

// Current is the current entry, as an index into the part's options; the
// number of options is Later.
func (p *Picker) Current() int { return p.cur().current() }

// Above and Below are how many kept answers lie above and below the window.
func (p *Picker) Above() int { return p.cur().top }
func (p *Picker) Below() int {
	pt := p.cur()
	return max(0, len(pt.shown)-(pt.top+p.window))
}

func (p *Picker) cur() *part {
	if len(p.parts) == 0 {
		return &part{chosen: -1}
	}
	return p.parts[p.tab]
}

// Apply takes one key and returns what the list then waits on.
func (p *Picker) Apply(k Key) Outcome {
	if len(p.parts) == 0 {
		return Done
	}
	pt := p.cur()
	switch k.Kind {
	case KeyInterrupt:
		return Interrupted
	case KeySuspend:
		return Suspended
	case KeyLeft:
		p.tab = (p.tab + len(p.parts) - 1) % len(p.parts)
	case KeyRight, KeyTab:
		p.tab = (p.tab + 1) % len(p.parts)
	case KeyUp:
		pt.cur = max(0, pt.cur-1)
	case KeyDown:
		pt.cur = min(len(pt.shown), pt.cur+1)
	case KeyPageUp:
		pt.cur = max(0, pt.cur-p.window)
	case KeyPageDown:
		pt.cur = min(len(pt.shown), pt.cur+p.window)
	case KeyHome:
		pt.cur = 0
	case KeyEnd:
		pt.cur = len(pt.shown)
	case KeyRune:
		pt.typed += string(k.Rune)
		pt.refilter(!isNumber(pt.typed))
	case KeyBackspace:
		if pt.typed != "" {
			r := []rune(pt.typed)
			pt.typed = string(r[:len(r)-1])
			pt.refilter(false)
		}
	case KeyEscape:
		pt.typed = ""
		pt.refilter(false)
	case KeyEnter:
		return p.enter(pt)
	}
	pt.scroll(p.window)
	return Pending
}

// enter chooses for the part shown: the typed number's entry, or the current
// one. A number out of range chooses nothing and says so.
func (p *Picker) enter(pt *part) Outcome {
	choice := pt.current()
	if isNumber(pt.typed) {
		n, err := strconv.Atoi(pt.typed)
		if err != nil || n < 1 || n > len(pt.all)+1 {
			pt.note = fmt.Sprintf("there is no number %s; the numbers run 1 to %d", pt.typed, len(pt.all)+1)
			return Pending
		}
		choice = n - 1
	}
	pt.chosen = choice
	pt.note = ""
	for i := 1; i <= len(p.parts); i++ {
		next := (p.tab + i) % len(p.parts)
		if p.parts[next].chosen < 0 {
			p.tab = next
			return Pending
		}
	}
	return Done
}

// Answers is every answered part's answer, in the Ask's order.
func (p *Picker) Answers() []Answer {
	var out []Answer
	for i, pt := range p.parts {
		if pt.chosen < 0 {
			continue
		}
		q, safe := p.orig.Questions[i], p.ask.Questions[i]
		all, safeAll := entries(q), entries(safe)
		out = append(out, Answer{
			ID:    q.ID,
			Value: all[pt.chosen].Value,
			Label: safeAll[pt.chosen].Label,
			Later: pt.chosen == len(pt.all),
		})
	}
	return out
}

// current is the current entry as an index into all; len(all) is Later.
func (pt *part) current() int {
	if pt.cur >= len(pt.shown) {
		return len(pt.all)
	}
	return pt.shown[pt.cur]
}

// refilter recomputes what the typed text keeps. Narrowing makes the first
// match current; widening, clearing and a number keep the current entry where
// it is still kept.
func (pt *part) refilter(narrowing bool) {
	prev := pt.current()
	filter := pt.typed
	if isNumber(filter) {
		filter = ""
	}
	pt.shown = narrow(filter, pt.all)
	pt.note = ""
	pt.top = 0
	if narrowing {
		pt.cur = 0
		return
	}
	pt.cur = len(pt.shown)
	if prev == len(pt.all) {
		return
	}
	pt.cur = 0
	for i, idx := range pt.shown {
		if idx == prev {
			pt.cur = i
			break
		}
	}
}

// scroll keeps the current entry inside the window of size w.
func (pt *part) scroll(w int) {
	if pt.cur < len(pt.shown) {
		if pt.cur < pt.top {
			pt.top = pt.cur
		}
		if pt.cur >= pt.top+w {
			pt.top = pt.cur - w + 1
		}
	}
	pt.top = max(0, min(pt.top, len(pt.shown)-w))
}

// narrow returns the indices of the options filter keeps, in order, by the
// one narrowing rule, question.Matches, which guided connect's session shares.
func narrow(filter string, opts []question.Option) []int {
	out := make([]int, 0, len(opts))
	for i, o := range opts {
		if question.Matches(filter, o) {
			out = append(out, i)
		}
	}
	return out
}

// isNumber reports whether s is a number to choose by: one or more digits.
func isNumber(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// Lines draws the list in the frame f: head is the part above the answers
// (the chip line, the material and the ask, then a blank line), drawn once
// per part; live is the rest, redrawn as keys arrive: the filter line, the
// window of kept answers with how many lie above and below, Later, and the
// Now: and Change later: lines. f's Tab and Current are the picker's.
func (p *Picker) Lines(f Frame) (head, live []string) {
	if len(p.parts) == 0 {
		return nil, nil
	}
	f.Tab, f.Current = p.tab, p.Current()
	d := newDrawer(f)
	q, pt := p.ask.Questions[p.tab], p.cur()
	head = append(d.head(p.ask, p.tab), "")

	all := entries(q)
	pad := strings.Repeat(" ", indent)
	switch {
	case isNumber(pt.typed):
		live = append(live, pad+"number: "+pt.typed)
	case pt.typed != "":
		live = append(live, pad+"filter: "+termsafe.Sanitize(pt.typed))
	}
	if pt.note != "" {
		live = append(live, pad+pt.note)
	}
	if len(pt.shown) == 0 {
		live = append(live, pad+fmt.Sprintf("nothing matches %q", termsafe.Sanitize(pt.typed)))
	}
	if pt.top > 0 {
		live = append(live, pad+fmt.Sprintf("%d more above", pt.top))
	}
	end := min(len(pt.shown), pt.top+p.window)
	live = append(live, d.optionLines(all, pt.shown[pt.top:end], f.Current)...)
	if below := len(pt.shown) - end; below > 0 {
		live = append(live, pad+fmt.Sprintf("%d more below", below))
	}
	live = append(live, d.optionLines(all, []int{len(all) - 1}, f.Current)...)
	if state := d.state(q); len(state) > 0 {
		live = append(live, "")
		live = append(live, state...)
	}
	return head, live
}

// Collapsed is what stays on screen once the Ask is answered: one plain line
// per part, "› <chip>: <label>".
func (p *Picker) Collapsed(f Frame) []string {
	d := newDrawer(f)
	var out []string
	for i, pt := range p.parts {
		if pt.chosen < 0 {
			continue
		}
		q := p.ask.Questions[i]
		out = append(out, d.glyph+" "+q.Chip+": "+entries(q)[pt.chosen].Label)
	}
	return out
}
