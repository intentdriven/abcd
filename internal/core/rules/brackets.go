package rules

import (
	"fmt"
	"strings"
)

// brackets.go — the context brackets (spc-2610111627213138 step 6, "Context
// brackets"): the one table that says how much the loader sends as the
// session fills. The rules injection (InjectWithin) and the memory recall
// (internal/core/recall) both read it from here, so no payload grows
// thresholds of its own (the facilitator's ruling of 2026-10-11, "Share the
// brackets", and that the brackets gate rules too, with the same thresholds).
//
// The table reads the fill as the share of the context window still free.
// Three rows are brackets of fill; the fourth, unknown, is what the loader
// does when no source tells how full the session is (the technical
// facilitator's ruling of 2026-10-11, "Rules full, notes short"). There is no
// row in which recall sends nothing: notes and pages come back in full, then
// as one line each, and stay at one line to the end of the session, since a
// compaction or a new session resets recall anyway ("never send nothing").
//
// The figures are the spec's starting points, each set here alone.

// Bracket is a row of the bracket table. The zero value is BracketUnknown,
// so a caller that hands no fill gets the unknown row: rules in full, as
// before the brackets, and notes and pages as one line each.
type Bracket int

// The brackets.
const (
	BracketUnknown Bracket = iota
	BracketFresh
	BracketModerate
	BracketDepleted
)

// String is the bracket's name, as the hook's envelope and `rules --prompt`
// name it.
func (b Bracket) String() string {
	switch b {
	case BracketFresh:
		return "fresh"
	case BracketModerate:
		return "moderate"
	case BracketDepleted:
		return "depleted"
	}
	return "unknown"
}

// MemoryAll is a memory figure that is bounded by the per-turn budget alone
// (InjectionBudgetBytes, less what the rules took), not by a bracket's own.
const MemoryAll = -1

// BracketRow is one row of the table: when it holds and what it sends.
type BracketRow struct {
	Bracket Bracket
	// MinFreePercent is the least share of the window, 0 to 100, still free
	// at which a bracket of fill holds. The unknown row has none.
	MinFreePercent float64
	// ShrinkRules sends each matched domain as its heading, its first rule
	// and a line naming how many rules it holds and the command that renders
	// the rest, instead of in full.
	ShrinkRules bool
	// MemoryFullBytes bounds the notes and pages sent in full, the
	// top-ranked first: MemoryAll for the per-turn budget alone, 0 for none.
	MemoryFullBytes int
	// MemoryLineBytes bounds the notes and pages sent as one line each, the
	// ones not sent in full, the top-ranked first: MemoryAll for the
	// per-turn budget alone, 0 for none. Where it is not 0 the top-ranked
	// candidate always comes back, as one line at least, past this bound
	// (though never past the per-turn budget).
	MemoryLineBytes int
}

// bracketTable is the table: the three brackets of fill, freest first, so
// the first whose MinFreePercent the free share reaches holds, then the
// unknown row.
var bracketTable = [...]BracketRow{
	{Bracket: BracketFresh, MinFreePercent: 70, ShrinkRules: false, MemoryFullBytes: MemoryAll, MemoryLineBytes: 2 * 1024},
	{Bracket: BracketModerate, MinFreePercent: 40, ShrinkRules: true, MemoryFullBytes: 8 * 1024, MemoryLineBytes: MemoryAll},
	{Bracket: BracketDepleted, MinFreePercent: 0, ShrinkRules: true, MemoryFullBytes: 0, MemoryLineBytes: 2 * 1024},
	{Bracket: BracketUnknown, ShrinkRules: false, MemoryFullBytes: 0, MemoryLineBytes: 2 * 1024},
}

// BracketTable returns the table's rows, a copy, in the order they are read.
func BracketTable() []BracketRow {
	return append([]BracketRow(nil), bracketTable[:]...)
}

