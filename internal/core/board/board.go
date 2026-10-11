// Package board is the status board's one renderer (spc-2610031844142274): two
// views, the product thinker's and the facilitator's, each in two forms, text
// for a Terminal or a pipe and a markdown list a host session pastes unchanged.
//
// It takes the board's data and returns lines. It never writes to a stream and
// never reads the environment: the front door reads the window, the colour rung
// and the locale and hands them in a Frame, so every view and form is a golden
// test. It holds its own colour rungs rather than importing internal/term,
// which carries raw-mode terminal code the transport-agnostic core does not
// depend on (spc-2610030944505997, open question 2).
package board

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/statusblock"
	"github.com/intentdriven/abcd/internal/core/statusline"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/intentdriven/abcd/internal/textwidth"
)

// View is whose view of the board is drawn.
type View int

const (
	// Product is the product thinker's view, the default: what is being
	// built, the next three things, and how many more wait.
	Product View = iota
	// Facilitator is the facilitator's view: every row of the full board, with
	// each planned row's spec and a lane's in-flight mark.
	Facilitator
)

// Form is how the lines are drawn.
type Form int

const (
	// Text is for a Terminal or a pipe: fitted or wrapped to the window.
	Text Form = iota
	// Markdown is a list for a host session to paste unchanged: no escape
	// byte, no fitted title, every line the label, a blank or a list item.
	Markdown
)

// Rung is how much colour the Terminal shows, the front door's reading of the
// colour ladder (term.ResolveColorMode) in the core's own terms.
type Rung int

const (
	// Mono paints nothing: a pipe, NO_COLOR, --no-color, a dumb terminal.
	Mono Rung = iota
	// Ansi16 paints the sixteen colours.
	Ansi16
	// Ansi256 paints the 256-colour set.
	Ansi256
	// TrueColor paints 24-bit colour, the only rung the view label is painted at.
	TrueColor
)

// Frame is what the front door read about where the board is drawn.
type Frame struct {
	View  View
	Form  Form
	Width int  // the window's columns; ignored by Markdown
	Rung  Rung // Mono in a pipe, under NO_COLOR and --no-color
	ASCII bool // no UTF-8 locale: the plain-text symbols and ellipsis
}

// Row is one of the facilitator's labelled rows, already worded by the front
// door (the presence, peers, inbox, oracle and reviews readers live there):
// a label and its value, and the lines nested one level under it.
type Row struct {
	Label string
	Text  string
	Items []string
}

// Input is everything the views draw from.
type Input struct {
	Dir    string
	Status *statusblock.Block
	Rows   []Row
	// Version is the installed version, the same core.VersionInfo
	// `abcd --version` reports; it is drawn as the board's last line in both
	// views and both forms (spc-2610100613109045, decision 4), and the line is
	// left out only when no version is handed in.
	Version core.VersionInfo
}

// productMeasure caps how wide a title runs in the product thinker's view: the
// window, up to 100 columns (open question 4).
const productMeasure = 100

// label is the first line of a view: whose view it is.
func label(v View) string {
	if v == Facilitator {
		return "view for the facilitator"
	}
	return "view for the product thinker"
}

// Render draws the board in the frame's view and form.
func Render(in Input, f Frame) []string {
	name := label(f.View)
	head := name
	if f.Form == Text && f.Rung == TrueColor {
		state := statusline.StateProductThinker
		if f.View == Facilitator {
			state = statusline.StateFacilitator
		}
		head = statusline.PaintRole(state, name)
	}
	out := []string{head}
	if f.Form == Markdown {
		out = append(out, "")
	}
	if f.View == Facilitator {
		out = append(out, facilitator(in, f)...)
	} else {
		out = append(out, product(in, f)...)
	}
	if l := versionLine(in.Version, f); l != "" {
		out = append(out, l)
	}
	if f.Form == Text {
		out = fitWindow(out, f.Width)
	}
	return out
}

