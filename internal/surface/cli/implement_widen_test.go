package cli

import (
	"strings"
	"testing"
)

// TestImplementWidenTakesAnIntentsRunAndRefusesAMissingLane: `implement widen`
// is wired into the command tree (iss-2610311224444576): an intent's run is
// judged as an issue's is (iss-2611101316537446), refused only for what its
// lane is doing, never for its footprint being its spec's, and a call without
// --lane is refused before any run is read.
func TestImplementWidenTakesAnIntentsRunAndRefusesAMissingLane(t *testing.T) {
	buildRepo(t)
	mustImplement(t, "build", "itd-10", "--json")
	ref := refusalDocs(t, 2, "implement", "widen", "internal/widget/b.go", "--lane", "lane-1", "--json")
	reason := ref["reason"].(string)
	if ref["stage"] != "reservation" || strings.Contains(reason, "spec") || !strings.Contains(reason, "where no implementer edits") {
		t.Fatalf("an intent's run is refused only for its lane's stage: %v", ref)
	}
	if code, out, _ := implementCLI(t, "implement", "widen", "internal/widget/b.go", "--json"); code != 2 || !strings.Contains(out, "--lane is required") {
		t.Fatalf("a widening without --lane is refused, exit 2: %d %s", code, out)
	}
}
