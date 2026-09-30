package history

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/gitleaks"
	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/adapter/scanner/augmenttest"
	"github.com/intentdriven/abcd/internal/core/tools"
)

// TestCaptureExplainsTheMissingGitleaks is itd-63 criterion 4 at the transcript
// store of a repository that armed gitleaks: the capture writes (the
// 2026-09-25 ruling on iss-2608291814575788 makes the missing binary a
// recorded gap, not a refusal), and the gap in its receipt says what gitleaks
// is, that this repository requires it, the exact install step, and the way
// back to the native scanner, rather than a bare error.
func TestCaptureExplainsTheMissingGitleaks(t *testing.T) {
	repoRoot, _ := setupStore(t)
	augmenttest.Install(t, &augmenttest.Func{
		Err: fmt.Errorf("%w: not on PATH and no path configured", gitleaks.ErrConfiguredNotFound),
	})
	res, err := Capture(repoRoot, testRootSHA, []byte("user: hi\n"), CaptureMeta{SessionID: "sess-explain", Kind: "native"})
	if err != nil {
		t.Fatalf("capture refused on the gap: %v", err)
	}
	e := tools.Explain("gitleaks", tools.TranscriptScanArmed)
	for _, want := range []string{"gitleaks configured but not found", "required for", e.StepText(), "enabled to false"} {
		if !strings.Contains(res.ScanGap, want) {
			t.Errorf("the recorded gap lacks %q:\n%s", want, res.ScanGap)
		}
	}
}

// TestCaptureDoesNotExplainARefusedPath: a binary that exists but is refused
// is not a missing tool, and gets no install offer.
func TestCaptureDoesNotExplainARefusedPath(t *testing.T) {
	repoRoot, _ := setupStore(t)
	setAugmenter(t, func(_, _ string) ([]scanner.Finding, error) {
		return nil, gitleaks.ErrConfiguredPathRefused
	})
	_, err := Capture(repoRoot, testRootSHA, []byte("user: hi\n"), CaptureMeta{SessionID: "sess-refused", Kind: "native"})
	var missing *tools.MissingError
	if err == nil || errors.As(err, &missing) {
		t.Fatalf("a refused path was explained as a missing tool: %v", err)
	}
}
