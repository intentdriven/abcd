package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/landing"
)

// TestALandingAReleaseHoldsExitsThree: a landing held by a release being cut
// (itd-2609151838312703 C7) is contention, exit 3, as every verb that backs
// off exits; any other refusal stays exit 2.
func TestALandingAReleaseHoldsExitsThree(t *testing.T) {
	for _, c := range []struct {
		held bool
		code int
	}{{true, 3}, {false, 2}} {
		err := landRefusal(landing.Result{}, &landing.Refusal{Reason: "release v0.16.0 is being cut by quinn", Fix: "run abcd land again", Held: c.held})
		var ee *exitError
		if !errors.As(err, &ee) || ee.Code != c.code || !strings.Contains(ee.Msg, "release v0.16.0 is being cut by quinn") {
			t.Fatalf("held %v: %v; want exit %d naming the hold", c.held, err, c.code)
		}
	}
}
