package scanner

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The container decoder closes the GHSA-9wv7 residual (iss-2608291832160371).
// The byte branch reads a skip-listed payload file and scans its RAW bytes,
// which covers a plaintext region (an uncompressed tar's entries, PNG tEXt)
// and nothing else: a token inside a gzip member, a zip entry or a DEFLATEd
// PNG zTXt chunk shipped with not even its prefix visible. This file decodes
// the formats the standard library gives us — zip, gzip, bzip2, zlib, tar and
// PNG's compressed chunks — and scans what comes out with the SAME byte rules
// (scanBytes), so the verdict on an entry is the verdict its bytes would get
// loose in the payload. Everything else stays exactly as it was: reported
// ContentUnverified, with the reason and the magic-keyed format saying which
// of "decoded and clean" and "could not be read" this file is.
//
// Two properties are load-bearing.
//
// MAGIC BYTES, NEVER THE EXTENSION. detectFormat sniffs the leading bytes, so
// a zip renamed .png is detected as a zip, decoded as one, and labelled one —
// the record names the old name-keyed label as the defect.
//
// BOUNDED WORK. Decompression is an amplification surface: a few hundred
// kilobytes of gzip inflates to gigabytes, and an unbounded decompressor here
// would be a denial of service this change introduced. Every decoded byte is
// drawn from one decodeBudget per payload file, nesting is depth-bounded, and
// every decode operation — an archive entry, a compressed PNG chunk — is
// counted, because a bomb can be built out of operations that cost no bytes at
// all. A container that would exceed any of the three is REFUSED — never
// decoded, never called clean — and says which bound it hit.
//
// ACCOUNTED COVERAGE. "Decoded" claims the whole of a file was read, so each
// walk proves that rather than assuming it: a zip's entries must tile the
// archive and its directory must tile the region up to its end record, a tar
// must end where its end-of-archive marker says and pad its entries with
// zeros, a stream's trailer is covered like the stream, a region counts as
// covered by the byte scan only if ALL of it reads as text, and every span
// whose length or content the format lets the file choose — names, comments,
// extra fields, gzip header fields, tar header and PAX records, a PNG chunk
// the spec calls plaintext — is judged rather than taken on the spec's word.
// Where a walk cannot account for a byte, the file stays ContentUnverified
// with the reason: the bytes may be innocent, but nothing read them. What is
// NOT judged is stated at coverStructuralField and at IDAT, because a residual
// that is written down can be argued with.

// The decode bounds, per payload file.
//
// maxDecodedBytes is the TOTAL inflated output one payload file may produce,
// across every entry and every nesting level. It equals maxBinaryScanBytes,
// the existing byte-scan read cap, which is the argument for the number: a
// payload file already costs the scanner one maxBinaryScanBytes read plus the
// byte scan over it, and decoding adds at most one more such buffer of BYTES —
// the same 4 MiB figure the surface already reports and tests hold equal to
// the sibling privacy lint's cap. Reads draw from the remaining budget with an
// io.LimitReader, so a bomb is stopped AT the bound rather than measured after
// blowing through it: a gzip member that inflates to 256 MiB allocates 4 MiB
// and is refused. Bytes are only one of the two costs, though; see
// maxDecodeEntries for the operation count, without which a file that draws no
// budget at all can still cost minutes.
//
// maxDecodeDepth bounds nesting (a .tgz is gzip inside tar, two levels; a zip
// inside a .tgz is three), so a container nested inside itself thousands of
// times cannot drive recursion. Four levels covers the real shapes with one to
// spare; deeper is a refusal, not a stack.
//
// maxDecodeEntries bounds the number of DECODE OPERATIONS one payload file may
// cost: archive entries and PNG compressed chunks alike. The output budget
// bounds their bytes and nothing else, so on its own it bounds nothing here —
// an archive of empty entries, or a 4 MiB PNG of empty zTXt chunks, draws zero
// budget while paying one inflate and one rule pass apiece, which measured 41 s
// for a single file before this bound existed. Both walks charge an entry
// BEFORE deciding to skip it, so a skipped entry is never free.
//
// With all three bounds in force, a payload file's decode costs at most one
// further maxBinaryScanBytes of inflated output and a rule-pass count that is
// a small multiple of maxDecodeEntries — two per archive entry (its header and
// its body), one per decoded region, one per covered structural field — over
// inputs that together fit in that same budget. The same order as the byte
// scan the file already pays for, rather than the unbounded multiple an
// operation count would otherwise allow.
const (
	maxDecodedBytes  = maxBinaryScanBytes
	maxDecodeDepth   = 4
	maxDecodeEntries = 1024
)

// errDecodeBudget is returned by a budgeted read whose stream did not end
// within the remaining output budget: the signature of a decompression bomb.
var errDecodeBudget = errors.New("scanner: decode output budget exceeded")

// Format labels. They name what the MAGIC BYTES say, not what the extension
// claims, and travel into ScanResult.ContentFormat so the report can be read
// against the filename.
const (
	formatZip     = "zip"
	formatGzip    = "gzip"
	formatBzip2   = "bzip2"
	formatZlib    = "zlib"
	formatTar     = "tar"
	formatPNG     = "png"
	formatUnknown = "unrecognised"
)

// decodableFormats is the closed set this file can open. Its polarity matches
// the byte branch's: only a format named here is ever promoted out of
// ContentUnverified, so a format nobody taught the decoder stays flagged
// rather than passing by omission.
var decodableFormats = map[string]struct{}{
	formatZip: {}, formatGzip: {}, formatBzip2: {}, formatZlib: {}, formatTar: {}, formatPNG: {},
}

