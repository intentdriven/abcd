package mode

import (
	"errors"
	"strings"
	"testing"
)

// TestBadgeRefusedWhileAway: while the person is away, parking the badge on
// either person is refused, naming `abcd decide`, since a badge parked on the
// person is a question waiting for them; `managed` is never refused, and
// nothing is refused while the person is present (dcn-2610180913431493).
func TestBadgeRefusedWhileAway(t *testing.T) {
	for _, s := range []State{ProductThinker, Facilitator} {
		err := RefuseParkWhileAway(s, true)
		if !errors.Is(err, ErrAway) {
			t.Errorf("%s while away: err = %v, want ErrAway", s, err)
			continue
		}
		if !strings.Contains(err.Error(), "abcd decide") || !strings.Contains(err.Error(), "abcd mode queue") {
			t.Errorf("%s while away: the refusal does not name `abcd decide` and `abcd mode queue`: %v", s, err)
		}
		if err := RefuseParkWhileAway(s, false); err != nil {
			t.Errorf("%s while present refused: %v", s, err)
		}
	}
	if err := RefuseParkWhileAway(Managed, true); err != nil {
		t.Errorf("managed while away refused: %v", err)
	}
}
