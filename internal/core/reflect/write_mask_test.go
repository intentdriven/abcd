package reflect

import (
	"os"
	"strings"
	"testing"
)

// Answers come from a host-run composer, so they may carry bytes that make the
// rendered record read differently from what it says: an ESC that opens a
// terminal escape, a U+202E that reverses the line. They are masked before the
// write, after the scanner has redacted, and the answer's lines survive.
func TestWriteMasksControlAndBidiBytesInAnswers(t *testing.T) {
	r := releaseRepo(t)
	a := fullAnswers()
	a.WentWell.Text = "The cut was quick \x1b[31mand red\x1b[0m\nand the ‮seed‬ read well."
	a.Lessons.FollowUp = "Keep \x1b]0;title\x07 audits current."
	res, err := Write(r.Root(), WriteRequest{Tag: "v0.2.0", Answers: a, ProceedDespiteUnshipped: true, Now: fixedNow})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	data, err := os.ReadFile(abs(r, res.Path))
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"\x1b", "‮", "‬", "\x07"} {
		if strings.Contains(string(data), bad) {
			t.Errorf("the retrospective carries %q raw:\n%s", bad, data)
		}
	}
	if !strings.Contains(string(data), "The cut was quick ?[31mand red?[0m\nand the ?seed? read well.") {
		t.Errorf("the answer was not masked in place with its lines kept:\n%s", data)
	}
}
