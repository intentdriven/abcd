package cli

import (
	"strings"
	"testing"
)

// TestReserveRefusesACommaJoinedFootprint (iss-2611260633438543): a
// --footprint value that joins several paths with commas, and names no path
// in the checkout, is refused (exit 2) naming the repeated-flag form, and
// nothing reaches origin; before it the value was stored as one path named
// with a comma, which no overlap check ever matched. A path whose own name
// holds a comma, and exists, is reserved as written.
func TestReserveRefusesACommaJoinedFootprint(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	writeRel(t, p, "odd,name.md", "a\n")
	gitCmd(t, p, "add", "-A")
	gitCommit(t, p, "commit", "-q", "-m", "odd")
	gitCmd(t, p, "push", "-q", "origin", "main")

	code, stdout, stderr := reserveRun(t, p, "reserve", "iss-1", "--footprint", "a.go,b.go")
	if code != 2 || !strings.Contains(stderr, "--footprint a.go --footprint b.go") || !strings.Contains(stderr, "nothing was pushed") {
		t.Fatalf("a comma-joined footprint: exit %d, stdout %q, stderr %q; want exit 2 naming the repeated-flag form", code, stdout, stderr)
	}
	if w.remoteTip(t, "iss-1") != "" {
		t.Fatal("the refused reserve reached origin")
	}
	if code, stdout, stderr := reserveRun(t, p, "reserve", "iss-2", "--footprint", "odd,name.md"); code != 0 {
		t.Fatalf("an existing path named with a comma: exit %d, stdout %q, stderr %q; want it reserved", code, stdout, stderr)
	}
}
