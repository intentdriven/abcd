package release

import (
	"slices"
	"strings"
	"testing"

	refusalcode "github.com/intentdriven/abcd/internal/core/refusal"
	"github.com/intentdriven/abcd/internal/gittest"
)

// writeRefusalLedger commits nothing; it writes a ledger of the given codes.
func writeRefusalLedger(t *testing.T, r *gittest.Repo, codes ...refusalcode.Code) {
	t.Helper()
	l := refusalcode.Ledger{FormatVersion: refusalcode.LedgerFormatVersion}
	for _, c := range codes {
		l.Codes = append(l.Codes, refusalcode.LedgerEntry{Code: c})
	}
	data, err := refusalcode.EncodeLedger(l)
	if err != nil {
		t.Fatal(err)
	}
	r.Write(refusalcode.LedgerPath, string(data))
}

// TestEmitRefusesACutThatDropsAPublishedRefusalCode: a code the last release
// carried and HEAD's ledger lacks refuses the cut under refusal-codes, naming
// the code (adr-2610301638558260); a cut that keeps every code is not refused
// on that ground.
func TestEmitRefusesACutThatDropsAPublishedRefusalCode(t *testing.T) {
	r := gittest.NewRepo(t)
	writeSurface(t, r, liveSurface())
	r.Write("CHANGELOG.md", "# Changelog\n\n## [0.4.0] - 2026-07-01\n\n### Added\n\n- the base.\n")
	r.Write(shippedDir+"README.md", "# shipped\n")
	writeRefusalLedger(t, r, "capture/missing-remedy", "usage/unknown-flag")
	r.Commit("the released state")
	r.Git("tag", "v0.4.0")
	r.Write(shippedDir+"itd-73-derived-versioning.md",
		"---\nid: itd-73\nimpact: fix\n---\n\n# A fix\n\nfixed.\n")
	r.Commit("ship a fix")

	if kinds := refusalKinds(emit(t, r)); slices.Contains(kinds, string(RefusalRefusalCodes)) {
		t.Fatalf("a cut keeping every code was refused under %s: %v", RefusalRefusalCodes, kinds)
	}

	writeRefusalLedger(t, r, "usage/unknown-flag")
	r.Commit("drop a published code")
	cut := emit(t, r)
	var got *Refusal
	for i := range cut.Refusals {
		if cut.Refusals[i].Kind == RefusalRefusalCodes {
			got = &cut.Refusals[i]
		}
	}
	if got == nil || cut.Ready {
		t.Fatalf("the cut did not refuse under %s: %v", RefusalRefusalCodes, refusalKinds(cut))
	}
	if !strings.Contains(got.Reason, "capture/missing-remedy") {
		t.Fatalf("the refusal does not name the dropped code: %s", got.Reason)
	}
}
