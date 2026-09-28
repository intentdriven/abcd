package launch

import "testing"

// TestProseLinesReadsTheBlockOnTheOneDelimiterRule: the prose gate takes the
// frontmatter block's extent from frontmatter.Close, so an indented rule does
// not close it and the line after it is metadata, not prose — as Fields reads
// it (iss-2608270908348042).
func TestProseLinesReadsTheBlockOnTheOneDelimiterRule(t *testing.T) {
	t.Parallel()
	got := proseLines([]byte("---\ntitle: x\n  ---\nkind: y\n---\nbody line\n"))
	for _, l := range got {
		if l.text == "kind: y" || l.text == "  ---" {
			t.Fatalf("proseLines read frontmatter line %d (%q) as prose: the indented rule closed the block", l.n, l.text)
		}
	}
	if len(got) == 0 || got[0].text != "body line" {
		t.Fatalf("proseLines = %+v, want the body to open at \"body line\"", got)
	}
}