// versionLine is the board's last line, the installed version as `abcd
// --version` names it (abcd v0.13.4): masked, since it is stamped at build
// time, and a list item in markdown, so the page's fence rule holds for it as
// for every other line. The first line stays the view label alone.
func versionLine(v core.VersionInfo, f Frame) string {
	if v.Version == "" {
		return ""
	}
	l := termsafe.Sanitize(strings.TrimSpace(v.Name + " " + v.Version))
	if f.Form == Markdown {
		return "- " + l
	}
	return l
}

// escape matches one SGR colour sequence, the only escape the board draws.
var escape = regexp.MustCompile("\x1b\\[[0-9;]*m")

// fitWindow keeps every text line within width columns: a line already inside
// is left as drawn, and one wider (a label or a count line in a window too
// narrow for it) is drawn without its colour and broken by display width, so
// the window bounds the board whatever its size (A7).
func fitWindow(lines []string, width int) []string {
	if width < 1 {
		return lines
	}
	var out []string
	for _, l := range lines {
		plain := escape.ReplaceAllString(l, "")
		if textwidth.Columns(plain) <= width {
			out = append(out, l)
			continue
		}
		out = append(out, textwidth.Break(plain, width)...)
	}
	return out
}

// symbols are the three state marks and the ellipsis, in UTF-8 or plain text.
type symbols struct{ building, next, count, ellipsis string }

func symbolsFor(f Frame) symbols {
	if f.ASCII && f.Form == Text {
		return symbols{"*", "o", "-", "..."}
	}
	return symbols{"●", "○", "•", "…"}
}

// sgr wraps text in one of the sixteen foreground colours, at any rung above
// Mono, and in nothing in the markdown form.
func sgr(f Frame, code int, text string) string {
	if f.Form != Text || f.Rung == Mono {
		return text
	}
	return fmt.Sprintf("\x1b[%dm%s\x1b[39m", code, text)
}

// title is a row's title as drawn: masked, then fitted to the room left on a
// text line, whole in markdown.
func title(f Frame, s string, room int, sym symbols) string {
	s = termsafe.Sanitize(s)
	if f.Form == Markdown {
		return s
	}
	return textwidth.Fit(s, room, sym.ellipsis)
}

// product is the product thinker's view below its label: one line per intent
// being built, the head and the next READY intents up to three, and a count.
func product(in Input, f Frame) []string {
	sym := symbolsFor(f)
	measure := min(f.Width, productMeasure)
	item := func(s string) string {
		if f.Form == Markdown {
			return "- " + s
		}
		return s
	}
	line := func(mark, word string, code int, t string) string {
		prefix := mark + " " + word + ": "
		return item(sgr(f, code, mark+" "+word+":") + " " + title(f, t, measure-textwidth.Columns(prefix), sym))
	}

	var building, next []string
	ready, parked := 0, 0
	if b := in.Status; b != nil {
		seen := map[string]bool{}
		for _, r := range b.Now {
			switch {
			case r.Lane != nil && !seen[r.ID]:
				seen[r.ID] = true
				building = append(building, r.Title)
			case r.NextUp:
				next = append(next, r.Title)
				ready++
			}
		}
		for _, r := range b.Next {
			next = append(next, r.Title)
			ready++
		}
		parked = len(b.Later)
	}

	var out []string
	if len(building) == 0 {
		out = append(out, item(sgr(f, 32, sym.building+" building:")+" nothing right now"))
	}
	for _, t := range building {
		out = append(out, line(sym.building, "building", 32, t))
	}
	shown := min(len(next), 3)
	if shown == 0 {
		out = append(out, item(sgr(f, 36, sym.next+" next:")+" nothing is ready"))
	}
	for _, t := range next[:shown] {
		out = append(out, line(sym.next, "next", 36, t))
	}
	more := "no more ready"
	if n := ready - shown; n > 0 {
		more = fmt.Sprintf("%d more ready", n)
	}
	waiting := "nothing parked"
	if parked > 0 {
		waiting = fmt.Sprintf("%d parked", parked)
	}
	return append(out, item(sym.count+" "+more+", "+waiting))
}

// entry is one line of the facilitator's view at its nesting depth.
type entry struct {
	depth int
	text  string
}

// labelWidth is the column a labelled row's value starts at, the full board's
// own alignment ("work tiers: " is the longest label).
const labelWidth = 12

