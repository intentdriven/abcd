package scanner

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/testsecret"
)

// syntheticPAT builds a GitHub-PAT-shaped value at runtime from seed, so no
// secret-shaped literal enters source.
func syntheticPAT(seed uint64) string { return "ghp_" + testsecret.Synthetic(seed, 36) }

// iss-2610090821502084: the text branch sniffed the first 8 KiB and then
// counted the whole file as scanned by the text rules. A page of prose with a
// compressed member appended past the window shipped as fully scanned, the
// member never decoded and never named as a coverage gap.
func TestTextFileWithACompressedTailIsNotCountedScanned(t *testing.T) {
	root := t.TempDir()
	token := syntheticPAT(2610090821502084)
	prose := strings.Repeat("An ordinary line of documentation prose.\n", 260) // ~10.6 KiB
	raw := append([]byte(prose), gzipOf(t, secretBody(token))...)
	mustNotBeVerbatim(t, raw, token)
	abs := writeFile(t, root, "docs/notes.md", string(raw))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "docs/notes.md", abs)
	if res.HardFails > 0 {
		return
	}
	if !contains(res.Unscanned, "docs/notes.md") {
		t.Fatalf("a text file whose tail is not text was counted scanned with no finding and no coverage gap: %+v", res)
	}
	if res.FilesScanned != 0 {
		t.Fatalf("the file the text rules could not read whole was counted toward FilesScanned: %+v", res)
	}
	if res.UnscannedWhy["docs/notes.md"] == "" {
		t.Fatalf("the coverage gap carries no reason: %+v", res)
	}
}

// The controls: a long markdown file that is text all the way through still
// takes the text branch, and the same compressed bytes as a .gz still
// hard-fail through the decoder.
func TestLongTextFileStillScansAndGzipControlStillHardFails(t *testing.T) {
	root := t.TempDir()
	token := syntheticPAT(2610090821502085)
	long := strings.Repeat("A line of prose with a multibyte rune € in it.\n", 400)
	md := writeFile(t, root, "docs/long.md", long)
	gz := writeFile(t, root, "docs/notes.gz", string(gzipOf(t, secretBody(token))))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "docs/long.md", md)
	if res.FilesScanned != 1 || len(res.Unscanned) != 0 {
		t.Fatalf("a long text file must still scan with the text rules: %+v", res)
	}
	if res := scanOne(t, sc, "docs/notes.gz", gz); res.HardFails == 0 {
		t.Fatalf("the .gz control must hard-fail on the token: %+v", res)
	}
}
