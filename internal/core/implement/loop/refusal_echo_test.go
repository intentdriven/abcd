package loop

import (
	"strings"
	"testing"
)

// TestARefusalOfATypedIDDoesNotRepeatIt: a run id given to --run and a lane id
// given to --restart reach the refusal document build's --json writes, which
// the CLI's error boundary does not read, so the refusal describes the value
// and never repeats it.
func TestARefusalOfATypedIDDoesNotRepeatIt(t *testing.T) {
	typed := "Tk_" + strings.Repeat("Q7", 12)
	_, err := ReadState(t.TempDir(), typed)
	if r := mustRefusal(t, err); strings.Contains(r.Reason, typed) || !strings.Contains(r.Reason, "not a run id") {
		t.Errorf("a run id that is not one = %+v; want it refused without repeating it", r)
	}
	repo, runID, _, _, _ := deadLane(t)
	_, err = Restart(repo.Root(), runID, typed, "", Options{})
	if r := mustRefusal(t, err); strings.Contains(r.Reason, typed) || !strings.Contains(r.Reason, "has no lane") {
		t.Fatalf("a restart of a lane the run does not have = %+v; want it refused without repeating it", r)
	}
}
