package scanner

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// iss-2608291832160371 (the GHSA-9wv7 residual): a payload file in a
// compressed or container format was byte-scanned and labelled
// content_unverified by its NAME, so a token inside a gzip member, a zip entry
// or a PNG zTXt chunk shipped invisible to the gate and a zip renamed .png was
// labelled a PNG. The ruling: decode the formats abcd knows, within the
// existing scan cap, and scan their entries; anything still unreadable stays
// flagged as it is today.

// secretBody wraps a token in enough compressible filler that DEFLATE encodes
// it rather than emitting a stored block: on a short input Go's deflate stores
// the bytes verbatim, and a control asserting the token is invisible in the raw
// container would pass for the wrong reason.
func secretBody(token string) []byte {
	return []byte("# notes\n" + strings.Repeat("filler line for the deflate window\n", 20) + "TOKEN=" + token + "\n")
}

// gzipOf returns body compressed as a single gzip member.
func gzipOf(t *testing.T, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// zipOf returns a zip archive holding one deflated entry.
func zipOf(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// tarOf returns an uncompressed tar holding one regular-file entry.
func tarOf(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// pngChunk frames one PNG chunk (length, type, data, CRC32).
func pngChunk(typ string, data []byte) []byte {
	var out bytes.Buffer
	var l [4]byte
	binary.BigEndian.PutUint32(l[:], uint32(len(data)))
	out.Write(l[:])
	out.WriteString(typ)
	out.Write(data)
	crc := crc32.NewIEEE()
	crc.Write([]byte(typ))
	crc.Write(data)
	var c [4]byte
	binary.BigEndian.PutUint32(c[:], crc.Sum32())
	out.Write(c[:])
	return out.Bytes()
}

// pngWithZTXt returns a valid PNG carrying text in a zTXt chunk — zlib
// DEFLATEd, so not a byte of it is visible to a raw byte scan.
func pngWithZTXt(t *testing.T, text string) []byte {
	t.Helper()
	base := pngBytes(t)
	var z bytes.Buffer
	zw := zlib.NewWriter(&z)
	if _, err := zw.Write([]byte(text)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	payload := append([]byte("Comment\x00\x00"), z.Bytes()...)
	chunk := pngChunk("zTXt", payload)
	// Splice the chunk in ahead of IEND (the last 12 bytes).
	cut := len(base) - 12
	out := append([]byte{}, base[:cut]...)
	out = append(out, chunk...)
	return append(out, base[cut:]...)
}

// TestTokenInsideGzipMemberIsFound: a secret inside a gzip member is invisible
// to the raw byte scan and must be found by decoding the member.
func TestTokenInsideGzipMemberIsFound(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	raw := gzipOf(t, secretBody(token))
	if bytes.Contains(raw, []byte(token)) {
		t.Fatalf("control: the token must not survive verbatim in the gzip member")
	}
	abs := writeFile(t, root, "docs/pack.gz", string(raw))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "docs/pack.gz", abs)
	if res.HardFails == 0 {
		t.Fatalf("a token inside a gzip member must hard-fail: %+v", res)
	}
	if !contains(res.ContentDecoded, "docs/pack.gz") || contains(res.ContentUnverified, "docs/pack.gz") {
		t.Errorf("a decoded gzip is ContentDecoded, not ContentUnverified: %+v", res)
	}
	if got := res.ContentFormat["docs/pack.gz"]; got != "gzip" {
		t.Errorf("format keyed on magic bytes = %q, want gzip", got)
	}
}

// TestTokenInsideZipEntryIsFound: same for a deflated zip entry, and the
// finding names the entry it sits in rather than only the archive.
func TestTokenInsideZipEntryIsFound(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	raw := zipOf(t, "conf/secrets.env", secretBody(token))
	if bytes.Contains(raw, []byte(token)) {
		t.Fatalf("control: the token must not survive verbatim in the zip entry")
	}
	abs := writeFile(t, root, "docs/pack.zip", string(raw))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "docs/pack.zip", abs)
	if res.HardFails == 0 {
		t.Fatalf("a token inside a zip entry must hard-fail: %+v", res)
	}
	if !contains(res.ContentDecoded, "docs/pack.zip") {
		t.Errorf("a decoded zip is ContentDecoded: %+v", res)
	}
	var named bool
	for _, f := range res.Findings {
		if strings.Contains(f.File, "docs/pack.zip") && strings.Contains(f.File, "conf/secrets.env") {
			named = true
		}
	}
	if !named {
		t.Errorf("the finding must name the entry inside the archive: %+v", res.Findings)
	}
}

// TestTokenInsideTarEntryIsFound covers the tar walk on its own (a .tgz is the
// gzip member above with a tar inside, so both layers are exercised).
func TestTokenInsideTarEntryIsFound(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	inner := tarOf(t, "conf/.env", secretBody(token))
	abs := writeFile(t, root, "docs/pack.tgz", string(gzipOf(t, inner)))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "docs/pack.tgz", abs)
	if res.HardFails == 0 {
		t.Fatalf("a token inside a gzipped tar entry must hard-fail: %+v", res)
	}
	if !contains(res.ContentDecoded, "docs/pack.tgz") {
		t.Errorf("a decoded .tgz is ContentDecoded: %+v", res)
	}
}

// TestTokenInsidePNGZTXtIsFound is the record's own example: a valid PNG can
// carry a token in a DEFLATEd zTXt chunk with not even its prefix visible in
// the raw bytes.
func TestTokenInsidePNGZTXtIsFound(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	raw := pngWithZTXt(t, string(secretBody(token)))
	if bytes.Contains(raw, []byte("ghp_")) {
		t.Fatalf("control: not even the token prefix may be visible in the raw PNG")
	}
	abs := writeFile(t, root, "docs/assets/shot.png", string(raw))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "docs/assets/shot.png", abs)
	if res.HardFails == 0 {
		t.Fatalf("a token inside a PNG zTXt chunk must hard-fail: %+v", res)
	}
	if !contains(res.ContentDecoded, "docs/assets/shot.png") {
		t.Errorf("a decoded PNG is ContentDecoded: %+v", res)
	}
}

// TestValidPNGDecodesClean: the anti-vacuity partner of the zTXt test — a
// genuine image decodes, scans clean, and is reported decoded rather than
// unverified.
func TestValidPNGDecodesClean(t *testing.T) {
	root := t.TempDir()
	abs := writeFile(t, root, "logo.png", string(pngBytes(t)))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "logo.png", abs)
	if len(res.Findings) != 0 {
		t.Fatalf("decoding a real image must not trip any rule: %+v", res.Findings)
	}
	if !contains(res.ContentDecoded, "logo.png") {
		t.Errorf("a valid PNG must be reported decoded: %+v", res)
	}
}

// TestArchiveIsClassifiedByMagicNotName: the record's explicit requirement —
// the label keyed on the name, so a zip renamed .png was labelled by its name.
func TestArchiveIsClassifiedByMagicNotName(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	abs := writeFile(t, root, "docs/assets/logo.png", string(zipOf(t, "payload.env", secretBody(token))))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "docs/assets/logo.png", abs)
	if got := res.ContentFormat["docs/assets/logo.png"]; got != "zip" {
		t.Errorf("a zip renamed .png must be classified zip by magic bytes, got %q", got)
	}
	if res.HardFails == 0 {
		t.Fatalf("its entry's token must still be found: %+v", res)
	}
}

