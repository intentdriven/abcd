package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/statusblock"
)

// A lane waiting for its full check (ruling DR6d-2) reads "waiting for its
// full check (since HH:MM)" in `implement status`, with how its receipt is
// minted, and on the status block's Now row.
func TestALaneWaitingForItsFullCheckIsShownWithItsSinceTime(t *testing.T) {
	since := time.Date(2026, 10, 2, 7, 18, 0, 0, time.UTC)
	l := loop.Lane{ID: "lane-2", SpecStep: 2, StepTitle: "Two", Stage: loop.StageLand,
		Landing: &loop.Landing{RecordsDone: true, CheckWaitSince: &since}}
	var b bytes.Buffer
	renderLaneLine(&b, l)
	if !strings.Contains(b.String(), "waiting for its full check (since 07:18): run the repository's preflight") {
		t.Fatalf("implement status names the wait and its since time:\n%s", b.String())
	}
	row := statusblock.Row{Lane: &statusblock.Lane{Run: "run-1", Lane: "lane-2", Stage: "land", Waiting: "waiting for its full check (since 07:18)"}}
	if got := statusRowPlace(row); got != "lane-2: land, waiting for its full check (since 07:18) (run-1)" {
		t.Fatalf("the Now row: %q", got)
	}
}
