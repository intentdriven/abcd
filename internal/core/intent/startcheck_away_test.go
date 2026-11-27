package intent

import (
	"strings"
	"testing"
)

// TestOpenQuestionsRemedyRoutesWhileAway: an open question names the planning
// interview for a person who is there, and the route that keeps work moving
// while they are away (iss-2611260602175153): a technical question decided and
// recorded, one only the person may answer queued, and the next ready item
// taken, so no open question stops autonomous work.
func TestOpenQuestionsRemedyRoutesWhileAway(t *testing.T) {
	content := "---\nid: itd-10\n---\n# T\n\n## Open Questions\n\n- which cache?\n"
	row := startOpenQuestionsRow("itd-10", content)
	if row.OK {
		t.Fatalf("an open question refuses: %+v", row)
	}
	for _, want := range []string{"planning interview", "away", "abcd decide", "abcd mode queue", "next ready item"} {
		if !strings.Contains(row.Remedy, want) {
			t.Errorf("the remedy leaves out %q:\n%s", want, row.Remedy)
		}
	}
}
