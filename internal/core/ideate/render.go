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
// cell() closes the first; the line-cleaning and the fixed prefixes close the rest.
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
	fmt.Fprintf(&b, "%s\n\n", blockText(v.idea))

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
		fmt.Fprintf(b, "| %s | %s | %s |\n", cell(c.Claim), cell(c.PrimarySource), c.Status)
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
		fmt.Fprintf(b, "| %s | %s | %s |\n", h.Record, h.Relation, cell(note))
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

// cell escapes a value for a markdown table cell.
//
// The BACKSLASH is escaped alongside the pipe, and it is not optional. GFM reads
// `\|` inside a cell as an escaped delimiter, so escaping the pipe alone turns a
// prose `\|` into `\\|` — an escaped backslash followed by a LIVE delimiter. That
// splits the claim into extra columns and pushes the core-owned Finding column off
// the end of the row, where GFM silently drops it: the record would then show
// whatever word the prose put in the third position instead of the status the
// payload declared. One replacer does both in a single pass, so the pipe's own
// escape can never be re-escaped.
var cellEscaper = strings.NewReplacer(`\`, `\\`, "|", `\|`)

func cell(s string) string { return cellEscaper.Replace(s) }

// blockText escapes a value that starts a line of its own. Every other untrusted
// field in this document sits behind a fixed prefix ("| ", "- ", "**"), so its
// first character cannot begin a block; the idea is the one field rendered as a
// bare paragraph, where a leading marker WOULD open a heading, a list, a quote, or
// a code fence. A backslash is markdown's own escape for exactly these.
//
// The value arrives already cleaned by termsafe, and the cleaner's code-span
// exemption rests on the field being parsed as the exact string it was cleaned as
// (see the invariant note in internal/termsafe/prose.go). Escaping a leading
// backtick unconditionally broke that: it kills the span the cleaner relied on
// and republishes its sheltered content as live markup — a quoted `<details>`
// became a real disclosure widget, concealing every later section of the record.
// A leading run that opens a BALANCED span opens no block (a backtick fence's
// info string may not contain backticks, so a run with a matching closer on the
// same line is an inline span by construction), so only an unbalanced run is
// escaped — and the cleaner no longer emits one.
func blockText(s string) string {
	if s == "" {
		return s
	}
	if s[0] == '`' {
		if termsafe.OpensBalancedCodeSpan(s) {
			return s
		}
		return `\` + s
	}
	switch s[0] {
	// '[' is here for a subtler reason than the rest: a paragraph shaped like a
	// link reference definition (`[x]: https://…`) is CONSUMED by CommonMark and
	// renders as nothing at all — so an idea in that shape would erase the record's
	// own subject while the verdict and the three legs still read normally.
	case '#', '-', '*', '+', '>', '|', '~', '=', '_', '[':
		return `\` + s
	}
	// An ordered-list opener ("1. ", "12) ") is the one multi-character marker.
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			continue
		}
		if i > 0 && (s[i] == '.' || s[i] == ')') {
			return `\` + s
		}
		break
	}
	return s
}
