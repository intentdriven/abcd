package scanner

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
	"unicode/utf16"
)

// exifEntry is one tag of a synthetic TIFF IFD: its tag, its field type
// (1 BYTE, 2 ASCII, 7 UNDEFINED) and its value bytes as stored.
type exifEntry struct {
	tag, typ uint16
	value    []byte
}

// asciiValue is an EXIF ASCII value: the text and its terminating NUL.
func asciiValue(s string) []byte { return append([]byte(s), 0) }

// utf16LEValue is an EXIF XP* value: UTF-16LE with no byte-order mark and a
// terminating zero unit, the way Windows writes XPAuthor.
func utf16LEValue(s string) []byte {
	var b bytes.Buffer
	for _, u := range utf16.Encode([]rune(s)) {
		b.Write([]byte{byte(u), byte(u >> 8)})
	}
	b.Write([]byte{0, 0})
	return b.Bytes()
}

// tiffBytes builds a TIFF stream (the payload of a JPEG APP1 Exif segment, a
// PNG eXIf chunk, or a .tif file): the byte-order header, IFD0 holding ifd0,
// and, when sub is non-empty, an Exif sub-IFD that IFD0's 0x8769 entry points
// at. Values over four bytes go after the IFDs, at the offset their entry names.
func tiffBytes(bigEndian bool, ifd0, sub []exifEntry) []byte {
	var bo binary.AppendByteOrder = binary.LittleEndian
	head := []byte("II*\x00")
	if bigEndian {
		bo, head = binary.BigEndian, []byte("MM\x00*")
	}
	if len(sub) > 0 {
		ifd0 = append(ifd0, exifEntry{tag: 0x8769, typ: 4})
	}
	ifdLen := func(n int) int { return 2 + 12*n + 4 }
	ifd0At := 8
	subAt := ifd0At + ifdLen(len(ifd0))
	valuesAt := subAt
	if len(sub) > 0 {
		valuesAt += ifdLen(len(sub))
	}
	var values []byte
	writeIFD := func(entries []exifEntry) []byte {
		var out []byte
		out = bo.AppendUint16(out, uint16(len(entries)))
		for _, e := range entries {
			out = bo.AppendUint16(out, e.tag)
			out = bo.AppendUint16(out, e.typ)
			if e.tag == 0x8769 {
				out = bo.AppendUint32(out, 1)
				out = bo.AppendUint32(out, uint32(subAt))
				continue
			}
			out = bo.AppendUint32(out, uint32(len(e.value)))
			if len(e.value) <= 4 {
				field := make([]byte, 4)
				copy(field, e.value)
				out = append(out, field...)
				continue
			}
			out = bo.AppendUint32(out, uint32(valuesAt+len(values)))
			values = append(values, e.value...)
		}
		return bo.AppendUint32(out, 0)
	}
	out := append([]byte{}, head...)
	out = bo.AppendUint32(out, uint32(ifd0At))
	out = append(out, writeIFD(ifd0)...)
	if len(sub) > 0 {
		out = append(out, writeIFD(sub)...)
	}
	return append(out, values...)
}

// jpegWithExif wraps a TIFF stream as a JPEG's APP1 Exif segment, with a
// camera-shaped head and tail around it and no XMP packet anywhere.
func jpegWithExif(tiff []byte) []byte {
	seg := append([]byte("Exif\x00\x00"), tiff...)
	out := []byte{0xff, 0xd8, 0xff, 0xe1}
	out = binary.BigEndian.AppendUint16(out, uint16(len(seg)+2))
	out = append(out, seg...)
	out = append(out, 0xff, 0xdb, 0x00, 0x04, 0x01, 0x02)
	return append(out, 0xff, 0xd9)
}

// pngWithExif carries a TIFF stream in a PNG eXIf chunk (the CRC is not
// checked by the byte scan, so it is left zero).
func pngWithExif(tiff []byte) []byte {
	out := []byte("\x89PNG\r\n\x1a\n")
	out = binary.BigEndian.AppendUint32(out, uint32(len(tiff)))
	out = append(out, "eXIf"...)
	out = append(out, tiff...)
	return append(out, 0, 0, 0, 0)
}

