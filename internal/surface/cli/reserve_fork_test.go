package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// TestReserveSaysWhenTheForkCheckWasNotMade: a reservation made while the
// forge could not say whether origin is a fork is reported with the check it
// skipped, in the text and in the JSON (spc-2610161044461448, step 10).
func TestReserveSaysWhenTheForkCheckWasNotMade(t *testing.T) {
	out := reserveView(claim.Result{Act: claim.ActReserved, Record: "iss-5", Holder: "octo", Copy: claim.CopyThis,
		Footprint: []string{"a.go"}, ForkUnchecked: "octo/abcd: gh did not answer in time"})
	var buf bytes.Buffer
	renderReserve(&buf, out)
	if !strings.Contains(buf.String(), "  not checked: whether origin is a fork (octo/abcd: gh did not answer in time)\n") {
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