// TestDecompressionBombIsRefused: a gzip member that inflates to far more than
// the decode budget is REFUSED — bounded output, bounded time, reported
// unverified with the reason, never silently called clean.
func TestDecompressionBombIsRefused(t *testing.T) {
	root := t.TempDir()
	bomb := gzipOf(t, bytes.Repeat([]byte{'A'}, 64<<20)) // 64 MiB, 16x the decode budget
	if len(bomb) > maxBinaryScanBytes {
		t.Fatalf("the bomb must be under the read cap to reach the decoder: %d", len(bomb))
	}
	abs := writeFile(t, root, "bomb.gz", string(bomb))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "bomb.gz", abs)
	if contains(res.ContentDecoded, "bomb.gz") {
		t.Errorf("a bomb must never be reported decoded: %+v", res)
	}
	if !contains(res.ContentUnverified, "bomb.gz") {
		t.Fatalf("a refused bomb stays ContentUnverified: %+v", res)
	}
	if why := res.ContentUnverifiedWhy["bomb.gz"]; !strings.Contains(why, "budget") {
		t.Errorf("the reason must say the decode budget was exceeded, got %q", why)
	}

	// Bounded output is measured as output, not as time: the same decode on a
	// budget the test holds must pull at most one byte past the budget out of
	// the decompressor. A read that inflated the whole member and refused it
	// afterwards would say "budget" just the same, so the reason alone cannot
	// tell the bound from its absence; a wall-clock ceiling could, but it read
	// the machine's load as well (iss-2609232048579579).
	b := &decodeBudget{bytesLeft: maxDecodedBytes, entriesLeft: maxDecodeEntries}
	var out []Finding
	if ok, why := sc.decodeInto(bomb, secretPatterns(sc.patterns), "bomb.gz", b, 1, &out); ok || !strings.Contains(why, "budget") {
		t.Fatalf("the decode of a bomb returned (%t, %q), want the budget's refusal", ok, why)
	}
	if b.drained > maxDecodedBytes+1 {
		t.Fatalf("a bomb of 64 MiB drained %d bytes from its decompressor, want at most %d", b.drained, maxDecodedBytes+1)
	}
}

// TestNestingBombIsRefused: nesting depth is bounded too, so a container
// wrapped past the depth limit is refused rather than recursed.
func TestNestingBombIsRefused(t *testing.T) {
	root := t.TempDir()
	body := []byte("harmless\n")
	for i := 0; i < maxDecodeDepth+2; i++ {
		body = gzipOf(t, body)
	}
	abs := writeFile(t, root, "nest.gz", string(body))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "nest.gz", abs)
	if contains(res.ContentDecoded, "nest.gz") {
		t.Errorf("nesting past the depth bound must not be called decoded: %+v", res)
	}
	if why := res.ContentUnverifiedWhy["nest.gz"]; !strings.Contains(why, "depth") {
		t.Errorf("the reason must name the nesting bound, got %q", why)
	}
}

// TestUndecodableFormatStaysContentUnverified: a format abcd has no decoder
// for is reported exactly as it is today — flagged, with the reason naming the
// format, never folded into the decoded (green) tier.
func TestUndecodableFormatStaysContentUnverified(t *testing.T) {
	root := t.TempDir()
	cases := []struct{ name, body, format string }{
		{"photo.jpg", "\xff\xd8\xff\xe0\x00\x10JFIF entropy", "jpeg"},
		{"paper.pdf", "%PDF-1.4\nstream opaque", "pdf"},
		{"anim.gif", "GIF89a\x01\x00\x01\x00 lzw", "gif"},
		{"arch.7z", "7z\xbc\xaf\x27\x1c opaque", "7z"},
		{"arch.xz", "\xfd7zXZ\x00 opaque", "xz"},
		{"data.db", "SQLite format 3\x00opaque", "sqlite"},
		{"opaque.exe", "\x00\x01\x02opaque bytes", "unrecognised"},
	}
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		abs := writeFile(t, root, c.name, c.body)
		res := scanOne(t, sc, c.name, abs)
		if contains(res.ContentDecoded, c.name) {
			t.Errorf("%s has no decoder and must not be reported decoded: %+v", c.name, res)
		}
		if !contains(res.ContentUnverified, c.name) {
			t.Errorf("%s must stay ContentUnverified: %+v", c.name, res)
		}
		if why := res.ContentUnverifiedWhy[c.name]; !strings.Contains(why, c.format) {
			t.Errorf("%s: reason %q must name the detected format %q", c.name, why, c.format)
		}
	}
}

// TestOpaqueEntryKeepsTheArchiveUnverified: an archive whose entry is itself
// an undecodable binary is decoded as far as it goes — its plaintext entries
// are scanned — but the archive is NOT promoted to decoded, because part of
// its content was never read.
func TestOpaqueEntryKeepsTheArchiveUnverified(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range []struct{ name, body string }{
		{"notes.txt", string(secretBody(token))},
		{"photo.jpg", "\xff\xd8\xff\xe0\x00\x10JFIF entropy"},
	} {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: e.name, Method: zip.Deflate})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	abs := writeFile(t, root, "mixed.zip", buf.String())
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "mixed.zip", abs)
	if res.HardFails == 0 {
		t.Fatalf("the readable entry's token must still be found: %+v", res)
	}
	if contains(res.ContentDecoded, "mixed.zip") {
		t.Errorf("an archive holding an opaque entry is not content-verified: %+v", res)
	}
	if why := res.ContentUnverifiedWhy["mixed.zip"]; !strings.Contains(why, "photo.jpg") {
		t.Errorf("the reason must name the entry that could not be read, got %q", why)
	}
}

// TestCorruptContainerIsUnverifiedNotClean: a file whose magic says gzip but
// whose stream will not inflate is a loud unverified, never a silent pass.
func TestCorruptContainerIsUnverifiedNotClean(t *testing.T) {
	root := t.TempDir()
	abs := writeFile(t, root, "broken.tgz", "\x1f\x8b\x08\x00opaque deflate bytes\n")
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "broken.tgz", abs)
	if contains(res.ContentDecoded, "broken.tgz") || !contains(res.ContentUnverified, "broken.tgz") {
		t.Errorf("a corrupt gzip stays unverified: %+v", res)
	}
	if why := res.ContentUnverifiedWhy["broken.tgz"]; !strings.Contains(why, "gzip") {
		t.Errorf("the reason must name the format that failed to decode, got %q", why)
	}
}

