package cli

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner/augmenttest"
)

// TestHistoryCaptureRendersTheScanGap: a repository whose configured scanner
// augmenter (gitleaks) is not installed still has the transcript stored, and
// the text render says what coverage is missing (iss-2608291814575788).
func TestHistoryCaptureRendersTheScanGap(t *testing.T) {
	historySourceRepo(t)
	augmenttest.Install(t, augmenttest.NotFound())

	out := string(runCLIStdin(t, "user: hi\n", "history", "capture", "--session", "sess-gap"))
	if !strings.Contains(out, "stored sess-gap") {
		t.Fatalf("the transcript was not stored:\n%s", out)
	}
	if !strings.Contains(out, "fake augmenter not on PATH") {
		t.Fatalf("the render does not name the gap:\n%s", out)
	}
}