// detectFormat names the format of data from its leading bytes. It recognises
// more formats than it can decode: naming "jpeg" or "7z" is what lets the
// unverified report say WHY a file could not be read, and what makes the
// mislabelling in the record visible — a zip renamed .png reports "zip".
func detectFormat(data []byte) string {
	switch {
	case len(data) >= 4 && data[0] == 'P' && data[1] == 'K' &&
		(data[2] == 3 && data[3] == 4 || data[2] == 5 && data[3] == 6 || data[2] == 7 && data[3] == 8):
		return formatZip
	case len(data) >= 3 && data[0] == 0x1f && data[1] == 0x8b && data[2] == 8:
		return formatGzip
	case len(data) >= 4 && data[0] == 'B' && data[1] == 'Z' && data[2] == 'h' && data[3] >= '1' && data[3] <= '9':
		return formatBzip2
	case bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")):
		return formatPNG
	case len(data) >= 2 && data[0] == 0x78 && (data[1] == 0x01 || data[1] == 0x5e || data[1] == 0x9c || data[1] == 0xda) &&
		(uint16(data[0])<<8|uint16(data[1]))%31 == 0:
		return formatZlib
	case len(data) >= 263 && bytes.Equal(data[257:262], []byte("ustar")):
		return formatTar
	case bytes.HasPrefix(data, []byte("\xff\xd8\xff")):
		return "jpeg"
	case bytes.HasPrefix(data, []byte("%PDF-")):
		return "pdf"
	case bytes.HasPrefix(data, []byte("GIF87a")), bytes.HasPrefix(data, []byte("GIF89a")):
		return "gif"
	case len(data) >= 12 && bytes.HasPrefix(data, []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return "webp"
	case len(data) >= 12 && bytes.HasPrefix(data, []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WAVE")):
		return "wav"
	case bytes.HasPrefix(data, []byte("7z\xbc\xaf\x27\x1c")):
		return "7z"
	case bytes.HasPrefix(data, []byte("\xfd7zXZ\x00")):
		return "xz"
	case bytes.HasPrefix(data, []byte("\x28\xb5\x2f\xfd")):
		return "zstd"
	case len(data) >= 12 && bytes.Equal(data[4:8], []byte("ftyp")):
		return "mp4"
	case bytes.HasPrefix(data, []byte("ID3")), len(data) >= 2 && data[0] == 0xff && data[1]&0xfe == 0xfa:
		// A tagged mp3, or an MPEG-1 Layer III frame sync. The frame-sync form
		// is kept narrow on purpose: a loose sync test (any 11 set bits) reads
		// half the binaries in the world as mp3, and this label is what the
		// unverified report tells a reader the file IS.
		return "mp3"
	case bytes.HasPrefix(data, []byte("\x1a\x45\xdf\xa3")):
		return "matroska"
	case bytes.HasPrefix(data, []byte("SQLite format 3\x00")):
		return "sqlite"
	}
	return formatUnknown
}

// knownFormats is every format detectFormat can name. It is a declared list,
// not a derivation, for the reason byteScanPolicy's table is: a format added
// to the sniff without a signature in containerSignatures is a member that
// hides in any structural field, and TestEveryDetectedFormatHasASignature
// fails until the two agree.
func knownFormats() []string {
	return []string{
		formatZip, formatGzip, formatBzip2, formatZlib, formatTar, formatPNG,
		"jpeg", "pdf", "gif", "webp", "wav", "7z", "xz", "zstd", "mp4", "mp3",
		"matroska", "sqlite",
	}
}

// decodeBudget is one payload file's allowance: inflated bytes and entries.
// It is threaded through every level of the walk, so nesting cannot reset it —
// a bomb hidden three archives deep draws from the same 4 MiB.
//
// reads and drained are not allowances but tallies: every decode operation,
// whichever walk asks for it, drains its stream through read, so the number
// of reads and the bytes they pulled out of the decompressors are the work the
// bounds exist to cap. A test judges a bound by them rather than by a clock,
// which reports the machine's load as readily as the scan's cost.
type decodeBudget struct {
	bytesLeft   int
	entriesLeft int
	reads       int
	drained     int
}

// read drains r into memory, never allocating past the remaining budget: it
// reads one byte MORE than the budget allows, and a stream that still has not
// ended is refused as errDecodeBudget. That one byte is the whole bomb
// defence — the decompressor is never asked for its full output.
func (b *decodeBudget) read(r io.Reader) ([]byte, error) {
	b.reads++
	data, err := io.ReadAll(io.LimitReader(r, int64(b.bytesLeft)+1))
	b.drained += len(data)
	if err != nil {
		return nil, err
	}
	if len(data) > b.bytesLeft {
		return nil, errDecodeBudget
	}
	b.bytesLeft -= len(data)
	return data, nil
}

// entry claims one entry from the budget.
func (b *decodeBudget) entry() bool {
	if b.entriesLeft <= 0 {
		return false
	}
	b.entriesLeft--
	return true
}

// inflate decompresses a zlib stream under the budget.
func (b *decodeBudget) inflate(data []byte) ([]byte, error) {
	zr, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return b.read(zr)
}

// decodeContent decodes one skip-listed payload file as far as the known
// formats allow, scanning every decoded region with the byte rules. It returns
// the magic-keyed format, the findings from the decoded regions (the caller
// has already scanned the raw bytes), whether the WHOLE of the content was
// read, and — when it was not — why.
//
// The caller keeps a decoded file out of ContentUnverified only on a true
// second return: a zip whose one entry is a JPEG is decoded as far as the zip
// goes, its readable entries scanned, and still reported unverified naming the
// entry, because part of what ships was never read.
func (s *Scanner) decodeContent(data []byte, secrets []Pattern, logical string) (string, []Finding, bool, string) {
	format := detectFormat(data)
	budget := &decodeBudget{bytesLeft: maxDecodedBytes, entriesLeft: maxDecodeEntries}
	var findings []Finding
	decoded, why := s.decodeInto(data, secrets, logical, budget, 1, &findings)
	return format, findings, decoded, why
}

// decodeInto decodes data at one nesting level, appending the findings of
// every decoded region to out. It reports whether the whole of data was read
// and, if not, the reason.
func (s *Scanner) decodeInto(data []byte, secrets []Pattern, label string, b *decodeBudget, depth int, out *[]Finding) (bool, string) {
	if depth > maxDecodeDepth {
		return false, "nested past the " + strconv.Itoa(maxDecodeDepth) + "-level decode depth bound"
	}
	format := detectFormat(data)
	switch format {
	case formatZip:
		return s.decodeZip(data, secrets, label, b, depth, out)
	case formatTar:
		return s.decodeTar(data, secrets, label, b, depth, out)
	case formatPNG:
		return s.decodePNG(data, secrets, label, b, depth, out)
	case formatGzip:
		src := bytes.NewReader(data)
		zr, err := gzip.NewReader(src)
		if err != nil {
			return false, formatGzip + ": stream would not decode"
		}
		defer zr.Close()
		// A gzip header carries three fields of the writer's choosing before
		// any compressed byte: FEXTRA, which is length-prefixed rather than
		// NUL-terminated and holds up to 64 KiB, plus the stored filename and
		// comment. They are structural fields like a zip's.
		for _, f := range []struct {
			name  string
			bytes []byte
		}{{"extra", zr.Extra}, {"name", []byte(zr.Name)}, {"comment", []byte(zr.Comment)}} {
			if ok, why := s.coverStructuralField(f.bytes, label+"!"+f.name); !ok {
				return false, why
			}
		}
		return s.decodeStream(data, src, zr, secrets, label, formatGzip, b, depth, out)
	case formatBzip2:
		src := bytes.NewReader(data)
		return s.decodeStream(data, src, bzip2.NewReader(src), secrets, label, formatBzip2, b, depth, out)
	case formatZlib:
		src := bytes.NewReader(data)
		zr, err := zlib.NewReader(src)
		if err != nil {
			return false, formatZlib + ": stream would not decode"
		}
		defer zr.Close()
		return s.decodeStream(data, src, zr, secrets, label, formatZlib, b, depth, out)
	case formatUnknown:
		return false, "unrecognised format"
	}
	return false, "no decoder for " + format
}

// decodeStream reads one compressed stream under the budget and covers both
// what comes out AND what the decompressor left behind. The trailer matters:
// gzip refuses trailing bytes, but zlib stops at its Adler-32 and bzip2 at its
// footer without complaining, so a valid stream with a second, unrelated member
// appended would otherwise be waved through on the strength of the first.
//
// src is the bytes.Reader the decompressor drank from. What is left in it is
// exactly the trailer for ZLIB, which stops on its checksum; gzip and bzip2
// over-read by a few bytes looking for a further member, but both then FAIL —
// "invalid header", "bad magic value in continuation file" — so b.read returns
// an error and the file is unverified before this branch is reached. The
// branch is what makes zlib safe and what would catch a future format that
// leaves a trailer without complaining; a format that over-reads and does not
// error would need its own boundary rather than this one.
func (s *Scanner) decodeStream(data []byte, src *bytes.Reader, r io.Reader, secrets []Pattern, label, format string, b *decodeBudget, depth int, out *[]Finding) (bool, string) {
	inner, err := b.read(r)
	if err != nil {
		return false, streamWhy(format, err)
	}
	if ok, why := s.cover(inner, secrets, label, b, depth+1, out); !ok {
		return false, why
	}
	if left := src.Len(); left > 0 {
		return s.cover(data[len(data)-left:], secrets, label+"!trailer", b, depth+1, out)
	}
	return true, ""
}

// cover treats one decoded region exactly as ScanBundle treats a top-level
// payload file: its raw bytes are scanned FIRST, then the format walk reads
// what those bytes hide. Scanning first is the point — a walk that leans on
// "the caller already scanned these bytes" is right only at the top level, and
// a nested PNG's plaintext tEXt chunk, a nested zip's directory and archive
// comment, or a nested gzip's stored filename would otherwise be read by
// nobody while the file was reported decoded. An uncompressed region pays for
// this by being scanned twice (the parent's raw pass and its own), which
// duplicates a finding rather than hiding one.
//
// A region that is not a container abcd decodes is covered only when the WHOLE
// of it reads as text. The 8 KiB sniff is not enough for that claim: a region
// that opens with prose and carries a compressed member past the window would
// be called fully covered on the strength of its first 8 KiB.
func (s *Scanner) cover(data []byte, secrets []Pattern, label string, b *decodeBudget, depth int, out *[]Finding) (bool, string) {
	*out = append(*out, s.scanBytes(data, secrets, label)...)
	format := detectFormat(data)
	if _, ok := decodableFormats[format]; ok {
		return s.decodeInto(data, secrets, label, b, depth, out)
	}
	if isTextWhole(data) {
		return true, ""
	}
	if format == formatUnknown {
		return false, label + ": opaque content in an unrecognised format"
	}
	return false, label + ": no decoder for " + format
}

// isTextWhole is isText's stricter sibling: every byte, not a sniff. isText
// answers "should this file take the text branch", where a sniff is the right
// trade; this answers "did the byte scan see all of this", where it is not.
func isTextWhole(data []byte) bool {
	return bytes.IndexByte(data, 0) < 0 && utf8.Valid(data)
}

// decodeZip walks a zip archive's entries and then checks that those entries
// account for the archive's bytes.
//
// Every entry is READ, including one whose mode bits claim it is a directory:
// the directory bit comes from the external attributes, not from the entry's
// content, so an entry can call itself a directory and still carry a body the
// reader will happily inflate. A real directory entry is empty, so reading it
// costs nothing.
func (s *Scanner) decodeZip(data []byte, secrets []Pattern, label string, b *decodeBudget, depth int, out *[]Finding) (bool, string) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return false, formatZip + ": archive would not open"
	}
	verified, why := true, ""
	for _, f := range zr.File {
		if !b.entry() {
			return false, formatZip + ": more than " + strconv.Itoa(maxDecodeEntries) + " entries"
		}
		name := entryLabel(label, f.Name)
		s.scanEntryHeader(secrets, name, out, f.Name, f.Comment)
		// A name is up to 65535 bytes of the archive's choosing, accounted by
		// the tiling through its length field and byte-scanned as if it were a
		// path — a structural field like the comment beside it.
		if ok, w := s.coverStructuralField([]byte(f.Name), name+"!name"); !ok && verified {
			verified, why = false, w
		}
		body, tail, err := b.zipEntryBody(data, f)
		if err != nil {
			if errors.Is(err, errDecodeBudget) {
				return false, streamWhy(formatZip, err)
			}
			// An entry compressed with a method the standard library does not
			// implement (zstd, bzip2-in-zip), or one whose extent does not sit
			// inside the file: unread, so unverified.
			verified, why = false, name+": entry could not be read"
			continue
		}
		if ok, w := s.cover(body, secrets, name, b, depth+1, out); !ok && verified {
			verified, why = false, w
		}
		if len(tail) > 0 {
			if ok, w := s.cover(tail, secrets, name+"!trailer", b, depth+1, out); !ok && verified {
				verified, why = false, w
			}
		}
		// An entry's comment and extra field are bytes the archive carries and
		// the accounting steps over. They are ordinary regions, covered as such.
		if ok, w := s.coverZipMetadata(f, secrets, name, b, depth, out); !ok && verified {
			verified, why = false, w
		}
	}
	if !verified {
		return verified, why
	}
	// The walk reads the central directory, so it sees the bytes the archive
	// ADMITS to. An archive can carry more: a local record ahead of the
	// catalogued ones, a gap between entries, or anything appended past the end
	// record — none of it catalogued, none of it read, and a zip whose entries
	// all scanned clean would otherwise be reported decoded on that evidence.
	return s.zipAccountsForItsBytes(data, zr, secrets, label, b, depth, out)
}

