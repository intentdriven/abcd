package ideate

// render.go turns a validated verdict into the durable research record a human
// reads. Two properties make it worth its own file.
//
// It is DETERMINISTIC: the same document and the same date render byte-identical
// output, in a fixed order, with no wall-clock read of its own.
//
// And it is the last place untrusted prose can forge structure. Every field
// arrives already line-cleaned (no newline, no HTML comment marker, no terminal
// escape), so the one hazard left is markdown's own syntax — a pipe inside a table
// cell would split the cell, and a leading marker would open a list or a heading.
// termsafe.TableCell closes the first; termsafe.BlockText, the line-cleaning and the
// fixed prefixes close the rest.
//
// The prose is British English, because it lands in a document a person reads.

import (
	"fmt"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// render writes the whole verdict record.
func render(v verdictDoc, at time.Time) string {
	var b strings.Builder

	// The slug is already constrained to lower-case kebab-case, so it is the one
	// value in this document that needs no escaping at all.
	fmt.Fprintf(&b, "# Ideate verdict — %s\n\n", v.slug)
	fmt.Fprintf(&b, "**Verdict: %s.** Recorded on %s by abcd's idea-admission protocol —\n", v.verdict, recordDate(at))
	b.WriteString("primary-source research, a grill against the existing record, and an\n")
	b.WriteString("independent adversarial review. This record exists so the idea is not\n")
	b.WriteString("re-litigated: it stands whether the idea lived or died.\n\n")

	b.WriteString("## The idea\n\n")
	fmt.Fprintf(&b, "%s\n\n", termsafe.BlockText(v.idea))

	renderResearch(&b, v)
	renderGrill(&b, v)
	renderAdversarial(&b, v)
	renderRejected(&b, v)
	renderOutcome(&b, v)

	return b.String()
}

// renderResearch writes leg 1 as the claims table: one row per load-bearing
// claim, the primary source it was checked against, and what that established.
func renderResearch(b *strings.Builder, v verdictDoc) {
	b.WriteString("## Leg 1 — Primary-source research\n\n")
	b.WriteString("Every load-bearing claim checked against its primary source, never a\n")
	b.WriteString("secondary citation.\n\n")
	b.WriteString("| Claim | Primary source | Finding |\n|---|---|---|\n")
	for _, c := range v.claims {
		fmt.Fprintf(b, "| %s | %s | %s |\n", termsafe.TableCell(c.Claim), termsafe.TableCell(c.PrimarySource), c.Status)
	}
	b.WriteString("\n")
}

// renderGrill writes leg 2. An empty grill is stated rather than omitted: "nothing
// in the record touches this" is the finding a later reader most needs to see.
func renderGrill(b *strings.Builder, v verdictDoc) {
	b.WriteString("## Leg 2 — Record grill\n\n")
	b.WriteString("Does the brief, an intent, an ADR, or a principle already cover,\n")
	b.WriteString("contradict, or supersede this idea? Every hit is cited by record id, and\n")
	b.WriteString("every id resolved in this repository when the verdict was recorded.\n\n")
	if len(v.hits) == 0 {
		b.WriteString("No entry in the record covers, contradicts, or supersedes this idea.\n\n")
		return
	}
	b.WriteString("| Record | Relation | Note |\n|---|---|---|\n")
	for _, h := range v.hits {
		note := h.Note
		if note == "" {
			note = "—"
		}
		fmt.Fprintf(b, "| %s | %s | %s |\n", h.Record, h.Relation, termsafe.TableCell(note))
	}
	b.WriteString("\n")
}

// renderAdversarial writes leg 3, and states the conduct the leg is defined by —
// the fresh-context, off-policy, unknown-authorship pattern that is the only
// measured debiasing effect, and the reason this leg is worth running at all.
func renderAdversarial(b *strings.Builder, v verdictDoc) {
	b.WriteString("## Leg 3 — Adversarial review\n\n")
	b.WriteString("Conducted fresh-context and off-policy by an evaluator that did not carry\n")
	b.WriteString("out the research and received the idea as an artefact of unknown\n")
	b.WriteString("authorship — the evaluator-outside-the-loop principle applied to ideas.\n\n")
	for _, k := range v.kills {
		fmt.Fprintf(b, "- **%s** — %s\n", k.Outcome, k.Attempt)
	}
	b.WriteString("\n")
}

// renderRejected writes the alternatives turned down. A deliberate absence is
// stated in the same words the payload's explicit marker declares, so a reader
// can tell "none were considered" from "nobody wrote them down".
func renderRejected(b *strings.Builder, v verdictDoc) {
	b.WriteString("## Rejected alternatives\n\n")
	if v.noneRejected {
		b.WriteString("None were considered. The absence is recorded deliberately, so a later\n")
		b.WriteString("session knows nothing was weighed and discarded here.\n\n")
		return
	}
	// A list item is not a table cell: escaping its pipes would print a visible
	// backslash for no gain, since nothing here can start a table.
	for _, r := range v.rejected {
		fmt.Fprintf(b, "- **%s** — %s\n", r.Alternative, r.WhyRejected)
	}
	b.WriteString("\n")
}

// renderOutcome states what the verdict means for the idea's future, so the
// record answers the question its next reader arrives with.
func renderOutcome(b *strings.Builder, v verdictDoc) {
	b.WriteString("## What follows\n\n")
	switch v.verdict {
	case VerdictSurvives:
		b.WriteString("The idea survives the gauntlet and may graduate to a draft intent through\n")
		b.WriteString("the ordinary quoted-text create (`abcd intent \"<text>\"`). Ideate mints no\n")
		b.WriteString("intent itself; graduation stays a deliberate act.\n")
	case VerdictKilled:
		b.WriteString("The idea is closed. Before proposing it again, read this record: the\n")
		b.WriteString("findings above are why it did not survive, and re-proposing it costs\n")
		b.WriteString("nothing only if something above has changed.\n")
	case VerdictReframed:
		b.WriteString("The idea as posed does not survive, but the reframing recorded above\n")
		b.WriteString("does. Any graduation to a draft intent carries the reframing, not the\n")
		b.WriteString("original wording.\n")
	}
}
