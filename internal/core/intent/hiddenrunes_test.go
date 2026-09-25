package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/grounds"
)

// TestCreateFromTextEncodesHiddenRunes: an `abcd intent "<text>"` headline and
// press release reached the committed draft with a bidi override or a
// zero-width rune verbatim (iss-2608301206073609). Both are encoded at the one
// draft-mint primitive, losslessly.
func TestCreateFromTextEncodesHiddenRunes(t *testing.T) {
	root := t.TempDir()
	it, err := CreateFromText(root, "The card ‮respects​ the reader. More prose here.", TextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, it.Path))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(data)
	for _, r := range []string{"‮", "​"} {
		if strings.Contains(raw, r) {
			t.Errorf("hidden rune %q reached the draft verbatim:\n%q", r, raw)
		}
	}
	if !strings.Contains(raw, "# The card %E2%80%AErespects%E2%80%8B the reader\n") {
		t.Errorf("the headline does not carry the encoded runes:\n%s", raw)
	}
}

// TestRecordGroundsEncodesHiddenRunes: intent grounds are the same boundary.
func TestRecordGroundsEncodesHiddenRunes(t *testing.T) {
	root := t.TempDir()
	const rel = plannedDir + "/itd-10-alpha.md"
	writeFile(t, root, rel, plannedUnlinked("itd-10", "alpha"))
	g := mustGrounds(t, grounds.Pursued, "we expect a stamped ‮identity to survive rewording")
	if _, err := RecordGrounds(root, "itd-10", g); err != nil {
		t.Fatal(err)
	}
	raw := readIntent(t, root, rel)
	if strings.Contains(raw, "‮") || !strings.Contains(raw, "stamped %E2%80%AEidentity") {
		t.Fatalf("intent grounds did not encode the bidi override:\n%q", raw)
	}
}
