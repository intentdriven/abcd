// Package ask draws a question of the shared type (internal/core/question) in
// a plain Terminal (spc-2610030911534855, itd-2610030810370060): the layout
// from a question to lines at a width and a colour rung, and the sanitising
// every part passes before it is measured or drawn.
//
// The layout is pure: it takes the window's width and the colour rung and
// returns the lines, so every layout is a golden test with an injected width.
// It is a front door, so it holds no limit of its own: how much each part may
// hold is the question package's check, and a question taller than the window
// is drawn whole for the Terminal to scroll.
package ask

import (
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/term"
	"github.com/intentdriven/abcd/internal/textwidth"
)

// Frame is how a question is drawn: the window's width, the colour rung, the
// locale, and which part and which option are current.
type Frame struct {
	Width int
	Mode  term.ColorMode
	// ASCII draws the marks without a UTF-8 locale (term.UTF8Locale false):
	// '>' for the glyph and '|' between tabs.
	ASCII bool
	// Tab is the part drawn, counted from zero, of an Ask with more than one.
	Tab int
	// Current is the current option, counted from zero; the number of options
	// is Later. A value outside that range marks no option.
	Current int
}

// The layout's fixed measures.
const (
	// proseColumns is the widest a run of prose is drawn, at any width (WCAG
	// 1.4.8): a wide window earns columns, not longer lines.
	proseColumns = 80
	// indent is the body's left margin, the width of the marker column.
	indent = 2
	// gap separates the label column from the meaning column.
	gap = 2
	// meaningColumns is the narrowest meaning column drawn beside the labels.
	meaningColumns = 60
	// hang is a list item's and a wrapped line's hanging indent.
	hang = 2
)

// The colours the drawing composes: the 16 basic colours only, whatever
// higher rung the terminal offers (adr-49 decision 2: attribute sequences the
// renderer composes are trusted). Each repeats what a glyph already says.
const (
	chipColour    = "36" // cyan: the chip and the current tab
	currentColour = "33" // yellow: the current option
	reset         = "\x1b[0m"
)

// Layout draws a for a window width columns wide at the colour rung mode: the
// first part, its first option current, the marks of a UTF-8 locale.
func Layout(a question.Ask, width int, mode term.ColorMode) []string {
	return Draw(a, Frame{Width: width, Mode: mode})
}

// Draw draws a in the frame f, top to bottom: the chip line (the chip, then
// the tab strip when a has more than one part), the material (each paragraph
// wrapped on its own, each list item with a hanging indent), the ask, the
// options numbered from 1 with Later as the last number, and the Now: and
// Change later: lines the question carries. One blank line separates each.
//
// Every part is sanitised first (Safe), so widths are of what is drawn. Prose
// is wrapped at min(width, 80) less its indent. An option's label sits on its
// numbered line with its meaning beneath it, unless the window is wider than
// 80 and holds the label column and a meaning column of at least 60, when the
// meaning sits beside the label, wrapped at a measure of at most 80.
func Draw(a question.Ask, f Frame) []string {
	a = Safe(a)
	if len(a.Questions) == 0 {
		return nil
	}
	tab := f.Tab
	if tab < 0 || tab >= len(a.Questions) {
		tab = 0
	}
	q := a.Questions[tab]
	d := newDrawer(f)
	all := entries(q)
	out := d.head(a, tab)
	out = append(out, "")
	out = append(out, d.optionLines(all, span(len(all)), f.Current)...)
	out = append(out, d.typed(q)...)
	if state := d.state(q); len(state) > 0 {
		out = append(out, "")
		out = append(out, state...)
	}
	return out
}

// newDrawer is a drawer for the frame f, its marks chosen for the locale.
func newDrawer(f Frame) drawer {
	d := drawer{f: f, glyph: "›", sep: " · "}
	if f.ASCII {
		d.glyph, d.sep = ">", " | "
	}
	return d
}

// measure is the width prose is wrapped at: min(width, 80) less the indent.
func (d drawer) measure() int {
	return min(d.f.Width, proseColumns) - indent
}

// head draws the part of the question above its answers: the chip line, the
// material, and the ask, one blank line between each.
func (d drawer) head(a question.Ask, tab int) []string {
	q := a.Questions[tab]
	measure := d.measure()
	sections := [][]string{{d.chipLine(a, tab)}}
	var material []string
	for i, b := range q.Material {
		if i > 0 {
			material = append(material, "")
		}
		material = append(material, d.block(b, measure)...)
	}
	if len(material) > 0 {
		sections = append(sections, material)
	}
	sections = append(sections, hanging(q.Ask, indent, 0, measure))
	var out []string
	for i, s := range sections {
		if i > 0 {
			out = append(out, "")
		}
		out = append(out, s...)
	}
	return out
}

// TypedPrefix opens the line the typed part is drawn on.
const TypedPrefix = "or type:"

// typed draws the question's typed part, when it has one, as one line after
// the options: "or type: <prompt>" (spc-2610031241482088).
func (d drawer) typed(q question.Question) []string {
	if q.Typed == "" {
		return nil
	}
	return hanging(TypedPrefix+" "+q.Typed, indent, hang, d.measure())
}