// errZipEntryUnreadable is an entry whose bytes the walk cannot get at: a
// compression method abcd does not implement, or an extent outside the file.
var errZipEntryUnreadable = errors.New("scanner: zip entry unreadable")

// zipEntryBody reads one entry from its PHYSICAL extent — the compressed bytes
// the directory points at — rather than through the archive reader's own
// notion of an entry. The reader is the wrong instrument here: it hands back an
// empty body for any entry whose name ends in "/" and whose recorded
// uncompressed size is zero, however much compressed data the entry actually
// carries, so an entry can name itself a directory and smuggle a whole DEFLATE
// stream past a walk that trusts the reader. Reading the extent asks what is
// there instead of what the archive says is there.
func (b *decodeBudget) zipEntryBody(data []byte, f *zip.File) ([]byte, []byte, error) {
	off, err := f.DataOffset()
	if err != nil {
		return nil, nil, errZipEntryUnreadable
	}
	if f.CompressedSize64 > uint64(len(data)) {
		return nil, nil, errZipEntryUnreadable
	}
	end := off + int64(f.CompressedSize64)
	if off < 0 || end < off || end > int64(len(data)) {
		return nil, nil, errZipEntryUnreadable
	}
	raw := data[off:end]
	switch f.Method {
	case zip.Store:
		body, err := b.read(bytes.NewReader(raw))
		return body, nil, err
	case zip.Deflate:
		src := bytes.NewReader(raw)
		fr := flate.NewReader(src)
		defer fr.Close()
		body, err := b.read(fr)
		if err != nil {
			return nil, nil, err
		}
		// A DEFLATE stream ends at its final block, and the entry's catalogued
		// extent can be longer than that: the reader stops at the end of the
		// stream while the accounting steps over the whole declared size, so
		// what sits between them is tiled over and never read. Same shape as a
		// stream's trailer, one level down.
		if left := src.Len(); left > 0 {
			return body, raw[len(raw)-left:], nil
		}
		return body, nil, nil
	}
	return nil, nil, errZipEntryUnreadable
}

