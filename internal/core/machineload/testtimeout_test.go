package machineload

import (
	"testing"
	"time"
)

// TestGoTestTimeoutScalesWithTheLoad: a test stage started on a loaded
// machine gives each package more time, in proportion to the one-minute load
// per online core, between one and TestTimeoutMaxFactor times the base and
// rounded up to a whole minute (iss-2610241911052191: three packages hit
// their timeout at load 17.8 on ten cores, with no test hung).
func TestGoTestTimeoutScalesWithTheLoad(t *testing.T) {
	base := 40 * time.Minute
	for _, c := range []struct {
		load    float64
		has     bool
		cores   int
		want    time.Duration
		comment string
	}{
		{2, true, 16, 40 * time.Minute, "an idle machine: the base"},
		{16, true, 16, 40 * time.Minute, "one runnable per core: the base"},
		{17.8, true, 10, 72 * time.Minute, "the issue's load: 1.78 times, rounded up to the minute"},
		{20, true, 10, 80 * time.Minute, "twice the cores: twice the base"},
		{300, true, 16, 80 * time.Minute, "never past the cap"},
		{0, false, 16, 40 * time.Minute, "an unreadable load: the base"},
	} {
		snap := Snapshot{Load1: c.load, HasLoad: c.has, Cores: c.cores}
		if got := GoTestTimeout(snap, base); got != c.want {
			t.Errorf("%s: GoTestTimeout(load %v on %d, %s) = %s, want %s", c.comment, c.load, c.cores, base, got, c.want)
		}
	}
	if TestTimeoutMaxFactor != 2 {
		t.Errorf("TestTimeoutMaxFactor = %d; land's run bound (internal/core/landing) holds two stages at this cap", TestTimeoutMaxFactor)
	}
}
