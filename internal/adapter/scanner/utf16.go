package scanner

import (
	"sort"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// utf16.go — the UTF-16 view of a byte scan (iss-2609261827066511).
//
// A PDF text string that holds anything outside PDFDocEncoding is written as
// UTF-16 behind the FE FF byte-order mark, and a Windows writer stores its
// strings little-endian behind FF FE. Either way a zero byte stands between
// every two ASCII letters, so the byte scan, which reads the file as one long
// string, never saw a name, a home path or an email written that way, however
// long. The fix is a decode, not a format parser: every run of UTF-16 text a
// byte-order mark opens is decoded to UTF-8, the runs are joined one to a line
// into a single view, and the view is scanned with the same byte rules as the
// raw bytes. Each hit is mapped back to the raw offset its first unit sits at,
// so the person-key rule (metadataFields) judges a short name by the raw bytes
// before it — a PDF's "/Author (" stands right before the mark.
//
// A PDF writer may also spell the same string in hex, <FEFF005A...>, which
// never puts the UTF-16 bytes in the file at all. The hex pairs of such a
// string are decoded to bytes, each mapped to the raw offset of its first
// digit, and the bytes are handed to the same view (pdfHexView); that needs
// no escape grammar (iss-2609261909108726). A string written with octal
// escapes inside parentheses (\376\377...) needs the literal-string syntax,
// which pdfLiteralView (metaview.go) decodes before handing the bytes here.
//
// A run without a mark is not read here: telling UTF-16 from chance bytes
// needs the structure the text sits in, which is exifView's IFD walk for the
// EXIF XP* tags (metaview.go).

// minUTF16Run is the fewest code units a run needs to be read: a mark before
// a single unit is chance far more often than text.
const minUTF16Run = 2

// utf16View returns the UTF-16 runs a byte-order mark opens in data, decoded
// and joined one run to a line, with the position map of each decoded byte to
// the raw offset its code unit began at (a separator maps to the end of the
// run before it, and the sentinel to len(data)). ok is false when data holds
// no such run. Every raw byte is decoded at most once, so the view costs what
// the data does.
func utf16View(data []byte) (decodedView, bool) {
	var text []byte
	var pos []int
	for i := 0; i+1 < len(data); {
		var bigEndian bool
		switch {
		case data[i] == 0xfe && data[i+1] == 0xff:
			bigEndian = true
		case data[i] == 0xff && data[i+1] == 0xfe:
		default:
			i++
			continue
		}
		start, mark := len(text), i+2
		j := mark
		for j+1 < len(data) {
			r, width := utf16RuneAt(data, j, bigEndian)
			if width == 0 {
				break
			}
			var enc [utf8.UTFMax]byte
			for _, c := range enc[:utf8.EncodeRune(enc[:], r)] {
				text = append(text, c)
				pos = append(pos, j)
			}
			j += width
		}
		scanMeter.charge(stageUTF16, j-i)
		if (j-mark)/2 < minUTF16Run {
			text, pos = text[:start], pos[:start]
		} else {
			text = append(text, '\n')
			pos = append(pos, j)
		}
		i = j
	}
	if len(text) == 0 {
		return decodedView{}, false
	}
	return decodedView{text: string(text), posMap: append(pos, len(data))}, true
}

// utf16RuneAt decodes the character whose first code unit starts at data[j]
// and reports the bytes it spans: 2 for a text unit, 4 for a surrogate pair
// that makes a character outside the Basic Multilingual Plane (an emoji, a
// historic script, a supplementary CJK ideograph), and 0 for anything that
// ends a run. A valid pair continues the run, so a name after an emoji in one
// string is read (iss-2609261909101409); a lone surrogate, a pair spelling a
// noncharacter, and every unit utf16TextRune refuses still end it.
func utf16RuneAt(data []byte, j int, bigEndian bool) (rune, int) {
	unit := func(k int) rune {
		if bigEndian {
			return rune(data[k])<<8 | rune(data[k+1])
		}
		return rune(data[k+1])<<8 | rune(data[k])
	}
	r := unit(j)
	if utf16TextRune(r) {
		return r, 2
	}
	if r < 0xd800 || r >= 0xdc00 || j+3 >= len(data) {
		return 0, 0
	}
	pair := utf16.DecodeRune(r, unit(j+2))
	if pair == unicode.ReplacementChar || pair&0xfffe == 0xfffe {
		return 0, 0
	}
	return pair, 4
}

// utf16TextRune reports whether a code unit reads as text inside a run: the
// layout controls, printable Latin through the IPA block, and beyond it the
// letters, marks, digits, punctuation and spaces of any script. A control, a
// surrogate (no BMP rune on its own; utf16RuneAt reads a valid pair), a
// noncharacter and a symbol end the run.
// The symbol clause is what ends a big-endian PDF string: the ')' closing it
// pairs with the byte after it into a unit in the arrows block.
func utf16TextRune(r rune) bool {
	switch {
	case r == '\t' || r == '\n' || r == '\r':
		return true
	case r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) || utf16.IsSurrogate(r) || r >= 0xfffe:
		return false
	case r < 0x300:
		return true
	}
	return unicode.IsLetter(r) || unicode.IsMark(r) || unicode.IsDigit(r) || unicode.IsPunct(r) || unicode.IsSpace(r)
}

