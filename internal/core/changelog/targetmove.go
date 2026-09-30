package changelog

// targetmove.go — the changelog's note of the targets a cut moved
// (itd-2609212103572513 criterion 3, ruling BS1 of 2026-09-29).
//
// A cut that passes a targeted intent without shipping it rewrites the target
// to `next` in the change that rolls the changelog, and the dated section says
// so in one line directly under its notice, ahead of every change-type
// heading. The line names planned intents, which this release did NOT ship, so
// every reader that credits a release with the records its section names —
// the site's release stamp — passes over it through IsTargetMoveNote. Its
// place ahead of the first `###` heading keeps it out of the delivery
// sections record-lint's delivery_state rule judges.

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/launch"
)

// targetMoveLead opens the note, spelled once for the writer and the reader.
const targetMoveLead = "Targeted and not shipped in this release, so each targets the next release (`next`): "

// TargetMoveNote renders the note for the moves a cut makes, or "" when it
// makes none. Each move reads `<id> (targeted <what the record carried>)`.
func TargetMoveNote(moves []launch.TargetMove) string {
	if len(moves) == 0 {
		return ""
	}
	parts := make([]string, 0, len(moves))
	for _, m := range moves {
		from := m.From
		if from == launch.TargetNext {
			from = "`" + from + "`"
		}
		parts = append(parts, m.ID+" (targeted "+from+")")
	}
	return targetMoveLead + strings.Join(parts, ", ") + "."
}

// IsTargetMoveNote reports whether a changelog line is a move note: a line
// that names intents a release did not ship, so no reader credits that
// release with them.
func IsTargetMoveNote(line string) bool {
	return strings.HasPrefix(strings.TrimRight(line, "\r"), targetMoveLead)
}
