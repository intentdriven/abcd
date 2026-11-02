package cli

import (
	"strings"
	"testing"
)

// TestImplementWidenRefusesAnIntentsRunAndAMissingLane: `implement widen` is
// wired into the command tree (iss-2610311224444576): an intent's run, whose
// footprint is its spec's, is refused at the reservation stage with exit 2,
// naming the spec, and a call without --lane is refused before any run is
// read.
func TestImplementWidenRefusesAnIntentsRunAndAMissingLane(t *testing.T) {
	buildRepo(t)
	mustImplement(t, "build", "itd-10", "--json")
	ref := refusalDocs(t, 2, "implement", "widen", "internal/widget/b.go", "--lane", "lane-1", "--json")
	if ref["stage"] != "reservation" || !strings.Contains(ref["reason"].(string), "holds the footprint its spec names") {
		t.Fatalf("an intent's run is refused at the reservation stage: %v", ref)
	}
	if remedy := ref["remedy"].(string); !strings.Contains(remedy, "## Footprint") {
		t.Fatalf("the remedy names the spec's footprint section: %q", remedy)
	}
	if code, out, _ := implementCLI(t, "implement", "widen", "internal/widget/b.go", "--json"); code != 2 || !strings.Contains(out, "--lane is required") {
		t.Fatalf("a widening without --lane is refused, exit 2: %d %s", code, out)
	}
}
