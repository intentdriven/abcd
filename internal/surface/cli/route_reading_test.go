package cli

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
)

// TestReadingIngestCarriesTheReceipt is step 4's golden receipt for `reading
// ingest`: the agent is the cold-reading position the payload names, the
// receipt carries the model the payload's instrument reports (AC 5), and a
// --route for another position is refused before anything is written.
func TestReadingIngestCarriesTheReceipt(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	repo := readingRepo(t)
	t.Setenv("HOME", t.TempDir())
	t.Chdir(repo)
	runID, manifestHash, def := parkedRunForIngest(t, srcRoot, repo, "detection")
	outPath := detectionPayloadFile(t, runID, manifestHash, def.Regime, def)

	_, _, err := runCLISplit(t, "reading", "ingest", "--reading-json", outPath, "--route", "cold-reading-widening=economy")
	if exitCodeOf(err) != 2 || !strings.Contains(err.Error(), `does not dispatch "cold-reading-widening"`) {
		t.Fatalf("a --route for a position the payload does not name: err %v", err)
	}

	stdout, stderr, err := runCLISplit(t, "reading", "ingest", "--reading-json", outPath, "--json",
		"--route", "cold-reading-detection=economy")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	var rc oracle.ReceiptRoute
	member(t, []byte(stdout), "route", &rc)
	want := oracle.ReceiptRoute{Agent: "cold-reading-detection", TierAsked: oracle.Economy, ConnectionUsed: oracle.Harness,
		FallbackReason: rc.FallbackReason, Override: "cold-reading-detection=economy", SettingsSent: oracle.Settings{},
		ModelReported: "a-model"}
	if rc.FallbackReason == "" || !receiptEqual(rc, want) {
		t.Fatalf("route = %+v", rc)
	}
}
