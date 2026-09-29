package scanner

import (
	"bytes"
	"encoding/binary"
	"unicode/utf8"
)

// metaview.go — the byte scan's views of document metadata that a plain byte
// read cannot see (iss-2609261659051539, iss-2609261831352258).
//
// Two metadata stores hide their text from the raw bytes, each in its own way,
// and a third hides the key that makes a short name in it a person field.
//
// EXIF keeps its tags in a TIFF image file directory: a binary table whose
// twelve-byte entries name a tag, a type, a count and the offset of the value.
// An Artist value is ASCII in the file, so the raw scan reads a long name in
// it, but no person key is written as text before it, so a short name there
// was dropped as chance; the XP* tags are UTF-16LE with no byte-order mark,
// which no view read at all. exifView walks the directory structurally —
// every TIFF header in the data (a JPEG APP1 Exif segment, a PNG eXIf chunk,
// a TIFF file's own head, a WebP or HEIC Exif item), IFD0 and the Exif
// sub-IFD it points at — and decodes the text tags one value to a line. The
// person tags (Artist, Copyright, XPAuthor, CameraOwnerName) register their
// raw value span with metadataFields, which is what keeps a short name there.
// The IPTC-IIM record in a JPEG's APP13 segment has the same shape for a
// person: its By-line and kin are binary datasets with no key text, so
// iptcPersonSpans records their value spans too.
//
// A PDF literal string is written between balanced parentheses with backslash
// escapes, so a UTF-16 /Author spelled \376\377\000Z..., a raw UTF-16 string
// whose escaped parenthesis knocks every later unit out of step, and a
// PDFDocEncoding name with an escaped or raw accented letter never put the
// text's bytes in the file. pdfLiteralView decodes the string syntax and hands
// the bytes to the UTF-16 view, and reads an unmarked string as UTF-8 behind
// its PDF 2.0 mark or as PDFDocEncoding otherwise; pdfDocHexView does the
// same for a hex string with no UTF-16 mark.
//
// Every view feeds the SAME byte rules the raw bytes get (byteViewFindings),
// and each decoded byte maps to the raw offset it came from, so a finding is
// judged — and masked — where it sits on disk. These readers parse untrusted
// bytes: every offset is checked against the data before it is read, and every
// walk is bounded so the cost stays linear in the data.

// exifTextTag is how one EXIF tag's value is read, and whether it names a
// person.
type exifTextTag struct {
	enc    byte
	person bool
}

// The value encodings of the EXIF text tags.
const (
	exifASCII   byte = iota // NUL-separated text (EXIF says ASCII; writers put UTF-8)
	exifUTF16LE             // the Windows XP* tags: UTF-16LE, no byte-order mark
	exifComment             // UserComment: an eight-byte charset code, then the text
)

// exifTags are the text tags the IFD walk reads, from IFD0 and the Exif
// sub-IFD alike. The person tags are the ones a camera or an editor stamps
// with the owner's or the author's name.
var exifTags = map[uint16]exifTextTag{
	0x010e: {exifASCII, false},   // ImageDescription
	0x013b: {exifASCII, true},    // Artist
	0x8298: {exifASCII, true},    // Copyright
	0x9c9b: {exifUTF16LE, false}, // XPTitle
	0x9c9c: {exifUTF16LE, false}, // XPComment
	0x9c9d: {exifUTF16LE, true},  // XPAuthor
	0x9c9e: {exifUTF16LE, false}, // XPKeywords
	0x9c9f: {exifUTF16LE, false}, // XPSubject
	0x9286: {exifComment, false}, // UserComment
	0xa430: {exifASCII, true},    // CameraOwnerName
}

// exifSubIFDTag is IFD0's pointer to the Exif sub-IFD.
const exifSubIFDTag = 0x8769

// maxIFDEntries bounds one directory's walk: a camera's IFD0 holds a few
// dozen entries, and the entry count is a two-byte field a hostile file sets
// to 65535.
const maxIFDEntries = 512