// pdfHexView returns the UTF-16 text of every PDF hex string in data whose
// bytes open with a byte-order mark: the digits between '<' and '>' (white
// space between them skipped, an odd last digit standing for its byte's high
// nibble) decoded to bytes and read by utf16View, with each decoded byte of
// the text mapped to the raw offset of the first hex digit of its code unit
// (a separator to the closing '>'). A '<' that opens a dictionary ("<<") or
// meets any other byte before its '>' opens no hex string, and the walk goes
// on from the byte that ended it, so every raw byte is read at most twice.
func pdfHexView(data []byte) (decodedView, bool) {
	var text []byte
	var pos []int
	pdfHexStrings(data, func(raw []byte, off []int) {
		if len(raw) < 2 || !(raw[0] == 0xfe && raw[1] == 0xff || raw[0] == 0xff && raw[1] == 0xfe) {
			return
		}
		v, ok := utf16View(raw)
		if !ok {
			return
		}
		for k := 0; k < len(v.text); k++ {
			text = append(text, v.text[k])
			pos = append(pos, off[v.posMap[k]])
		}
	})
	if len(text) == 0 {
		return decodedView{}, false
	}
	return decodedView{text: string(text), posMap: append(pos, len(data))}, true
}

// pdfHexStrings hands fn the bytes of every closed PDF hex string in data,
// with the raw offset of the first hex digit of each byte and, one entry past
// them, the offset of the closing '>'. The slices are reused between calls.
func pdfHexStrings(data []byte, fn func(raw []byte, off []int)) {
	var raw []byte
	var off []int
	for i := 0; i < len(data); i++ {
		if data[i] != '<' {
			continue
		}
		if i+1 < len(data) && data[i+1] == '<' {
			i++
			continue
		}
		raw, off = raw[:0], off[:0]
		hi, hiAt, closed := -1, 0, false
		j := i + 1
		for ; j < len(data); j++ {
			c := data[j]
			if c == '>' {
				closed = true
				break
			}
			if isPDFSpace(c) {
				continue
			}
			if !isHexDigit(c) {
				break
			}
			if hi < 0 {
				hi, hiAt = int(hexNibble(c)), j
				continue
			}
			raw = append(raw, byte(hi<<4)|hexNibble(c))
			off = append(off, hiAt)
			hi = -1
		}
		scanMeter.charge(stageUTF16, j-i)
		if !closed {
			i = j - 1
			continue
		}
		i = j
		if hi >= 0 {
			raw = append(raw, byte(hi<<4))
			off = append(off, hiAt)
		}
		if len(raw) == 0 {
			continue
		}
		fn(raw, append(off, j))
	}
}

// isPDFSpace reports whether c is one of the six bytes PDF reads as white
// space, which a hex string may carry between its digits.
func isPDFSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == 0
}

// byteViewFindings scans the decoded views of data with the byte rules and
// re-homes each finding onto the raw bytes: Line and Column name the raw
// position of the value's first source byte (meta's line starts), while
// Matched and the snippet stay the decoded text, so the short-name length rule
// counts the name's own bytes rather than its spelling on disk, and a
// serialized finding masks the name as it reads.
//
// The views are the UTF-16 runs a mark opens (as written, and as a PDF hex
// string spells them), then the metadata views (metaview.go): the EXIF text
// tags, the PDF literal strings, and the PDF hex strings with no mark. A
// metadata view adds only what nothing before it found: its finding at the
// same raw span and kind as one the raw scan or a UTF-16 view already made is
// the same finding, not a second one. The IPTC person datasets and exifView
// record their person spans in meta here, before any finding is judged.
func (s *Scanner) byteViewFindings(data []byte, id Identity, secrets []Pattern, logical string, meta *metadataFields, prior []Finding) []Finding {
	meta.persons = iptcPersonSpans(data, meta.persons)
	views := []func([]byte) (decodedView, bool){
		utf16View, pdfHexView,
		func(d []byte) (decodedView, bool) { return exifView(d, meta) },
		pdfLiteralView, pdfDocHexView,
	}
	const firstMetadataView = 2
	type key struct {
		kind         string
		line, column int
		matchLen     int
	}
	var seen map[key]bool
	var out []Finding
	for n, view := range views {
		if n == firstMetadataView {
			seen = map[key]bool{}
			for _, list := range [][]Finding{prior, out} {
				for _, f := range list {
					seen[key{f.Kind, f.Line, f.Column, len(f.Matched)}] = true
				}
			}
		}
		v, ok := view(data)
		if !ok {
			continue
		}
		starts := lineStarts([]byte(v.text))
		for _, f := range scanText(v.text, id, secrets, s.identSev, logical, true) {
			if f.Line < 1 || f.Line > len(starts) {
				continue
			}
			at := starts[f.Line-1] + f.Column - 1
			if at < 0 || at >= len(v.text) {
				continue
			}
			f.Line, f.Column = meta.position(v.posMap[at])
			if seen != nil {
				k := key{f.Kind, f.Line, f.Column, len(f.Matched)}
				if seen[k] {
					continue
				}
				seen[k] = true
			}
			out = append(out, f)
		}
	}
	return out
}

// lineStarts returns the offset each '\n'-separated line of data starts at.
func lineStarts(data []byte) []int {
	starts := []int{0}
	for i, b := range data {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// position is the 1-based line and column of raw offset at in m's data, in
// the coordinates scanText gives a finding on the same bytes.
func (m *metadataFields) position(at int) (int, int) {
	if m.starts == nil {
		m.starts = lineStarts(m.data)
	}
	line := sort.Search(len(m.starts), func(i int) bool { return m.starts[i] > at })
	return line, at - m.starts[line-1] + 1
}
