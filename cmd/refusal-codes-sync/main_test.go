package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/refusal"
)

// TestSyncWritesTheLedgerAndCheckNamesDrift: the sync writes the ledger into
// a fresh checkout, a second run finds nothing to do, and -check on a stale
// ledger reports it and writes nothing.
func TestSyncWritesTheLedgerAndCheckNamesDrift(t *testing.T) {
	root := t.TempDir()
	l := refusal.BuildLedger()

	if _, err := run(root, l, true); !errors.Is(err, errOutOfDate) {
		t.Fatalf("-check on a checkout with no ledger = %v, want out of date", err)
	}
	path := filepath.Join(root, filepath.FromSlash(refusal.LedgerPath))
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("-check wrote the ledger: %v", err)
	}
	wrote, err := run(root, l, false)
	if err != nil || !wrote {
		t.Fatalf("sync = %v, %v; want the ledger written", wrote, err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := refusal.EncodeLedger(l)
	if string(got) != string(want) {
		t.Fatalf("the written ledger differs from the registry's:\n%s", got)
	}
	if wrote, err := run(root, l, false); err != nil || wrote {
		t.Fatalf("a second sync = %v, %v; want nothing to do", wrote, err)
	}
	if _, err := run(root, l, true); err != nil {
		t.Fatalf("-check on a current ledger = %v", err)
	}

	stale := refusal.Ledger{FormatVersion: refusal.LedgerFormatVersion, Codes: l.Codes[1:]}
	if wrote, err := run(root, stale, true); !errors.Is(err, errOutOfDate) || wrote {
		t.Fatalf("-check against a different registry = %v, %v; want out of date and nothing written", wrote, err)
	}
	if again, _ := os.ReadFile(path); string(again) != string(want) {
		t.Fatal("-check changed the ledger")
	}
}
