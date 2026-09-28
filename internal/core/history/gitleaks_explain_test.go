package history

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/gitleaks"
	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/tools"
)

// TestCaptureExplainsTheMissingGitleaks is itd-63 criterion 4 at the one
// missing-scanner path on main: the transcript store of a repository that armed
// gitleaks. The refusal stands (fail-closed, nothing stored), and it now says
// what gitleaks is, that this repository requires it, the exact install step,
// and the way back to the native scanner, rather than a bare error.
func TestCaptureExplainsTheMissingGitleaks(t *testing.T) {
	repoRoot, _ := setupStore(t)
	restore := scanGitleaks
	t.Cleanup(func() { scanGitleaks = restore })
	scanGitleaks = func(_, _, _ string) ([]scanner.Finding, error) {
		return nil, fmt.Errorf("%w: not on PATH and no path configured", gitleaks.ErrConfiguredNotFound)
	}
	_, err := Capture(repoRoot, testRootSHA, []byte("user: hi\n"), CaptureMeta{SessionID: "sess-explain", Kind: "native"})
	if err == nil {
		t.Fatal("capture did not fail closed")
	}
	if !errors.Is(err, gitleaks.ErrConfiguredNotFound) {
		t.Fatalf("the explanation lost the sentinel: %v", err)
	}
	var missing *tools.MissingError
	if !errors.As(err, &missing) || missing.Explanation.Capability != tools.TranscriptScanArmed {
		t.Fatalf("error carries no registry explanation: %v", err)
	}
	e := tools.Explain("gitleaks", tools.TranscriptScanArmed)
	for _, want := range []string{"gitleaks configured but not found", "required for", e.StepText(), "enabled to false"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal lacks %q:\n%s", want, err)
		}
	}
}

// TestCaptureDoesNotExplainARefusedPath: a binary that exists but is refused
// is not a missing tool, and gets no install offer.
func TestCaptureDoesNotExplainARefusedPath(t *testing.T) {
	repoRoot, _ := setupStore(t)
	restore := scanGitleaks
	t.Cleanup(func() { scanGitleaks = restore })
	scanGitleaks = func(_, _, _ string) ([]scanner.Finding, error) {
		return nil, gitleaks.ErrConfiguredPathRefused
	}
	_, err := Capture(repoRoot, testRootSHA, []byte("user: hi\n"), CaptureMeta{SessionID: "sess-refused", Kind: "native"})
	var missing *tools.MissingError
	if err == nil || errors.As(err, &missing) {
		t.Fatalf("a refused path was explained as a missing tool: %v", err)
	}
}
