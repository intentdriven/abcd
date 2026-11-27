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

// TestLibraryLedgerFlipBelowOneNamesFlip is iss-2611271406524430: --flip given
// as 0, or as any value below 1, is a refusal naming --flip and the line
// numbers it takes, never the bare form's "nothing to append". The value typed
// is not echoed, and nothing is written. The refusal is a usage error, so it
// stands with no corpus at all (dcn-2611271408204240): exit 2, not the
// no-corpus exit 3, which a flip of a real line number still gets.
func TestLibraryLedgerFlipBelowOneNamesFlip(t *testing.T) {
	sourceCheckout(t)
	for _, args := range [][]string{{"ledger", "--flip", "0"}, {"ledger", "--flip", "-7"}} {
		code, out, errs := runLibrary(t, args...)
		if code != 2 {
			t.Errorf("no corpus, %v: exit %d, want 2\n%s%s", args, code, out, errs)
		}
		if !strings.Contains(out+errs, "--flip") {
			t.Errorf("no corpus, %v: the refusal does not name --flip\n%s%s", args, out, errs)
		}
		if strings.Contains(out+errs, "no sources corpus") {
			t.Errorf("no corpus, %v: a usage error reads as the absent corpus\n%s%s", args, out, errs)
		}
	}
	if code, out, errs := runLibrary(t, "ledger", "--flip", "1"); code != 3 || !strings.Contains(out+errs, "no sources corpus") {
		t.Errorf("no corpus, --flip 1: exit %d, want 3 naming the absent corpus\n%s%s", code, out, errs)
	}
	if code, out, errs := runLibrary(t, "init"); code != 0 {
		t.Fatalf("init: %d %s%s", code, out, errs)
	}
	for _, args := range [][]string{{"ledger", "--flip", "0"}, {"ledger", "--flip=0", "--json"}, {"ledger", "--flip", "-7"}} {
		code, out, errs := runLibrary(t, args...)
		if code != 2 {
			t.Errorf("%v: exit %d, want 2\n%s%s", args, code, out, errs)
		}
		if strings.Contains(out+errs, "nothing to append") {
			t.Errorf("%v: --flip given reads as the bare form\n%s%s", args, out, errs)
		}
		if !strings.Contains(out+errs, "--flip") || !strings.Contains(out+errs, "1 or more") {
			t.Errorf("%v: the refusal does not name --flip and its range\n%s%s", args, out, errs)
		}
		if strings.Contains(out+errs, "-7") {
			t.Errorf("%v: the refusal echoes the value typed\n%s%s", args, out, errs)
		}
	}
	code, out, errs := runLibrary(t, "ledger", "--list")
	if code != 0 || !strings.Contains(out, "0 lines") {
		t.Fatalf("list after the --flip refusals: %d %s%s", code, out, errs)
	}
}
