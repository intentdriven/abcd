package extent

import (
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// GapsFlag is the one flag a declared command takes to list every gap.
const GapsFlag = "--gaps"

// RenderOptions shapes the text of a statement.
type RenderOptions struct {
	// AllGaps lists every gap in full under the count line, which then names
	// no flag: the person passed GapsFlag, or no one can pass it (the sweep
	// abcd runs on its own).
	AllGaps bool
}

var causeWords = map[Cause]string{
	CauseHostRun:       "run by the host",
	CauseOtherRun:      "left to another way of running it",
	CauseNotArmed:      "not armed",
	CauseNothingToRead: "nothing to read",
}

var reasonWords = map[Reason]string{
	ReasonUnreadable:  "unreadable",
	ReasonUnreachable: "unreachable",
	ReasonToolMissing: "tool missing",
	ReasonTimedOut:    "timed out",
	ReasonCheckFailed: "check failed",
}

// words returns the text for an identifier, or the identifier itself where it
// has none, so a statement Validate would refuse still renders.
func words[T ~string](table map[T]string, v T) string {
	if w, ok := table[v]; ok {
		return w
	}
	return termsafe.SanitizeLine(string(v))
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

// Lines renders the statement as the lines every command prints, in this
// order:
//
//   - "Looked at:" and the labels, or "Looked at: nothing";
//   - "Left out in this run:" and each left-out kind with its cause, where
//     there is one;
//   - "Known not looked at:" and the exclusions and the hand-written list,
//     only on the verdicts clear and clear_except_gaps;
//   - "Could not check:" and one count of checks for each reason, naming
//     GapsFlag, or, with AllGaps, naming no flag and followed by every gap;
//     no such line when there is no gap;
//   - one "Judged by:" line for each depth entry;
//   - a summary: clear except for the gaps on clear_except_gaps, or, where no
//     kind was looked at, that nothing was checked and why.
//
// Every value taken from the statement is masked for one line of a terminal.
func (s Statement) Lines(o RenderOptions) []string {
	var lines []string

	var looked []string
	for _, l := range s.LookedAt {
		looked = append(looked, termsafe.SanitizeLine(l.Label))
	}
	if len(looked) == 0 {
		lines = append(lines, "Looked at: nothing")
	} else {
		lines = append(lines, "Looked at: "+strings.Join(looked, ", "))
	}

	var left, known []string
	for _, n := range s.NotLookedAt {
		label := termsafe.SanitizeLine(n.Label)
		if n.Source == SourceLeftOut {
			left = append(left, label+" ("+words(causeWords, n.Cause)+")")
		} else {
			known = append(known, label)
		}
	}
	if len(left) > 0 {
		lines = append(lines, "Left out in this run: "+strings.Join(left, ", "))
	}
	verdict, hasVerdict := s.Verdict()
	if len(known) > 0 && hasVerdict && (verdict == VerdictClear || verdict == VerdictClearExceptGaps) {
		lines = append(lines, "Known not looked at: "+strings.Join(known, ", "))
	}

	if len(s.CouldNotCheck) > 0 {
		line := "Could not check: " + s.gapCounts()
		if !o.AllGaps {
			line += "; " + GapsFlag + " lists them"
		}
		lines = append(lines, line)
		if o.AllGaps {
			for _, g := range s.CouldNotCheck {
				lines = append(lines, "  - "+gapLine(g))
			}
		}
	}

	for _, d := range s.Depth {
		lines = append(lines, "Judged by: "+d.line())
	}

	switch {
	case len(s.LookedAt) == 0:
		lines = append(lines, "Nothing was checked: "+s.whyNothing()+".")
	case hasVerdict && verdict == VerdictClearExceptGaps:
		lines = append(lines, "Clear except for "+plural(len(s.CouldNotCheck), "check", "checks")+" that could not be made.")
	}
	return lines
}

// gapCounts is one count of checks for each reason, in the order of the
// closed set.
func (s Statement) gapCounts() string {
	counts := map[Reason]int{}
	var order []Reason
	for _, g := range s.CouldNotCheck {
		if counts[g.Reason] == 0 {
			order = append(order, g.Reason)
		}
		counts[g.Reason]++
	}
	var parts []string
	for _, r := range reasons {
		if counts[r] > 0 {
			parts = append(parts, plural(counts[r], "check", "checks")+" ("+reasonWords[r]+")")
		}
	}
	for _, r := range order { // a reason outside the set, which Validate refuses
		if _, known := reasonWords[r]; !known {
			parts = append(parts, plural(counts[r], "check", "checks")+" ("+words(reasonWords, r)+")")
		}
	}
	return strings.Join(parts, ", ")
}

// gapLine is one gap in full: the check, the identifier of the kind it bears
// on where it names one, the reason and the text. The kind is named by its
// identifier and not its label, so the text of many gaps grows with the gaps
// and not with the label of the one kind they share; a gap whose check
// stopped its kind being read names a kind no list labels in any case.
func gapLine(g Gap) string {
	line := termsafe.SanitizeLine(g.Check)
	if g.Kind != "" {
		line += " (" + termsafe.SanitizeLine(g.Kind) + ")"
	}
	return line + ": " + words(reasonWords, g.Reason) + ": " + termsafe.SanitizeLine(g.Text)
}

// whyNothing says why a statement that looked at no kind checked nothing: the
// causes of what was left out, the reasons nothing could be read, or that
// there was no kind to look at.
func (s Statement) whyNothing() string {
	var parts []string
	leftOut, seen := 0, map[Cause]bool{}
	for _, n := range s.NotLookedAt {
		if n.Source == SourceLeftOut {
			leftOut++
			seen[n.Cause] = true
		}
	}
	if leftOut > 0 {
		var why []string
		for _, c := range causes {
			if seen[c] {
				why = append(why, causeWords[c])
			}
		}
		parts = append(parts, plural(leftOut, "kind", "kinds")+" left out ("+strings.Join(why, ", ")+")")
	}
	if len(s.CouldNotCheck) > 0 {
		var why []string
		for _, r := range reasons {
			for _, g := range s.CouldNotCheck {
				if g.Reason == r {
					why = append(why, reasonWords[r])
					break
				}
			}
		}
		parts = append(parts, plural(len(s.CouldNotCheck), "check", "checks")+" could not be made ("+strings.Join(why, ", ")+")")
	}
	if len(parts) == 0 {
		return "there was no kind to look at"
	}
	return strings.Join(parts, "; ")
}

// line is one depth entry: its role and key, then the classes, the number of
// judging runs and the effort, with not told and none asked in words.
func (d Depth) line() string {
	head := termsafe.SanitizeLine(d.Role)
	switch {
	case d.Lane != "":
		head += ", lane " + termsafe.SanitizeLine(d.Lane)
	case d.Issue != "":
		head += ", issue " + termsafe.SanitizeLine(d.Issue)
	}
	if d.Runs.Mark == "" && d.Runs.Count == 0 {
		return head + ": no judging run"
	}
	if d.Class != nil && d.Class.Mark == MarkNotTold && d.Runs.Mark == MarkNotTold && d.Effort != nil && d.Effort.Mark == MarkNotTold {
		return head + ": abcd was not told the class, the number of runs or the effort"
	}
	var parts []string
	if d.Class != nil {
		parts = append(parts, told(d.Class, "class", "classes"))
	}
	switch d.Runs.Mark {
	case MarkNotTold:
		parts = append(parts, "abcd was not told the number of runs")
	case MarkNoneAsked:
		parts = append(parts, "no number of runs asked")
	default:
		parts = append(parts, plural(d.Runs.Count, "judging run", "judging runs")+fromWords(d.Runs.From))
	}
	if d.Effort != nil {
		parts = append(parts, told(d.Effort, "effort", "efforts"))
	}
	return head + ": " + strings.Join(parts, ", ")
}

func told(t *Told, one, many string) string {
	switch t.Mark {
	case MarkNotTold:
		return "abcd was not told the " + one
	case MarkNoneAsked:
		return "no " + one + " asked"
	}
	name := one
	if len(t.Values) > 1 {
		name = many
	}
	return name + " " + termsafe.SanitizeLine(strings.Join(t.Values, ", ")) + fromWords(t.From)
}

// fromWords marks a value a receipt carries, which is the host's own account
// relayed; a value from the run's own state needs no mark.
func fromWords(o Origin) string {
	if o == OriginReceipt {
		return " (from a receipt)"
	}
	return ""
}
