// Package termsafe holds the one canonical sanitiser for a string built from
// untrusted content (commit subjects, refs, file paths, repo prose, error text
// echoing a malformed file) before it is written to a terminal or a human report.
// It is the single primitive every render path shares — the terminal analogue of
// fsutil's guarded read: a hostile or archived repository controls this text, and
// left raw it can spoof, corrupt, or visually rewrite the report.
package termsafe

import "strings"

// Sanitize replaces every terminal-display attack rune with a visible '?' (tab
// becomes a space). It neutralises:
//   - C0 controls (<0x20) and DEL (0x7f) — these carry ESC, so a raw ANSI escape
//     in a commit subject could recolour, move the cursor, or corrupt the report;
//     a newline is masked too, so an injected line break cannot forge extra lines;
//   - the C1 range (0x80–0x9F) — U+009B (CSI) acts like ESC[ on an 8-bit terminal,
//     so masking ESC (a C0 control) alone would leave an equivalent path open;
//   - bidirectional override/isolate controls (the "Trojan Source" class) and
//     zero-width characters, which reorder or hide text so the rendered line
//     differs from the bytes — the reader sees something the file does not say.
//
// JSON output is NOT covered by encoding/json's own escaping: it escapes only
// C0 (below 0x20) and U+2028/9 — DEL, the C1 range, bidi overrides and
// zero-width runes pass through raw (pinned by TestJSONLeavesC1AndBidiRaw).
// A JSON surface whose strings reach a terminal or a committed record needs
// its values shape-checked or encoded at the boundary that produced them
// (iss-359); this function is the mask for the human/terminal render path.
func Sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\t':
			return ' '
		case r < 0x20 || r == 0x7f:
			return '?'
		case r >= 0x80 && r <= 0x9f:
			return '?'
		case isBidiControl(r) || isZeroWidth(r):
			return '?'
		}
		return r
	}, s)
}

// SanitizeBlock sanitises multi-line text while keeping its line structure: each
// line is sanitised on its own and the line terminators are preserved. Sanitize
// masks a newline (an injected line break must not forge extra report lines),
// which is right for a value interpolated into one line and wrong for output
// that IS lines — a rendered diff or a quoted file excerpt, where flattening the
// terminators destroys the artefact the reader needs. Use this only where the
// line breaks are the render's own, never for a single untrusted value.
//
// The carriage return of a CRLF pair survives; every other carriage return is
// masked. That distinction is load-bearing in both directions: a BARE CR moves
// the cursor to the column zero of the line being written, so it can overprint
// what the reader already saw — the attack Sanitize masks — whereas the CR of a
// CRLF is immediately committed by its newline and can overprint nothing. And a
// rendered patch against a CRLF file must carry those pairs or no patch tool
// will apply it, so dropping them would silently void the artefact.
func SanitizeBlock(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		// A trailing carriage return is half of a CRLF pair only when a newline
		// actually follows it — that is, on every element except the last, which
		// by construction has no newline after it. A trailing CR there is bare
		// and is masked like any other.
		if i < len(lines)-1 && strings.HasSuffix(l, "\r") {
			lines[i] = Sanitize(l[:len(l)-1]) + "\r"
			continue
		}
		lines[i] = Sanitize(l)
	}
	return strings.Join(lines, "\n")
}

// SanitizeAll sanitises every member of a slice, returning a new slice.
func SanitizeAll(in []string) []string {
	if in == nil {
		return nil
	}
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = Sanitize(s)
	}
	return out
}

// IsHidden reports whether r is a bidirectional control or a zero-width rune:
// a rune that makes rendered text read differently from its bytes, or hides
// text altogether. It is the predicate Sanitize masks these with, exported so a
// parser that refuses such runes outright judges them by the same set.
func IsHidden(r rune) bool { return isBidiControl(r) || isZeroWidth(r) }

// isBidiControl reports whether r is a Unicode bidirectional override/embedding/
// isolate or directional mark — the runes a "Trojan Source" attack uses to make a
// rendered line read differently from its bytes. Code points are written
// numerically so this source file carries none of the invisible characters it
// defends against.
func isBidiControl(r rune) bool {
	switch {
	case r >= 0x202A && r <= 0x202E: // LRE RLE PDF LRO RLO
		return true
	case r >= 0x2066 && r <= 0x2069: // LRI RLI FSI PDI
		return true
	case r == 0x200E || r == 0x200F: // LRM RLM
		return true
	case r == 0x061C: // Arabic letter mark
		return true
	}
	return false
}

// isZeroWidth reports whether r is a zero-width character that can hide or splice
// text invisibly. It covers ZWSP/ZWNJ/ZWJ and the BOM/ZWNBSP (U+FEFF), the WORD
// JOINER (U+2060) that Unicode designates as U+FEFF's non-BOM successor, the
// invisible-operator block (U+2061–U+2064), and SOFT HYPHEN (U+00AD) — each
// renders zero-width, so two distinct byte strings can display identically.
func isZeroWidth(r rune) bool {
	switch {
	case r == 0x200B || r == 0x200C || r == 0x200D || r == 0xFEFF:
		return true
	case r >= 0x2060 && r <= 0x2064: // WORD JOINER + invisible operators
		return true
	case r == 0x00AD: // SOFT HYPHEN
		return true
	}
	return false
}
