package loop

import (
	"fmt"
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

// TestWithoutSessionNamesTheFlagWhereTheRefusalNamesTheSession: the shared
// run's refusals name the session in four phrases, and each is replaced by the
// flag, whatever the id: a value that is not valid UTF-8, one letter, or a word
// the refusal itself uses. The rest of the refusal stays as it was.
func TestWithoutSessionNamesTheFlagWhereTheRefusalNamesTheSession(t *testing.T) {
	for _, c := range []struct{ session, msg, want string }{
		{"\xff" + strings.Repeat("Q7", 8),
			fmt.Sprintf("refused: session %q is not a name (letters, digits)", "\xff"+strings.Repeat("Q7", 8)),
			"refused: the --session given is not a name (letters, digits)"},
		{"a",
			"refused: session a has not joined this run (run `abcd implement join` first)",
			"refused: the --session given has not joined this run (run `abcd implement join` first)"},
		{"join",
			"refused: session join has not joined this run (run `abcd implement join` first)",
			"refused: the --session given has not joined this run (run `abcd implement join` first)"},
		{"record",
			"the run state is locked: the session record for record is unreadable; remove sessions/record.json by hand to rejoin",
			"the run state is locked: the session record for the --session given is unreadable; remove sessions/<the --session given>.json by hand to rejoin"},
		{"host-a",
			"lane-1 is claimed by session host-ab, not host-a",
			"lane-1 is claimed by session host-ab, not host-a"},
	} {
		if got := withoutSession(c.msg, c.session); got != c.want {
			t.Errorf("withoutSession(%q):\n got %q\nwant %q", c.session, got, c.want)
		}
	}
}
