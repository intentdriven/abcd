package loop

import (
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/assistedby"
)

// The commits the loop composes from record facts (the pick's record-only
// commit and the sync's merge commit) declare abcd as what composed them,
// with the running binary's version: ruling PC1, the third label beside a
// model's and None. These tests hold each composer to the exact trailer line.

// TestComposedAssistedByNamesTheRunningBinary: the composer's line is the
// shared label for the running binary's version (the grammar and its tie to the
// gate are internal/core/assistedby's).
func TestComposedAssistedByNamesTheRunningBinary(t *testing.T) {
	if got, want := composedAssistedBy(), "Assisted-by: "+assistedby.ComposedValue(core.Version); got != want {
		t.Errorf("composedAssistedBy() = %q, want %q", got, want)
	}
}

// TestThePickMessageDeclaresAbcd: the pick commit's message ends with the abcd
// label, never None, which would claim no tool touched text abcd computed.
func TestThePickMessageDeclaresAbcd(t *testing.T) {
	msg := pickMessage(State{RunID: "run-1", Intent: "itd-10"})
	lines := strings.Split(strings.TrimRight(msg, "\n"), "\n")
	if last := lines[len(lines)-1]; last != "Assisted-by: abcd:dev" {
		t.Fatalf("the pick message's trailer is %q, want %q:\n%s", last, "Assisted-by: abcd:dev", msg)
	}
	if strings.Contains(msg, "Assisted-by: None") {
		t.Fatalf("the pick message still declares None:\n%s", msg)
	}
}

// TestTheSyncMessageDeclaresAbcd: the sync's merge message ends with the abcd
// label, never None.
func TestTheSyncMessageDeclaresAbcd(t *testing.T) {
	lane := Lane{ID: "lane-2", Branch: "abcd/run-1/lane-2"}
	msg := syncMessage(lane, "run-1", "main", strings.Repeat("a", 40), []string{"lane-1"})
	lines := strings.Split(strings.TrimRight(msg, "\n"), "\n")
	if lines[0] != syncSubject(lane, "main", []string{"lane-1"}) {
		t.Fatalf("the sync message opens with its subject:\n%s", msg)
	}
	if last := lines[len(lines)-1]; last != "Assisted-by: abcd:dev" {
		t.Fatalf("the sync message's trailer is %q, want %q:\n%s", last, "Assisted-by: abcd:dev", msg)
	}
	if strings.Contains(msg, "Assisted-by: None") {
		t.Fatalf("the sync message still declares None:\n%s", msg)
	}
}

// TestAReceiptCannotClaimTheAbcdLabel: the abcd label says abcd composed the
// text from record facts, so a receipt reporting abcd as the model that wrote
// a lane's prose is refused as a gap, never copied into the records commit.
func TestAReceiptCannotClaimTheAbcdLabel(t *testing.T) {
	for _, m := range []string{"abcd:dev", "abcd:v0.12.0", "ABCD:v0.12.0", "Abcd:dev"} {
		got, gap := assistedByTrailers([]ReceiptRecord{{Role: RoleImplementer, Receipt: "r.json", Model: m}})
		if gap == "" || len(got) != 0 {
			t.Errorf("a receipt reporting %q is a gap, got %q", m, got)
		}
		if slices.ContainsFunc(got, func(s string) bool { return strings.Contains(strings.ToLower(s), "abcd:") }) {
			t.Errorf("the abcd label reached the records commit from a receipt: %q", got)
		}
	}
}
