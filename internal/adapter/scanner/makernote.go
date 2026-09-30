package scanner

import (
	"bytes"
	"encoding/binary"
)

// makernote.go — the EXIF walk's reader of a camera vendor's MakerNote
// (iss-2609291653241690).
//
// The Exif sub-IFD's MakerNote (tag 0x927c, UNDEFINED; CIPA DC-008, 4.6.5)
// is a block each vendor lays out its own way: the standard names no header,
// byte order or offset base for it, so no generic pointer walk can follow it.
// A Canon camera writes the owner's name there (OwnerName, tag 0x0009, ASCII,
// padded to 32 bytes), where no person key is written as text before it, so a
// short single-token owner name was dropped as chance, exactly as a short
// EXIF Artist was before exifView read it.
//
// What is read. Canon's layout, as ExifTool documents and reads it
// (Image::ExifTool::MakerNotes, MakerNoteCanon, and the Canon tag table
// Image::ExifTool::Canon::Main, https://exiftool.org/TagNames/Canon.html):
//
//   - a MakerNote is Canon's when the IFD0 Make of its TIFF begins "Canon"
//     (a Make on any page of the chain opens the gate, and a later page's
//     never closes it, since opening it cannot hide a name);
//   - it starts with an IFD and no header of its own, whose value offsets are
//     relative to the TIFF header, like the EXIF IFDs around it;
//   - its byte order is normally the TIFF's, but an editor may rewrite it in
//     the other one, so the order is the one whose entry count fits the block
//     (ExifTool's ByteOrder 'Unknown');
//   - a writer may end it with a TIFF footer — the byte-order mark, 42, and
//     the offset the block stood at when its offsets were written — which an
//     editor that moves the block leaves behind; the values are then read at
//     the base the footer implies as well as at the TIFF header (ExifTool's
//     FixBase applies the footer and falls back to the header when a buggy
//     editor updated the offsets but not the footer; reading both covers
//     either without judging which the writer did);
//   - a CR3 file carries the MakerNote as its own TIFF stream, header and
//     all, in a CMT3 box, with no Make beside it (ExifTool's
//     Image::ExifTool::Canon uuid table, CMT3 => MakerNoteCanon).
//
// Of Canon's tags only OwnerName is read: it is the one that names a person,
// and the rest (the model, the firmware, the lens, binary sub-tables) are not
// text a person wrote.
//
// What is not read. Every other vendor's MakerNote is skipped, with no
// finding: each lays its block out differently (Nikon's carries its own TIFF
// header and base, Olympus, Pentax, Fujifilm, Sony, Panasonic and Apple open
// with a signature and set their own base), so reading one needs its own
// reader and its own primary source. So is a MakerNote copied into a DNG's
// Adobe MakN private data, and a Canon one whose Make is missing. The raw
// byte scan still reads a name of eight bytes or more, or of several words,
// wherever it sits in any of them; what the skip leaves is a short
// single-token name in a vendor block nobody has read.
//
// The block is untrusted file bytes. Every offset and count is checked against
// the data before it is read (w.value, w.offset), the entry count is capped at
// maxIFDEntries and charged against the walk's shared entry budget, the value
// bytes against its shared byte budget, and each MakerNote directory is walked
// once, so no header-declared size allocates or loops beyond what the data
// holds, and a malformed block is skipped rather than half-read.

// canonOwnerNameTag is Canon's MakerNote OwnerName.
const canonOwnerNameTag = 0x0009

// cr3CanonMakerNoteBox is the CR3 box whose payload is Canon's MakerNote as a
// TIFF stream (ExifTool: Canon uuid table, CMT3).
const cr3CanonMakerNoteBox = "CMT3"

// makerNote is a MakerNote value's raw offset and length in the data.
type makerNote struct {
	at, size int
}

// isCanonMake reports whether a Make value names Canon ("Canon", or
// "Canon Inc." and kin), the condition ExifTool reads a MakerNote as Canon's on.
func isCanonMake(val []byte) bool {
	return bytes.HasPrefix(val, []byte("Canon"))
}

