package termsafe

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// EncodeHiddenRunes percent-encodes every rune the terminal sanitizer would mask
// — C0/DEL, the 2-byte-encoded C1 range, bidi overrides and zero-width runes — so
// a value built from an untrusted source (a redirect-supplied final address, say)
// is recorded losslessly but can no longer smuggle terminal escapes or
// Trojan-Source reordering into a JSON surface or a committed baseline. This is
// the canonical encoder for the JSON/record boundary that Sanitize's doc note
// points at (iss-359); a value the terminal render path masks with '?' is encoded
// here instead, so the byte is preserved rather than substituted.
//
// It is lossless in both directions a mask is not: a hidden rune is percent-encoded
// as its UTF-8 bytes, and a byte that is not valid UTF-8 — which strings.Map would
// silently rewrite to U+FFFD — is percent-encoded raw. A canonical address never
// carries these runes (net/url rejects C0 outright and percent-encodes the path
// itself), so encoding them cannot break a legitimate final URL.
func EncodeHiddenRunes(s string) string {
	if Sanitize(s) == s && utf8.ValidString(s) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && width == 1 {
			// An invalid byte, which strings.Map would silently rewrite to
			// U+FFFD — percent-encode the RAW byte so the record stays lossless
			// on non-UTF-8 input too.
			fmt.Fprintf(&b, "%%%02X", s[i])
			i++
			continue
		}
		rs := s[i : i+width]
		if Sanitize(rs) != rs {
			for j := 0; j < width; j++ {
				fmt.Fprintf(&b, "%%%02X", rs[j])
			}
		} else {
			b.WriteString(rs)
		}
		i += width
	}
	return b.String()
}

// EncodeHiddenRunesBlock is EncodeHiddenRunes for multi-line prose bound for a
// committed record — a capture body, a resolution note, a press release. It
// encodes every rune EncodeHiddenRunes encodes EXCEPT the three that are the
// prose's own structure: the line feed, the tab, and a carriage return that is
// half of a CRLF pair. A bare carriage return is encoded, because it is the
// terminal overwrite a record must not carry (iss-2608301206073609).
//
// It is SanitizeBlock's counterpart at the record boundary: the render path masks,
// the record path encodes, and both keep the line structure the prose was
// written with.
func EncodeHiddenRunesBlock(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		cr := ""
		if i < len(lines)-1 && strings.HasSuffix(l, "\r") {
			l, cr = l[:len(l)-1], "\r"
		}
		cells := strings.Split(l, "\t")
		for j, c := range cells {
			cells[j] = EncodeHiddenRunes(c)
		}
		lines[i] = strings.Join(cells, "\t") + cr
	}
	return strings.Join(lines, "\n")
}
