package mode

import (
	"errors"
	"fmt"
)

// ErrAway is the badge's refusal while the person is away: a loop parked on
// the facilitator or the product thinker is a question waiting for someone
// who is not there (dcn-2610180913431493).
var ErrAway = errors.New("mode: the person is away")

// RefuseParkWhileAway refuses parking the badge on a person while the away
// mode holds (away true), naming where the question goes instead. Managed is
// never refused, and nothing is refused while the person is present. The
// three-word vocabulary is untouched: the away mode is not a fourth word, and
// whether it holds is the caller's to read (internal/core/autonomy), so this
// package keeps no knowledge of the away store.
func RefuseParkWhileAway(s State, away bool) error {
	if !away || s == Managed {
		return nil
	}
	return fmt.Errorf("%w, so no stop parks on the %s: decide it and record it with `abcd decide`, "+
		"or queue an act only the person may approve with `abcd mode queue \"<step>\"`, and carry on", ErrAway, s.Addressee())
}
