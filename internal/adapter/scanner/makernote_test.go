package scanner

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// canonNote is one synthetic camera JPEG's EXIF with a vendor MakerNote: the
// Make IFD0 names, the MakerNote's own IFD entries, and how the MakerNote was
// last rewritten. The names are fake and no value comes from a real photo.
type canonNote struct {
	bigEndian bool
	make      string
	// noteBigEndian writes the MakerNote in the other byte order from the
	// TIFF's when it differs from bigEndian (an editor may do so).
	noteBigEndian bool
	entries       []exifEntry
	// moved, when non-zero, writes the MakerNote's value offsets as they
	// stood before an editor moved the block moved bytes later, with the
	// Canon TIFF footer naming that original offset.
	moved int
	// footer appends the Canon TIFF footer even when the block did not move.
	footer bool
}

// build lays the TIFF stream out by hand, every offset relative to the TIFF
// header (CIPA DC-008): IFD0 holding Make and the Exif sub-IFD pointer, the
// sub-IFD holding the MakerNote (0x927c, UNDEFINED), the Make text, then the
// MakerNote itself — an IFD with no header of its own whose values follow it.
func (c canonNote) build() []byte {
	order := func(big bool) binary.AppendByteOrder {
		if big {
			return binary.BigEndian
		}
		return binary.LittleEndian
	}
	bo, nbo := order(c.bigEndian), order(c.noteBigEndian)
	head := []byte("II*\x00")
	if c.bigEndian {
		head = []byte("MM\x00*")
	}
	const ifd0At, subAt, makeAt = 8, 8 + 30, 8 + 30 + 18
	makeVal := asciiValue(c.make)
	noteAt := makeAt + len(makeVal)
	// The MakerNote: its IFD, then its long values, then the footer.
	dirLen := 2 + 12*len(c.entries) + 4
	var note, values []byte
	note = nbo.AppendUint16(note, uint16(len(c.entries)))
	for _, e := range c.entries {
		note = nbo.AppendUint16(note, e.tag)
		note = nbo.AppendUint16(note, e.typ)
		note = nbo.AppendUint32(note, uint32(len(e.value)))
		if len(e.value) <= 4 {
			field := make([]byte, 4)
			copy(field, e.value)
			note = append(note, field...)
			continue
		}
		note = nbo.AppendUint32(note, uint32(noteAt-c.moved+dirLen+len(values)))
		values = append(values, e.value...)
	}
	note = nbo.AppendUint32(note, 0)
	note = append(note, values...)
	if c.moved != 0 || c.footer {
		if c.noteBigEndian {
			note = append(note, "MM\x00*"...)
		} else {
			note = append(note, "II*\x00"...)
		}
		note = nbo.AppendUint32(note, uint32(noteAt-c.moved))
	}

	out := append([]byte{}, head...)
	out = bo.AppendUint32(out, ifd0At)
	out = bo.AppendUint16(out, 2)
	out = bo.AppendUint16(out, 0x010f)
	out = bo.AppendUint16(out, 2)
	out = bo.AppendUint32(out, uint32(len(makeVal)))
	if len(makeVal) <= 4 {
		field := make([]byte, 4)
		copy(field, makeVal)
		out = append(out, field...)
	} else {
		out = bo.AppendUint32(out, makeAt)
	}
	out = bo.AppendUint16(out, 0x8769)
	out = bo.AppendUint16(out, 4)
	out = bo.AppendUint32(out, 1)
	out = bo.AppendUint32(out, subAt)
	out = bo.AppendUint32(out, 0)
	out = bo.AppendUint16(out, 1)
	out = bo.AppendUint16(out, 0x927c)
	out = bo.AppendUint16(out, 7)
	out = bo.AppendUint32(out, uint32(len(note)))
	out = bo.AppendUint32(out, uint32(noteAt))
	out = bo.AppendUint32(out, 0)
	out = append(out, makeVal...)
	return append(out, note...)
}

// ownerName is Canon's OwnerName (MakerNote tag 0x0009, ASCII) as a camera
// writes it: the name padded with NULs to 32 bytes.
func ownerName(name string) exifEntry {
	v := make([]byte, 32)
	copy(v, name)
	return exifEntry{0x0009, 2, v}
}

// cr3WithCMT3 carries a Canon MakerNote the way a CR3 file does: a CMT3 box
// whose payload is a TIFF stream with its own header, IFD0 being the
// MakerNote's directory and no Make beside it (that is in CMT1).
func cr3WithCMT3(bigEndian bool, entries []exifEntry) []byte {
	tiff := tiffBytes(bigEndian, entries, nil)
	out := binary.BigEndian.AppendUint32(nil, 16)
	out = append(out, "ftypcrx \x00\x00\x00\x01"...)
	out = binary.BigEndian.AppendUint32(out, uint32(8+len(tiff)))
	out = append(out, "CMT3"...)
	return append(out, tiff...)
}