// exifView returns the EXIF text tags of every TIFF header in data, decoded
// one value to a line with each byte mapped to the raw offset of its source
// byte, and records each person tag's raw value span (identity.go's span, as
// raw offsets) in meta. The walk reads
// at most len(data)/12 entries plus a slack of one directory in all, each
// directory once and each value once, so a file dense in headers that share
// one directory costs what its bytes do.
func exifView(data []byte, meta *metadataFields) (decodedView, bool) {
	w := exifWalk{
		data:    data,
		entries: len(data)/12 + maxIFDEntries,
		dirs:    map[int]bool{},
		values:  map[int]bool{},
		meta:    meta,
	}
	for b := 0; b+8 <= len(data); b++ {
		var bo binary.ByteOrder
		switch {
		case data[b] == 'I' && data[b+1] == 'I' && data[b+2] == 42 && data[b+3] == 0:
			bo = binary.LittleEndian
		case data[b] == 'M' && data[b+1] == 'M' && data[b+2] == 0 && data[b+3] == 42:
			bo = binary.BigEndian
		default:
			continue
		}
		scanMeter.charge(stageEXIF, 8)
		if at, ok := w.offset(b, bo.Uint32(data[b+4:])); ok {
			w.dir(b, at, bo, true)
		}
	}
	meta.persons = mergeSpans(meta.persons)
	if len(w.text) == 0 {
		return decodedView{}, false
	}
	return decodedView{text: string(w.text), posMap: append(w.pos, len(data))}, true
}

// exifWalk is one exifView's state: the entry budget left, the directories
// and values already read, and the view built so far.
type exifWalk struct {
	data    []byte
	entries int
	dirs    map[int]bool
	values  map[int]bool
	meta    *metadataFields
	text    []byte
	pos     []int
}

// offset resolves a TIFF offset, relative to the header at base, to a raw
// offset inside the data, refusing one that points past its end. The check is
// made in the offset's own width so a hostile value cannot overflow an int.
func (w *exifWalk) offset(base int, off uint32) (int, bool) {
	if uint64(off) >= uint64(len(w.data)-base) {
		return 0, false
	}
	return base + int(off), true
}

// dir walks one directory at raw offset at, reading the text tags and, from
// IFD0 alone, following the Exif sub-IFD pointer one level down.
func (w *exifWalk) dir(base, at int, bo binary.ByteOrder, top bool) {
	if at+2 > len(w.data) || w.dirs[at] {
		return
	}
	w.dirs[at] = true
	n := min(int(bo.Uint16(w.data[at:])), maxIFDEntries)
	for k := 0; k < n && w.entries > 0; k++ {
		e := at + 2 + 12*k
		if e+12 > len(w.data) {
			return
		}
		w.entries--
		scanMeter.charge(stageEXIF, 12)
		tag, typ, count := bo.Uint16(w.data[e:]), bo.Uint16(w.data[e+2:]), bo.Uint32(w.data[e+4:])
		if tag == exifSubIFDTag && top && (typ == 4 || typ == 13) {
			if sub, ok := w.offset(base, bo.Uint32(w.data[e+8:])); ok {
				w.dir(base, sub, bo, false)
			}
			continue
		}
		t, ok := exifTags[tag]
		if !ok || (typ != 1 && typ != 2 && typ != 7) {
			continue
		}
		v, size, ok := w.value(base, e, bo, count)
		if !ok || w.values[v] {
			continue
		}
		w.values[v] = true
		w.read(w.data[v:v+size], v, t.enc, bo)
		if t.person {
			w.meta.persons = append(w.meta.persons, span{v, v + size})
		}
	}
}

// value locates the value of the byte-sized entry at raw offset e: inline in
// the entry when it fits in four bytes, at its offset otherwise, and refused
// when any byte of it lies outside the data.
func (w *exifWalk) value(base, e int, bo binary.ByteOrder, count uint32) (int, int, bool) {
	if count == 0 || uint64(count) > uint64(len(w.data)) {
		return 0, 0, false
	}
	size := int(count)
	if size <= 4 {
		return e + 8, size, true
	}
	v, ok := w.offset(base, bo.Uint32(w.data[e+8:]))
	if !ok || size > len(w.data)-v {
		return 0, 0, false
	}
	return v, size, true
}