// TestEXIFPersonTagIsRead pins iss-2609261659051539: a short single-token name
// in an EXIF person tag has no person key written as text before it (the tag
// is a binary IFD entry), so the byte scan dropped it as chance noise, and an
// XP* tag is UTF-16LE with no byte-order mark, which no view read at all. The
// IFD is walked structurally, the person tags (Artist, Copyright, XPAuthor, and
// the Exif sub-IFD's CameraOwnerName) are read as person fields, and the text
// tags are read with the long rules. The names are fake.
func TestEXIFPersonTagIsRead(t *testing.T) {
	id := synthIdentity()
	artist := func(v string) []exifEntry {
		return []exifEntry{{0x010f, 2, asciiValue("Camco")}, {0x013b, 2, asciiValue(v)}}
	}
	cases := []struct {
		name, logical, identity string
		body                    []byte
		want                    bool
	}{
		{"short Artist in a JPEG, little-endian", "shot.jpg", "Zedqx",
			jpegWithExif(tiffBytes(false, artist("Zedqx"), nil)), true},
		{"short Artist in a JPEG, big-endian", "shot.jpg", "Zedqx",
			jpegWithExif(tiffBytes(true, artist("Zedqx"), nil)), true},
		{"short Artist in a PNG eXIf chunk", "shot.png", "Zedqx",
			pngWithExif(tiffBytes(true, artist("Zedqx"), nil)), true},
		{"short Copyright in a JPEG", "shot.jpg", "Zedqx",
			jpegWithExif(tiffBytes(false, []exifEntry{{0x8298, 2, asciiValue("(c) 2026 Zedqx")}}, nil)), true},
		{"short XPAuthor (UTF-16LE, no mark) in a JPEG", "shot.jpg", "Zedqx",
			jpegWithExif(tiffBytes(false, []exifEntry{{0x9c9d, 1, utf16LEValue("Zedqx")}}, nil)), true},
		{"short CameraOwnerName in the Exif sub-IFD", "shot.jpg", "Zedqx",
			jpegWithExif(tiffBytes(true, []exifEntry{{0x010f, 2, asciiValue("Camco")}},
				[]exifEntry{{0xa430, 2, asciiValue("Zedqx")}})), true},
		{"multi-word name in XPComment (UTF-16LE)", "shot.jpg", id.GitUserName,
			jpegWithExif(tiffBytes(false, []exifEntry{{0x9c9c, 1, utf16LEValue("shot by " + id.GitUserName)}}, nil)), true},
		{"UNICODE UserComment in the Exif sub-IFD", "shot.jpg", id.GitUserName,
			jpegWithExif(tiffBytes(false, []exifEntry{{0x010f, 2, asciiValue("Camco")}},
				[]exifEntry{{0x9286, 7, append([]byte("UNICODE\x00"), utf16LEValue(id.GitUserName)...)}})), true},
		// The short-name rule still holds outside a person tag: a model name
		// that happens to equal a short name is chance, not a stamp.
		{"short name in the Model tag", "shot.jpg", "Zedqx",
			jpegWithExif(tiffBytes(false, []exifEntry{{0x0110, 2, asciiValue("Zedqx")}}, nil)), false},
		{"short name in XPComment", "shot.jpg", "Zedqx",
			jpegWithExif(tiffBytes(false, []exifEntry{{0x9c9c, 1, utf16LEValue("Zedqx")}}, nil)), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			sc, err := New(root)
			if err != nil {
				t.Fatal(err)
			}
			sc.identity = Identity{GitUserName: c.identity}
			res := scanOne(t, sc, c.logical, writeFile(t, root, c.logical, string(c.body)))
			if got := hasKind(res.Findings, kindRealName); got != c.want {
				t.Errorf("real_name reported = %v, want %v: %+v", got, c.want, res.Findings)
			}
			if c.want && res.HardFails == 0 {
				t.Errorf("the name is not a hard fail: %+v", res.Findings)
			}
		})
	}
}

// TestEXIFBareTIFFIsRead pins the third home of an IFD: a TIFF file's own
// header, read by the byte scan wherever its bytes reach it (a decoded archive
// entry, a skip-listed file).
func TestEXIFBareTIFFIsRead(t *testing.T) {
	sc := &Scanner{identity: Identity{GitUserName: "Zedqx"}, identSev: DefaultIdentitySeverities()}
	for _, bigEndian := range []bool{false, true} {
		body := tiffBytes(bigEndian, []exifEntry{{0x013b, 2, asciiValue("Zedqx")}}, nil)
		if !hasKind(sc.scanBytes(body, secretPatterns(DefaultPatterns()), "scan.tif"), kindRealName) {
			t.Errorf("bigEndian=%v: a short Artist in a bare TIFF is not reported", bigEndian)
		}
	}
}