// coverZipMetadata covers the two regions an entry carries beside its data: its
// comment, which is any run of bytes the archive likes, and its extra field.
func (s *Scanner) coverZipMetadata(f *zip.File, secrets []Pattern, label string, b *decodeBudget, depth int, out *[]Finding) (bool, string) {
	if ok, why := s.coverStructuralField([]byte(f.Comment), label+"!comment"); !ok {
		return false, why
	}
	return s.coverStructuralField(f.Extra, label+"!extra")
}

// containerSignatures are the leading bytes of every container format
// detectFormat names — decodable or not — and the structural-field rule
// searches for them anywhere in a region rather than only at its head. Every
// name knownFormats returns must appear here: a format the sniff can name with
// no signature in this list is a member that hides in every field the rule
// guards, which is exactly how zlib slipped through the first version of this
// list. TestEveryDetectedFormatHasASignature holds the two together.
type containerSignature struct {
	format string
	bytes  []byte
}

var containerSignatures = []containerSignature{
	{formatZip, []byte("PK\x03\x04")}, {formatZip, []byte("PK\x05\x06")}, {formatZip, []byte("PK\x07\x08")},
	{formatGzip, []byte("\x1f\x8b\x08")}, {formatBzip2, []byte("BZh")},
	{formatPNG, []byte("\x89PNG\r\n\x1a\n")}, {formatTar, []byte("ustar")},
	// The four zlib CMF/FLG pairs detectFormat accepts. Two bytes is a loose
	// signature and a chance hit in binary data is expected; the file is then
	// reported unverified naming the field, which is the honest verdict for
	// bytes nothing read, and never a false green.
	{formatZlib, []byte("\x78\x01")}, {formatZlib, []byte("\x78\x5e")},
	{formatZlib, []byte("\x78\x9c")}, {formatZlib, []byte("\x78\xda")},
	{"jpeg", []byte("\xff\xd8\xff")}, {"pdf", []byte("%PDF-")},
	{"gif", []byte("GIF87a")}, {"gif", []byte("GIF89a")},
	{"webp", []byte("RIFF")}, {"7z", []byte("7z\xbc\xaf\x27\x1c")},
	{"xz", []byte("\xfd7zXZ\x00")}, {"zstd", []byte("\x28\xb5\x2f\xfd")},
	{"mp4", []byte("ftyp")}, {"mp3", []byte("ID3")},
	{"matroska", []byte("\x1a\x45\xdf\xa3")}, {"sqlite", []byte("SQLite format 3\x00")},
}

