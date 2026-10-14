package main

import (
	"bytes"
	"testing"

	"github.com/intentdriven/abcd/internal/core/machineload"
)

// TestPrintsTheDerivedParallelism: the tool prints the -p make preflight's test
// stages take, one number and a newline, so `go test -p "$p"` reads it whole.
func TestPrintsTheDerivedParallelism(t *testing.T) {
	var out bytes.Buffer
	run(&out, func(string) string { return "" }, func() (machineload.Snapshot, error) {
		return machineload.Snapshot{Load1: 20, HasLoad: true, Cores: 16}, nil
	})
	if got := out.String(); got != "12\n" {
		t.Fatalf("printed %q, want %q", got, "12\n")
	}
}
