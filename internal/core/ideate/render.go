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

	renderOrigins(&b, v)
	renderResearch(&b, v)
	renderGrill(&b, v)
	renderAdversarial(&b, v)
	renderRejected(&b, v)
	renderReframing(&b, v)
	renderOutcome(&b, v)

	return b.String()
}

// renderOrigins writes the sources the idea came from, which is what a quoted
// claim was checked against. An idea with none says so, in the words the
// explicit empty list declares.
func renderOrigins(b *strings.Builder, v verdictDoc) {
	b.WriteString("## Where the idea came from\n\n")
	if len(v.origins) == 0 {
		b.WriteString("No source: the idea entered the gauntlet with nothing behind it but its own words.\n\n")
		return
	}
	// A list item's text can still open a block ("- # x" is a heading inside
	// the item), so each is escaped as the start of a block.
	for _, o := range v.origins {
		fmt.Fprintf(b, "- %s\n", termsafe.BlockText(o))
	}
	b.WriteString("\n")
}

// renderResearch writes leg 1 as the claims table: one row per load-bearing
// claim, the primary source it was checked against, and what that established.
func renderResearch(b *strings.Builder, v verdictDoc) {
	b.WriteString("## Leg 1 — Primary-source research\n\n")
	b.WriteString("Every load-bearing claim checked against its primary source, never a\n")
	b.WriteString("secondary citation. A claim marked quoted was checked only against the\n")
	b.WriteString("source the idea came from, which proves the quotation and not the claim.\n\n")
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
	} else {
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
	renderSearches(b, v)
}

// renderSearches writes what the grill searched, so a reader can tell an empty
// grill from a broken one: the terms, the places, and how many entries each
// search returned. A grill with hits may keep no search record, and says so.
func renderSearches(b *strings.Builder, v verdictDoc) {
	b.WriteString("### Searches\n\n")
	if len(v.searches) == 0 {
		b.WriteString("The grill recorded no searches; its findings are the cited hits above.\n\n")
		return
	}
	b.WriteString("| Terms | Places | Count |\n|---|---|---|\n")
	for _, s := range v.searches {
		fmt.Fprintf(b, "| %s | %s | %d |\n",
			termsafe.TableCell(strings.Join(s.Terms, "; ")), termsafe.TableCell(strings.Join(s.Places, "; ")), *s.Count)
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
	b.WriteString("authorship — the evaluator-outside-the-loop principle applied to ideas.\n")
	b.WriteString("Each reviewer's attempts, verdict and sources are kept under its own label.\n\n")
	// The label sits after a fixed prefix, so it cannot open a block of its own.
	for _, r := range v.reviewers {
		fmt.Fprintf(b, "### Reviewer %s — %s\n\n", r.Label, r.Verdict)
		for _, k := range r.KillAttempts {
			fmt.Fprintf(b, "- **%s** — %s\n", k.Outcome, k.Attempt)
		}
		if len(r.Sources) == 0 {
			b.WriteString("\nSources consulted: none named.\n\n")
		} else {
			fmt.Fprintf(b, "\nSources consulted: %s.\n\n", strings.Join(r.Sources, ", "))
		}
	}
	renderRuling(b, v)
}

// renderRuling writes the ruling that settled a split between reviewers. Where
// the reviewers agreed and nobody ruled, the agreement is stated, so a reader
// can tell "no split" from "a split nobody settled" (which the validator
// refuses).
func renderRuling(b *strings.Builder, v verdictDoc) {
	if v.ruling == nil {
		if len(v.reviewers) > 1 {
			b.WriteString("The reviewers agree, so no ruling was needed.\n\n")
		}
		return
	}
	b.WriteString("### Ruling\n\n")
	fmt.Fprintf(b, "**%s**, ruled by %s — %s\n\n", v.ruling.Verdict, v.ruling.By, v.ruling.Reason)
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

// renderReframing writes a reframed idea's surviving shape in a section of its
// own, which is what the outcome below tells a later reader to graduate.
func renderReframing(b *strings.Builder, v verdictDoc) {
	if v.verdict != VerdictReframed {
		return
	}
	b.WriteString("## The reframing\n\n")
	fmt.Fprintf(b, "%s\n\n", termsafe.BlockText(v.reframing))
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