// signatureIn names the format of the EARLIEST known container signature in a
// region — at its head or buried inside it — and reports whether one is there
// at all. "wav" is deliberately absent from the list it searches: RIFF covers
// both, and the field rule only needs to know that something is there.
func signatureIn(region []byte) (string, bool) {
	at, name := -1, ""
	for _, sig := range containerSignatures {
		i := signatureSearch(region, sig.bytes)
		if i >= 0 && (at < 0 || i < at) {
			at, name = i, sig.format
		}
	}
	return name, at >= 0
}

// signatureSearch is the field rule's one search primitive. It is a variable
// so that a test can count the bytes the rule searches, which is how the
// rule's one-pass cost is held without a clock; nothing else assigns it.
var signatureSearch = bytes.Index

// hidesAContainer is signatureIn's predicate half, which is what the detectors
// holding the signature list to the format sniff assert on.
func hidesAContainer(region []byte) bool {
	_, ok := signatureIn(region)
	return ok
}

// coverStructuralField judges one of the regions a format's STRUCTURE carries
// beside its content. Every span whose length or content the format lets the
// file choose goes through here: a zip's entry names (both copies), its extra
// fields (both copies), its entry and archive comments; a gzip header's
// FEXTRA, stored filename and comment; a tar header's name, linkname, uname
// and gname, and every PAX record and extended attribute it carries; and every
// PNG chunk the spec calls plaintext, an uncompressed iTXt included. The
// accounting counts these as part of the format and the byte scan reads every
// byte of them — but "the spec says this field is a timestamp" is a claim about
// well-formed files, and a payload is whatever its length field says it is, so
// a compressed member fits in any of them.
//
// The rule is deliberately the blunt one: a field in which no signature of a
// known container appears anywhere is covered; a field carrying one refuses to
// promote the file, naming the field and the format. It does NOT try to decode
// what it finds. The sophisticated version — decode the member, and treat one
// that will not decode as a false alarm — was written twice and shipped a hole
// both times, because "I could not read these bytes" was answering "so they
// are fine" where every other walk in this file answers "so the file is
// unverified"; one byte appended to a member inverts it, and an over-budget
// member inverts it too. The blunt rule cannot be inverted, costs one pass
// over the field rather than one decode per signature found in it (which was
// quadratic in the field's length, since each decode covered the whole tail),
// and fails closed by construction.
//
// The price is noise, and it is worth being concrete about where. Several
// signatures are short ASCII, so an ordinary source path inside an archive
// trips them: internal/ftype.go reads as mp4, docs/ID3-tags.md as mp3,
// RIFF.md as webp, cmd/ustar/ as tar. A chance two-byte zlib pair in a palette
// trips at roughly 1% for a 192-byte field, and the JPEG thumbnail a
// camera-export PNG carries in its eXIf chunk trips honestly. All of them
// report ContentUnverified naming the field. That is the honest verdict —
// nothing read those bytes — and it is the tier those files sat in before any
// of this existed.
//
// Three residuals, written down rather than papered over: a member encoded so
// that no signature abcd knows appears anywhere in it; a member buried inside
// inflated pixel data or an ICC profile, which are scanned as bytes because
// asking a region rule to vouch for them would refuse every real image; and
// the short NUL-terminated sub-fields inside a compressed PNG text chunk (a
// keyword, a language tag), which afterNulThen steps over — a member that
// survives NUL-termination there is a shape nobody has built.
func (s *Scanner) coverStructuralField(payload []byte, label string) (bool, string) {
	format, ok := signatureIn(payload)
	if !ok {
		return true, ""
	}
	return false, label + ": a " + format + " member nothing read"
}