// TestCanonMakerNoteOwnerNameIsRead pins iss-2609291653241690: the EXIF walk
// followed only the Exif sub-IFD pointer, so the owner's name a Canon camera
// writes into its MakerNote (tag 0x0009 OwnerName, offsets relative to the
// TIFF header) was never read as a person field, and a short single-token
// owner name was dropped as chance. The MakerNote of a file whose Make is
// Canon is now walked, and so is a CR3 file's CMT3 box; any other vendor's is
// skipped, and so is anything but OwnerName in Canon's.
func TestCanonMakerNoteOwnerNameIsRead(t *testing.T) {
	owner := []exifEntry{{0x0006, 2, asciiValue("Canon EOS Qx")}, ownerName("Zedqx")}
	cases := []struct {
		name, logical string
		body          []byte
		want          bool
	}{
		{"Canon MakerNote OwnerName, little-endian", "shot.jpg",
			jpegWithExif(canonNote{make: "Canon", entries: owner}.build()), true},
		{"Canon MakerNote OwnerName, big-endian", "shot.jpg",
			jpegWithExif(canonNote{bigEndian: true, noteBigEndian: true, make: "Canon", entries: owner}.build()), true},
		{"Canon MakerNote in the other byte order from the TIFF", "shot.jpg",
			jpegWithExif(canonNote{make: "Canon", noteBigEndian: true, entries: owner}.build()), true},
		{"Canon MakerNote with a footer that matches", "shot.jpg",
			jpegWithExif(canonNote{make: "Canon", entries: owner, footer: true}.build()), true},
		// The move is longer than OwnerName's 32 bytes, so the value read at
		// the TIFF header's base cannot overlap the name: only the footer's
		// base reaches it.
		{"Canon MakerNote moved by an editor, footer names the old offset", "shot.jpg",
			jpegWithExif(canonNote{make: "Canon", entries: owner, moved: 48}.build()), true},
		{"Canon MakerNote in a bare CR2-shaped TIFF", "shot.cr2",
			canonNote{bigEndian: false, make: "Canon", entries: owner}.build(), true},
		{"Make with the full company name", "shot.jpg",
			jpegWithExif(canonNote{make: "Canon Inc.", entries: owner}.build()), true},
		{"CR3 CMT3 box, little-endian", "shot.cr3", cr3WithCMT3(false, owner), true},
		{"CR3 CMT3 box, big-endian", "shot.cr3", cr3WithCMT3(true, owner), true},
		// Not a person field: another Canon tag, or another vendor's MakerNote
		// whose tag numbers mean something else.
		{"short name in Canon ImageType", "shot.jpg",
			jpegWithExif(canonNote{make: "Canon", entries: []exifEntry{{0x0006, 2, asciiValue("Zedqx")}}}.build()), false},
		{"another vendor's MakerNote", "shot.jpg",
			jpegWithExif(canonNote{make: "Nikqx", entries: owner}.build()), false},
		{"no Make at all", "shot.jpg",
			jpegWithExif(canonNote{make: "", entries: owner}.build()), false},
		{"OwnerName tag in a box that is not CMT3", "shot.cr3",
			bytes.Replace(cr3WithCMT3(false, owner), []byte("CMT3"), []byte("CMT1"), 1), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sc := &Scanner{identity: Identity{GitUserName: "Zedqx"}, identSev: DefaultIdentitySeverities()}
			got := hasKind(sc.scanBytes(c.body, secretPatterns(DefaultPatterns()), c.logical), kindRealName)
			if got != c.want {
				t.Errorf("real_name reported = %v, want %v", got, c.want)
			}
		})
	}
}

// TestCanonMakerNoteFindingIsNotDoubled pins that the MakerNote adds only
// what the raw scan missed: a long owner name the raw bytes already hold is
// reported once, the footer's second base included.
func TestCanonMakerNoteFindingIsNotDoubled(t *testing.T) {
	id := synthIdentity()
	sc := &Scanner{identity: Identity{GitUserName: id.GitUserName}, identSev: DefaultIdentitySeverities()}
	for _, moved := range []int{0, 48} {
		body := jpegWithExif(canonNote{make: "Canon", entries: []exifEntry{ownerName(id.GitUserName)}, moved: moved}.build())
		n := 0
		for _, f := range sc.scanBytes(body, secretPatterns(DefaultPatterns()), "shot.jpg") {
			if f.Kind == kindRealName {
				n++
			}
		}
		if n != 1 {
			t.Errorf("moved=%d: a long OwnerName the raw bytes hold was reported %d times, want once", moved, n)
		}
	}
}

