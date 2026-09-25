package cli

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
)

// TestLaunchShipCarriesTheRequestAndTheReceipt is step 4's golden request
// block and receipt for `launch ship`: the emit step hands the composer's
// routing to the host, and the ingest step's result carries the route block.
func TestLaunchShipCarriesTheRequestAndTheReceipt(t *testing.T) {
	r := shipReadyRepo(t)
	t.Setenv("HOME", t.TempDir())
	t.Chdir(r.Root())

	stdout, stderr, err := runCLISplit(t, "launch", "ship", "--json")
	if err != nil || stderr != "" {
		t.Fatalf("emit: %v stderr %q", err, stderr)
	}
	var rr oracle.RequestRouting
	member(t, []byte(stdout), "routing", &rr)
	if rr != (oracle.RequestRouting{Agent: "release-changelog-composer", Tier: oracle.HostDecides, FanOut: 1, Source: "none", Origin: "none", Connection: "harness"}) {
		t.Fatalf("routing = %+v", rr)
	}
	text, _, err := runCLISplit(t, "launch", "ship", "--route", "release-changelog-composer=frontier")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "routing: release-changelog-composer at tier frontier") {
		t.Fatalf("the emit's human rendering carries no routing line:\n%s", text)
	}

	payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")
	stdout, stderr, err = runCLISplit(t, "launch", "ship", "--changelog-json", payload, "--json",
		"--route", "release-changelog-composer=frontier")
	if err != nil {
		t.Fatalf("ingest: %v\n%s", err, stderr)
	}
	var rc oracle.ReceiptRoute
	member(t, []byte(stdout), "route", &rc)
	if rc.Agent != "release-changelog-composer" || rc.TierAsked != oracle.Frontier || rc.ConnectionUsed != oracle.Harness ||
		rc.Override != "release-changelog-composer=frontier" || rc.FallbackReason == "" {
		t.Fatalf("route = %+v", rc)
	}
	if !strings.Contains(stderr, "serves tier frontier") {
		t.Fatalf("stderr %q", stderr)
	}

	if _, _, err := runCLISplit(t, "launch", "ship", "--route", "scribe=economy"); exitCodeOf(err) != 2 {
		t.Fatalf("a --route for an agent ship does not dispatch: err %v", err)
	}
}
