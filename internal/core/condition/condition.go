// Package condition is the scope-condition disposition vocabulary as DATA: the
// four values a condition is dispositioned with, the identity marker a condition
// carries, the two block grammars a disposition is written under, and the one
// reader that folds those blocks into a condition's standing disposition.
//
// Two writers put dispositions into an intent's `## Audit Notes`: the fidelity
// verdict ingest (one block per receipt, covering every condition) and the
// condition verb (one dated block per write, covering one condition and naming
// what occasioned it). Both, and every reader of what they wrote — the verb's
// render, the verdict ingest's report of what it leaves standing, and the record
// lint the principles spec builds over the same dispositions — read one
// vocabulary from here, so no two of them can disagree about what a disposition
// is (spc-2609020626046252).
//
// It is a leaf for the reason core/grounds and core/issueschema are: core/lint
// must read dispositions, core/intent's tests import core/lint, and a lint that
// imported intent back would be an import cycle. It imports core/mdrecord for
// the one notion of where a section starts and stops, and otherwise only the
// standard library: no filesystem, no transport, no record store.
package condition

import (
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/mdrecord"
)

// The four disposition values (spc-59). Narrowed is the one that requires a
// stated narrowing and the only one permitted to carry one; Untested is the
// word for the absence of a judgement.
const (
	Survived  = "survived"
	Narrowed  = "narrowed"
	Falsified = "falsified"
	Untested  = "untested"
)

// Enum is the closed set, in the order a refusal names it.
var Enum = []string{Survived, Narrowed, Falsified, Untested}

// Valid reports whether v is one of the four values.
func Valid(v string) bool {
	for _, e := range Enum {
		if v == e {
			return true
		}
	}
	return false
}

var (
	// MarkerRe matches a condition's identity marker anywhere inside its bullet
	// (spc-55). It is not line-anchored: an editor that rewraps a bullet moves the
	// marker, and a positional read would orphan every disposition keyed on it.
	MarkerRe = regexp.MustCompile(`<!-- cond: (cond-[0-9]{16}) -->`)
	// MarkerIDRe is the identity alone, whole-string.
	MarkerIDRe = regexp.MustCompile(`^cond-[0-9]{16}$`)
	// ReviewMarkerRe is the verdict ingest's block marker: one line, whole-line,
	// carrying the receipt state and id.
	ReviewMarkerRe = regexp.MustCompile(`(?m)^<!-- abcd-review: (OWED|INGESTED|DEAD_LETTER) receipt=(rcp-[0-9a-f]+) -->\r?$`)
	// BlockMarkerRe is the condition verb's block marker: the one identity the
	// block dispositions and what occasioned it, a reading item or a delivered
	// intent and nothing else.
	BlockMarkerRe = regexp.MustCompile(`(?m)^<!-- abcd-condition: (cond-[0-9]{16}) occasion=((?:rdi|itd)-[0-9]+) -->\r?$`)
)

var (
	auditHeadingRe  = regexp.MustCompile(`^#{1,6}\s+Audit Notes\s*$`)
	bulletRe        = regexp.MustCompile(`^- (cond-[0-9]{16}) — ([a-z]+)(?:: (.*))?$`)
	narrowingLineRe = regexp.MustCompile(`^  narrowing: (.*)$`)
	blockDateRe     = regexp.MustCompile(`^Condition disposition — ([0-9]{4}-[0-9]{2}-[0-9]{2}), occasioned by ((?:rdi|itd)-[0-9]+)\.$`)
)

// dispositionsLabel opens the disposition list inside a verdict block; the
// bullets above it are per-criterion verdicts, not dispositions.
const dispositionsLabel = "Scope-condition dispositions:"

// IsBlockMarker reports whether line opens a disposition-bearing block under
// either grammar. It is the boundary both writers end a block at, so a verdict
// block replaced in place never swallows a condition block that follows it.
func IsBlockMarker(line string) bool {
	line = strings.TrimRight(line, "\r")
	return ReviewMarkerRe.MatchString(line) || BlockMarkerRe.MatchString(line)
}

