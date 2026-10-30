package claim

import (
	"strings"
	"testing"
)

// TestANotEndedSentenceMarksEveryMentionOfAClaimedHolder: each mention of a
// holder the reader only has the trailer's word for is marked claimed.
func TestANotEndedSentenceMarksEveryMentionOfAClaimedHolder(t *testing.T) {
	l := notEndedLine("rel-v0.16.0", mayaB, true)
	if n := strings.Count(l, mayaB+" ("+SenderClaimed+")"); n != strings.Count(l, mayaB) || n < 2 {
		t.Fatalf("%q marks %d of %d mentions claimed; want every one", l, n, strings.Count(l, mayaB))
	}
	if l := notEndedLine("rel-v0.16.0", mayaB, false); strings.Contains(l, SenderClaimed) {
		t.Fatalf("%q marks an unclaimed holder", l)
	}
}
