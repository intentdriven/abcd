package cli

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement"
)

// TestTheWaitingLineNamesUngatedTestRuns (iss-2610241130157455): when the load
// holds the gate shut, the waiting line names the go test runs outside the
// gate that add to it, by pid, so the session waiting can see what holds it;
// with none, the load reads as before.
func TestTheWaitingLineNamesUngatedTestRuns(t *testing.T) {
	res := implement.WaitResult{Still: []string{implement.WaitOnLoad}, Load: 33.7, WaitLoad: 20, Cores: 10,
		UngatedTests: []int{70000, 70100}}
	got := waitConditions(res)
	if !strings.Contains(got, "load 33.7 is not below 20") ||
		!strings.Contains(got, ", with 2 go test runs outside the gate adding to it (pids 70000, 70100)") {
		t.Fatalf("the waiting line = %q", got)
	}
	res.UngatedTests = []int{70000}
	if got = waitConditions(res); !strings.Contains(got, ", with a go test run outside the gate adding to it (pid 70000)") {
		t.Fatalf("one ungated run: %q", got)
	}
	res.UngatedTests = nil
	if got = waitConditions(res); strings.Contains(got, "outside the gate") {
		t.Fatalf("no ungated run is named when there is none: %q", got)
	}
}
