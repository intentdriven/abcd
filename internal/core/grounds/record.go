package grounds

// record.go is the RECORD FORM of a recorded ground: the `## Grounds` section, the
// reader that finds the entries in it, and the append that puts one there.
//
// It lives beside the vocabulary for the reason the vocabulary lives here at
// all. Both record families accumulate grounds — the intent record at its
// readiness gate, the issue record at each of its three triage routes — and the
// shape is a schema decision, not a per-family convenience. Spelled twice, the
// two halves drift: the ledger's grounds began as a single frontmatter scalar
// while the intent's were an append-only section, so a resolve overwrote the
// conjecture the promote before it had recorded, silently and with a success
// result (iss-2608301657354776). One definition of how grounds accumulate is
// what makes that class unrepresentable.
//
// Recording is APPEND-ONLY. The earlier conjecture is precisely what a later
// reader checks the outcome against, so rewriting it would leave the record
// saying only what was believed last — which is the evaporation this whole
// argument exists to close, moved one step later.

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/mdrecord"
)

// Body returns the part of a record FILE the `## Grounds` section lives in —
// everything after the leading frontmatter block. It is the ONE answer the
// writer and every reader ask, so the bytes a write is judged over are the bytes
// a reader will consult.
//
// Handed the whole file instead, a reader looks for its heading in the
// frontmatter too, where `# Grounds` is a legal YAML comment the block parser
// skips and an ATX heading pattern matches. Writer and reader could then agree
// an entry had landed while disagreeing about where (iss-2608301805069999).
//
// A text that carries no frontmatter is already a body and comes back unchanged,
// which is what lets a caller holding either shape ask.
func Body(content string) string {
	_, body := frontmatter.Split(content)
	return body
}

// Heading is the section a record carries its grounds under. It is spelled once
// for the two that have to agree: the writer that creates the section and the
// reader that locates it, across both record families. Record-lint names the
// section in a remedy as literal prose, because core/lint does not import this
// package and one message is not a reason to (iss-2608301836222858).
const Heading = "Grounds"

// headingRe matches the `## Grounds` heading at any heading depth.
var headingRe = regexp.MustCompile(`^#{1,6}\s+` + Heading + `\s*$`)

// ParseSection reads a record's recorded grounds, in the order they were
// written. It is the single reader every consumer asks, so no two of them can
// disagree about what an entry is.
//
// A bullet that does not parse is not an entry: it is prose under the heading,
// and reporting it as a malformed ground would put a gate verdict on a sentence
// somebody wrote for a human.
//
// It checks the GRAMMAR and stops there, which is what lets the ledger use it. A
// wontfix stamps its grounds from a reason whose own contract is merely
// non-empty, so a floor here would make the reader skip entries the ledger has
// always accepted and a gate then report no recorded grounds about a record that
// visibly carries one. A consumer that claims the floor asks for
// ParseSectionAboveFloor instead.
func ParseSection(content string) []Grounds {
	lines := strings.Split(content, "\n")
	mask := mdrecord.Mask(lines)
	start, end, ok := mdrecord.SectionLineRangeIn(lines, mask, headingRe)
	if !ok {
		return nil
	}
	var out []Grounds
	for _, b := range mdrecord.BulletBlocks(lines, mask, start, end) {
		g, err := Parse(blockText(lines, b))
		if err != nil {
			continue
		}
		out = append(out, g)
	}
	return out
}

// ParseSectionAboveFloor is ParseSection keeping only the entries that also
// clear the SUBSTANCE FLOOR, so a consumer that CLAIMS the floor holds a
// hand-typed bullet to exactly what New holds a supplied one to. Reading the
// grammar alone let `- pursued: yes` satisfy the intent readiness check, which
// claims the floor and would then have been enforcing only a colon
// (iss-2608300930057882).
//
// It is expressed on top of ParseSection rather than beside it: what an ENTRY is
// stays one definition, and this adds a filter over it rather than a second
// reading of the section.
func ParseSectionAboveFloor(content string) []Grounds {
	entries := ParseSection(content)
	out := entries[:0:0]
	for _, g := range entries {
		if ValidateText(g.Text) != nil {
			continue
		}
		out = append(out, g)
	}
	return out
}