// TestCanonGateIsNotClosedByALaterPage pins the drainScan review's MINOR-2:
// the Canon gate took the Make of the last top-level directory the chain
// reached, so an IFD1 naming another maker closed the gate IFD0's "Canon"
// opened and the OwnerName was skipped. A Make on any page opens the gate and
// no later page closes it.
func TestCanonGateIsNotClosedByALaterPage(t *testing.T) {
	b := canonNote{make: "Canon", entries: []exifEntry{ownerName("Zedqx")}}.build()
	// IFD1, appended after the MakerNote and linked from IFD0's next-IFD
	// field (IFD0 at 8: count, two entries, link): one Make naming another maker.
	const ifd0Link = 8 + 2 + 2*12
	ifd1At := len(b)
	bo := binary.LittleEndian
	bo.PutUint32(b[ifd0Link:], uint32(ifd1At))
	b = bo.AppendUint16(b, 1)
	b = bo.AppendUint16(b, 0x010f)
	b = bo.AppendUint16(b, 2)
	b = bo.AppendUint32(b, 6)
	b = bo.AppendUint32(b, uint32(ifd1At+2+12+4))
	b = bo.AppendUint32(b, 0)
	b = append(b, asciiValue("Nikqx")...)
	sc := &Scanner{identity: Identity{GitUserName: "Zedqx"}, identSev: DefaultIdentitySeverities()}
	if !hasKind(sc.scanBytes(jpegWithExif(b), secretPatterns(DefaultPatterns()), "shot.jpg"), kindRealName) {
		t.Error("an IFD1 Make naming another maker hid IFD0's Canon OwnerName")
	}
}

// TestMalformedMakerNoteIsSkipped pins the trust boundary: a MakerNote is
// untrusted file bytes, and one that is truncated, declares more entries than
// it holds, names values past the data, or carries a footer pointing anywhere
// is skipped with no finding and no panic.
func TestMalformedMakerNoteIsSkipped(t *testing.T) {
	good := canonNote{make: "Canon", entries: []exifEntry{ownerName("Zedqx")}}.build()
	// noteAt is where build puts the MakerNote: header, IFD0, sub-IFD, "Canon\0".
	const noteAt = 8 + 30 + 18 + 6
	mutate := func(f func(b []byte) []byte) []byte {
		return jpegWithExif(f(append([]byte{}, good...)))
	}
	cases := map[string][]byte{
		"entry count 65535": mutate(func(b []byte) []byte {
			binary.LittleEndian.PutUint16(b[noteAt:], 0xffff)
			return b
		}),
		"entry count zero": mutate(func(b []byte) []byte {
			binary.LittleEndian.PutUint16(b[noteAt:], 0)
			return b
		}),
		"value offset past the data": mutate(func(b []byte) []byte {
			binary.LittleEndian.PutUint32(b[noteAt+2+8:], 0xfffffff0)
			return b
		}),
		"value count past the data": mutate(func(b []byte) []byte {
			binary.LittleEndian.PutUint32(b[noteAt+2+4:], 0x7fffffff)
			return b
		}),
		"MakerNote truncated mid-entry": mutate(func(b []byte) []byte {
			return b[:noteAt+7]
		}),
		"MakerNote length past the data": mutate(func(b []byte) []byte {
			binary.LittleEndian.PutUint32(b[8+30+2+4:], 0xffffffff)
			return b
		}),
		"footer naming an offset past the data": func() []byte {
			b := canonNote{make: "Canon", entries: []exifEntry{{0x0006, 2, asciiValue("Canon EOS Qx")}, {0x0009, 2, asciiValue("Zedqx")}}, footer: true}.build()
			// Break the OwnerName's own offset so only a footer base could reach it.
			binary.LittleEndian.PutUint32(b[noteAt+2+12+8:], 0xfffffff0)
			binary.LittleEndian.PutUint32(b[len(b)-4:], 0xfffffff0)
			return jpegWithExif(b)
		}(),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			sc := &Scanner{identity: Identity{GitUserName: "Zedqx"}, identSev: DefaultIdentitySeverities()}
			if hasKind(sc.scanBytes(body, secretPatterns(DefaultPatterns()), "shot.jpg"), kindRealName) {
				t.Error("a malformed MakerNote produced a real_name finding")
			}
		})
	}
	// A footer whose original offset is far past the MakerNote gives a base
	// below zero; it is ignored rather than wrapped.
	b := canonNote{make: "Canon", entries: []exifEntry{ownerName("Zedqx")}, footer: true}.build()
	binary.LittleEndian.PutUint32(b[len(b)-4:], 0xffffff00)
	meta := metadataFields{data: b}
	exifView(b, &meta)
	for _, s := range meta.persons {
		if s.start < 0 || s.end > len(b) || s.start > s.end {
			t.Fatalf("person span %v outside the data (%d bytes)", s, len(b))
		}
	}
}