// TestDecodedEntriesRespectTheReadCap: the decoder never widens the existing
// scan cap — an oversized container is still an Unscanned refusal, read at all.
func TestDecodedEntriesRespectTheReadCap(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(root, "big.zip")
	f, err := os.Create(abs)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxBinaryScanBytes + 1); err != nil {
		t.Fatal(err)
	}
	f.Close()
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "big.zip", abs)
	if !contains(res.Unscanned, "big.zip") {
		t.Fatalf("the read cap still governs: %+v", res)
	}
	if contains(res.ContentDecoded, "big.zip") {
		t.Errorf("an unread file must not claim to be decoded: %+v", res)
	}
}

// TestPNGWithAppendedArchiveIsCovered: a PNG ends at its IEND chunk, and
// appending an archive after it is the oldest polyglot there is. The trailer is
// covered like any other decoded region — the appended zip's entry is scanned
// and its token found — so "the image decoded" cannot vouch for bytes that are
// not part of the image.
func TestPNGWithAppendedArchiveIsCovered(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	body := append(pngBytes(t), zipOf(t, "stow.env", secretBody(token))...)
	abs := writeFile(t, root, "poly.png", string(body))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "poly.png", abs)
	if res.HardFails == 0 {
		t.Fatalf("a token in an archive appended to a PNG must hard-fail: %+v", res)
	}
	if !contains(res.ContentDecoded, "poly.png") {
		t.Errorf("image plus a decodable trailer is fully decoded: %+v", res)
	}
}

// TestPNGWithOpaqueTrailerIsUnverified: the other half of the same rule — an
// appended blob abcd cannot read leaves the file unverified, naming the
// trailer, rather than passing on the strength of the image.
func TestPNGWithOpaqueTrailerIsUnverified(t *testing.T) {
	root := t.TempDir()
	body := append(pngBytes(t), []byte("\xff\xd8\xff\xe0\x00\x10JFIF entropy")...)
	abs := writeFile(t, root, "poly2.png", string(body))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "poly2.png", abs)
	if contains(res.ContentDecoded, "poly2.png") {
		t.Errorf("an opaque trailer must keep the file unverified: %+v", res)
	}
	if why := res.ContentUnverifiedWhy["poly2.png"]; !strings.Contains(why, "trailer") {
		t.Errorf("the reason must name the trailer, got %q", why)
	}
}

// TestEntryCountBombIsRefused: an archive can be a bomb by COUNT as well as by
// size — thousands of empty entries cost no output budget at all — so the
// entry bound is its own refusal, with its own reason.
func TestEntryCountBombIsRefused(t *testing.T) {
	root := t.TempDir()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for i := 0; i <= maxDecodeEntries; i++ {
		if _, err := zw.Create("e" + strconv.Itoa(i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	abs := writeFile(t, root, "many.zip", buf.String())
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "many.zip", abs)
	if contains(res.ContentDecoded, "many.zip") {
		t.Errorf("an archive past the entry bound must not be reported decoded: %+v", res)
	}
	if why := res.ContentUnverifiedWhy["many.zip"]; !strings.Contains(why, "entries") {
		t.Errorf("the reason must name the entry bound, got %q", why)
	}
}

// TestEntryNamesInsideACompressedArchiveAreScanned: an entry's NAME is content
// too — a tarball built from a home directory carries the path in every header
// — and the walk scans an entry's body rather than the archive blob it came
// from, so the headers have to be scanned in their own right. In a loose tar
// the raw byte scan sees them; inside a .tgz nothing else does.
func TestEntryNamesInsideACompressedArchiveAreScanned(t *testing.T) {
	root := t.TempDir()
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	sc.identity = synthIdentity()
	inner := tarOf(t, sc.identity.HomePath+"/deck/notes.md", []byte("harmless\n"))
	abs := writeFile(t, root, "home.tgz", string(gzipOf(t, inner)))
	res := scanOne(t, sc, "home.tgz", abs)
	if !hasKind(res.Findings, kindHomeSelf) || res.HardFails == 0 {
		t.Fatalf("the caller's home path in a gzipped tar's entry name must hard-fail: %+v", res)
	}
}

// TestMalformedContainersDoNotPanic: the decoder reads attacker-supplied
// lengths and offsets out of the payload, so every truncation and byte flip of
// a valid container must come back as a verdict — never a panic, and never a
// hang. A malformed file is unverified or decoded; either is a report, and a
// panic in the scanner takes the launch gate down with it.
func TestMalformedContainersDoNotPanic(t *testing.T) {
	root := t.TempDir()
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	originals := map[string][]byte{
		"m.png": pngWithZTXt(t, "harmless text"),
		"m.zip": zipOf(t, "a.txt", secretBody("x")),
		"m.gz":  gzipOf(t, tarOf(t, "a.txt", []byte("harmless\n"))),
		"m.tar": tarOf(t, "a.txt", []byte("harmless\n")),
	}
	for name, orig := range originals {
		for _, cut := range []int{1, 9, 16, 33, len(orig) / 3, len(orig) / 2, len(orig) - 1} {
			if cut <= 0 || cut > len(orig) {
				continue
			}
			body := append([]byte{}, orig[:cut]...)
			abs := writeFile(t, root, name, string(body))
			res := scanOne(t, sc, name, abs)
			if contains(res.Unscanned, name) {
				t.Errorf("%s truncated to %d became an unreadable file: %+v", name, cut, res)
			}
		}
		for _, at := range []int{9, 20, 40} {
			if at >= len(orig) {
				continue
			}
			body := append([]byte{}, orig...)
			body[at] ^= 0xff
			abs := writeFile(t, root, name, string(body))
			_ = scanOne(t, sc, name, abs)
		}
	}
}

// The six shapes below are one defect wearing six hats: the walk decided "read
// whole" from evidence that did not cover the whole, so a token nothing scanned
// shipped in a file reported ContentDecoded. Each plants a token that is NOT in
// the file's raw bytes (asserted as a control), so the raw byte scan provably
// cannot be what catches it; each requires either the finding or a refusal to
// promote — never a silent green.

// mustNotBeVerbatim is the control every false-green detector needs: if the
// token survives in the raw bytes, the raw byte scan catches it and the test
// would pass without the decoder reading anything.
func mustNotBeVerbatim(t *testing.T, raw []byte, token string) {
	t.Helper()
	if bytes.Contains(raw, []byte(token)) {
		t.Fatalf("control: the token must not be visible in the raw container bytes")
	}
}

// assertCaughtOrRefused: the file either hard-fails (the content was read) or
// is kept out of the decoded tier with a reason (the walk admitted it could not
// read it). Reporting it decoded and clean is the false green.
func assertCaughtOrRefused(t *testing.T, res ScanResult, path string) {
	t.Helper()
	if res.HardFails > 0 {
		return
	}
	if contains(res.ContentDecoded, path) {
		t.Fatalf("%s was reported decoded and clean while carrying an unscanned token: %+v", path, res)
	}
	if why := res.ContentUnverifiedWhy[path]; why == "" {
		t.Fatalf("%s was neither read nor refused with a reason: %+v", path, res)
	}
}

// TestNestedPNGPlaintextChunkIsScanned: decodePNG left the uncompressed chunks
// to "the caller's raw byte scan", which is only true at the top level. A
// screenshot carrying a session URL in a plaintext tEXt chunk, zipped into the
// payload, was reported decoded with the URL never scanned.
func TestNestedPNGPlaintextChunkIsScanned(t *testing.T) {
	root := t.TempDir()
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	sc.identity = synthIdentity()
	leak := sc.identity.HomePath + "/secret/deck.key"
	base := pngBytes(t)
	// The tEXt payload is padded with compressible filler so the zip entry is
	// DEFLATEd rather than stored: on incompressible input Go's deflate stores
	// the bytes, and the control below would then pass for the wrong reason.
	chunk := pngChunk("tEXt", []byte("Comment\x00"+strings.Repeat("filler for the deflate window ", 200)+leak))
	cut := len(base) - 12
	shot := append(append(append([]byte{}, base[:cut]...), chunk...), base[cut:]...)

	loose := writeFile(t, root, "shot.png", string(shot))
	if control := scanOne(t, sc, "shot.png", loose); control.HardFails == 0 {
		t.Fatalf("control: the home path must hard-fail in a loose PNG: %+v", control)
	}
	raw := zipOf(t, "shot.png", shot)
	mustNotBeVerbatim(t, raw, leak)
	abs := writeFile(t, root, "pack.zip", string(raw))
	res := scanOne(t, sc, "pack.zip", abs)
	assertCaughtOrRefused(t, res, "pack.zip")
	if !hasKind(res.Findings, kindHomeSelf) {
		t.Errorf("a nested PNG's plaintext chunk must be scanned like a loose one: %+v", res)
	}
}

// TestZipEntryWithDirectoryModeIsRead: Mode() derives the directory bit from
// the external attributes, not from a trailing slash, so an entry can claim to
// be a directory and still carry a body the zip reader will inflate.
func TestZipEntryWithDirectoryModeIsRead(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	h := &zip.FileHeader{Name: "notes.txt", Method: zip.Deflate}
	h.SetMode(fs.ModeDir | 0o755)
	w, err := zw.CreateHeader(h)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(secretBody(token)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	mustNotBeVerbatim(t, buf.Bytes(), token)
	abs := writeFile(t, root, "dirmode.zip", buf.String())
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "dirmode.zip", abs), "dirmode.zip")
}

// TestTarContiguousEntryIsRead: typeflag '7' is not TypeReg but still carries a
// body, so a type filter that keeps only TypeReg skips real content. The body
// is itself a gzip member, because a skipped PLAINTEXT body is still caught by
// the raw scan of the tar it sits in — what a skipped entry really costs is the
// walk INTO it, and only a compressed body shows that.
func TestTarContiguousEntryIsRead(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	body := gzipOf(t, secretBody(token))
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(&tar.Header{Name: "conf/.env", Mode: 0o644, Size: int64(len(body)), Typeflag: '7'}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	raw := gzipOf(t, buf.Bytes())
	mustNotBeVerbatim(t, raw, token)
	abs := writeFile(t, root, "contig.tgz", string(raw))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "contig.tgz", abs), "contig.tgz")
}