// facilitator is the full board below its label, in its own order: the
// directory, each labelled row with its nested lines, then the status block,
// each Now and Next row carrying its spec and a lane's in-flight mark.
func facilitator(in Input, f Frame) []string {
	es := []entry{{0, "abcd — " + termsafe.Sanitize(in.Dir)}}
	// Every row is masked here, whoever worded it, so a row can never add a
	// line or touch the page's fence.
	for _, r := range in.Rows {
		text := termsafe.Sanitize(r.Text)
		if r.Label != "" {
			text = fmt.Sprintf("%-*s%s", labelWidth, termsafe.Sanitize(r.Label)+":", text)
		}
		es = append(es, entry{1, text})
		for _, it := range r.Items {
			es = append(es, entry{2, termsafe.Sanitize(it)})
		}
	}
	if b := in.Status; b != nil {
		es = append(es, entry{1, fmt.Sprintf("%-*sNow %d · Next %d · Later %d", labelWidth, "status:", len(b.Now), len(b.Next), len(b.Later))})
		for _, list := range []struct {
			name string
			rows []statusblock.Row
		}{{"Now", b.Now}, {"Next", b.Next}} {
			es = append(es, entry{2, list.name + ":"})
			if len(list.rows) == 0 {
				es = append(es, entry{3, "(none)"})
			}
			for _, r := range list.rows {
				es = append(es, entry{3, statusRow(r)})
			}
		}
		noun := "intents"
		if len(b.Later) == 1 {
			noun = "intent"
		}
		es = append(es, entry{2, fmt.Sprintf("Later: %d %s", len(b.Later), noun)})
	}

	var out []string
	for _, e := range es {
		indent := strings.Repeat("  ", e.depth)
		if f.Form == Markdown {
			out = append(out, indent+"- "+e.text)
			continue
		}
		out = append(out, wrap(indent+e.text, f.Width)...)
	}
	return out
}

// wrap lays one text line at width columns with a hanging indent four columns
// past its own, breaking at spaces and, where one word is wider than its line,
// inside the word by display width, so no line runs past the window.
func wrap(s string, width int) []string {
	if width < 1 {
		return []string{s}
	}
	hang := len(s) - len(strings.TrimLeft(s, " ")) + 4
	rest := max(width-hang, 1)
	var out []string
	for i, l := range textwidth.Hang(s, width, rest) {
		limit, pad := width, ""
		if i > 0 {
			limit, pad = rest, strings.Repeat(" ", hang)
		}
		for _, piece := range textwidth.Break(l, limit) {
			out = append(out, pad+piece)
			limit, pad = rest, strings.Repeat(" ", hang)
		}
	}
	return out
}

// statusRow is one Now or Next row of the full board: its id, its spec, its
// title, and in brackets what places it there.
func statusRow(r statusblock.Row) string {
	s := termsafe.Sanitize(r.ID)
	if r.SpecID != "" {
		s += "  " + termsafe.Sanitize(r.SpecID)
	}
	s += "  " + termsafe.Sanitize(r.Title)
	if tag := rowTag(r); tag != "" {
		s += "  [" + tag + "]"
	}
	return s
}

// rowTag is what places a row where it is, in words: its lane state or
// "next up", "in flight" for a lane whose branch exists and whose spec is
// open, then the release the intent targets.
func rowTag(r statusblock.Row) string {
	var parts []string
	switch {
	case r.Lane != nil:
		l := r.Lane
		tag := termsafe.Sanitize(l.Stage)
		if l.Lane != "" {
			tag = termsafe.Sanitize(l.Lane) + ": " + tag
		}
		if l.Awaiting != "" {
			tag += ", awaiting the " + termsafe.Sanitize(l.Awaiting)
		}
		if l.Waiting != "" {
			tag += ", " + termsafe.Sanitize(l.Waiting)
		}
		parts = append(parts, tag+" ("+termsafe.Sanitize(l.Run)+")")
		if l.InFlight {
			parts = append(parts, "in flight")
		}
	case r.NextUp:
		parts = append(parts, "next up")
	}
	if r.Target != "" {
		parts = append(parts, "target "+termsafe.Sanitize(r.Target))
	}
	return strings.Join(parts, "; ")
}