// Row is the bracket's row of the table.
func (b Bracket) Row() BracketRow {
	for _, r := range bracketTable {
		if r.Bracket == b {
			return r
		}
	}
	return bracketTable[len(bracketTable)-1]
}

// BracketForUsed is the bracket of fill a used share of the window, 0 to 100,
// falls in. A share outside that range, or not a number, is no fill at all:
// BracketUnknown.
func BracketForUsed(usedPct float64) Bracket {
	if !validPercentage(usedPct) {
		return BracketUnknown
	}
	free := 100 - usedPct
	for _, r := range bracketTable {
		if r.Bracket == BracketUnknown {
			continue
		}
		if free >= r.MinFreePercent {
			return r.Bracket
		}
	}
	return BracketDepleted
}

// BracketFor is the bracket a fill falls in: BracketUnknown when no source
// told how full the session is.
func BracketFor(f Fill) Bracket {
	if !f.Known() {
		return BracketUnknown
	}
	return BracketForUsed(*f.UsedPercentage)
}

// Describe says, in words, when the row holds and what it sends, for
// `rules --prompt`: When, a colon, then Sends. The figures come from the row,
// so the words cannot state a threshold the table does not hold.
func (r BracketRow) Describe() string {
	return r.When() + ": " + r.Sends()
}

// When says when the row holds: the share of the window still free, or, for
// the unknown row, that nothing tells.
func (r BracketRow) When() string {
	switch r.Bracket {
	case BracketUnknown:
		return "no source tells how full the session is"
	case BracketDepleted:
		return fmt.Sprintf("below %s%% of the window free, to the end of the session", trimFloat(r.upperFree()))
	}
	if up := r.upperFree(); up < 100 {
		return fmt.Sprintf("%s%% to %s%% of the window free", trimFloat(r.MinFreePercent), trimFloat(up))
	}
	return fmt.Sprintf("%s%% or more of the window free", trimFloat(r.MinFreePercent))
}

// Sends says what the row sends, as a sentence: the rules' form, then the
// notes' and pages'.
func (r BracketRow) Sends() string {
	rulesForm := "Rules in full"
	if r.ShrinkRules {
		rulesForm = "Each rule domain as its heading and first rule, with a line naming the command that renders the rest"
	}
	var mem string
	switch {
	case r.MemoryFullBytes == MemoryAll:
		mem = fmt.Sprintf("notes and pages in full, in ranked order, within the per-turn budget, and one line each within %s for those that do not fit", kib(r.MemoryLineBytes))
	case r.MemoryFullBytes > 0 && r.MemoryLineBytes == MemoryAll:
		mem = fmt.Sprintf("the top-ranked notes and pages in full within %s, the rest as one line each", kib(r.MemoryFullBytes))
	case r.MemoryFullBytes > 0:
		mem = fmt.Sprintf("the top-ranked notes and pages in full within %s, then one line each within %s", kib(r.MemoryFullBytes), kib(r.MemoryLineBytes))
	case r.MemoryLineBytes == MemoryAll:
		mem = "notes and pages as one line each, within the per-turn budget"
	default:
		mem = fmt.Sprintf("notes and pages as one line each, the top-ranked lines within %s and always at least the top-ranked one", kib(r.MemoryLineBytes))
	}
	return rulesForm + "; " + mem
}

// upperFree is the free share at which the next freer bracket of fill takes
// over: 100 for the freest.
func (r BracketRow) upperFree() float64 {
	up := 100.0
	for _, o := range bracketTable {
		if o.Bracket != BracketUnknown && o.MinFreePercent > r.MinFreePercent && o.MinFreePercent < up {
			up = o.MinFreePercent
		}
	}
	return up
}

func trimFloat(f float64) string {
	return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%.2f", f), "0"), ".")
}

func kib(n int) string {
	if n%1024 == 0 {
		return fmt.Sprintf("%d KiB", n/1024)
	}
	return fmt.Sprintf("%d bytes", n)
}
