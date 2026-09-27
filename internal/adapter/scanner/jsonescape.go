package scanner

import (
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// jsonescape.go — the JSON-escape decoded views of a line
// (iss-2609261647358395, iss-2609251639263391).
//
// The transcript store scans raw JSONL, so every string on the lines it hands
// the scanner is JSON-escaped, and an escape changes what the detectors see
// without changing what the text says. Three shapes were invisible on the raw
// line:
//
//   - a control escape before a value. Every bundled token pattern anchors on
//     a leading \b, and in "\nghp_…" the byte before the token is the escape's
//     'n', a word byte, so the anchor never holds; the home-path anchor reads
//     the same letter as a path byte and declines "\n/home/<name>".
//   - an escape after a value. The home-path trailing boundary has no
//     backslash in its set, so "/home/<name>\n" and "\"/home/<name>\"" were
//     declined as a name that goes on.
//   - an escaped separator or letter inside a value. "\/home\/<name>" and
//     "\u002fhome\u002f<name>" spell a home path with no '/' on the line, and
//     an ASCII-only encoder (Python json's default) writes a non-ASCII name as
//     "\u00e9", which no configured name matches.
//
// Rather than teach every detector every escape, each line with an escape is
// decoded one JSON layer at a time and each layer is scanned as a view of the
// line, exactly as the percent-decode pre-pass scans its decoded copy: every
// hit is mapped back to the raw bytes it came from, so Redact masks the live
// spelling on disk. Decode-equivalent spellings therefore match by
// construction, for every detector at once.
//
// EVERY layer is scanned, not only the last. A tool result that is itself
// JSON is escaped again by the transcript line that quotes it, so the second
// layer is where its strings read plainly; but a literal backslash the first
// layer uncovers (a Windows path, a regex) is read again by the next as an
// escape — "C:\Users\bob" decodes "\b" to a backspace — so the last layer alone
// can be wrong where an earlier one was right. The layers are bounded like the
// percent passes, and a layer that decodes nothing ends the walk.
//
// The percent spelling is the percent pre-pass's business (percent.go), and
// the two do not compose: a JSON escape never carries a '%' sequence that a
// percent decode would need unescaped first, and a percent-encoded JSON escape
// is not a spelling any encoder that feeds the scanner writes.

// maxJSONDecodeLayers bounds the JSON-unescape walk: one layer for a
// transcript line, a second for JSON quoted inside it (a tool result), a third
// for slack. Each layer strictly shrinks the line, so the walk ends early on
// ordinary input.
const maxJSONDecodeLayers = 3

// jsonEscapeLayers returns the JSON-decoded views of s, outermost first, each
// with its position map back to s (the percentDecodeBounded shape: posMap[i]
// is the offset in s at which decoded byte i began, with a sentinel at
// len(decoded)). It returns nil when s carries no JSON escape.
func jsonEscapeLayers(s string) []decodedView {
	if strings.IndexByte(s, '\\') < 0 {
		return nil
	}
	var out []decodedView
	cur := s
	var m []int // offsets in cur -> offsets in s; nil means identity
	for layer := 0; layer < maxJSONDecodeLayers; layer++ {
		next, step, changed := jsonUnescapeOnce(cur)
		if !changed {
			break
		}
		composed := make([]int, len(next)+1)
		for i := range composed {
			if m == nil {
				composed[i] = step[i]
			} else {
				composed[i] = m[step[i]]
			}
		}
		out = append(out, decodedView{text: next, posMap: composed})
		cur, m = next, composed
	}
	return out
}

// decodedView is one decoded spelling of a raw line and its position map.
type decodedView struct {
	text   string
	posMap []int
}

// jsonUnescapeOnce decodes one layer of JSON string escapes in s — the two-byte
// escapes \" \\ \/ \b \f \n \r \t and the \uXXXX escape, a UTF-16 surrogate
// pair combined into one rune — and copies everything else as it stands: a
// backslash before any other byte is not a JSON escape (a Windows path read
// at the wrong depth, a regex) and is kept, as is a lone surrogate, which no
// rune stands for. The map sends each decoded byte to the offset in s its
// escape began at, with a trailing sentinel == len(s).
func jsonUnescapeOnce(s string) (string, []int, bool) {
	scanMeter.charge(stageJSONEscape, len(s))
	b := make([]byte, 0, len(s))
	pos := make([]int, 0, len(s)+1)
	changed := false
	emit := func(at int, bs ...byte) {
		for _, c := range bs {
			pos = append(pos, at)
			b = append(b, c)
		}
	}
	for i := 0; i < len(s); {
		if s[i] != '\\' || i+1 >= len(s) {
			emit(i, s[i])
			i++
			continue
		}
		if c, ok := jsonShortEscape(s[i+1]); ok {
			emit(i, c)
			i += 2
			changed = true
			continue
		}
		if r, n := jsonUnicodeEscape(s[i:]); n > 0 {
			var enc [utf8.UTFMax]byte
			emit(i, enc[:utf8.EncodeRune(enc[:], r)]...)
			i += n
			changed = true
			continue
		}
		emit(i, s[i])
		i++
	}
	pos = append(pos, len(s))
	return string(b), pos, changed
}

// jsonShortEscape returns the byte a two-byte JSON escape stands for.
func jsonShortEscape(c byte) (byte, bool) {
	switch c {
	case '"', '\\', '/':
		return c, true
	case 'b':
		return '\b', true
	case 'f':
		return '\f', true
	case 'n':
		return '\n', true
	case 'r':
		return '\r', true
	case 't':
		return '\t', true
	}
	return 0, false
}

// jsonUnicodeEscape decodes the \uXXXX escape at the start of s, or a
// surrogate pair written as two of them, returning the rune and the bytes it
// consumed; n == 0 when s does not start with one that stands for a rune.
func jsonUnicodeEscape(s string) (rune, int) {
	u, ok := hex4(s)
	if !ok {
		return 0, 0
	}
	r := rune(u)
	if !utf16.IsSurrogate(r) {
		return r, 6
	}
	if lo, ok := hex4(s[6:]); ok {
		if pair := utf16.DecodeRune(r, rune(lo)); pair != utf8.RuneError {
			return pair, 12
		}
	}
	return 0, 0
}

// hex4 reads "\uXXXX" at the start of s.
func hex4(s string) (uint16, bool) {
	if len(s) < 6 || s[0] != '\\' || s[1] != 'u' {
		return 0, false
	}
	var v uint16
	for _, c := range []byte(s[2:6]) {
		if !isHexDigit(c) {
			return 0, false
		}
		v = v<<4 | uint16(hexNibble(c))
	}
	return v, true
}