// blockText folds one top-level bullet — and the continuation lines wrapped into
// it — back into the single line the grammar is written on.
func blockText(lines []string, b mdrecord.BulletBlock) string {
	parts := make([]string, 0, b.End-b.Start)
	for i := b.Start; i < b.End; i++ {
		ln := strings.TrimRight(lines[i], "\r")
		if i == b.Start {
			ln = mdrecord.TrimBulletPrefix(ln)
		}
		parts = append(parts, ln)
	}
	return Fold(strings.Join(parts, " "))
}

// AppendToRecord appends one grounds entry to a record FILE, creating the
// `## Grounds` section when it is absent, and returns the file to write.
//
// The append and its read-back run over the record BODY, and the result is
// spliced back onto the frontmatter unchanged. That is what makes the guard a
// property of the section the reader will actually consult rather than of the
// whole file: judged over the whole file, the writer matched a frontmatter
// `# Grounds` comment as its heading, wrote the bullet there, read it back
// happily, and reported success about a value the body reader could not see
// (iss-2608301805069999).
//
// It refuses a write that would leave the record LESS readable than it was
// (iss-2608300927577980) rather than performing it. A grounds text is operator
// prose, and prose can carry an unclosed `<!--`. The comment mask then runs to
// end of file: the entry disappears, every line after it disappears with it, and
// the result still reports success, so nothing anywhere says the record was
// blinded. The check is two questions asked of the bytes about to be written,
// before they are written, rather than a pattern ban on the text: a text that
// leaves a comment open, and a write whose entry does not actually arrive.
//
// The count question subsumes more than the comment one: a bullet swallowed by a
// fence, by a mask this reader does not yet model, or by anything else, fails to
// raise the count and is refused for the same reason. The comment question is
// still asked separately because it is the one whose remedy the caller can act
// on — "close the comment, or drop the marker" — and a refusal that can name the
// cause is worth more than one that can only say the entry did not arrive.
//
// The refusal names no record family; the caller wraps it with its own.
func AppendToRecord(content string, g Grounds) (string, error) {
	if mdrecord.OpensComment(g.Bullet()) {
		return "", fmt.Errorf(
			"the grounds text leaves an HTML comment open (`<!--` with no `-->`); "+
				"written, it would hide the entry and every line below it from every reader of this record — "+
				"close the comment or drop the marker; nothing written (text: %q)", g.Text)
	}
	head, body := frontmatter.Split(content)
	// Two live headings leave no single section to append to. The reader takes
	// the FIRST, so every entry under a second is invisible to every surface, and
	// a writer that picks one is guessing which the operator meant. The intent
	// half refuses to stamp an ambiguous `## Scope Conditions` on the same
	// ground; this is grounds' half of it (iss-2608301803425790). The READER
	// still takes the first and says nothing — a reader with no verdict to
	// report has nowhere to raise this, so the refusal lives at the write.
	bodyLines := strings.Split(body, "\n")
	if n := mdrecord.CountHeadings(bodyLines, mdrecord.Mask(bodyLines), headingRe); n > 1 {
		return "", fmt.Errorf(
			"the record's body carries %d live `## %s` headings and the reader takes the first, so "+
				"entries under the others are invisible to every surface; merge them into one section "+
				"before recording another; nothing written", n, Heading)
	}
	updated := appendBullet(body, g)
	if want, got := len(ParseSection(body))+1, len(ParseSection(updated)); got != want {
		return "", readBackRefusal(body, got, want)
	}
	return head + updated, nil
}