// TestZipWithUncataloguedEntryIsNotVerified: the walk reads the central
// directory, so bytes that are physically present but not catalogued — a local
// record prepended ahead of the catalogued one — are never read at all.
func TestZipWithUncataloguedEntryIsNotVerified(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	smuggled := zipOf(t, "smuggled.env", secretBody(token))
	// The local record of the smuggled archive, without its directory.
	end := bytes.Index(smuggled, []byte("PK\x01\x02"))
	if end < 0 {
		t.Fatal("could not locate the central directory of the fixture")
	}
	benign := zipOf(t, "readme.txt", []byte("harmless\n"))
	raw := append(append([]byte{}, smuggled[:end]...), benign...)
	mustNotBeVerbatim(t, raw, token)
	abs := writeFile(t, root, "prepend.zip", string(raw))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "prepend.zip", abs), "prepend.zip")
}

// TestTarWithAppendedDataIsNotVerified: a tar ends at its end-of-archive
// marker, exactly as a PNG ends at IEND. Bytes after it are not part of the
// archive and cannot be vouched for by it.
func TestTarWithAppendedDataIsNotVerified(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	raw := append(tarOf(t, "readme.txt", []byte("harmless\n")), gzipOf(t, secretBody(token))...)
	mustNotBeVerbatim(t, raw, token)
	abs := writeFile(t, root, "appended.tar", string(raw))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "appended.tar", abs), "appended.tar")
}

// TestZlibWithTrailingStreamIsNotVerified: zlib stops at its Adler-32 without
// complaining about what follows, so a trailer needs its own accounting.
func TestZlibWithTrailingStreamIsNotVerified(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	var z bytes.Buffer
	zw := zlib.NewWriter(&z)
	if _, err := zw.Write([]byte("harmless\n")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	raw := append(z.Bytes(), gzipOf(t, secretBody(token))...)
	mustNotBeVerbatim(t, raw, token)
	abs := writeFile(t, root, "trailer.png", string(raw))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "trailer.png", abs), "trailer.png")
}

// TestTextSniffDoesNotVouchForTheWholeRegion: isText sniffs the first 8 KiB, so
// a decoded region that opens with text and carries a compressed member past
// the window was called fully covered on the strength of its first 8 KiB.
func TestTextSniffDoesNotVouchForTheWholeRegion(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	inner := append([]byte(strings.Repeat("harmless prose line\n", 600)), gzipOf(t, secretBody(token))...)
	raw := gzipOf(t, inner)
	mustNotBeVerbatim(t, raw, token)
	abs := writeFile(t, root, "sniff.gz", string(raw))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "sniff.gz", abs), "sniff.gz")
}

// TestPNGChunkCountBombIsBounded: the three declared bounds count inflated
// bytes, nesting and archive entries — none of which a PNG chunk draws on. A
// file of empty compressed text chunks costs zero budget and one inflate plus
// one rule pass each, so the chunk loop needs a bound of its own.
func TestPNGChunkCountBombIsBounded(t *testing.T) {
	root := t.TempDir()
	var empty bytes.Buffer
	zw := zlib.NewWriter(&empty)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	chunk := pngChunk("zTXt", append([]byte("k\x00\x00"), empty.Bytes()...))
	base := pngBytes(t)
	cut := len(base) - 12
	body := append([]byte{}, base[:cut]...)
	for len(body) < maxBinaryScanBytes-len(base)-len(chunk) {
		body = append(body, chunk...)
	}
	body = append(body, base[cut:]...)
	abs := writeFile(t, root, "chunkbomb.png", string(body))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "chunkbomb.png", abs)

	// The property is the refusal, not the clock. A bounded loop stops and says
	// why; an unbounded one never reaches this line at all. Asserting the reason
	// distinguishes a working bound from a slow machine, which a duration cannot
	// (iss-2609091215552981).
	why := res.ContentUnverifiedWhy["chunkbomb.png"]
	if !strings.Contains(why, "compressed chunks") {
		t.Fatalf("a chunk-count bomb was not refused by the chunk bound; reason %q: %+v", why, res)
	}

	// The refusal says the bound fired, not that it fired in time: a walk that
	// inflated every chunk and refused at the end would pass the check above.
	// So the same decode is run again on a budget the test holds, and the work
	// it did is counted, not timed — a count is the same on a loaded machine
	// and under the race detector, where a wall-clock ceiling reddened a run
	// whose bound held (iss-2609232048579579). The bomb carries about 180,000
	// chunks; a working bound inflates at most maxDecodeEntries of them.
	chunks := (len(body) - len(base)) / len(chunk)
	b := &decodeBudget{bytesLeft: maxDecodedBytes, entriesLeft: maxDecodeEntries}
	var out []Finding
	if ok, why := sc.decodeInto(body, secretPatterns(sc.patterns), "chunkbomb.png", b, 1, &out); ok || !strings.Contains(why, "compressed chunks") {
		t.Fatalf("the decode of a chunk-count bomb returned (%t, %q), want the chunk bound's refusal", ok, why)
	}
	if b.reads > maxDecodeEntries {
		t.Fatalf("a chunk-count bomb of %d chunks cost %d decode operations, want at most %d", chunks, b.reads, maxDecodeEntries)
	}
}