// TestCanonDirIsWalkedOnce pins that a MakerNote directory named by many
// headers or MakerNote entries is walked once: the walk's entry budget is
// spent on it once, not once per name.
func TestCanonDirIsWalkedOnce(t *testing.T) {
	b := canonNote{make: "Canon", entries: []exifEntry{ownerName("Zedqx"), {0x0006, 2, asciiValue("Canon EOS Qx")}}}.build()
	const noteAt = 8 + 30 + 18 + 6
	w := exifWalk{data: b, entries: 100, bytes: len(b), dirs: map[[2]int]bool{}, walked: map[int]bool{}, values: map[int]bool{}, notes: map[int]bool{}, meta: &metadataFields{data: b}}
	for range 3 {
		w.canonDir([]int{0}, noteAt, len(b), binary.LittleEndian)
	}
	if w.entries != 98 {
		t.Errorf("three walks of one two-entry directory spent %d entries, want 2", 100-w.entries)
	}
	if len(w.meta.persons) != 1 {
		t.Errorf("three walks recorded %d person spans, want 1", len(w.meta.persons))
	}
}

// TestMakerNoteWorkIsLinear holds the MakerNote walk to the byte scan's cost
// class: MakerNotes that all name one far directory with a large entry count
// are walked once, not once per MakerNote, and footers that each imply a
// second base cost no more than the bytes do.
func TestMakerNoteWorkIsLinear(t *testing.T) {
	if raceEnabled {
		t.Skip("a deterministic count gains nothing under -race; the uninstrumented run asserts it")
	}
	sc := &Scanner{identity: Identity{GitUserName: "Zedqx"}, identSev: DefaultIdentitySeverities()}
	secrets := secretPatterns(DefaultPatterns())
	shapes := map[string]func(n int) []byte{
		"Canon MakerNotes": func(n int) []byte {
			one := canonNote{make: "Canon", entries: []exifEntry{ownerName("Qqqqq"), {0x0006, 2, asciiValue("Canon EOS Qx")}}, moved: 48}.build()
			return bytes.Repeat(jpegWithExif(one), n)
		},
		"CMT3 boxes": func(n int) []byte {
			return bytes.Repeat(cr3WithCMT3(false, []exifEntry{ownerName("Qqqqq")}), n)
		},
	}
	for name, build := range shapes {
		t.Run(name, func(t *testing.T) {
			charge := func(data []byte) int {
				total := 0
				scanMeter.tally = func(_ string, n int) { total += n }
				defer func() { scanMeter.tally = nil }()
				sc.scanBytes(data, secrets, "f")
				return total
			}
			lo, hi := charge(build(64)), charge(build(256))
			if lo == 0 {
				t.Fatal("the shape charged nothing; it pins nothing")
			}
			if growth := float64(hi) / float64(lo); growth > linearCostBar {
				t.Errorf("quadrupling the payload multiplied the charge by %.2fx, want at most %.1fx", growth, linearCostBar)
			}
		})
	}
}

// FuzzCanonMakerNote feeds hostile bytes shaped like Canon MakerNotes to the
// EXIF walk: none may panic, every position the view maps to lies inside the
// data, and every person span it records does too. The seeds are synthetic
// headers built above; no real photo, name or serial is in them.
func FuzzCanonMakerNote(f *testing.F) {
	owner := []exifEntry{{0x0006, 2, asciiValue("Canon EOS Qx")}, ownerName("Zedqx")}
	f.Add(canonNote{make: "Canon", entries: owner}.build())
	f.Add(canonNote{bigEndian: true, noteBigEndian: true, make: "Canon", entries: owner}.build())
	f.Add(canonNote{make: "Canon", noteBigEndian: true, entries: owner, moved: 48}.build())
	f.Add(canonNote{make: "Canon", entries: []exifEntry{ownerName("Zedqx")}, footer: true}.build())
	f.Add(cr3WithCMT3(false, owner))
	f.Add(cr3WithCMT3(true, owner))
	f.Fuzz(func(t *testing.T, data []byte) {
		meta := metadataFields{data: data}
		v, ok := exifView(data, &meta)
		if ok {
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