// canonMakerNote walks the Canon MakerNote n found under the TIFF header at
// base: its directory at the block's start, in whichever byte order its entry
// count fits the block, its values at the header's base and at the one a
// matching footer implies.
func (w *exifWalk) canonMakerNote(base int, bo binary.ByteOrder, n makerNote) {
	block := w.data[n.at : n.at+n.size]
	bo, ok := canonByteOrder(block, bo)
	if !ok {
		return
	}
	bases := []int{base}
	if fb, ok := canonFooterBase(block, bo, base, n.at); ok && fb != base {
		bases = append(bases, fb)
	}
	w.canonDir(bases, n.at, n.at+n.size, bo)
}

// canonByteOrder returns the byte order a MakerNote block's directory is
// written in: the TIFF's when its entry count fits the block, the other one
// when only that fits, and neither when no count does. A count fits when it
// is at least one, no more than maxIFDEntries, and its entries lie inside the
// block.
func canonByteOrder(block []byte, bo binary.ByteOrder) (binary.ByteOrder, bool) {
	if len(block) < 2 {
		return nil, false
	}
	fits := func(o binary.ByteOrder) bool {
		c := int(o.Uint16(block))
		return c >= 1 && c <= maxIFDEntries && 2+12*c <= len(block)
	}
	other := binary.ByteOrder(binary.BigEndian)
	if bo == binary.BigEndian {
		other = binary.LittleEndian
	}
	switch {
	case fits(bo):
		return bo, true
	case fits(other):
		return other, true
	}
	return nil, false
}

// canonFooterBase reads the Canon TIFF footer a MakerNote block may end with
// — the byte-order mark of the block's own order, 42, and the offset the
// block stood at, relative to the TIFF header, when its value offsets were
// written — and returns the header base those offsets resolve against now:
// base moved by as far as the block moved since. A footer in the other byte
// order is not Canon's (ExifTool checks the same), and a base that would lie
// outside the data is refused rather than wrapped.
func canonFooterBase(block []byte, bo binary.ByteOrder, base, at int) (int, bool) {
	if len(block) < 2+12+8 {
		return 0, false
	}
	f := block[len(block)-8:]
	mark := "II*\x00"
	if bo == binary.BigEndian {
		mark = "MM\x00*"
	}
	if string(f[:4]) != mark {
		return 0, false
	}
	// at-base is the block's offset now; the footer holds the old one. The
	// sum is made in int64 so a hostile 32-bit offset cannot overflow it.
	moved := int64(at-base) - int64(bo.Uint32(f[4:]))
	nb := int64(base) + moved
	if nb < 0 || nb > int64(at) {
		return 0, false
	}
	return int(nb), true
}

// canonDir walks one Canon MakerNote directory at raw offset at, whose
// entries must end by end, reading OwnerName as a person tag with its value
// resolved against each of bases. The directory is walked once however many
// headers or MakerNote entries name it.
func (w *exifWalk) canonDir(bases []int, at, end int, bo binary.ByteOrder) {
	if at+2 > end || w.notes[at] {
		return
	}
	w.notes[at] = true
	n := min(int(bo.Uint16(w.data[at:])), maxIFDEntries)
	for k := 0; k < n && w.entries > 0; k++ {
		e := at + 2 + 12*k
		if e+12 > end {
			return
		}
		w.entries--
		scanMeter.charge(stageEXIF, 12)
		tag, typ, count := bo.Uint16(w.data[e:]), bo.Uint16(w.data[e+2:]), bo.Uint32(w.data[e+4:])
		if tag != canonOwnerNameTag || typ != 2 {
			continue
		}
		for _, b := range bases {
			v, size, ok := w.value(b, e, bo, count)
			if !ok {
				continue
			}
			if !w.values[v] {
				if size > w.bytes {
					continue
				}
				w.values[v] = true
				w.bytes -= size
				w.read(w.data[v:v+size], v, exifASCII, bo)
			}
			w.meta.persons = append(w.meta.persons, span{v, v + size})
		}
	}
}
