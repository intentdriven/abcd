package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
)

// TestReserveSaysWhenTheForkCheckWasNotMade: a reservation made while the
// forge could not say whether origin is a fork is reported with the check it
// skipped, in the text and in the JSON (spc-2610161044461448, step 10).
func TestReserveSaysWhenTheForkCheckWasNotMade(t *testing.T) {
	out := reserveView(claim.Result{Act: claim.ActReserved, Record: "iss-5", Holder: "octo", Copy: claim.CopyThis,
		Footprint: []string{"a.go"}, ForkUnchecked: "octo/abcd: gh did not answer in time"})
	var buf bytes.Buffer
	renderReserve(&buf, out)
	if !strings.Contains(buf.String(), "  "+claim.ForkUncheckedLine("octo/abcd: gh did not answer in time")+"\n") {
		t.Fatalf("the text names the skipped check:\n%s", buf.String())
	}
	doc, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(doc), `"fork_unchecked":"octo/abcd: gh did not answer in time"`) {
		t.Fatalf("the JSON names the skipped check: %s", doc)
	}

	buf.Reset()
	renderReserve(&buf, reserveView(claim.Result{Act: claim.ActReserved, Record: "iss-5", Holder: "octo"}))
	if strings.Contains(buf.String(), "fork") {
		t.Fatalf("a checked origin says nothing of forks:\n%s", buf.String())
	}
}

// TestDrainSaysWhenALanesForkCheckWasNotMade: a drain move whose lane's
// reservation could not ask the forge says so in its text as reserve does,
// and carries the reason in its JSON (spc-2610161044461448, step 10).
func TestDrainSaysWhenALanesForkCheckWasNotMade(t *testing.T) {
	start := &loop.StartResult{RunID: "run-1", ForkUnchecked: "octo/abcd: gh did not answer in time"}
	res := loop.DrainResult{State: loop.DrainStateRel, Start: start}
	var buf bytes.Buffer
	renderDrainRun(&buf, res)
	if !strings.Contains(buf.String(), "  "+claim.ForkUncheckedLine("octo/abcd: gh did not answer in time")+"\n") {
		t.Fatalf("the drain's text names the skipped check:\n%s", buf.String())
	}
	doc, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(doc), `"fork_unchecked":"octo/abcd: gh did not answer in time"`) {
		t.Fatalf("the drain's JSON names the skipped check: %s", doc)
	}

	buf.Reset()
	renderDrainRun(&buf, loop.DrainResult{State: loop.DrainStateRel, Start: &loop.StartResult{RunID: "run-1"}})
	if strings.Contains(buf.String(), "fork") {
		t.Fatalf("a checked origin says nothing of forks:\n%s", buf.String())
	}
}

// TestReserveAndDrainSayWhenTheForkCheckWasWaived (dcn-2610161223212218): a
// reservation let through on the fork the repository allows says so, in
// reserve's text and JSON and in a drain move's, in the one wording claim
// gives it.
func TestReserveAndDrainSayWhenTheForkCheckWasWaived(t *testing.T) {
	line := "  " + claim.ForkAllowedLine("octo/abcd") + "\n"
	out := reserveView(claim.Result{Act: claim.ActReserved, Record: "iss-5", Holder: "octo", Copy: claim.CopyThis,
		Footprint: []string{"a.go"}, ForkAllowed: "octo/abcd"})
	var buf bytes.Buffer
	renderReserve(&buf, out)
	if !strings.Contains(buf.String(), line) {
		t.Fatalf("reserve's text names the allowed fork:\n%s", buf.String())
	}
	if doc, err := json.Marshal(out); err != nil || !strings.Contains(string(doc), `"fork_allowed":"octo/abcd"`) {
		t.Fatalf("reserve's JSON names the allowed fork: %s %v", doc, err)
	}

	res := loop.DrainResult{State: loop.DrainStateRel, Start: &loop.StartResult{RunID: "run-1", ForkAllowed: "octo/abcd"}}
	buf.Reset()
	renderDrainRun(&buf, res)
	if !strings.Contains(buf.String(), line) {
		t.Fatalf("the drain's text names the allowed fork:\n%s", buf.String())
	}
	if doc, err := json.Marshal(res); err != nil || !strings.Contains(string(doc), `"fork_allowed":"octo/abcd"`) {
		t.Fatalf("the drain's JSON names the allowed fork: %s %v", doc, err)
	}
}
