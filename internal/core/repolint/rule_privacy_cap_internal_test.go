package repolint

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadTrackedFileCapBoundary pins readTrackedFile's cap handling after the
// cap+1 change: a file at exactly the cap is still scanned whole (so a trailing
// leak is not missed), and a file already over the cap at fstat time is reported
// not-scanned rather than truncated-and-reported-clean.
//
// The remaining case — a file that grows PAST the cap between the fstat and the
// read — cannot be raced deterministically, so it is pinned through the capRead
// seam instead: a reader carrying more bytes than the fstat-time cap stands in
// for the grown file, and the cap+1 probe must refuse it whole rather than
// return the truncated prefix as scanned-clean.
func TestReadTrackedFileCapBoundary(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	// A file of exactly maxScanBytes whose final bytes carry a path-shaped leak:
	// it must be read whole (ok=true) so the trailing content is scannable.
	atCap := filepath.Join(dir, "at-cap.txt")
	buf := make([]byte, maxScanBytes)
	for i := range buf {
		buf[i] = 'a'
	}
	tail := []byte("\n/home/somebody/secret\n") // abcd-lint:allow
	copy(buf[len(buf)-len(tail):], tail)
	if err := os.WriteFile(atCap, buf, 0o644); err != nil {
		t.Fatal(err)
	}
	data, ok, oversize, _ := readTrackedFile(root, "at-cap.txt")
	if !ok {
		t.Fatalf("a file exactly at the cap must scan whole; ok=false oversize=%v", oversize)
	}
	if int64(len(data)) != maxScanBytes {
		t.Errorf("at-cap read length = %d, want %d (cap+1 must not truncate an at-cap file)", len(data), maxScanBytes)
	}
	if !strings.Contains(string(data), "/home/somebody/secret") { // abcd-lint:allow
		t.Error("the trailing content of an at-cap file was not read")
	}

	// A file already over the cap at fstat time is not scanned, and (being
	// textual) is reported as such so the caller warns instead of staying silent.
	overCap := filepath.Join(dir, "over-cap.txt")
	if err := os.WriteFile(overCap, append(buf, 'z'), 0o644); err != nil {
		t.Fatal(err)
	}
	data, ok, oversize, _ = readTrackedFile(root, "over-cap.txt")
	if ok || data != nil {
		t.Errorf("an over-cap file must not be scanned; ok=%v len=%d", ok, len(data))
	}
	if !oversize {
		t.Error("an over-cap textual file must report oversizeText so the caller warns")
	}

	// The grown-file TOCTOU, via the capRead seam: the reader carries one byte
	// more than the cap the fstat saw. A capped read (no +1 probe) would return
	// the truncated prefix as scanned-clean; the probe must refuse whole and
	// report not-scanned-but-textual.
	data, ok, oversize = capRead(bytes.NewReader(append(buf, 'z')))
	if ok || data != nil {
		t.Errorf("a grown file must not be scanned as a truncated prefix; ok=%v len=%d", ok, len(data))
	}
	if !oversize {
		t.Error("a grown textual file must report oversizeText so the caller warns")
	}
}