// read appends one value to the view as a line, each decoded byte mapped to
// the raw offset of its source byte (at is the value's raw offset). A NUL
// ends a line, since Copyright holds the photographer's and the editor's
// notices as two NUL-terminated strings.
func (w *exifWalk) read(val []byte, at int, enc byte, bo binary.ByteOrder) {
	scanMeter.charge(stageEXIF, len(val))
	switch enc {
	case exifComment:
		if len(val) < 8 {
			return
		}
		switch string(val[:8]) {
		case "ASCII\x00\x00\x00":
			w.read(val[8:], at+8, exifASCII, bo)
		case "UNICODE\x00":
			w.utf16(val[8:], at+8, bo == binary.BigEndian)
		}
		return
	case exifUTF16LE:
		w.utf16(val, at, false)
		return
	}
	for i, c := range val {
		if c == 0 {
			c = '\n'
		}
		w.text = append(w.text, c)
		w.pos = append(w.pos, at+i)
	}
	w.text = append(w.text, '\n')
	w.pos = append(w.pos, at+len(val))
}

// utf16 appends a UTF-16 value with no byte-order mark, decoded with the same
// unit reader the marked view uses, up to the first unit that is not text.
func (w *exifWalk) utf16(val []byte, at int, bigEndian bool) {
	for j := 0; j+1 < len(val); {
		r, width := utf16RuneAt(val, j, bigEndian)
		if width == 0 {
			break
		}
		var enc [utf8.UTFMax]byte
		for _, c := range enc[:utf8.EncodeRune(enc[:], r)] {
			w.text = append(w.text, c)
			w.pos = append(w.pos, at+j)
		}
		j += width
	}
	w.text = append(w.text, '\n')
	w.pos = append(w.pos, at+len(val))
}

// iptcPersonDatasets are the IPTC-IIM application record (record 2) datasets
// that name a person: By-line (80), Credit (110), Copyright Notice (116),
// Contact (118) and Writer/Editor (122).
var iptcPersonDatasets = [256]bool{80: true, 110: true, 116: true, 118: true, 122: true}

// iptcPersonSpans appends to spans the raw value span of every IPTC-IIM
// person dataset in data: the tag marker 0x1C, record 2, the dataset number,
// and a two-byte length (an extended length, top bit set, is not a text
// field). Its value is text in the raw bytes, which the raw scan reads; what
// it lacks is a person key written as text, which the span stands in for, as
// an EXIF person tag's does. One pass, each marker read once.
func iptcPersonSpans(data []byte, spans []span) []span {
	for i := 0; i+5 <= len(data); i++ {
		if data[i] != 0x1c || data[i+1] != 2 || !iptcPersonDatasets[data[i+2]] {
			continue
		}
		n := int(binary.BigEndian.Uint16(data[i+3:]))
		if n&0x8000 != 0 || n > len(data)-i-5 {
			continue
		}
		spans = append(spans, span{i + 5, i + 5 + n})
	}
	scanMeter.charge(stageEXIF, len(data))
	return spans
}

// inPersonSpan reports whether raw offset at lies inside a person tag's value.
func (m *metadataFields) inPersonSpan(at int) bool {
	return inAnySpan(at, m.persons)
}

// maxPDFLiteral bounds one literal string's decode at the longest string PDF
// admits (ISO 32000-1, Annex C). A '(' that never closes — a stray byte in a
// binary stream — decodes at most that far, and the walk resumes where the
// decode stopped, so every raw byte is decoded at most once.
const maxPDFLiteral = 32767

// isPDFData reports whether data carries a PDF (or FDF) header anywhere: the
// literal-string and PDFDocEncoding views read PDF syntax, and on any other
// bytes a '(' is chance.
func isPDFData(data []byte) bool {
	return bytes.Contains(data, []byte("%PDF-")) || bytes.Contains(data, []byte("%FDF-"))
}

