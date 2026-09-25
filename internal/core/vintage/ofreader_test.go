package vintage

import (
	"bytes"
	"os"
	"testing"
)

// TestOfReaderReadsABinaryWithoutRunningIt: a Go binary on disk yields its build
// metadata, and a go-test binary (no vcs stamping) is never Known; bytes that
// are not a Go binary are an error, never a vintage.
func TestOfReaderReadsABinaryWithoutRunningIt(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Skipf("no executable path: %v", err)
	}
	f, err := os.Open(exe)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cur, err := OfReader(f)
	if err != nil {
		t.Fatalf("OfReader on the test binary: %v", err)
	}
	if cur.Known {
		t.Errorf("a go-test binary carries no vcs stamp, so its vintage must not be Known: %+v", cur)
	}
	if _, err := OfReader(bytes.NewReader([]byte("#!/bin/sh\necho not a go binary\n"))); err == nil {
		t.Error("OfReader accepted bytes that are not a Go binary")
	}
}