// zipAccountsForItsBytes reports whether the catalogued entries, the central
// directory and the end record tile the archive with no bytes left over. It
// walks the entries in file order and requires each one's local header to
// begin exactly where the previous entry's data ended, which is what a zip
// nobody has tampered with looks like. Anything it cannot account for — a
// prepended record, a gap, a trailer, a zip64 layout it does not model — is a
// refusal to promote, not a finding: the bytes may be innocent, but nothing
// read them.
func (s *Scanner) zipAccountsForItsBytes(data []byte, zr *zip.Reader, secrets []Pattern, label string, b *decodeBudget, depth int, out *[]Finding) (bool, string) {
	end, ok := zipEndRecord(data)
	if !ok {
		return false, formatZip + ": its end-of-archive record does not close the file"
	}
	cdOffset := end.directoryOffset
	files := append([]*zip.File(nil), zr.File...)
	offsets := make(map[*zip.File]int64, len(files))
	for _, f := range files {
		off, err := f.DataOffset()
		if err != nil {
			return false, formatZip + ": an entry's data could not be located"
		}
		offsets[f] = off
	}
	sort.Slice(files, func(i, j int) bool { return offsets[files[i]] < offsets[files[j]] })
	pos := 0
	for _, f := range files {
		if pos+30 > len(data) || !bytes.Equal(data[pos:pos+4], []byte("PK\x03\x04")) {
			return false, formatZip + ": bytes outside its catalogued entries"
		}
		nameLen := int(binary.LittleEndian.Uint16(data[pos+26 : pos+28]))
		extraLen := int(binary.LittleEndian.Uint16(data[pos+28 : pos+30]))
		if pos+30+nameLen+extraLen != int(offsets[f]) {
			return false, formatZip + ": bytes outside its catalogued entries"
		}
		// The LOCAL extra field, which need not match the one the directory
		// carries, is a region of the entry's choosing sitting inside the
		// tiling. It is covered here because this is where its extent is known.
		if ok, why := s.coverStructuralField(data[pos+30:pos+30+nameLen], entryLabel(label, f.Name)+"!name"); !ok {
			return false, why
		}
		if ok, why := s.coverStructuralField(data[pos+30+nameLen:int(offsets[f])], entryLabel(label, f.Name)+"!extra"); !ok {
			return false, why
		}
		// The size comes from the directory, which is attacker-supplied: check
		// it against the file BEFORE the addition, so a size saturated through
		// a zip64 extra field cannot wrap int64 into a negative offset and
		// index the slice out of bounds. Reaching that needs a size the entry
		// read itself would have refused first (it runs off the end of the
		// file and fails), so this is a precondition on the arithmetic rather
		// than a reachable path — which is why no detector covers it.
		if f.CompressedSize64 > uint64(len(data)) {
			return false, formatZip + ": an entry runs past the end of the file"
		}
		end := offsets[f] + int64(f.CompressedSize64)
		if end > int64(len(data)) || end < 0 {
			return false, formatZip + ": an entry runs past the end of the file"
		}
		pos = int(end)
		if f.Flags&0x8 != 0 { // a streamed entry: its sizes follow the data
			n := zipDescriptorLen(data, pos, f.CompressedSize64)
			if n < 0 {
				return false, formatZip + ": an entry's trailing size record could not be read"
			}
			pos += n
		}
	}
	if pos != cdOffset {
		return false, formatZip + ": bytes between its entries and its directory"
	}
	// The directory must reach the end record, and the end record must reach
	// the end of the file. Without both, the region between them is free space
	// an archive can fill with anything: the reader looks for its end record
	// from the back and never minds what sits in front of it.
	//
	// The directory is WALKED rather than measured. Its declared size is a
	// field the archive writes, so checking that field against the end record's
	// position is an identity between two numbers the same hand chose — inflate
	// the size by exactly what you splice in and it balances again. Walking its
	// records is a statement about bytes.
	if cdOffset+end.directorySize != end.offset {
		return false, formatZip + ": its directory does not reach its end record"
	}
	p := cdOffset
	for i := 0; i < end.records; i++ {
		if p+46 > end.offset || !bytes.Equal(data[p:p+4], []byte("PK\x01\x02")) {
			return false, formatZip + ": its directory does not tile the region it declares"
		}
		nameLen := int(binary.LittleEndian.Uint16(data[p+28 : p+30]))
		extraLen := int(binary.LittleEndian.Uint16(data[p+30 : p+32]))
		commentLen := int(binary.LittleEndian.Uint16(data[p+32 : p+34]))
		p += 46 + nameLen + extraLen + commentLen
		if p > end.offset {
			return false, formatZip + ": a directory record runs past its end record"
		}
	}
	if p != end.offset {
		return false, formatZip + ": bytes between its directory and its end record"
	}
	// The archive comment is the one region the end record legitimately
	// carries, and it is a run of bytes of the archive's choosing.
	if end.comment > 0 {
		return s.coverStructuralField(data[end.offset+22:], label+"!comment")
	}
	return true, ""
}

// zipEnd is the end-of-archive record the accounting works from.
type zipEnd struct {
	offset          int // where the record starts
	directoryOffset int
	directorySize   int
	records         int
	comment         int
}

// zipEndRecord finds the end-of-archive record and reads the directory's
// extent from it. The record must CLOSE the file — its comment length has to
// reach the last byte — so bytes appended after it are refused rather than
// ignored the way a zip reader ignores them. A zip64 layout (a saturated size,
// offset or record count, any of which sends a reader to a record this does
// not model) is reported as unaccounted rather than guessed at.
func zipEndRecord(data []byte) (zipEnd, bool) {
	const endLen = 22
	low := len(data) - endLen - 65535
	if low < 0 {
		low = 0
	}
	for i := len(data) - endLen; i >= low; i-- {
		if !bytes.Equal(data[i:i+4], []byte("PK\x05\x06")) {
			continue
		}
		comment := int(binary.LittleEndian.Uint16(data[i+20 : i+22]))
		if i+endLen+comment != len(data) {
			continue
		}
		records := binary.LittleEndian.Uint16(data[i+10 : i+12])
		size := binary.LittleEndian.Uint32(data[i+12 : i+16])
		offset := binary.LittleEndian.Uint32(data[i+16 : i+20])
		if records == 0xffff || size == 0xffffffff || offset == 0xffffffff {
			return zipEnd{}, false
		}
		if int64(offset)+int64(size) > int64(i) {
			return zipEnd{}, false
		}
		return zipEnd{offset: i, directoryOffset: int(offset), directorySize: int(size),
			records: int(records), comment: comment}, true
	}
	return zipEnd{}, false
}

// zipDescriptorLen measures the data descriptor a streamed entry writes after
// its data — optionally signed, with 32-bit or 64-bit sizes — by requiring its
// compressed-size field to match what the directory recorded. A descriptor it
// cannot match is a refusal, not a guess.
func zipDescriptorLen(data []byte, at int, compressed uint64) int {
	off := at
	if at+4 <= len(data) && bytes.Equal(data[at:at+4], []byte("PK\x07\x08")) {
		off += 4
	}
	if off+12 <= len(data) && uint64(binary.LittleEndian.Uint32(data[off+4:off+8])) == compressed {
		return off - at + 12
	}
	if off+20 <= len(data) && binary.LittleEndian.Uint64(data[off+4:off+12]) == compressed {
		return off - at + 20
	}
	return -1
}