// state draws the Now: and Change later: lines the question carries.
func (d drawer) state(q question.Question) []string {
	measure := d.measure()
	var state []string
	if q.Now != "" {
		state = append(state, hanging(question.Default.NowPrefix+" "+q.Now, indent, hang, measure)...)
	}
	if q.ChangeLater != "" {
		state = append(state, hanging(question.Default.ChangeLaterPrefix+" "+q.ChangeLater, indent, hang, measure)...)
	}
	return state
}

// entries is what a question offers, as drawn and numbered: its options (a
// long list's choices), then Later.
func entries(q question.Question) []question.Option {
	opts := q.Options
	if q.List != nil {
		opts = q.List.Choices
	}
	return append(append([]question.Option(nil), opts...), q.Later)
}

// span is the indices 0 to n-1.
func span(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

// drawer carries one drawing's frame and marks.
type drawer struct {
	f          Frame
	glyph, sep string
}

// colour wraps s in the SGR colour code, or leaves it plain in Mono.
func (d drawer) colour(code, s string) string {
	if d.f.Mode == term.Mono {
		return s
	}
	return "\x1b[" + code + "m" + s + reset
}

// chipLine is the glyph and the chip, then the tab strip ("1 B1 · › 2 B2")
// when the Ask has more than one part, the current tab marked by the glyph.
func (d drawer) chipLine(a question.Ask, tab int) string {
	line := d.colour(chipColour, d.glyph+" "+a.Questions[tab].Chip)
	if len(a.Questions) < 2 {
		return line
	}
	parts := make([]string, len(a.Questions))
	for i, q := range a.Questions {
		label := fmt.Sprintf("%d %s", i+1, q.ID)
		if i == tab {
			label = d.colour(chipColour, d.glyph+" "+label)
		}
		parts[i] = label
	}
	return line + "   " + strings.Join(parts, d.sep)
}

// block draws one material block: a paragraph wrapped at measure, or a list
// with each item's continuation lines hanging beneath its text.
func (d drawer) block(b question.Block, measure int) []string {
	if b.Kind != question.KindList {
		return hanging(b.Text, indent, 0, measure)
	}
	var out []string
	for _, it := range b.Items {
		for i, l := range textwidth.Wrap(it, measure-hang) {
			lead := "- "
			if i > 0 {
				lead = strings.Repeat(" ", hang)
			}
			out = append(out, strings.Repeat(" ", indent)+lead+l)
		}
	}
	return out
}

// optionLines draws the entries of all at idx, each numbered by its place in
// all from 1 (so a narrowed or paged list keeps the numbers the full list
// has), the one at current marked by the glyph. The label column is measured
// over every entry, so it holds still as the list narrows or scrolls.
func (d drawer) optionLines(all []question.Option, idx []int, current int) []string {
	numW := len(fmt.Sprint(len(all))) + 2 // "12. "
	labelAt := indent + numW
	longest := 0
	for _, o := range all {
		longest = max(longest, textwidth.Columns(o.Label))
	}
	meaningAt := labelAt + longest + gap
	// The narrow window is always stacked (spc-2610030911534855: "At 80
	// columns the stacked form is drawn"), however short the labels; a wider
	// one sits side by side once it holds a 60-column meaning column.
	beside := d.f.Width > proseColumns && d.f.Width-meaningAt >= meaningColumns
	labelMeasure := min(d.f.Width, proseColumns) - labelAt

	var out []string
	for _, i := range idx {
		o := all[i]
		marker := strings.Repeat(" ", indent)
		if i == current {
			marker = d.glyph + " "
		}
		number := fmt.Sprintf("%*d. ", numW-2, i+1)
		var rows []string
		if beside {
			meaning := textwidth.Wrap(o.Meaning, min(d.f.Width-meaningAt, proseColumns))
			first := o.Label
			if len(meaning) > 0 {
				first += strings.Repeat(" ", longest-textwidth.Columns(o.Label)+gap) + meaning[0]
			}
			rows = append(rows, first)
			for _, m := range meaning[min(1, len(meaning)):] {
				rows = append(rows, strings.Repeat(" ", meaningAt-labelAt)+m)
			}
		} else {
			label := textwidth.Wrap(o.Label, labelMeasure)
			if len(label) == 0 {
				label = []string{""}
			}
			rows = append(rows, label...)
			rows = append(rows, textwidth.Wrap(o.Meaning, labelMeasure)...)
		}
		head := marker + number + rows[0]
		if i == current {
			// The colour repeats the glyph: the marker, the number and the
			// first row of the current option.
			head = d.colour(currentColour, head)
		}
		out = append(out, head)
		for _, r := range rows[1:] {
			out = append(out, strings.Repeat(" ", labelAt)+r)
		}
	}
	return out
}

// hanging wraps s at measure less hangBy, the first line at the indent at and
// every following line hangBy columns further in, so no line runs past at
// plus measure.
func hanging(s string, at, hangBy, measure int) []string {
	lines := textwidth.Wrap(s, measure-hangBy)
	out := make([]string, len(lines))
	for i, l := range lines {
		pad := at
		if i > 0 {
			pad += hangBy
		}
		out[i] = strings.Repeat(" ", pad) + l
	}
	return out
}
