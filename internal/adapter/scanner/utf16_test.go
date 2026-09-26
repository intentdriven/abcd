package scanner

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf16"
)

// utf16Bytes encodes s as UTF-16 behind the byte-order mark of the given
// order, the way a PDF text string (big-endian, FE FF) or a Windows writer
// (little-endian, FF FE) stores it.
func utf16Bytes(s string, bigEndian bool) []byte {
	var b bytes.Buffer
	if bigEndian {
		b.Write([]byte{0xfe, 0xff})
	} else {
		b.Write([]byte{0xff, 0xfe})
	}
	for _, u := range utf16.Encode([]rune(s)) {
		if bigEndian {
			b.Write([]byte{byte(u >> 8), byte(u)})
		} else {
			b.Write([]byte{byte(u), byte(u >> 8)})
		}
	}
	return b.Bytes()
}

// TestUTF16TextOnBytesIsRead pins the byte scan to UTF-16 text a byte-order
// mark opens (iss-2609261827066511): a PDF /Author written as a
// UTF-16 text string interleaves a zero byte with every letter, so no name,
// home path or email in it matched at all, whatever its length. The run is
// decoded and scanned with the byte rules, and a short name in it is kept by
// the person-key rule exactly as the plain-text /Author is.
func TestUTF16TextOnBytesIsRead(t *testing.T) {
	id := synthIdentity()
	cases := []struct {
		name, logical string
		body          []byte
		kind          string
	}{
		{"short name in a UTF-16 /Author", "deck.pdf",
			append(append([]byte("%PDF-1.7\n1 0 obj\n<< /Author ("), utf16Bytes("Zedqx", true)...), []byte(") >>\nendobj\n")...), kindRealName},
		{"multi-word name in a UTF-16 /Author", "deck.pdf",
			append(append([]byte("%PDF-1.7\n<< /Author ("), utf16Bytes(id.GitUserName, true)...), []byte(") >>\n")...), kindRealName},
		{"home path in a little-endian string", "shot.png",
			append(append([]byte("\x89PNG\r\n\x1a\n\x00\x01"), utf16Bytes(id.HomePath+"/deck.key", false)...), 0, 0, 0x7f), kindHomeSelf},
		{"email in a UTF-16 /Creator", "deck.pdf",
			append(append([]byte("%PDF-1.7\n<< /Creator ("), utf16Bytes("by "+id.GitUserEmail, true)...), []byte(") >>\n")...), kindRealEmail},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			sc, err := New(root)
			if err != nil {
				t.Fatal(err)
			}
			sc.identity = id
			if c.name == "short name in a UTF-16 /Author" {
				sc.identity = Identity{GitUserName: "Zedqx"}
			}
			res := scanOne(t, sc, c.logical, writeFile(t, root, c.logical, string(c.body)))
			if !hasKind(res.Findings, c.kind) || res.HardFails == 0 {
				t.Errorf("no hard_fail %s in the UTF-16 text: %+v", c.kind, res.Findings)
			}
		})
	}
}

// TestUTF16TextOnBytesKeepsTheShortNameRule pins the other half: a short name
// in UTF-16 text with no person key within reach is chance noise on bytes and
// is dropped, as it is in plain bytes, and a byte-order mark followed by
// binary content decodes nothing that raises a finding.
func TestUTF16TextOnBytesKeepsTheShortNameRule(t *testing.T) {
	var noise bytes.Buffer
	noise.WriteString("\x89PNG\r\n\x1a\n")
	noise.Write(append([]byte("\x00\x00"), utf16Bytes("scanned page "+strings.Repeat("q", 60)+" Zedqx seen", true)...))
	for i := 0; i < 64; i++ {
		noise.Write([]byte{0xfe, 0xff, byte(i), 0x00, 0xff, 0xfe, 0x00, byte(i)})
	}
	root := t.TempDir()
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	sc.identity = Identity{GitUserName: "Zedqx"}
	res := scanOne(t, sc, "noise.png", writeFile(t, root, "noise.png", noise.String()))
	if hasKind(res.Findings, kindRealName) {
		t.Errorf("a short name with no person key in reach was reported: %+v", res.Findings)
	}
}

// TestUTF16ViewWorkIsLinear holds the byte scan with its UTF-16 view to the
// cost class the line scan is held to: quadrupling a payload dense in marks,
// in short runs and in one long run at most multiplies the charged work by
// linearCostBar, because each raw byte is decoded at most once and the runs
// are scanned as one view.
func TestUTF16ViewWorkIsLinear(t *testing.T) {
	if raceEnabled {
		t.Skip("a deterministic count gains nothing under -race; the uninstrumented run asserts it")
	}
	sc := &Scanner{identity: Identity{GitUserName: "Zedqx"}, identSev: DefaultIdentitySeverities()}
	secrets := secretPatterns(DefaultPatterns())
	shapes := []struct {
		name string
		unit []byte
	}{
		{"marks with nothing after them", []byte{0xfe, 0xff, 0xff, 0xfe}},
		{"short runs", append(utf16Bytes("ab", true), 0, 0)},
		{"short names by a key", append([]byte("/Author ("), append(utf16Bytes("Zedqx", true), ')', ' ')...)},
		{"one long run", utf16Bytes("Zedqx ", false)[2:]},
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
			build := func(n int) []byte {
				out := []byte{0xff, 0xfe}
				return append(out, bytes.Repeat(s.unit, n)...)
			}
			base := max(4096/len(s.unit), 1)
			lo, hi := charge(build(base)), charge(build(4*base))
			if lo == 0 {
				t.Fatal("the shape charged nothing; it pins nothing")
			}
			if growth := float64(hi) / float64(lo); growth > linearCostBar {
				t.Errorf("quadrupling the payload multiplied the byte scan's charge by %.2fx, want at most %.1fx", growth, linearCostBar)
			}
		})
	}
}