// decodeTar walks a tar archive's entries and then accounts for what sits
// after it.
//
// Every entry's body is READ, whatever its typeflag says: a filter that keeps
// only TypeReg skips typeflag '7' (contiguous), which carries a body the tar
// reader will hand over, and any typeflag a future tar invents. A header-only
// entry — a directory, a symlink — yields nothing, so reading it costs nothing.
//
// A tar ends at its end-of-archive marker, exactly as a PNG ends at IEND, and
// the reader stops there without minding what follows. The counting source is
// how the walk learns where that was: every decoder here reads its source
// exactly, so the count after the walk IS the archive's extent.
func (s *Scanner) decodeTar(data []byte, secrets []Pattern, label string, b *decodeBudget, depth int, out *[]Finding) (bool, string) {
	counted := &countingReader{r: bytes.NewReader(data)}
	tr := tar.NewReader(counted)
	verified, why := true, ""
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return false, formatTar + ": archive would not read"
		}
		if !b.entry() {
			return false, formatTar + ": more than " + strconv.Itoa(maxDecodeEntries) + " entries"
		}
		name := entryLabel(label, h.Name)
		s.scanEntryHeader(secrets, name, out, h.Name, h.Linkname, h.Uname, h.Gname)
		// The header's own fields are the tar counterpart of a zip's names and
		// comments. PAX lifts the classic length limits and forbids NUL only in
		// the four the format defines, so an extended record's key and value
		// are arbitrary bytes of the archive's choosing.
		if ok, why := s.coverTarHeader(h, name); !ok {
			return false, why
		}
		body, err := b.read(tr)
		if err != nil {
			return false, streamWhy(formatTar, err)
		}
		// An entry's body is padded to the 512-byte block, and the reader
		// discards that padding while the counting walk includes it in the
		// archive's extent: up to 511 bytes per entry inside no region at all.
		// Every real writer zero-fills, so requiring it costs nothing.
		// Measured from what the reader CONSUMED, not from the header's size:
		// a sparse entry expands to more bytes than it occupies, and measuring
		// from the expanded length would look at the wrong window.
		if pad := tarPadding(data, counted.n); !allZero(pad) {
			return false, name + ": its padding to the block boundary is not zero"
		}
		if len(body) == 0 {
			continue // a header-only entry: nothing to cover
		}
		if ok, w := s.cover(body, secrets, name, b, depth+1, out); !ok && verified {
			verified, why = false, w
		}
	}
	if !verified {
		return verified, why
	}
	tail := data[min(counted.n, len(data)):]
	for len(tail) > 0 && tail[0] == 0 {
		tail = tail[1:] // the archive's own zero padding
	}
	if len(tail) > 0 {
		return s.cover(tail, secrets, label+"!trailer", b, depth+1, out)
	}
	return true, ""
}

// coverTarHeader judges the fields a tar header carries: the four the format
// names, and every PAX extended record and extended attribute, key and value
// alike.
func (s *Scanner) coverTarHeader(h *tar.Header, label string) (bool, string) {
	fields := []struct {
		name  string
		bytes []byte
	}{
		{"name", []byte(h.Name)}, {"linkname", []byte(h.Linkname)},
		{"uname", []byte(h.Uname)}, {"gname", []byte(h.Gname)},
	}
	for key, value := range h.PAXRecords {
		fields = append(fields, struct {
			name  string
			bytes []byte
		}{"pax", []byte(key + "\x00" + value)})
	}
	for key, value := range h.Xattrs { //nolint:staticcheck // PAXRecords does not carry every xattr on every archive
		fields = append(fields, struct {
			name  string
			bytes []byte
		}{"xattr", []byte(key + "\x00" + value)})
	}
	for _, f := range fields {
		if ok, why := s.coverStructuralField(f.bytes, label+"!"+f.name); !ok {
			return false, why
		}
	}
	return true, ""
}

// tarPadding returns the bytes between where an entry's data physically ends
// and its next 512-byte block boundary, as they sit in the archive.
func tarPadding(data []byte, end int) []byte {
	if end < 0 || end > len(data) {
		return nil
	}
	pad := (512 - end%512) % 512
	if end+pad > len(data) {
		pad = len(data) - end
	}
	return data[end : end+pad]
}

// allZero reports whether every byte is zero.
func allZero(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}

// countingReader records how much of its source a decoder actually consumed.
// It deliberately implements Read alone: a tar reader handed an io.Seeker
// skips over entry data instead of reading it, and a skipped byte is one this
// count would miss.
type countingReader struct {
	r io.Reader
	n int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += n
	return n, err
}

// scanEntryHeader scans an entry's own header fields — its name, and whatever
// else the format records about it. An entry's NAME is content: a tarball built
// from a home directory carries the caller's home path in every header, and a
// path is a hard-fail wherever it sits. In a loose tar or zip the headers are
// plaintext in the raw bytes the caller already scanned; inside a COMPRESSED
// archive nothing else reads them, because the walk scans each entry's body
// rather than the inflated blob those bodies came from (which would double
// every finding).
func (s *Scanner) scanEntryHeader(secrets []Pattern, label string, out *[]Finding, fields ...string) {
	joined := strings.Join(fields, "\n")
	if strings.TrimSpace(joined) == "" {
		return
	}
	*out = append(*out, s.scanBytes([]byte(joined), secrets, label)...)
}

// pngPlainChunks are the PNG chunk types whose payload is NOT compressed, so
// the caller's raw byte scan already covers them in full. The list is an
// allow-list for the same reason the byte branch's is: a chunk type nobody
// classified leaves the file unverified rather than passing by omission — an
// APNG's fdAT frames and any private chunk land there.
var pngPlainChunks = toSet([]string{
	"IHDR", "PLTE", "IEND", "tRNS", "cHRM", "gAMA", "sBIT", "sRGB", "bKGD",
	"hIST", "pHYs", "sPLT", "tIME", "eXIf", "oFFs", "pCAL", "sCAL", "sTER",
	"cICP", "mDCv", "cLLi", "tEXt",
})

