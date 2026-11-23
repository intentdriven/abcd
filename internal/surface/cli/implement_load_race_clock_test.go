package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
)

// TestImplementLoadRaceGateSaysItsClockIsPaused: with two slots, a race gate
// behind a race run it has not yet measured says on its waiting line that the
// timeout's clock is paused, and one that gives up at the ceiling says how
// much of its wait was paused (iss-2611232155470105).
func TestImplementLoadRaceGateSaysItsClockIsPaused(t *testing.T) {
	waitFixture(t, machine(3, 3, 3, liveRace))
	t.Setenv(implement.PreflightSlotsEnv, "2")
	code, out, errOut := implementCLI(t, "implement", "load", "--race-gate", "--timeout", "1m")
	if code != 1 {
		t.Fatalf("exit %d, want 1 at the ceiling\n%s%s", code, out, errOut)
	}
	for _, want := range []string{
		"waiting for capacity: a go test -race run is live (pid 70000); holding: pid 70000 for 1m0s, not yet measured; the race queue ahead is progressing, so the timeout's clock is paused: 0s of the 1m0s timeout counted, 0s paused, 6m0s at most in all\n",
		"LOAD GATE TIMED OUT after 6m0s (5m0s of it paused while the race queue ahead progressed): exit 1, nothing ran: a go test -race run is live (pid 70000)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output lacks %q:\n%s", want, out)
		}
	}
}

// TestImplementLoadRaceGateSaysItsClockRuns: a race gate shut by the load
// alone says its timeout's clock runs, and times out at the timeout.
func TestImplementLoadRaceGateSaysItsClockRuns(t *testing.T) {
	waitFixture(t, machine(40, 40, 40))
	t.Setenv(implement.PreflightSlotsEnv, "2")
	code, out, errOut := implementCLI(t, "implement", "load", "--race-gate", "--timeout", "1m")
	if code != 1 {
		t.Fatalf("exit %d, want 1\n%s%s", code, out, errOut)
	}
	for _, want := range []string{
		"; nothing ahead is progressing, so the timeout's clock runs: 0s of the 1m0s timeout counted\n",
		"LOAD GATE TIMED OUT after 1m0s: exit 1, nothing ran: load 40.0 is not below 32",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output lacks %q:\n%s", want, out)
		}
	}
}

// TestRaceClockTextAtTheCeiling: a race queue still progressing once the wait
// has reached its ceiling is said as such, with the clock running.
func TestRaceClockTextAtTheCeiling(t *testing.T) {
	res := implement.WaitResult{Gate: implement.GateRace, AtCeiling: true, WaitedS: 11 * 3600, PausedS: 10 * 3600}
	got := raceClockText(res, 2*time.Hour)
	want := "; the race queue ahead is progressing, but the wait has reached its 12h0m0s ceiling, so the timeout's clock runs: 1h0m0s of the 2h0m0s timeout counted, 10h0m0s paused before"
	if got != want {
		t.Fatalf("raceClockText = %q, want %q", got, want)
	}
}