// TestEXIFFindingIsNotDoubled pins that the IFD view adds only what the raw
// scan missed: a long name the raw bytes already hold is reported once.
func TestEXIFFindingIsNotDoubled(t *testing.T) {
	id := synthIdentity()
	sc := &Scanner{identity: Identity{GitUserName: id.GitUserName}, identSev: DefaultIdentitySeverities()}
	body := jpegWithExif(tiffBytes(false, []exifEntry{{0x013b, 2, asciiValue(id.GitUserName)}}, nil))
	n := 0
	for _, f := range sc.scanBytes(body, secretPatterns(DefaultPatterns()), "shot.jpg") {
		if f.Kind == kindRealName {
			n++
		}
	}
	if n != 1 {
		t.Errorf("a long Artist name the raw bytes hold was reported %d times, want once", n)
	}
}

// pdfOctal spells b as the body of a PDF literal string with every byte
// outside printable ASCII, and every parenthesis and backslash, escaped as
// three octal digits — the spelling several writers use for a UTF-16 /Author.
func pdfOctal(b []byte) string {
	var s strings.Builder
	for _, c := range b {
		if c >= 0x20 && c < 0x7f && c != '(' && c != ')' && c != '\\' {
			s.WriteByte(c)
			continue
		}
		s.WriteByte('\\')
		s.WriteByte('0' + c>>6)
		s.WriteByte('0' + c>>3&7)
		s.WriteByte('0' + c&7)
	}
	return s.String()
}

// TestPDFLiteralTextStringIsRead pins iss-2609261831352258: a PDF literal
// string written with escapes never puts the text's bytes in the file as they
// read, so a UTF-16 /Author spelled with octal escapes (\376\377\000Z...), a
// raw UTF-16 string whose escaped parenthesis knocks the units out of step,
// and a PDFDocEncoding string with an escaped or raw non-ASCII letter all
// raised nothing. The literal string's escapes are decoded (the balanced
// parentheses, the octal and named escapes, a backslash before a line break)
// and the bytes are read as UTF-16 behind a mark and as PDFDocEncoding
// otherwise, each finding judged by the raw bytes before its first source
// byte. The names are fake.
func TestPDFLiteralTextStringIsRead(t *testing.T) {
	id := synthIdentity()
	rawUTF16 := func(s string) string { return string(utf16Bytes(s, true)[2:]) }
	cases := []struct {
		name, identity, body string
		want                 bool
	}{
		{"short name, octal UTF-16 behind /Author", "Zedqx",
			"%PDF-1.7\n<< /Author (" + pdfOctal(utf16Bytes("Zedqx", true)) + ") >>\n", true},
		{"multi-word name, octal UTF-16, continued lines", id.GitUserName,
			"%PDF-1.7\n<< /Creator (" + strings.Replace(pdfOctal(utf16Bytes(id.GitUserName, true)), `\000 `, "\\\n\\000 ", 1) + ") >>\n", true},
		{"raw UTF-16 after an escaped parenthesis", id.GitUserName,
			"%PDF-1.7\n<< /Author (\xfe\xff\x00\\(" + rawUTF16(id.GitUserName) + "\x00\\)) >>\n", true},
		{"PDFDocEncoding name with octal escapes", "Zoë Qüxbar",
			"%PDF-1.7\n<< /Author (Zo\\353 Q\\374xbar) >>\n", true},
		{"PDFDocEncoding name in raw bytes", "Zoë Qüxbar",
			"%PDF-1.7\n<< /Author (Zo\xeb Q\xfcxbar) >>\n", true},
		{"PDFDocEncoding letter above Latin-1", "Łuqx Zedqa",
			"%PDF-1.7\n<< /Author (\\225uqx Zedqa) >>\n", true},
		{"PDFDocEncoding name in a hex string", "Zoë Qüxbar",
			"%PDF-1.7\n<< /Author " + pdfHex([]byte("Zo\xeb Q\xfcxbar"), false, 0) + " >>\n", true},
		// A '(' in a compressed stream is chance and never closes there; the
		// string it opens ends with the stream body, so the Info dictionary
		// after it is still read.
		{"escaped name after a stray parenthesis in a stream", "Zoë Qüxbar",
			"%PDF-1.7\n1 0 obj\n<< /Length 7 >>\nstream\n\x78\x9c(\x01\x02\x03\nendstream\nendobj\n" +
				"2 0 obj\n<< /Author (Zo\\353 Q\\374xbar) >>\nendobj\n", true},
		{"short name with no person key in reach", "Zedqx",
			"%PDF-1.7\n<< /Title (" + pdfOctal(utf16Bytes("scanned "+strings.Repeat("q", 100)+" Zedqx", true)) + ") >>\n", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			sc, err := New(root)
			if err != nil {
				t.Fatal(err)
			}
			sc.identity = Identity{GitUserName: c.identity}
			res := scanOne(t, sc, "deck.pdf", writeFile(t, root, "deck.pdf", c.body))
			if got := hasKind(res.Findings, kindRealName); got != c.want {
				t.Errorf("real_name reported = %v, want %v: %+v", got, c.want, res.Findings)
			}
		})
	}
}