// TestCraftedZipDirectorySizesDoNotPanic: the sizes and offsets the byte
// accounting reads come from the archive's own directory, so a saturated or
// absurd value must be a verdict, not an out-of-range index.
func TestCraftedZipDirectorySizesDoNotPanic(t *testing.T) {
	root := t.TempDir()
	orig := zipOf(t, "a.txt", secretBody(fakeToken()))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	cd := bytes.Index(orig, []byte("PK\x01\x02"))
	if cd < 0 {
		t.Fatal("could not locate the central directory of the fixture")
	}
	for _, at := range []int{cd + 20, cd + 24, cd + 42} { // compressed size, uncompressed size, local header offset
		for _, v := range []uint32{0xffffffff, 0x7fffffff, 0x80000000} {
			body := append([]byte{}, orig...)
			if at+4 > len(body) {
				continue
			}
			binary.LittleEndian.PutUint32(body[at:at+4], v)
			abs := writeFile(t, root, "crafted.zip", string(body))
			res := scanOne(t, sc, "crafted.zip", abs)
			if contains(res.ContentDecoded, "crafted.zip") && res.HardFails == 0 {
				t.Errorf("a crafted directory field at %d=%#x was reported decoded and clean: %+v", at, v, res)
			}
		}
	}
}

// The shapes below are the second round of the same defect: a byte the walk
// ACCOUNTS for is not a byte the walk READ. Each plants a token absent from the
// raw bytes, in a region the archive's own structure explains away.

// TestZipEntryNamedAsADirectoryIsRead: the standard library's reader
// short-circuits an entry whose name ends in "/" and whose recorded
// uncompressed size is zero, handing back an empty reader however much
// compressed data the entry actually carries — the mode-bit case wearing a
// different hat.
func TestZipEntryNamedAsADirectoryIsRead(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	// The writer refuses to write a body under a directory name, so the name is
	// patched afterwards — same length, both headers — and the recorded
	// uncompressed size zeroed, which is the pair the reader short-circuits on.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.CreateHeader(&zip.FileHeader{Name: "dataX", Method: zip.Deflate})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(secretBody(token)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	raw := bytes.ReplaceAll(buf.Bytes(), []byte("dataX"), []byte("data/"))
	cd := bytes.Index(raw, []byte("PK\x01\x02"))
	if cd < 0 {
		t.Fatal("could not locate the central directory of the fixture")
	}
	binary.LittleEndian.PutUint32(raw[cd+24:cd+28], 0) // claim it decompresses to nothing
	// Control: the reader really does hand back an empty body for this entry.
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	rc, err := zr.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if len(got) != 0 {
		t.Skip("this Go version does not short-circuit a directory-named entry")
	}
	mustNotBeVerbatim(t, raw, token)
	abs := writeFile(t, root, "dirname.zip", string(raw))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "dirname.zip", abs), "dirname.zip")
}

// TestZipWithDataBeforeItsEndRecordIsNotVerified: the accounting proved the
// entries end where the directory starts and stopped there, so an arbitrary
// region between the directory and the end record was free space.
func TestZipWithDataBeforeItsEndRecordIsNotVerified(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	orig := zipOf(t, "readme.txt", []byte("harmless\n"))
	at := bytes.Index(orig, []byte("PK\x05\x06"))
	if at < 0 {
		t.Fatal("could not locate the end record of the fixture")
	}
	raw := append(append(append([]byte{}, orig[:at]...), gzipOf(t, secretBody(token))...), orig[at:]...)
	if _, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw))); err != nil {
		t.Skip("this Go version rejects the spliced archive outright")
	}
	mustNotBeVerbatim(t, raw, token)
	abs := writeFile(t, root, "splice.zip", string(raw))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "splice.zip", abs), "splice.zip")
}

// TestZipArchiveCommentIsCovered: the end record's comment is any length the
// archive likes, and the accounting counted it as end-record bytes.
func TestZipArchiveCommentIsCovered(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if err := zw.SetComment(string(gzipOf(t, secretBody(token)))); err != nil {
		t.Fatal(err)
	}
	w, err := zw.Create("readme.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("harmless\n")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	mustNotBeVerbatim(t, buf.Bytes(), token)
	abs := writeFile(t, root, "comment.zip", buf.String())
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "comment.zip", abs), "comment.zip")
}

// TestZipEntryExtraFieldIsCovered: an extra field is a length the header
// declares and the tiling steps over; an unknown record id can hold anything.
func TestZipEntryExtraFieldIsCovered(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	member := gzipOf(t, secretBody(token))
	extra := make([]byte, 4, 4+len(member))
	binary.LittleEndian.PutUint16(extra[0:2], 0x9901) // an id abcd does not know
	binary.LittleEndian.PutUint16(extra[2:4], uint16(len(member)))
	extra = append(extra, member...)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.CreateHeader(&zip.FileHeader{Name: "readme.txt", Method: zip.Deflate, Extra: extra})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("harmless\n")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	mustNotBeVerbatim(t, buf.Bytes(), token)
	abs := writeFile(t, root, "extra.zip", buf.String())
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "extra.zip", abs), "extra.zip")
}

// TestPNGPlainChunkPayloadIsCovered: tEXt is text BY SPEC, and nothing enforces
// the spec — a chunk type on the plaintext allow-list can carry a compressed
// member as easily as any other run of bytes.
func TestPNGPlainChunkPayloadIsCovered(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	base := pngBytes(t)
	chunk := pngChunk("tEXt", append([]byte("Comment\x00"), gzipOf(t, secretBody(token))...))
	cut := len(base) - 12
	raw := append(append(append([]byte{}, base[:cut]...), chunk...), base[cut:]...)
	mustNotBeVerbatim(t, raw, token)
	abs := writeFile(t, root, "text.png", string(raw))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "text.png", abs), "text.png")
}

