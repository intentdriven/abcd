package scanner

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"testing"
)

// rawTarBlockWith is rawTarBlock with a hook that edits the block before its
// checksum is written: the sparse fields an old GNU sparse header carries.
func rawTarBlockWith(name string, typeflag byte, size int, edit func([]byte)) []byte {
	blk := rawTarBlock(name, typeflag, size)
	edit(blk)
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

// padBlock zero-pads b to the 512-byte block.
func padBlock(b []byte) []byte {
	if p := (512 - len(b)%512) % 512; p > 0 {
		b = append(b, make([]byte, p)...)
	}
	return b
}

// paxRecord spells one PAX record, "%d %s\n", the length counting itself.
func paxRecord(kv string) []byte {
	n := len(kv) + 3
	for len(fmt.Sprint(n))+len(kv)+2 != n {
		n = len(fmt.Sprint(n)) + len(kv) + 2
	}
	return []byte(fmt.Sprintf("%d %s\n", n, kv))
}

// tarEntryCount reads raw with the standard reader and returns how many
// entries it hands over; any error fails the test as a broken control.
func tarEntryCount(t *testing.T, raw []byte) int {
	t.Helper()
	tr := tar.NewReader(bytes.NewReader(raw))
	n := 0
	for {
		_, err := tr.Next()
		if err == io.EOF {
			return n
		}
		if err != nil {
			t.Fatalf("control: the hand-built archive must read cleanly: %v", err)
		}
		if _, err := io.Copy(io.Discard, tr); err != nil {
			t.Fatalf("control: an entry body must read cleanly: %v", err)
		}
		n++
	}
}

// iss-2610090821512707, review round: an extension header with no entry after
// it, only the end marker, is consumed by the Next call that then reports
// io.EOF. The walk broke on io.EOF before covering what that call read, so a
// long name placed last in the archive was read by nobody.
func TestTarExtensionHeaderBeforeTheEndMarkerIsCovered(t *testing.T) {
	token := syntheticPAT(2610090821512709)
	member := gzipOf(t, secretBody(token))
	regular := rawTarEntry("readme.txt", tar.TypeReg, []byte("harmless\n"))
	dangling := rawTarEntry("././@LongLink", 'L', append([]byte("ghost.txt\x00"), member...))
	end := make([]byte, 1024)
	cases := []struct {
		name    string
		raw     []byte
		entries int
	}{
		{"after a regular entry", append(append(append([]byte{}, regular...), dangling...), end...), 1},
		{"with no regular entry", append(append([]byte{}, dangling...), end...), 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mustNotBeVerbatim(t, tc.raw, token)
			if n := tarEntryCount(t, tc.raw); n != tc.entries {
				t.Fatalf("control: the reader must see %d entries, saw %d", tc.entries, n)
			}
			root := t.TempDir()
			abs := writeFile(t, root, "dangling.tar", string(tc.raw))
			sc, err := New(root)
			if err != nil {
				t.Fatal(err)
			}
			res := scanOne(t, sc, "dangling.tar", abs)
			if res.HardFails == 0 {
				t.Fatalf("the member in the trailing long name was never read: %+v", res)
			}
		})
	}
}

// sparse10Archive builds a PAX sparse 1.0 entry: the sparse map is the first
// block of the entry's data, which tar.Reader.Next reads and parses for the
// numbers it needs, and slack is whatever follows the map in that block.
func sparse10Archive(slack []byte) []byte {
	var pax []byte
	for _, kv := range []string{"GNU.sparse.major=1", "GNU.sparse.minor=0", "GNU.sparse.name=readme.txt", "GNU.sparse.realsize=9"} {
		pax = append(pax, paxRecord(kv)...)
	}
	data := []byte("harmless\n")
	physical := append(padBlock(append([]byte("1\n0\n9\n"), slack...)), data...)
	raw := append(rawTarBlock("PaxHeaders/readme.txt", tar.TypeXHeader, len(pax)), padBlock(pax)...)
	raw = append(raw, rawTarBlock("GNUSparseFile.0/readme.txt", tar.TypeReg, len(physical))...)
	raw = append(raw, padBlock(physical)...)
	return append(raw, make([]byte, 1024)...)
}

// oldGNUSparseArchive builds an old GNU sparse entry (typeflag S) whose header
// says an extension block follows. ext is that block: the reader parses its
// 24-byte entries up to the first whose offset opens with a NUL and its
// isExtended byte, and looks at nothing else in it.
func oldGNUSparseArchive(ext []byte) []byte {
	data := []byte("harmless\n")
	hdr := rawTarBlockWith("readme.txt", tar.TypeGNUSparse, len(data), func(blk []byte) {
		copy(blk[386:398], "00000000000\x00") // entry 0: offset 0
		copy(blk[398:410], "00000000011\x00") // entry 0: length 9
		blk[482] = 1                          // isExtended
		copy(blk[483:495], "00000000011\x00") // realSize 9
	})
	raw := append(append(append([]byte{}, hdr...), ext...), padBlock(data)...)
	return append(raw, make([]byte, 1024)...)
}

// iss-2610090821512707, review round: Next consumes bytes after the entry's
// own header (the rest of a PAX sparse 1.0 map block, the slack of an old GNU
// sparse extension block) and parses only part of them. The walk stopped at
// the entry's header, so the unparsed rest was read by nobody.
func TestTarSparseMapSlackIsNotVouchedFor(t *testing.T) {
	token := syntheticPAT(2610090821512710)
	member := gzipOf(t, secretBody(token))
	if len(member) > 500 {
		t.Fatalf("control: the member must fit an extension block, is %d bytes", len(member))
	}
	ext := make([]byte, 512)
	copy(ext[1:], member) // byte 0 is NUL, so entry 0 ends the map
	cases := map[string][]byte{
		"pax sparse 1.0 map block":       sparse10Archive(member),
		"old gnu sparse extension block": oldGNUSparseArchive(ext),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			mustNotBeVerbatim(t, raw, token)
			if n := tarEntryCount(t, raw); n != 1 {
				t.Fatalf("control: the reader must see one entry, saw %d", n)
			}
			root := t.TempDir()
			abs := writeFile(t, root, "sparse.tar", string(raw))
			sc, err := New(root)
			if err != nil {
				t.Fatal(err)
			}
			assertCaughtOrRefused(t, scanOne(t, sc, "sparse.tar", abs), "sparse.tar")
		})
	}
}

// The controls: a sparse map whose block is zero past what the reader parses,
// in either spelling, and a long name whose body ends the archive with nothing
// after its NUL, still decode clean.
func TestTarSparseMapsAndATrailingLongNameStillDecode(t *testing.T) {
	ext := make([]byte, 512)
	copy(ext[0:12], "00000000011\x00")  // entry 0: offset 9, past the header's entry
	copy(ext[12:24], "00000000000\x00") // entry 0: length 0
	cases := map[string][]byte{
		"pax sparse 1.0":     sparse10Archive(nil),
		"old gnu sparse":     oldGNUSparseArchive(ext),
		"trailing long name": append(rawTarEntry("././@LongLink", 'L', []byte("ghost.txt\x00")), make([]byte, 1024)...),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			tarEntryCount(t, raw)
			root := t.TempDir()
			abs := writeFile(t, root, "clean.tar", string(raw))
			sc, err := New(root)
			if err != nil {
				t.Fatal(err)
			}
			res := scanOne(t, sc, "clean.tar", abs)
			if !contains(res.ContentDecoded, "clean.tar") || len(res.Findings) != 0 {
				t.Fatalf("%s must decode clean: %+v", name, res)
			}
		})
	}
}