// TestMetadataViewsWorkIsLinear holds the IFD and PDF literal views to the
// cost class the byte scan is held to: quadrupling a payload dense in TIFF
// headers, in IFDs that point at one another, and in literal strings that
// open and never close at most multiplies the charged work by linearCostBar.
func TestMetadataViewsWorkIsLinear(t *testing.T) {
	if raceEnabled {
		t.Skip("a deterministic count gains nothing under -race; the uninstrumented run asserts it")
	}
	sc := &Scanner{identity: Identity{GitUserName: "Zedqx"}, identSev: DefaultIdentitySeverities()}
	secrets := secretPatterns(DefaultPatterns())
	// Every header points at the same far IFD with a large entry count, so a
	// walk without a global bound reads it once per header.
	headers := func(n int) []byte {
		ifd := binary.LittleEndian.AppendUint16(nil, 0xffff)
		ifd = append(ifd, bytes.Repeat([]byte{0x3b, 0x01, 0x02, 0x00, 0x08, 0, 0, 0, 0, 0, 0, 0}, 4096)...)
		var out []byte
		for i := 0; i < n; i++ {
			at := uint32(n*8 - i*8)
			out = append(out, "II*\x00"...)
			out = binary.LittleEndian.AppendUint32(out, at)
		}
		return append(out, ifd...)
	}
	shapes := []struct {
		name  string
		build func(n int) []byte
	}{
		{"TIFF headers sharing one IFD", headers},
		{"literal strings that never close", func(n int) []byte {
			return append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte(`(\376\377\000Z`), n)...)
		}},
		{"nested literal strings", func(n int) []byte {
			return append([]byte("%PDF-1.7\n"), append(bytes.Repeat([]byte("(\xe9"), n), bytes.Repeat([]byte(")"), n)...)...)
		}},
		{"stream keywords with no end", func(n int) []byte {
			return append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte("stream\n(\xe9 "), n)...)
		}},
		{"short stream bodies", func(n int) []byte {
			return append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte("stream\n(\xe9\x01 endstream\n(Z\\351) "), n)...)
		}},
		{"PDFDocEncoding hex strings", func(n int) []byte {
			return append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte("/Author <5A65E9> "), n)...)
		}},
	}
	for _, s := range shapes {
		t.Run(s.name, func(t *testing.T) {
			charge := func(data []byte) int {
				total := 0
				scanMeter.tally = func(_ string, n int) { total += n }
				defer func() { scanMeter.tally = nil }()
				sc.scanBytes(data, secrets, "f")
				return total
			}
			lo, hi := charge(s.build(256)), charge(s.build(1024))
			if lo == 0 {
				t.Fatal("the shape charged nothing; it pins nothing")
			}
			if growth := float64(hi) / float64(lo); growth > linearCostBar {
				t.Errorf("quadrupling the payload multiplied the byte scan's charge by %.2fx, want at most %.1fx", growth, linearCostBar)
			}
		})
	}
}