// TestZipEntryCommentIsCovered: a per-entry comment is the same shape as the
// archive comment, one level down.
func TestZipEntryCommentIsCovered(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.CreateHeader(&zip.FileHeader{Name: "readme.txt", Method: zip.Deflate, Comment: string(gzipOf(t, secretBody(token)))})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("harmless\n")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	mustNotBeVerbatim(t, buf.Bytes(), token)
	abs := writeFile(t, root, "entrycomment.zip", buf.String())
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "entrycomment.zip", abs), "entrycomment.zip")
}

// TestZipLocalExtraFieldIsCovered: an entry has TWO extra fields, one in its
// local header and one in the directory, and nothing requires them to match.
// The directory's copy is what the archive reader exposes, so only the tiling
// walk — which is where the local extent is known — can cover the local one.
func TestZipLocalExtraFieldIsCovered(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	member := gzipOf(t, secretBody(token))
	blank := make([]byte, 4+len(member)) // what the directory will carry
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.CreateHeader(&zip.FileHeader{Name: "readme.txt", Method: zip.Deflate, Extra: blank})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("harmless\n")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	raw := buf.Bytes()
	// Patch the LOCAL copy only, in place and to the same length.
	nameLen := int(binary.LittleEndian.Uint16(raw[26:28]))
	at := 30 + nameLen
	binary.LittleEndian.PutUint16(raw[at:at+2], 0x9901)
	binary.LittleEndian.PutUint16(raw[at+2:at+4], uint16(len(member)))
	copy(raw[at+4:at+4+len(member)], member)
	mustNotBeVerbatim(t, raw, token)
	abs := writeFile(t, root, "localextra.zip", string(raw))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "localextra.zip", abs), "localextra.zip")
}

// zlibOf returns body as a zlib stream — the signature the search list missed.
func zlibOf(t *testing.T, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestZipEntryWithDataAfterItsStreamIsCovered: an entry's catalogued extent can
// be longer than the DEFLATE stream inside it. The stream ends at its final
// block and the reader stops there, while the accounting steps over the whole
// declared size — the trailer hole one level down from decodeStream.
func TestZipEntryWithDataAfterItsStreamIsCovered(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	member := gzipOf(t, secretBody(token))
	raw := zipOf(t, "a.txt", []byte("harmless\n"))
	cd := bytes.Index(raw, []byte("PK\x01\x02"))
	if cd < 0 {
		t.Fatal("could not locate the central directory of the fixture")
	}
	nameLen := int(binary.LittleEndian.Uint16(raw[26:28]))
	extraLen := int(binary.LittleEndian.Uint16(raw[28:30]))
	dataAt := 30 + nameLen + extraLen
	csize := int(binary.LittleEndian.Uint32(raw[cd+20 : cd+24]))
	// Splice the member in after the stream, inside the entry's declared extent.
	body := append([]byte{}, raw[:dataAt+csize]...)
	body = append(body, member...)
	body = append(body, raw[dataAt+csize:]...)
	grown := uint32(csize + len(member))
	binary.LittleEndian.PutUint32(body[cd+len(member)+20:cd+len(member)+24], grown) // directory
	if d := bytes.Index(body, []byte("PK\x07\x08")); d >= 0 {
		binary.LittleEndian.PutUint32(body[d+8:d+12], grown) // data descriptor
	}
	if e := bytes.LastIndex(body, []byte("PK\x05\x06")); e >= 0 {
		off := binary.LittleEndian.Uint32(body[e+16 : e+20])
		binary.LittleEndian.PutUint32(body[e+16:e+20], off+uint32(len(member)))
	}
	if _, err := zip.NewReader(bytes.NewReader(body), int64(len(body))); err != nil {
		t.Skip("this Go version rejects the crafted entry outright")
	}
	mustNotBeVerbatim(t, body, token)
	abs := writeFile(t, root, "fat.zip", string(body))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "fat.zip", abs), "fat.zip")
}

// TestZipWithAnInflatedDirectorySizeIsNotVerified: the directory's size is a
// number the archive writes, so checking it against the end record's position
// is an identity between two attacker-supplied fields. The directory has to be
// walked, not measured.
func TestZipWithAnInflatedDirectorySizeIsNotVerified(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	member := gzipOf(t, secretBody(token))
	orig := zipOf(t, "readme.txt", []byte("harmless\n"))
	at := bytes.LastIndex(orig, []byte("PK\x05\x06"))
	if at < 0 {
		t.Fatal("could not locate the end record of the fixture")
	}
	raw := append(append(append([]byte{}, orig[:at]...), member...), orig[at:]...)
	e := at + len(member)
	size := binary.LittleEndian.Uint32(raw[e+12 : e+16])
	binary.LittleEndian.PutUint32(raw[e+12:e+16], size+uint32(len(member))) // swallow the splice
	if _, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw))); err != nil {
		t.Skip("this Go version rejects the crafted archive outright")
	}
	mustNotBeVerbatim(t, raw, token)
	abs := writeFile(t, root, "fatdir.zip", string(raw))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "fatdir.zip", abs), "fatdir.zip")
}

