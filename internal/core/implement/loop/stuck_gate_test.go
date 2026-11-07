package loop

import (
	"testing"
	"time"
)

// TestAnUnreadGateDoesNotCountTowardStuck: a read of the forge that failed at the arming
// says nothing of the lane, so it waits rather than being handed back.
func TestAnUnreadGateDoesNotCountTowardStuck(t *testing.T) {
	st := State{Key: "iss-1", Lanes: []Lane{{ID: "lane-1", Stage: StageLand, Key: "iss-1"}}}
	if st.Issue() == "" {
		t.Fatalf("the fixture is an issue run: %+v", st)
	}
	r := contend(string(StageLand), "", "lane-1", "the merge gate could not read", "again")
	r.gateUnread = true
	if noteRefusal(&st, r, time.Now()) || st.Lanes[0].Refused != nil {
		t.Fatalf("the wait was logged: %+v", st.Lanes[0].Refused)
	}
}