// readBackRefusal explains an entry that did not read back. The count question
// is deliberately broad — a bullet swallowed by anything at all fails to raise
// the count — so the bare count is the honest thing to say when nothing more is
// known. When something more IS known, saying only the count misattributes:
// naming the appended entry sends the operator to rewrite the one part of the
// record that cannot be at fault, and this cycle's standing defect class is
// exactly that (iss-2608301803423101).
//
// What is known is what Mask already had to work out. An opener nothing closes
// runs to end of file, so every line below it — including a line appended at the
// end — is masked, and the reader skips it. The opener is in the record AS READ,
// before the append: a grounds bullet cannot be one, because a comment opener in
// its text is refused above and a fence delimiter must start its line, which a
// bullet's `- ` prefix rules out. So the line this names is the operator's to
// close, and the grounds text is not.
//
// The line is counted from the start of the BODY, and the opener's own text is
// quoted beside it. A file-relative number would be wrong: the triage verbs
// append after setting their note field, so the content in hand carries
// frontmatter lines the record on disk — the one the operator opens, since
// nothing is written — does not. The body is what those writes leave alone, so a
// body-relative count is the number that holds, and the quoted text is what the
// operator can search for either way.
func readBackRefusal(body string, got, want int) error {
	lines := strings.Split(body, "\n")
	if i, flag, ok := mdrecord.Unclosed(lines); ok {
		return fmt.Errorf(
			"the record's body leaves %s open: the opener is body line %d, %q. An unclosed opener runs "+
				"to end of file, so every line below it — the appended entry included — is masked and "+
				"does not read back. Close it or remove it; the grounds text is not the fault; nothing written",
			maskConstruct(flag), i+1, strings.TrimRight(lines[i], "\r"))
	}
	return fmt.Errorf(
		"the appended grounds entry does not read back (%d entries after the append, expected %d); "+
			"nothing written", got, want)
}

// maskConstruct names a mask flag the way the record spells it, so the operator
// searches the file for the thing the message named.
func maskConstruct(flag uint8) string {
	if flag&mdrecord.MaskComment != 0 {
		return "an HTML comment (`<!--` with no `-->`)"
	}
	return "a fenced code block"
}

// appendBullet puts one entry at the end of the record's `## Grounds` section,
// creating the section at end of file when it is absent. It performs the
// trailing link-reference peel: a `[ref]: url` definition parked at the end of a
// section belongs below its prose, and appending under it would detach the entry
// from the section it belongs to.
func appendBullet(content string, g Grounds) string {
	lines := strings.Split(content, "\n")
	mask := mdrecord.Mask(lines)
	start, end, ok := mdrecord.SectionLineRangeIn(lines, mask, headingRe)
	if !ok {
		body := strings.TrimRight(content, "\n")
		return body + "\n\n## " + Heading + "\n\n" + g.Bullet() + "\n"
	}
	section := append([]string{}, lines[start:end]...)
	for len(section) > 0 && strings.TrimSpace(section[len(section)-1]) == "" {
		section = section[:len(section)-1]
	}
	trailingRefs := mdrecord.PeelTrailingLinkRefs(&section)
	rebuilt := make([]string, 0, len(lines)+6)
	rebuilt = append(rebuilt, lines[:start]...)
	rebuilt = append(rebuilt, section...)
	if len(section) > 0 && strings.TrimSpace(section[len(section)-1]) != "" &&
		!mdrecord.IsTopLevelBullet(strings.TrimRight(section[len(section)-1], "\r")) {
		// Prose immediately above the first entry needs its blank line; a run of
		// bullets is one list and must not be split by one.
		rebuilt = append(rebuilt, "")
	}
	rebuilt = append(rebuilt, g.Bullet())
	if len(trailingRefs) > 0 {
		rebuilt = append(rebuilt, "")
		rebuilt = append(rebuilt, trailingRefs...)
	}
	rebuilt = append(rebuilt, "")
	rebuilt = append(rebuilt, lines[end:]...)
	return strings.Join(rebuilt, "\n")
}