// decodePNG walks a PNG's chunks and inflates the compressed ones — the text
// chunks first (zTXt, a compressed iTXt, iCCP) and IDAT last, so a big image's
// pixel data cannot spend the budget the metadata needs. The record's own
// example is the zTXt case: a valid 1x1 PNG can carry a token in a DEFLATEd
// zTXt chunk with not even its prefix visible in the raw bytes.
//
// A PNG must end at its IEND chunk: bytes appended after it are not part of
// the image, and appending a zip to a PNG is the oldest polyglot there is, so
// the trailer is covered like any other decoded region rather than waved
// through with the image.
func (s *Scanner) decodePNG(data []byte, secrets []Pattern, label string, b *decodeBudget, depth int, out *[]Finding) (bool, string) {
	pos := 8 // the signature
	var idat []byte
	sawIEND := false
	for pos+8 <= len(data) {
		length := int(binary.BigEndian.Uint32(data[pos : pos+4]))
		typ := string(data[pos+4 : pos+8])
		// Bounds-checked by SUBTRACTION rather than pos+12+length, which can
		// overflow int on a 32-bit build and turn an attacker-chosen length
		// into a slice past the end of the buffer.
		if length < 0 || length > len(data)-pos-12 {
			return false, formatPNG + ": malformed chunk"
		}
		payload := data[pos+8 : pos+8+length]
		pos += 12 + length
		switch {
		case typ == "IDAT":
			idat = append(idat, payload...)
			continue
		case typ == "zTXt":
			if !b.entry() {
				return false, formatPNG + ": more than " + strconv.Itoa(maxDecodeEntries) + " compressed chunks"
			}
			body, err := b.inflate(afterNulThen(payload, 1, 1))
			if err != nil {
				return false, chunkWhy("zTXt", err)
			}
			if ok, why := s.cover(body, secrets, label+"!zTXt", b, depth+1, out); !ok {
				return false, why
			}
		case typ == "iCCP":
			if !b.entry() {
				return false, formatPNG + ": more than " + strconv.Itoa(maxDecodeEntries) + " compressed chunks"
			}
			body, err := b.inflate(afterNulThen(payload, 1, 1))
			if err != nil {
				return false, chunkWhy("iCCP", err)
			}
			// An ICC profile is binary BY DEFINITION, so asking cover to vouch
			// for it would send every colour-managed PNG to ContentUnverified,
			// and a tier people learn to override has stopped working. It takes
			// IDAT's treatment: inflated, byte-scanned, not asked to be a
			// region — with IDAT's residual, a member buried inside it.
			*out = append(*out, s.scanBytes(body, secrets, label+"!iCCP")...)
		case typ == "iTXt":
			// keyword\0 flag method lang\0 translated\0 text; only a flag of 1
			// compresses the text, and an uncompressed one is already in the
			// raw bytes the caller scanned.
			rest := afterNulThen(payload, 1, 0)
			if len(rest) < 2 || rest[0] != 1 {
				// Uncompressed: the raw scan read it, but it is a field like
				// tEXt beside it and can hold a member just the same.
				if ok, why := s.coverStructuralField(payload, label+"!iTXt"); !ok {
					return false, why
				}
				continue
			}
			if !b.entry() {
				return false, formatPNG + ": more than " + strconv.Itoa(maxDecodeEntries) + " compressed chunks"
			}
			body, err := b.inflate(afterNulThen(rest[2:], 2, 0))
			if err != nil {
				return false, chunkWhy("iTXt", err)
			}
			if ok, why := s.cover(body, secrets, label+"!iTXt", b, depth+1, out); !ok {
				return false, why
			}
		default:
			if _, ok := pngPlainChunks[typ]; !ok {
				return false, formatPNG + ": chunk " + sanitiseLabel(typ) + " is not a chunk abcd decodes"
			}
			// "Plaintext by spec" is a claim about well-formed files, and a
			// payload is whatever the length field says it is: a tEXt chunk
			// holds a compressed member as readily as any other run of bytes.
			// A short fixed-shape chunk (a header, a gamma, a timestamp) has
			// no room for one and is covered by the raw scan; a long one is
			// covered as a region.
			if ok, why := s.coverStructuralField(payload, label+"!"+sanitiseLabel(typ)); !ok {
				return false, why
			}
		}
		if typ == "IEND" {
			sawIEND = true
			break
		}
	}
	if !sawIEND {
		return false, formatPNG + ": no IEND chunk"
	}
	if len(idat) == 0 {
		return false, formatPNG + ": no image data"
	}
	body, err := b.inflate(idat)
	if err != nil {
		return false, chunkWhy("IDAT", err)
	}
	// IDAT inflates to filtered pixel data: bytes, not a region with a format.
	// They are scanned, not covered — asking cover to vouch for them would
	// refuse every real image, since pixel data is neither text nor a container
	// at its head. The residual is a member buried inside pixel data, which is
	// the same residual a member buried mid-region has anywhere else.
	*out = append(*out, s.scanBytes(body, secrets, label+"!IDAT")...)
	if pos < len(data) {
		return s.cover(data[pos:], secrets, label+"!trailer", b, depth+1, out)
	}
	return true, ""
}

// afterNulThen skips n NUL-terminated fields and then skip further bytes,
// returning what remains — the shape every compressed PNG text chunk has.
func afterNulThen(payload []byte, nuls, skip int) []byte {
	for i := 0; i < nuls; i++ {
		j := bytes.IndexByte(payload, 0)
		if j < 0 {
			return nil
		}
		payload = payload[j+1:]
	}
	if len(payload) < skip {
		return nil
	}
	return payload[skip:]
}

// streamWhy names why a budgeted read of a stream ended badly, keeping the
// bomb case distinct from a corrupt one.
func streamWhy(format string, err error) string {
	if errors.Is(err, errDecodeBudget) {
		return format + ": over the " + strconv.Itoa(maxDecodedBytes>>20) + " MiB decode budget"
	}
	return format + ": stream would not decode"
}

// chunkWhy is streamWhy for a PNG chunk.
func chunkWhy(chunk string, err error) string {
	if errors.Is(err, errDecodeBudget) {
		return formatPNG + " " + chunk + ": over the " + strconv.Itoa(maxDecodedBytes>>20) + " MiB decode budget"
	}
	return formatPNG + " " + chunk + ": would not inflate"
}

// entryLabel names one entry inside a container, jar-style, so a finding says
// which member of which archive it sits in. The name comes from an archive
// header, which is attacker-controlled, so it is sanitised and bounded here
// rather than trusted into a report.
func entryLabel(container, name string) string {
	return container + "!" + sanitiseLabel(name)
}

// sanitiseLabel strips control bytes from an archive-supplied name and bounds
// its length.
func sanitiseLabel(name string) string {
	const maxLabel = 120
	var sb strings.Builder
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			sb.WriteByte('?')
			continue
		}
		sb.WriteRune(r)
		if sb.Len() >= maxLabel {
			sb.WriteString("...")
			break
		}
	}
	return sb.String()
}