// TestZlibMemberInAStructuralFieldIsCovered: the structural-field rule turns on
// a search for container signatures, so a format the search list forgets is a
// hole in every field it guards — and zlib is a format the decoder DOES open.
func TestZlibMemberInAStructuralFieldIsCovered(t *testing.T) {
	root := t.TempDir()
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	token := fakeToken()
	member := zlibOf(t, secretBody(token))

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if err := zw.SetComment(string(member)); err != nil {
		t.Fatal(err)
	}
	w, err := zw.Create("readme.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("harmless\n")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	mustNotBeVerbatim(t, buf.Bytes(), token)
	abs := writeFile(t, root, "zcomment.zip", buf.String())
	assertCaughtOrRefused(t, scanOne(t, sc, "zcomment.zip", abs), "zcomment.zip")

	base := pngBytes(t)
	chunk := pngChunk("tEXt", append([]byte("Comment\x00"), member...))
	cut := len(base) - 12
	png := append(append(append([]byte{}, base[:cut]...), chunk...), base[cut:]...)
	mustNotBeVerbatim(t, png, token)
	abs = writeFile(t, root, "ztext.png", string(png))
	assertCaughtOrRefused(t, scanOne(t, sc, "ztext.png", abs), "ztext.png")
}

// TestEveryDetectedFormatHasASignature is the totality guard the search list
// needs, and the reason the zlib hole existed: the list was maintained by hand
// beside detectFormat, and the two drifted. Every format detectFormat can name
// must be findable inside a region, or a member in it hides in any structural
// field.
func TestEveryDetectedFormatHasASignature(t *testing.T) {
	samples := map[string][]byte{
		formatZip:   []byte("PK\x03\x04rest of an archive"),
		formatGzip:  []byte("\x1f\x8b\x08\x00rest of a member"),
		formatBzip2: []byte("BZh9rest of a stream"),
		formatPNG:   []byte("\x89PNG\r\n\x1a\nrest of an image"),
		formatZlib:  []byte("\x78\x9crest of a stream"),
		formatTar:   append(append(make([]byte, 257), []byte("ustar")...), make([]byte, 10)...),
		"jpeg":      []byte("\xff\xd8\xff\xe0rest"),
		"pdf":       []byte("%PDF-1.4 rest"),
		"gif":       []byte("GIF89a rest"),
		"webp":      []byte("RIFF\x00\x00\x00\x00WEBPrest"),
		"wav":       []byte("RIFF\x00\x00\x00\x00WAVErest"),
		"7z":        []byte("7z\xbc\xaf\x27\x1crest"),
		"xz":        []byte("\xfd7zXZ\x00rest"),
		"zstd":      []byte("\x28\xb5\x2f\xfdrest"),
		"mp4":       []byte("\x00\x00\x00\x18ftypisom rest"),
		"mp3":       []byte("ID3\x04\x00rest"),
		"matroska":  []byte("\x1a\x45\xdf\xa3rest"),
		"sqlite":    []byte("SQLite format 3\x00rest"),
	}
	for _, name := range knownFormats() {
		sample, ok := samples[name]
		if !ok {
			t.Errorf("format %s has no sample here; add one so its signature is checked", name)
			continue
		}
		if got := detectFormat(sample); got != name {
			t.Errorf("sample for %s is detected as %s", name, got)
		}
		buried := append(append([]byte("some preceding bytes\n"), sample...), []byte("\ntrailing")...)
		if !hidesAContainer(buried) {
			t.Errorf("a %s member buried in a structural field is not found by the signature search", name)
		}
	}
}

// TestUndecodableMemberInAStructuralFieldRefuses: the other half of the
// structural-field rule. A member abcd cannot open — a JPEG thumbnail in an
// eXIf chunk is the ordinary camera-export shape — leaves the file unverified
// naming the field, because nothing read it. It is the same verdict the file
// would get if that JPEG were the payload itself.
func TestUndecodableMemberInAStructuralFieldRefuses(t *testing.T) {
	root := t.TempDir()
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	thumb := []byte("\xff\xd8\xff\xe0\x00\x10JFIF entropy that abcd cannot read")
	base := pngBytes(t)
	chunk := pngChunk("eXIf", thumb)
	cut := len(base) - 12
	raw := append(append(append([]byte{}, base[:cut]...), chunk...), base[cut:]...)
	abs := writeFile(t, root, "thumb.png", string(raw))
	res := scanOne(t, sc, "thumb.png", abs)
	if contains(res.ContentDecoded, "thumb.png") {
		t.Fatalf("a member abcd cannot open must keep the file unverified: %+v", res)
	}
	if why := res.ContentUnverifiedWhy["thumb.png"]; !strings.Contains(why, "jpeg") || !strings.Contains(why, "eXIf") {
		t.Errorf("the reason must name the field and the format, got %q", why)
	}
}

// TestStructuralFieldMemberThatWillNotDecodeStillRefuses: the field rule must
// be fail-closed BY CONSTRUCTION. A member that does not open — a gzip with one
// byte appended, or one the decode budget refuses — is not evidence that the
// field was harmless; it is the one case where nothing read the bytes at all.
func TestStructuralFieldMemberThatWillNotDecodeStillRefuses(t *testing.T) {
	root := t.TempDir()
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string][]byte{
		"broken.zip":     append(gzipOf(t, secretBody(fakeToken())), 'x'), // will not open
		"overbudget.zip": gzipOf(t, bytes.Repeat([]byte{'A'}, maxDecodedBytes+1)),
	}
	for name, member := range cases {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		if err := zw.SetComment(string(member)); err != nil {
			t.Fatal(err)
		}
		w, err := zw.Create("readme.txt")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("harmless\n")); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		abs := writeFile(t, root, name, buf.String())
		res := scanOne(t, sc, name, abs)
		if contains(res.ContentDecoded, name) && res.HardFails == 0 {
			t.Errorf("%s: a member that would not decode was accepted as covered: %+v", name, res)
		}
	}
}

// TestUncompressedITXtIsCovered: the tEXt hole in the branch beside it. An
// uncompressed iTXt payload was skipped before the field rule saw it.
func TestUncompressedITXtIsCovered(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	payload := append([]byte("Comment\x00\x00\x00en\x00c\x00"), gzipOf(t, secretBody(token))...)
	base := pngBytes(t)
	chunk := pngChunk("iTXt", payload)
	cut := len(base) - 12
	raw := append(append(append([]byte{}, base[:cut]...), chunk...), base[cut:]...)
	mustNotBeVerbatim(t, raw, token)
	abs := writeFile(t, root, "itxt.png", string(raw))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "itxt.png", abs), "itxt.png")
}

// TestZipEntryNameIsCovered: a name is up to 65535 bytes of the archive's
// choosing, accounted by the tiling through nameLen and byte-scanned as if it
// were a path. It is a structural field like the comment beside it.
func TestZipEntryNameIsCovered(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.CreateHeader(&zip.FileHeader{Name: "a" + string(gzipOf(t, secretBody(token))), Method: zip.Deflate})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("harmless\n")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len())); err != nil {
		t.Skip("this Go version rejects the crafted name")
	}
	mustNotBeVerbatim(t, buf.Bytes(), token)
	abs := writeFile(t, root, "name.zip", buf.String())
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "name.zip", abs), "name.zip")
}

// TestInflatedChunkOutputIsCovered: what comes out of a compressed PNG chunk is
// a decoded region like any other, so a member one layer inside it must be
// walked rather than only byte-scanned.
func TestInflatedChunkOutputIsCovered(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	var z bytes.Buffer
	zw := zlib.NewWriter(&z)
	if _, err := zw.Write(gzipOf(t, secretBody(token))); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	base := pngBytes(t)
	chunk := pngChunk("zTXt", append([]byte("Comment\x00\x00"), z.Bytes()...))
	cut := len(base) - 12
	raw := append(append(append([]byte{}, base[:cut]...), chunk...), base[cut:]...)
	mustNotBeVerbatim(t, raw, token)
	abs := writeFile(t, root, "nested.png", string(raw))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "nested.png", abs), "nested.png")
}

// TestManySignaturesInAStructuralFieldStayCheap: the field rule must cost one
// pass over the field, not one pass per signature it finds. A field packed with
// signatures ahead of a long tail is the shape that turns a per-hit walk
// quadratic, and no operation bound can save a bound whose operations are each
// O(field).
func TestManySignaturesInAStructuralFieldStayCheap(t *testing.T) {
	root := t.TempDir()
	var payload bytes.Buffer
	payload.WriteString("Comment\x00")
	for i := 0; i < 400; i++ {
		payload.Write(zlibOf(t, []byte("x")))
	}
	payload.Write(bytes.Repeat([]byte("filler for the tail\n"), 13000)) // ~256 KB
	base := pngBytes(t)
	chunk := pngChunk("tEXt", payload.Bytes())
	cut := len(base) - 12
	raw := append(append(append([]byte{}, base[:cut]...), chunk...), base[cut:]...)
	abs := writeFile(t, root, "packed.png", string(raw))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	// The cost is counted as bytes the field rule's search is handed, not
	// timed (iss-2609232048579579). One pass per signature KIND over each field
	// hands it about len(containerSignatures) times the file; one pass per
	// signature FOUND hands it the tail once per hit, 400 times over. The bound
	// allows four times the one-pass cost, so a second legitimate pass over an
	// overlapping region stays inside it while the per-hit walk lands two
	// orders of magnitude past it.
	searched := 0
	signatureSearch = func(s, sep []byte) int {
		searched += len(s)
		return bytes.Index(s, sep)
	}
	t.Cleanup(func() { signatureSearch = bytes.Index })
	res := scanOne(t, sc, "packed.png", abs)
	const passes = 4
	if bound := passes * len(containerSignatures) * len(raw); searched > bound {
		t.Fatalf("a signature-packed field of %d bytes had its search handed %d bytes, want at most %d (%d passes per signature kind over the file) — work that far past one pass per kind is the field rule walking the field per hit, not per field: %+v",
			len(raw), searched, bound, passes, res)
	}
	if searched == 0 {
		t.Fatal("the field rule's search was never handed the field; the count proves nothing")
	}
}