// FuzzMetadataViews feeds hostile bytes to the IFD and PDF literal readers:
// none may panic, and every position a view maps to lies inside the data.
func FuzzMetadataViews(f *testing.F) {
	f.Add(jpegWithExif(tiffBytes(false, []exifEntry{{0x013b, 2, asciiValue("Zedqx")}, {0x9c9d, 1, utf16LEValue("Zedqx")}}, nil)))
	f.Add(tiffBytes(true, []exifEntry{{0x8298, 2, asciiValue("Zedqx")}}, []exifEntry{{0xa430, 2, asciiValue("Zedqx")}, {0x9286, 7, append([]byte("UNICODE\x00"), 0, 'Z')}}))
	f.Add([]byte("%PDF-1.7\n<< /Author (" + pdfOctal(utf16Bytes("Zedqx", true)) + ") /Title (a\\(b\\\nc\\) \\0) >>"))
	f.Add([]byte("%PDF-1.7\n<< /Author <5A65E9> (\xfe\xff\x00\\(\x00Z) >>"))
	f.Fuzz(func(t *testing.T, data []byte) {
		meta := metadataFields{data: data}
		views := []func([]byte) (decodedView, bool){
			func(d []byte) (decodedView, bool) { return exifView(d, &meta) },
			pdfLiteralView, pdfDocHexView, pdfHexView,
		}
		for _, view := range views {
			v, ok := view(data)
			if !ok {
				continue
			}
			if len(v.posMap) != len(v.text)+1 {
				t.Fatalf("posMap has %d entries for %d bytes", len(v.posMap), len(v.text))
			}
			for _, p := range v.posMap {
				if p < 0 || p > len(data) {
					t.Fatalf("position %d outside the data (%d bytes)", p, len(data))
				}
			}
		}
		for _, s := range meta.persons {
			if s.start < 0 || s.end > len(data) || s.start > s.end {
				t.Fatalf("person span %v outside the data (%d bytes)", s, len(data))
			}
		}
		sc := &Scanner{identity: Identity{GitUserName: "Zedqx"}, identSev: DefaultIdentitySeverities()}
		sc.scanBytes(data, nil, "f")
	})
}

// TestTextMetadataOwnerKeysAreRead pins the text-keyed siblings of the EXIF
// person tags: a PNG text chunk's Copyright keyword, XMP's dc:rights and
// xmpRights:Owner, and XMP's CameraOwnerName carry the name the file was
// stamped with exactly as an Author does, so a short name after one is kept.
func TestTextMetadataOwnerKeysAreRead(t *testing.T) {
	bodies := map[string]string{
		"PNG tEXt Copyright":  "\x89PNG\r\n\x1a\n\x00\x00\x00\x10tEXtCopyright\x00Zedqx\x00\x00\x00\x00",
		"XMP dc:rights":       "\xff\xd8<x:xmpmeta><dc:rights>\n  <rdf:Alt>\n   <rdf:li xml:lang=\"x-default\">Zedqx</rdf:li>",
		"XMP CameraOwnerName": "\xff\xd8<x:xmpmeta><rdf:Description exifEX:CameraOwnerName=\"Zedqx\"/>",
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			sc := &Scanner{identity: Identity{GitUserName: "Zedqx"}, identSev: DefaultIdentitySeverities()}
			if !hasKind(sc.scanBytes([]byte(body), secretPatterns(DefaultPatterns()), "shot.jpg"), kindRealName) {
				t.Errorf("a short name after the %s key is not reported", name)
			}
		})
	}
}

// iptcDataset is one IPTC-IIM application record dataset: the 0x1C tag
// marker, record 2, the dataset number and a two-byte length.
func iptcDataset(n byte, v string) []byte {
	out := []byte{0x1c, 0x02, n}
	out = binary.BigEndian.AppendUint16(out, uint16(len(v)))
	return append(out, v...)
}

// TestIPTCPersonDatasetIsRead pins the EXIF reader's sibling store: a JPEG's
// IPTC-IIM record (inside the Photoshop APP13 segment) keeps the By-line,
// Credit, Copyright Notice, Contact and Writer/Editor as binary datasets with
// no key text, so a short name in one was dropped just as in EXIF Artist.
func TestIPTCPersonDatasetIsRead(t *testing.T) {
	sc := &Scanner{identity: Identity{GitUserName: "Zedqx"}, identSev: DefaultIdentitySeverities()}
	secrets := secretPatterns(DefaultPatterns())
	app13 := func(ds []byte) []byte {
		body := append([]byte("Photoshop 3.0\x008BIM\x04\x04\x00\x00\x00\x00\x00\x40"), iptcDataset(0x05, "Harbour")...)
		return append(append([]byte{0xff, 0xd8, 0xff, 0xed, 0x00, 0x60}, append(body, ds...)...), 0xff, 0xd9)
	}
	for _, n := range []byte{0x50, 0x6e, 0x74, 0x76, 0x7a} {
		if !hasKind(sc.scanBytes(app13(iptcDataset(n, "Zedqx")), secrets, "shot.jpg"), kindRealName) {
			t.Errorf("a short name in IPTC dataset 2:%d is not reported", n)
		}
	}
	// The object name (2:5) is not a person field.
	if hasKind(sc.scanBytes(app13(iptcDataset(0x05, "Zedqx")), secrets, "shot.jpg"), kindRealName) {
		t.Error("a short name in the IPTC object name was reported")
	}
}
