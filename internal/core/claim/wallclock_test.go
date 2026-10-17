package claim

import (
	"testing"
	"time"
)

// wallClockSlack is how far a test stretches a wall-clock budget or bound
// (iss-2610170730017996). The shipped budgets fit a quiet machine; a test run
// shares its machine with other test runs, and on a development machine
// running two preflights a read of 300 announcements took 280 to 540 ms,
// past a 500 ms bound and the 600 ms read budget, at one-minute loads of 14
// to 40 on 16 cores: the time went on flushing to a disk the other runs were
// also writing, which the load average does not count, so no reading of the
// load says when to stretch. A fixed slack still fails a change that makes
// the work an order of magnitude slower, which is the regression these
// bounds are there to catch, and each test logs what it measured, so the
// cost on a quiet machine stays readable.
const wallClockSlack = 10

// slack is d stretched by wallClockSlack.
func slack(d time.Duration) time.Duration { return d * wallClockSlack }

// shippedPromptReadBudget is promptReadBudget as the binary ships it, so a
// stretch is always taken from it and never compounds.
var shippedPromptReadBudget = promptReadBudget

// stretchPromptBudget stretches promptReadBudget by wallClockSlack for the
// rest of the test, from its shipped value however often it is called, and
// puts it back when the test ends: a read held to the shipped budget on a
// busy machine runs out of time and reports the status unavailable, which is
// the budget working, not what a test reading the status asks about. A test
// of the budget itself does not call it.
func stretchPromptBudget(t *testing.T) {
	t.Helper()
	promptReadBudget = slack(shippedPromptReadBudget)
	t.Cleanup(func() { promptReadBudget = shippedPromptReadBudget })
}
