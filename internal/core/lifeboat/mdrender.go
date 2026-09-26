package lifeboat

import (
	"strings"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// mdrender.go — the one render discipline for every markdown file the lifeboat
// writes (principles.md, press-release.md, review/review-<manifest12>.md and the
// packed brief section docs). It is the discipline the memory renderers got for
// iss-2609020539188868, applied to the lifeboat half (iss-2609251355497247):
//
//   - every untrusted field on a markdown line goes through the file-write
//     cleaner, termsafe.CleanProse, never Sanitize alone — Sanitize defangs a
//     terminal and leaves an HTML comment opener or link syntax live;
//   - no renderer wraps a cleaned value in a delimiter of its own (brackets,
//     emphasis, backticks): termsafe's guarantees hold over the exact string it
//     returned, and a wrapper the value can close parses a different string;
//     where a delimiter is wanted, termsafe.CodeSpan picks one the value cannot
//     close, without altering the value's bytes;
//   - a value that begins a block has its leading marker escaped, so a field
//     cannot turn itself into a heading, a list, a quote, a fence, a table or
//     a link reference definition.
//
// Re-cleaning a field its ingest already cleaned is a no-op (the cleaner is
// idempotent), so this costs a well-formed record nothing and holds for any
// field that reaches a renderer by another route.

// maxMDFieldBytes is the render-time cap: the largest any lifeboat field is
// cleaned to at ingest (the press-release body), so the render never cuts a
// field its ingest kept.
const maxMDFieldBytes = maxPressReleaseBodyBytes

// mdInline is an untrusted field placed mid-line: cleaned, nothing added.
func mdInline(s string) string { return termsafe.CleanProse(s, maxMDFieldBytes) }

// mdCode is an untrusted field set off as a code span whose fence the value
// cannot close. An empty value renders as nothing.
func mdCode(s string) string { return termsafe.CodeSpan(mdInline(s)) }

// mdCodeList renders refs as comma-separated code spans.
func mdCodeList(refs []string) string {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		if c := mdCode(r); c != "" {
			out = append(out, c)
		}
	}
	return strings.Join(out, ", ")
}

// mdBlock is an untrusted field that begins a block (a paragraph, a quote's
// first line): cleaned, then its leading marker escaped.
func mdBlock(s string) string { return escapeLeadingMarker(mdInline(s)) }

// escapeLeadingMarker backslash-escapes the character that would make s open a
// block construct rather than a paragraph: an ATX heading (#), a bullet (- * +),
// a block quote (>), a fence (` ~), a table row (|), a thematic break or setext
// underline (- _ * =), raw HTML (<), a link reference definition ([), or an
// ordered-list marker (digits then . or )). CommonMark renders a
// backslash-escaped ASCII punctuation character as the character itself, so the
// reader sees the value's text unchanged.
//
// The bracket is the subtle one: a paragraph shaped like `[label]: <dest>` is
// consumed as a definition and renders as nothing, and it arms a `[label]`
// shortcut reference in any other field, which the cleaner leaves alone, as a
// live link to the attacker's destination (iss-2609262237352137). ideate's
// blockText escapes it for the same reason.
func escapeLeadingMarker(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '#', '-', '*', '+', '>', '`', '~', '|', '=', '_', '<', '[':
		return `\` + s
	}
	digits := 0
	for digits < len(s) && digits < 10 && s[digits] >= '0' && s[digits] <= '9' {
		digits++
	}
	if digits > 0 && digits < len(s) && (s[digits] == '.' || s[digits] == ')') {
		return s[:digits] + `\` + s[digits:]
	}
	return s
}