// TestZipNamesDifferBetweenHeaders: an entry has two names, one in its local
// header and one in the directory, and nothing requires them to match. Each is
// a field of its own, so covering one is not covering the other — the fixture
// above, where the writer put the same name in both, would be satisfied by
// either check alone.
func TestZipNamesDifferBetweenHeaders(t *testing.T) {
	root := t.TempDir()
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, where := range []string{"local", "directory"} {
		token := fakeToken()
		member := gzipOf(t, secretBody(token))
		blank := "a" + strings.Repeat("b", len(member)-1) // same length, harmless
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		w, err := zw.CreateHeader(&zip.FileHeader{Name: blank, Method: zip.Deflate})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("harmless\n")); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		raw := buf.Bytes()
		at := 30 // the local header's name
		if where == "directory" {
			cd := bytes.Index(raw, []byte("PK\x01\x02"))
			if cd < 0 {
				t.Fatal("could not locate the central directory of the fixture")
			}
			at = cd + 46
		}
		copy(raw[at:at+len(member)], member)
		if _, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw))); err != nil {
			t.Skip("this Go version rejects the crafted name")
		}
		name := where + "name.zip"
		abs := writeFile(t, root, name, string(raw))
		assertCaughtOrRefused(t, scanOne(t, sc, name, abs), name)
	}
}

// The gzip header and the tar header are structural fields too — the field
// rule reached zip and PNG before it reached them.

// TestGzipHeaderFieldsAreCovered: a gzip header carries FEXTRA, FNAME and
// FCOMMENT, and FEXTRA is length-prefixed rather than NUL-terminated, so it
// holds up to 64 KiB of anything. Three lines of the standard library build it.
func TestGzipHeaderFieldsAreCovered(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	member := gzipOf(t, secretBody(token))
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Extra = member
	if _, err := zw.Write([]byte("harmless\n")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := gzip.NewReader(bytes.NewReader(buf.Bytes())); err != nil {
		t.Skip("this Go version rejects the crafted header")
	}
	abs := writeFile(t, root, "header.gz", buf.String())
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "header.gz", abs), "header.gz")
}

// TestTarPAXRecordIsCovered: a PAX record's key is the archive's choosing and
// its value may hold any bytes at all — the format forbids NULs only in the
// four records the walk already knew about.
func TestTarPAXRecordIsCovered(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	member := gzipOf(t, secretBody(token))
	body := []byte("harmless\n")
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	h := &tar.Header{Name: "readme.txt", Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg,
		Format: tar.FormatPAX, PAXRecords: map[string]string{"ABCD.stash": string(member)}}
	if err := tw.WriteHeader(h); err != nil {
		t.Skip("this Go version rejects the crafted PAX record")
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), member) {
		t.Skip("the writer did not carry the record into the archive")
	}
	abs := writeFile(t, root, "pax.tgz", string(gzipOf(t, buf.Bytes())))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "pax.tgz", abs), "pax.tgz")
}

// TestTarEntryPaddingIsCovered: an entry's body is padded to the 512-byte
// block, and the reader discards the padding while the counting walk includes
// it in the archive's extent — up to 511 bytes per entry that no rule sees.
func TestTarEntryPaddingIsCovered(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	member := gzipOf(t, secretBody(token))
	body := []byte("harmless\n")
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(&tar.Header{Name: "readme.txt", Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	raw := buf.Bytes()
	at := 512 + len(body) // the padding after the entry's body
	if at+len(member) > 1024 {
		t.Fatalf("the fixture's member does not fit in one block of padding")
	}
	copy(raw[at:], member)
	if tr := tar.NewReader(bytes.NewReader(raw)); func() bool { _, err := tr.Next(); return err != nil }() {
		t.Skip("this Go version rejects the crafted padding")
	}
	abs := writeFile(t, root, "pad.tgz", string(gzipOf(t, raw)))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "pad.tgz", abs), "pad.tgz")
}

// TestTarEntryNameIsCovered: the zip walk guards both copies of an entry name;
// the tar walk byte-scanned its four name-shaped fields and judged none of
// them. PAX lifts the classic length limit, so the field is as long as the
// archive likes.
func TestTarEntryNameIsCovered(t *testing.T) {
	root := t.TempDir()
	token := fakeToken()
	member := bytes.ReplaceAll(gzipOf(t, secretBody(token)), []byte{0}, []byte{'0'})
	body := []byte("harmless\n")
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	h := &tar.Header{Name: "a" + string(member), Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg, Format: tar.FormatPAX}
	if err := tw.WriteHeader(h); err != nil {
		t.Skip("this Go version rejects the crafted name")
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("\x1f\x8b\x08")) {
		t.Skip("the writer did not carry the name into the archive")
	}
	abs := writeFile(t, root, "paxname.tgz", string(gzipOf(t, buf.Bytes())))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "paxname.tgz", abs), "paxname.tgz")
}

// TestPNGWithAnICCProfileStillDecodes is the over-refusal guard for the same
// rule. An ICC profile is binary BY DEFINITION, so a rule that refuses a
// compressed chunk whose output is not text sends every colour-managed PNG to
// ContentUnverified — and a tier people learn to override has stopped working.
// iCCP takes IDAT's treatment: inflated, byte-scanned, not asked to be a
// region.
func TestPNGWithAnICCProfileStillDecodes(t *testing.T) {
	root := t.TempDir()
	profile := make([]byte, 600)
	for i := range profile {
		profile[i] = byte(i*7 + 3) // binary, no container signature
	}
	var z bytes.Buffer
	zw := zlib.NewWriter(&z)
	if _, err := zw.Write(profile); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	base := pngBytes(t)
	chunk := pngChunk("iCCP", append([]byte("ICC Profile\x00\x00"), z.Bytes()...))
	cut := len(base) - 12
	raw := append(append(append([]byte{}, base[:cut]...), chunk...), base[cut:]...)
	abs := writeFile(t, root, "icc.png", string(raw))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "icc.png", abs)
	if !contains(res.ContentDecoded, "icc.png") {
		t.Fatalf("a colour-managed PNG must still decode: %+v", res)
	}
	if len(res.Findings) != 0 {
		t.Errorf("an ICC profile must not trip a rule: %+v", res.Findings)
	}
}
