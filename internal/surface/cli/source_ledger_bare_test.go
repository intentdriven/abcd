package cli

import (
	"strings"
	"testing"
)

// TestLibraryLedgerBareNamesList is iss-2611040712366155: a bare `library
// ledger`, with no input to append, on a real corpus is a refusal that points at
// --list, not one about an invalid record; it appends nothing, and the ledger
// still lists empty afterwards.
func TestLibraryLedgerBareNamesList(t *testing.T) {
	sourceCheckout(t)
	if code, out, errs := runLibrary(t, "init"); code != 0 {
		t.Fatalf("init: %d %s%s", code, out, errs)
	}
	for _, args := range [][]string{{"ledger"}, {"ledger", "--json"}, {"ledger", "--repo", "abcdef012345"}} {
		code, out, errs := runLibrary(t, args...)
		if code != 2 {
			t.Errorf("%v: exit %d, want 2\n%s%s", args, code, out, errs)
		}
		if !strings.Contains(out+errs, "--list") {
			t.Errorf("%v: the refusal does not name --list\n%s%s", args, out, errs)
		}
		if strings.Contains(out+errs, "invalid ledger record") {
			t.Errorf("%v: the refusal reads as an invalid record\n%s%s", args, out, errs)
		}
	}
	code, out, errs := runLibrary(t, "ledger", "--list")
	if code != 0 || !strings.Contains(out, "0 lines") {
		t.Fatalf("list after the bare refusals: %d %s%s", code, out, errs)
	}
	// Some input given is still an append, and its refusal is the record's own.
	code, out, errs = runLibrary(t, "ledger", "--claim", "append-only")
	if code != 2 || strings.Contains(out+errs, "nothing to append") {
		t.Fatalf("partial append: %d %s%s", code, out, errs)
	}
}