// Disposition is one disposition as a block records it. Occasion is set only
// for an entry from a condition block, and is what tells the two sources apart;
// Date likewise.
type Disposition struct {
	ConditionID string `json:"condition_id"`
	Disposition string `json:"disposition"`
	Rationale   string `json:"rationale,omitempty"`
	Narrowing   string `json:"narrowing,omitempty"`
	// Source names the block: `verdict rcp-…` or `condition <occasion>`.
	Source   string `json:"source"`
	Date     string `json:"date,omitempty"`
	Occasion string `json:"occasion,omitempty"`
}

// ReadDispositions returns every disposition the `## Audit Notes` section
// records, in document order. A bullet whose value is outside the enum is not a
// disposition and is skipped, as is a condition-block bullet naming an identity
// other than the one its marker names: the marker is the block's key.
func ReadDispositions(content string) []Disposition {
	lines := strings.Split(content, "\n")
	start, end, ok := mdrecord.SectionLineRange(lines, auditHeadingRe)
	if !ok {
		return nil
	}
	var (
		out       []Disposition
		source    string // "" outside any disposition-bearing block
		inList    bool   // verdict block: past the dispositions label
		blockID   string // condition block: the identity its marker names
		occasion  string
		date      string
		lastIndex = -1 // the entry a narrowing line attaches to
	)
	for _, raw := range lines[start:end] {
		ln := strings.TrimRight(raw, "\r")
		if m := ReviewMarkerRe.FindStringSubmatch(ln); m != nil {
			source, inList, blockID, occasion, date, lastIndex = "verdict "+m[2], false, "", "", "", -1
			continue
		}
		if m := BlockMarkerRe.FindStringSubmatch(ln); m != nil {
			source, inList, blockID, occasion, date, lastIndex = "condition "+m[2], true, m[1], m[2], "", -1
			continue
		}
		if source == "" {
			continue
		}
		if blockID != "" && date == "" {
			if m := blockDateRe.FindStringSubmatch(ln); m != nil && m[2] == occasion {
				date = m[1]
				continue
			}
		}
		if blockID == "" && ln == dispositionsLabel {
			inList = true
			continue
		}
		if !inList {
			continue
		}
		if m := bulletRe.FindStringSubmatch(ln); m != nil {
			lastIndex = -1
			if !Valid(m[2]) || (blockID != "" && m[1] != blockID) {
				continue
			}
			out = append(out, Disposition{
				ConditionID: m[1], Disposition: m[2], Rationale: m[3],
				Source: source, Date: date, Occasion: occasion,
			})
			lastIndex = len(out) - 1
			continue
		}
		if m := narrowingLineRe.FindStringSubmatch(ln); m != nil && lastIndex >= 0 {
			out[lastIndex].Narrowing = m[1]
		}
	}
	return out
}

// Standing folds ReadDispositions into each identity's standing disposition.
//
// The last entry in document order stands, with one exception: an entry from a
// verdict block does not replace a standing entry from a condition block unless
// the verdict's rationale names that entry's occasion. A re-audit covers every
// condition by construction and knows nothing of the readings, so without the
// exception it would erase, silently, the one thing the join exists to keep; an
// auditor who has weighed the reading says so by naming it.
func Standing(content string) map[string]Disposition {
	standing := map[string]Disposition{}
	for _, d := range ReadDispositions(content) {
		prev, ok := standing[d.ConditionID]
		if ok && prev.Occasion != "" && d.Occasion == "" && !namesOccasion(d.Rationale, prev.Occasion) {
			continue
		}
		standing[d.ConditionID] = d
	}
	return standing
}

// namesOccasion reports whether text names occasion as a whole token, so
// rdi-12 is not named by a rationale citing rdi-123.
func namesOccasion(text, occasion string) bool {
	return regexp.MustCompile(`(?:^|[^A-Za-z0-9-])` + regexp.QuoteMeta(occasion) + `(?:$|[^A-Za-z0-9-])`).MatchString(text)
}
