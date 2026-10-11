package ahoy

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestReadmePartsRefuseWritingPastTheSizeCap: a README just under
// maxAhoyFileBytes must not be grown past it, or every later sync and removal
// refuses at the guarded read and abcd cannot take its own part back out
// (iss-2610111552464704). The composed file is measured before the write; one
// over the cap is refused naming the cap, and the file is left as it was.
func TestReadmePartsRefuseWritingPastTheSizeCap(t *testing.T) {
	head := "# Project\n\n"
	pad := strings.Repeat("x", maxAhoyFileBytes-len(head)-11) + "\n"
	path := writeReadme(t, head+pad)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) > maxAhoyFileBytes {
		t.Fatalf("fixture is %d bytes, over the %d-byte cap", len(before), maxAhoyFileBytes)
	}

	_, err = syncReadmeParts(path, []readmePartWant{{Name: "licence", Body: "Licensed under the MIT License; see [LICENSE](LICENSE)."}})
	if err == nil {
		t.Fatalf("sync grew a %d-byte README past the cap without refusing; it is now %d bytes", len(before), len(readReadme(t, path)))
	}
	if !strings.Contains(err.Error(), strconv.Itoa(maxAhoyFileBytes)) {
		t.Errorf("refusal %q does not name the %d-byte cap", err, maxAhoyFileBytes)
	}
	if after := []byte(readReadme(t, path)); !bytes.Equal(after, before) {
		t.Errorf("README changed (%d -> %d bytes) although the write was refused", len(before), len(after))
	}

	// The file stays readable, so a removal still reads it and has nothing to do.
	res, err := removeReadmeParts(path)
	if err != nil {
		t.Fatalf("removal after the refusal: %v", err)
	}
	if res.Wrote {
		t.Error("removal wrote a README that held no part")
	}
}
