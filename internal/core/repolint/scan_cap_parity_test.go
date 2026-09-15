package repolint

import (
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
)

// TestBinaryScanCapMatchesPrivacyLint: the launch byte scan and this privacy
// lint bound a file read by the same cap. The constant cannot be shared (this
// package imports the scanner), so parity is asserted here instead.
func TestBinaryScanCapMatchesPrivacyLint(t *testing.T) {
	if got, want := scanner.MaxBinaryScanBytesForTest(), MaxScanBytesForTest(); got != want {
		t.Fatalf("scanner byte-scan cap %d != repolint privacy cap %d", got, want)
	}
}
