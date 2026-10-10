package scanner

import (
	"encoding/binary"
	"testing"
)

// pngWithIDATTail rebuilds base with extra appended inside its (single) IDAT
// chunk, after the zlib stream the encoder wrote, the chunk CRC recomputed.
func pngWithIDATTail(t *testing.T, base, extra []byte) []byte {
	t.Helper()
	out := append([]byte{}, base[:8]...)
	pos := 8
	for pos+12 <= len(base) {
		length := int(binary.BigEndian.Uint32(base[pos : pos+4]))
		typ := string(base[pos+4 : pos+8])
		payload := base[pos+8 : pos+8+length]
		if typ == "IDAT" {
			payload = append(append([]byte{}, payload...), extra...)
		}
		out = append(out, pngChunk(typ, payload)...)
		pos += 12 + length
	}
	return out
}

// iss-2610090821499579: inflating a compressed PNG chunk stops at the zlib
// checksum, and the walk covered only what came out. The bytes the chunk's
// length field still claimed after the stream were read by nobody, and the
// PNG was reported decoded. Each compressed chunk kind carries the same tail.
func TestPNGCompressedChunkTailAfterTheStreamIsCovered(t *testing.T) {
	token := syntheticPAT(2610090821499579)
	tail := gzipOf(t, secretBody(token))
	harmless := zlibOf(t, []byte("harmless\n"))
	base := pngBytes(t)
	spliced := func(chunk []byte) []byte {
		cut := len(base) - 12
		return append(append(append([]byte{}, base[:cut]...), chunk...), base[cut:]...)
	}
	cases := map[string][]byte{
		"zTXt": spliced(pngChunk("zTXt", append(append([]byte("Comment\x00\x00"), harmless...), tail...))),
		"iTXt": spliced(pngChunk("iTXt", append(append([]byte("Comment\x00\x01\x00en\x00\x00"), harmless...), tail...))),
		"iCCP": spliced(pngChunk("iCCP", append(append([]byte("ICC Profile\x00\x00"), harmless...), tail...))),
		"IDAT": pngWithIDATTail(t, base, tail),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			mustNotBeVerbatim(t, raw, token)
			abs := writeFile(t, root, "img.png", string(raw))
			readme := writeFile(t, root, "README.md", "clean documentation\n")
			sc, err := New(root)
			if err != nil {
				t.Fatal(err)
			}
			res, err := sc.ScanBundle([]BundleFile{
				{LogicalPath: "README.md", ResolvedPath: readme},
				{LogicalPath: "img.png", ResolvedPath: abs},
			})
			if err != nil {
				t.Fatal(err)
			}
			if res.HardFails == 0 {
				t.Fatalf("the member after the %s stream was never read: %+v", name, res)
			}
		})
	}
}

// The control: a zTXt chunk that is only its zlib member still decodes clean.
func TestPNGZTXtWithoutATailStillDecodes(t *testing.T) {
	root := t.TempDir()
	abs := writeFile(t, root, "img.png", string(pngWithZTXt(t, "harmless\n")))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "img.png", abs)
	if !contains(res.ContentDecoded, "img.png") || len(res.Findings) != 0 {
		t.Fatalf("a PNG whose zTXt is only its zlib member must decode clean: %+v", res)
	}
}