// pdfLiteralView returns the text of every PDF literal string whose bytes the
// raw scan cannot read as written — one holding an escape or a byte above
// ASCII — decoded from the string syntax and read two ways: its byte-order
// marked runs as UTF-16 (utf16View), and the whole string as UTF-8 behind its
// mark or as PDFDocEncoding (appendPDFText).
// Each decoded byte maps to the raw offset of the first byte of its escape or
// code unit (a separator to the string's end).
//
// A stream body (between "stream" and "endstream") is mostly compressed
// bytes, where a '(' is chance: a string found there ends at the body's end,
// so a stray one cannot swallow the objects after it. A string is read as
// PDFDocEncoding only when it decodes to text (pdfTextLike), so the junk of a
// compressed stream is not scanned a second time; its marked UTF-16 runs are
// read either way.
func pdfLiteralView(data []byte) (decodedView, bool) {
	if !isPDFData(data) {
		return decodedView{}, false
	}
	var text []byte
	var pos []int
	var raw []byte
	var off []int
	streamEnd := -1
	for i := 0; i < len(data); i++ {
		if i >= streamEnd && data[i] == 's' {
			if at, ok := pdfStreamBody(data, i); ok {
				streamEnd = at
				continue
			}
		}
		if data[i] != '(' {
			continue
		}
		limit := len(data)
		if i < streamEnd {
			limit = streamEnd
		}
		var end int
		var escaped bool
		raw, off, end, escaped = pdfLiteral(data[:limit], i, raw[:0], off[:0])
		scanMeter.charge(stagePDFLiteral, end-i)
		i = end
		if !escaped || len(raw) == 0 {
			continue
		}
		off = append(off, end)
		if v, ok := utf16View(raw); ok {
			for k := 0; k < len(v.text); k++ {
				text = append(text, v.text[k])
				pos = append(pos, off[v.posMap[k]])
			}
		}
		if len(raw) >= 2 && (raw[0] == 0xfe && raw[1] == 0xff || raw[0] == 0xff && raw[1] == 0xfe) {
			continue
		}
		if !pdfTextLike(raw) {
			continue
		}
		text, pos = appendPDFText(text, pos, raw, off)
	}
	if len(text) == 0 {
		return decodedView{}, false
	}
	return decodedView{text: string(text), posMap: append(pos, len(data))}, true
}

// pdfStreamBody reports whether the "stream" keyword opens a stream body at
// data[i] (the keyword, not the tail of "endstream", followed by an end of
// line) and returns the offset of the "endstream" closing it, or len(data)
// when none does. The search runs forward from the keyword and the walk
// resumes past it, so each byte is searched at most once.
func pdfStreamBody(data []byte, i int) (int, bool) {
	const kw = "stream"
	if !bytes.HasPrefix(data[i:], []byte(kw)) || i+len(kw) >= len(data) {
		return 0, false
	}
	if c := data[i+len(kw)]; c != '\r' && c != '\n' {
		return 0, false
	}
	if i >= 3 && string(data[i-3:i]) == "end" {
		return 0, false
	}
	body := i + len(kw)
	e := bytes.Index(data[body:], []byte("endstream"))
	if e < 0 {
		scanMeter.charge(stagePDFLiteral, len(data)-body)
		return len(data), true
	}
	scanMeter.charge(stagePDFLiteral, e)
	return body + e, true
}

// pdfTextLike reports whether decoded string bytes read as text: no control
// byte but the layout ones. A compressed stream's chance string holds one
// within a few bytes.
func pdfTextLike(raw []byte) bool {
	for _, c := range raw {
		if c < 0x20 && c != '\t' && c != '\n' && c != '\r' && c != '\f' || c == 0x7f {
			return false
		}
	}
	return true
}

// pdfLiteral decodes the literal string opening at data[open] (a '('): the
// balanced parentheses (an escaped one does not count), the named escapes, one
// to three octal digits, a backslash before a line break that continues the
// line, and a backslash before any other byte, which PDF ignores. It appends
// the decoded bytes to raw and the raw offset each came from to off, and
// returns the offset of the closing ')' (or where the decode stopped) and
// whether the string held an escape or a byte above ASCII.
func pdfLiteral(data []byte, open int, raw []byte, off []int) ([]byte, []int, int, bool) {
	depth, escaped := 1, false
	j := open + 1
	for ; j < len(data) && j-open <= maxPDFLiteral; j++ {
		c := data[j]
		switch {
		case c == '(':
			depth++
		case c == ')':
			depth--
			if depth == 0 {
				return raw, off, j, escaped
			}
		case c >= 0x80:
			escaped = true
		case c == '\\':
			escaped = true
			if j+1 >= len(data) {
				return raw, off, len(data), escaped
			}
			at := j
			j++
			switch n := data[j]; {
			case n == '\r':
				if j+1 < len(data) && data[j+1] == '\n' {
					j++
				}
				continue
			case n == '\n':
				continue
			case n >= '0' && n <= '7':
				v := int(n - '0')
				for k := 0; k < 2 && j+1 < len(data) && data[j+1] >= '0' && data[j+1] <= '7'; k++ {
					j++
					v = v<<3 | int(data[j]-'0')
				}
				c = byte(v)
			default:
				c = pdfNamedEscape(n)
			}
			raw = append(raw, c)
			off = append(off, at)
			continue
		}
		raw = append(raw, c)
		off = append(off, j)
	}
	return raw, off, min(j, len(data)), escaped
}

