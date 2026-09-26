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
// A run without a mark (EXIF's XPAuthor tag, a legacy binary document) is not
// read: telling UTF-16 from chance bytes there needs the structure the text
// sits in, which is iss-2609261659051539's IFD reader, not a decode.

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

// utf16Findings scans the UTF-16 view of data with the byte rules and re-homes
// each finding onto the raw bytes: Line and Column name the raw position of
// the value's first code unit (meta's line starts), while Matched and the
// snippet stay the decoded text, so the short-name length rule counts the
// name's own bytes rather than its zero-interleaved spelling, and a serialized
// finding masks the name as it reads.
func (s *Scanner) utf16Findings(data []byte, id Identity, secrets []Pattern, logical string, meta *metadataFields) []Finding {
	v, ok := utf16View(data)
	if !ok {
		return nil
	}
	starts := lineStarts([]byte(v.text))
	var out []Finding
	for _, f := range scanText(v.text, id, secrets, s.identSev, logical, true) {
		if f.Line < 1 || f.Line > len(starts) {
			continue
		}
		at := starts[f.Line-1] + f.Column - 1
		if at < 0 || at >= len(v.text) {
			continue
		}
		f.Line, f.Column = meta.position(v.posMap[at])
		out = append(out, f)
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
