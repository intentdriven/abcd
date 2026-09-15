package statusline

// Package statusline composes abcd's row for a host harness's status line.
//
// It is one render, in core, consumed by two front doors — the status verb the
// harness invokes and the bare `/abcd` board — so that no front door invents
// its own words (spc-70). Like the rest of internal/core it writes to no
// stream, reads no argv and knows no transport: Render takes values and
// returns a value.
//
// The render is PURE. Payload, repository, counts and badge state in, a row
// out: no network, no writes, no clock, and no read of anything but its
// arguments. The one file this package touches at all is the user-level
// setting, behind Load, which is a separate call a front door makes once.
//
// TRUNCATION SAFETY COMES FROM ORDER ALONE. The harness hands the status
// command no terminal width and runs it with no controlling terminal, so the
// width is not merely unread but unknowable, and the right edge is what a
// narrow host cuts first (iss-168, settled by demonstration 2026-08-29). This
// package therefore implements no truncation of its own. It puts the badge
// first and makes every later element optional, so cutting the row at any
// point leaves the badge — the property ac-10 states and
// TestEveryPrefixOfTheRowBeginsWithTheBadge checks directly, over every prefix.

import (
	"math"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// Separator sits between two elements. It is the same one-line separator the
// rest of this codebase already uses for a run of short facts (capture's
// reading positions, lifeboat's coverage line, the intent audit's rollup),
// rather than a new one invented for this row.
const Separator = " · "

// ElementKey names one element of the row. The keys are also the names the
// per-element switches carry in the user-level setting, so a switch and the
// thing it switches cannot drift apart.
type ElementKey string

const (
	KeyPresence ElementKey = "presence"
	KeyRepo     ElementKey = "repo"
	KeyBranch   ElementKey = "branch"
	KeyModel    ElementKey = "model"
	KeyContext  ElementKey = "context"
	KeyFiveHour ElementKey = "five_hour"
	KeySevenDay ElementKey = "seven_day"
	KeyIntents  ElementKey = "intents"
	KeyIssues   ElementKey = "issues"
)

// order is the fixed element order itd-200's commitments set: the badge, the
// repository name, the branch, the model, the context percentage, the
// five-hour and seven-day usage percentages, and the record's intent and issue
// counts.
//
// The badge is element one and is not switchable. Everything after it is.
var order = []ElementKey{
	KeyPresence,
	KeyRepo,
	KeyBranch,
	KeyModel,
	KeyContext,
	KeyFiveHour,
	KeySevenDay,
	KeyIntents,
	KeyIssues,
}

// Order returns the fixed element order as a copy the caller may mutate.
func Order() []ElementKey { return append([]ElementKey(nil), order...) }

// Element is one rendered slot of the row: what it is, how it looks with
// colour, and how it reads without.
//
// Two forms rather than one because two consumers need different things from
// the same words. The status verb prints Rendered into a surface that
// interprets escapes; the `/abcd` board — the fallback where a host has no
// status surface — prints Plain. Neither invents the text.
type Element struct {
	Key      ElementKey `json:"key"`
	Rendered string     `json:"rendered"`
	Plain    string     `json:"plain"`
}

// Row is the composed line: elements in order, ready to join. It is
// deliberately not a string — a front door that received a string would have
// to parse it back apart to reuse a part of it, and the board does exactly
// that.
type Row struct {
	Elements []Element `json:"elements"`
}

// Empty reports whether the row carries nothing at all, which is what an
// unmanaged repository and a switched-off setting both produce.
func (r Row) Empty() bool { return len(r.Elements) == 0 }

// Element returns the element with the given key.
func (r Row) Element(k ElementKey) (Element, bool) {
	for _, el := range r.Elements {
		if el.Key == k {
			return el, true
		}
	}
	return Element{}, false
}

// String is the row with colour, for a surface that interprets escapes.
func (r Row) String() string { return r.join(func(e Element) string { return e.Rendered }) }

// Plain is the row without colour, for the board and for anything that must
// read the row back.
func (r Row) Plain() string { return r.join(func(e Element) string { return e.Plain }) }

func (r Row) join(pick func(Element) string) string {
	parts := make([]string, 0, len(r.Elements))
	for _, el := range r.Elements {
		parts = append(parts, pick(el))
	}
	return strings.Join(parts, Separator)
}

// Repo is the repository half of the render's input: the two facts that come
// from the checkout rather than from the harness or the record.
type Repo struct {
	Name   string `json:"name"`
	Branch string `json:"branch"`
}

// Counts is the record half: open issues, and intents not yet shipped.
//
// Zero is a fact about the record, not an absence, so a zero count renders. A
// caller that has no record to count switches the two elements off rather than
// passing a zero it does not mean.
type Counts struct {
	Intents int `json:"intents"`
	Issues  int `json:"issues"`
}

// Input is everything Render reads.
type Input struct {
	State   State   `json:"state"`
	Repo    Repo    `json:"repo"`
	Payload Payload `json:"payload"`
	Counts  Counts  `json:"counts"`
}

// Render composes the row.
//
// An element renders when it is switched on AND has something to say. The
// second half is itd-200's absent-field rule: a payload field the harness did
// not supply DROPS its element — no placeholder, no empty slot, and no
// separator left where the element was, because the separator is placed
// between the elements that survived rather than between the slots that might
// have existed.
func Render(in Input, set Settings) Row {
	if set.Disabled {
		return Row{}
	}
	row := Row{Elements: make([]Element, 0, len(order))}
	for _, k := range order {
		if k == KeyPresence {
			// The badge leads, and nothing abcd owns precedes it (ac-1). It is
			// placed from inside the ordered walk rather than prepended
			// afterwards, so its position is the SAME fact as every other
			// element's: move it in `order` and it moves in the row.
			row.Elements = append(row.Elements, renderBadge(in.State, set.Presence))
			continue
		}
		if !set.Enabled(k) {
			continue
		}
		text := elementText(k, in)
		if text == "" {
			continue
		}
		row.Elements = append(row.Elements, Element{Key: k, Rendered: text, Plain: text})
	}
	return row
}

// elementText is the plain text of one non-badge element, or "" when the
// element has nothing to say and must drop.
func elementText(k ElementKey, in Input) string {
	switch k {
	case KeyRepo:
		return termsafe.Sanitize(strings.TrimSpace(in.Repo.Name))
	case KeyBranch:
		return termsafe.Sanitize(strings.TrimSpace(in.Repo.Branch))
	case KeyModel:
		return strings.TrimSpace(in.Payload.Model)
	case KeyContext:
		return percent("ctx", in.Payload.ContextPct)
	case KeyFiveHour:
		return percent("5h", in.Payload.FiveHourPct)
	case KeySevenDay:
		return percent("7d", in.Payload.SevenDayPct)
	case KeyIntents:
		return "itd " + strconv.Itoa(in.Counts.Intents)
	case KeyIssues:
		return "iss " + strconv.Itoa(in.Counts.Issues)
	}
	return ""
}

// percent renders a usage figure as a whole percent behind its label, or ""
// when the harness did not supply it.
//
// It rounds rather than truncating, because a 99.6% context window reported as
// 99% is the one reading that matters being wrong in the reassuring direction.
// It does not clamp a plausible figure: a rate-limit window can read above 100
// once the limit is exceeded, and the row says what the payload says.
//
// It DOES drop an implausible one. A percentage above maxPercent is a
// payload abcd does not understand, and rendering it verbatim would let one
// field decide the width of the whole row — 1e300 formats to three hundred
// digits, which pushes every later element off a line this package is
// forbidden to truncate. A NEGATIVE percentage is a payload abcd does not
// understand either: no usage window can be less than empty, so "-7%" is not
// a reading, and negative zero — which a float carries and would render as
// "-0%" — is dropped with it by testing the sign bit rather than the value.
// Dropping either is the same answer an absent field gets, and for the same
// reason: the row says nothing rather than something wrong.
func percent(label string, v *float64) string {
	if v == nil || math.IsNaN(*v) || math.Signbit(*v) || *v > maxPercent {
		return ""
	}
	return label + " " + strconv.FormatFloat(math.Round(*v), 'f', -1, 64) + "%"
}

// maxPercent bounds a rendered percentage. It is far above any figure the
// three windows can legitimately report and far below the width at which one
// element could crowd out the rest of the row.
const maxPercent = 100000