// pdfNamedEscape is the byte a backslash before n stands for: a control for
// n, r, t, b and f, and n itself otherwise (a parenthesis, a backslash, or a
// byte PDF says the backslash before is ignored for).
func pdfNamedEscape(n byte) byte {
	switch n {
	case 'n':
		return '\n'
	case 'r':
		return '\r'
	case 't':
		return '\t'
	case 'b':
		return '\b'
	case 'f':
		return '\f'
	}
	return n
}

// pdfDocHigh is PDFDocEncoding from 0x80 to 0xA0, where it departs from
// Latin-1 (ISO 32000-1, Annex D); 0x9F is undefined and reads as nothing.
// Above 0xA0 PDFDocEncoding is Latin-1.
var pdfDocHigh = [...]rune{
	0x2022, 0x2020, 0x2021, 0x2026, 0x2014, 0x2013, 0x0192, 0x2044,
	0x2039, 0x203a, 0x2212, 0x2030, 0x201e, 0x201c, 0x201d, 0x2018,
	0x2019, 0x201a, 0x2122, 0xfb01, 0xfb02, 0x0141, 0x0152, 0x0160,
	0x0178, 0x017d, 0x0131, 0x0142, 0x0153, 0x0161, 0x017e, 0,
	0x20ac,
}

// appendPDFText appends the bytes of an unmarked PDF text string to text as
// one line: behind a UTF-8 mark (EF BB BF, which PDF 2.0 admits) as UTF-8,
// and otherwise read as PDFDocEncoding. Each UTF-8 byte of a character maps
// through off to its source's raw offset; off carries one entry past raw, the
// string's end, for the separator.
func appendPDFText(text []byte, pos []int, raw []byte, off []int) ([]byte, []int) {
	if len(raw) >= 3 && raw[0] == 0xef && raw[1] == 0xbb && raw[2] == 0xbf {
		for k := 3; k < len(raw); k++ {
			text = append(text, raw[k])
			pos = append(pos, off[k])
		}
		return append(text, '\n'), append(pos, off[len(raw)])
	}
	for k, c := range raw {
		r := rune(c)
		if c >= 0x80 && c <= 0xa0 {
			r = pdfDocHigh[c-0x80]
		}
		if r == 0 {
			continue
		}
		var enc [utf8.UTFMax]byte
		for _, b := range enc[:utf8.EncodeRune(enc[:], r)] {
			text = append(text, b)
			pos = append(pos, off[k])
		}
	}
	return append(text, '\n'), append(pos, off[len(raw)])
}

// pdfDocHexView returns every PDF hex string whose bytes open with no UTF-16
// mark, read as UTF-8 behind its mark or as PDFDocEncoding (appendPDFText): a
// hex /Author is spelled in digits, so its name is in no raw byte. The UTF-16
// ones are pdfHexView's.
func pdfDocHexView(data []byte) (decodedView, bool) {
	if !isPDFData(data) {
		return decodedView{}, false
	}
	var text []byte
	var pos []int
	pdfHexStrings(data, func(raw []byte, off []int) {
		if len(raw) >= 2 && (raw[0] == 0xfe && raw[1] == 0xff || raw[0] == 0xff && raw[1] == 0xfe) {
			return
		}
		text, pos = appendPDFText(text, pos, raw, off)
	})
	if len(text) == 0 {
		return decodedView{}, false
	}
	return decodedView{text: string(text), posMap: append(pos, len(data))}, true
}
