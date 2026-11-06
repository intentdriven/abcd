package landing

import (
	"slices"
	"time"

	"github.com/intentdriven/abcd/internal/core/runstop"
)

// stopReadingOff is set by the package's TestMain alone: a stop on the machine running the
// tests holds no test.
var stopReadingOff bool

// readStop is the read every other verb makes of a stop with no run to pin it to
// (itd-2611041549471712).
func readStop() runstop.Reading {
	if stopReadingOff {
		return runstop.Reading{}
	}
	return runstop.ReadHomes(runstop.MachineStopHome(), runstop.ThisAccountHome(), nil)
}

// stopHold is a person's stop at `abcd land`, made before anything runs, again after the
// preflight and again at the arming, each of which can follow the last by a long time: while
// a stop holds, or a read cannot be trusted to have seen one, nothing is pushed, armed or
// queued, held as contention as the verbs that take a stop are. A file the read refused and a
// stop another account left are no stop: each is named among the notices.
func stopHold(res *Result, o Options) error {
	r := o.StopRead()
	for _, line := range slices.Concat(r.Refused, r.Foreign) {
		if !slices.Contains(res.Notices, line) {
			res.Notices = append(res.Notices, line)
		}
	}
	switch {
	case r.Stop != nil:
		return &Refusal{
			Reason: clean("a stop set by " + r.Stop.SetBy + " at " + r.Stop.SetAt.UTC().Format(time.RFC3339) + " holds, so abcd land pushes, arms and queues nothing"),
			Fix:    "the person's own `abcd implement stop --lift` lifts it, then run abcd land again",
			Held:   true,
		}
	case r.Held != "":
		return &Refusal{
			Reason: clean(r.Held + ", so abcd land pushes, arms and queues nothing"),
			Fix:    "tell the person; they put it back with `abcd ahoy shared-folder`, then run abcd land again",
			Held:   true,
		}
	}
	return nil
}
