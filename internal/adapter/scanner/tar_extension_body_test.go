package scanner

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"testing"
)

// rawTarBlock hand-builds one GNU-magic tar header block. archive/tar's writer
// refuses to emit a type-L or type-K header on request, so the extension
// headers these tests need are framed by hand.
func rawTarBlock(name string, typeflag byte, size int) []byte {
	blk := make([]byte, 512)
	copy(blk[0:100], name)
	copy(blk[100:108], "0000644\x00")
	copy(blk[108:116], "0000000\x00")
	copy(blk[116:124], "0000000\x00")
	copy(blk[124:136], fmt.Sprintf("%011o\x00", size))
	copy(blk[136:148], "00000000000\x00")
	blk[156] = typeflag
	copy(blk[257:265], "ustar  \x00")
	for i := 148; i < 156; i++ {
		blk[i] = ' '
	}
	sum := 0
	for _, c := range blk {
		sum += int(c)
	}
	copy(blk[148:156], fmt.Sprintf("%06o\x00 ", sum))
	return blk
}

// rawTarEntry is a header block followed by its body, zero-padded to the block.
func rawTarEntry(name string, typeflag byte, body []byte) []byte {
	out := append(rawTarBlock(name, typeflag, len(body)), body...)
	if pad := (512 - len(body)%512) % 512; pad > 0 {
		out = append(out, make([]byte, pad)...)
	}
	return out
}

// iss-2610090821512707: an extension header's body (a GNU long name or long
// link, a PAX record set) is consumed inside tar.Reader.Next, which keeps only
// what it parses out of it. The rest of the body was read by nobody, and the
// archive was reported decoded.
func TestTarExtensionHeaderBodyIsCovered(t *testing.T) {
	token := syntheticPAT(2610090821512707)
	member := gzipOf(t, secretBody(token))
	regular := rawTarEntry("readme.txt", tar.TypeReg, []byte("harmless\n"))
	end := make([]byte, 1024)
	cases := map[string][]byte{
		"long name": append(append(rawTarEntry("././@LongLink", 'L', append([]byte("readme.txt\x00"), member...)), regular...), end...),
		"long link": append(append(rawTarEntry("././@LongLink", 'K', append([]byte("target.txt\x00"), member...)), regular...), end...),
		"long name overridden": append(append(append(
			rawTarEntry("././@LongLink", 'L', append([]byte("first.txt\x00"), member...)),
			rawTarEntry("././@LongLink", 'L', []byte("readme.txt\x00"))...), regular...), end...),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			mustNotBeVerbatim(t, raw, token)
			// The archive is what the standard reader sees: one regular entry.
			tr := tar.NewReader(bytes.NewReader(raw))
			h, err := tr.Next()
			if err != nil || h.Typeflag != tar.TypeReg {
				t.Fatalf("control: the hand-built archive must read as one regular entry: %v %+v", err, h)
			}
			if _, err := tr.Next(); err != io.EOF {
				t.Fatalf("control: one entry expected, got %v", err)
			}
			root := t.TempDir()
			abs := writeFile(t, root, "bundle.tar", string(raw))
			readme := writeFile(t, root, "README.md", "clean documentation\n")
			sc, err := New(root)
			if err != nil {
				t.Fatal(err)
			}
			res, err := sc.ScanBundle([]BundleFile{
				{LogicalPath: "README.md", ResolvedPath: readme},
				{LogicalPath: "bundle.tar", ResolvedPath: abs},
			})
			if err != nil {
				t.Fatal(err)
			}
			if res.HardFails == 0 {
				t.Fatalf("the member in the %s body was never read: %+v", name, res)
			}
		})
	}
}

// A PAX record set is parsed into a map, so a key written twice keeps only its
// last value: the first was consumed and never seen by the header walk.
func TestTarDuplicatePAXRecordIsNotVouchedFor(t *testing.T) {
	token := syntheticPAT(2610090821512708)
	member := gzipOf(t, secretBody(token))
	record := func(kv []byte) []byte {
		// "%d %s\n" where %d counts the whole record, itself included.
		n := len(kv) + 3
		for len(fmt.Sprint(n))+len(kv)+2 != n {
			n = len(fmt.Sprint(n)) + len(kv) + 2
		}
		return append(append([]byte(fmt.Sprint(n)+" "), kv...), '\n')
	}
	pax := append(record(append([]byte("comment="), member...)), record([]byte("comment=x"))...)
	raw := append(append(rawTarEntry("PaxHeaders/readme.txt", tar.TypeXHeader, pax),
		rawTarEntry("readme.txt", tar.TypeReg, []byte("harmless\n"))...), make([]byte, 1024)...)
	mustNotBeVerbatim(t, raw, token)
	tr := tar.NewReader(bytes.NewReader(raw))
	h, err := tr.Next()
	if err != nil || h.PAXRecords["comment"] != "x" {
		t.Fatalf("control: the reader must keep only the last comment: %v %+v", err, h)
	}
	root := t.TempDir()
	abs := writeFile(t, root, "pax.tar", string(raw))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	assertCaughtOrRefused(t, scanOne(t, sc, "pax.tar", abs), "pax.tar")
}

// The control: an ordinary GNU long name, and an ordinary archive of text,
// still decode clean.
func TestTarWithAPlainLongNameStillDecodes(t *testing.T) {
	long := bytes.Repeat([]byte("d/"), 80)
	long = append(long, "readme.txt"...)
	raw := append(append(rawTarEntry("././@LongLink", 'L', append(long, 0)),
		rawTarEntry("readme.txt", tar.TypeReg, []byte("harmless\n"))...), make([]byte, 1024)...)
	root := t.TempDir()
	abs := writeFile(t, root, "long.tar", string(raw))
	plain := writeFile(t, root, "plain.tar", string(tarOf(t, "notes.txt", []byte("harmless prose\n"))))
	// The standard writer's own long-name spellings: a GNU L header, and a PAX
	// path record.
	written := func(format tar.Format) string {
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		body := []byte("harmless prose\n")
		if err := tw.WriteHeader(&tar.Header{Name: string(long), Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg, Format: format}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	gnu := writeFile(t, root, "gnu.tar", written(tar.FormatGNU))
	pax := writeFile(t, root, "pax.tar", written(tar.FormatPAX))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []struct{ logical, abs string }{{"long.tar", abs}, {"plain.tar", plain}, {"gnu.tar", gnu}, {"pax.tar", pax}} {
		res := scanOne(t, sc, f.logical, f.abs)
		if !contains(res.ContentDecoded, f.logical) || len(res.Findings) != 0 {
			t.Fatalf("%s must decode clean: %+v", f.logical, res)
		}
	}
}
