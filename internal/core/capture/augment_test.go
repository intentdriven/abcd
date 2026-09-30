package capture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/adapter/scanner/augmenttest"
)

func captureWithValue(t *testing.T, slug string) (CaptureResult, string) {
	t.Helper()
	repo := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	res, err := Capture(CaptureRequest{
		RepoRoot:    repo,
		Text:        "the config holds " + augmenttest.Value + " in prose",
		Severity:    SeverityMinor,
		Category:    "process",
		Source:      "user-observation",
		Slug:        slug,
		FoundDuring: "unit-test",
	})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(repo, res.Path))
	if err != nil {
		t.Fatal(err)
	}
	return res, string(data)
}

// TestCaptureReportsTheAugmentersFinding: a repository's opt-in augmenter
// reaches the issue ledger's redactor, so what it flags is masked on write
// and counted (iss-2608291814575788).
func TestCaptureReportsTheAugmentersFinding(t *testing.T) {
	augmenttest.Install(t, augmenttest.Fake())
	res, data := captureWithValue(t, "augmented-value")
	if strings.Contains(data, augmenttest.Value) {
		t.Fatalf("the augmented value reached the ledger:\n%s", data)
	}
	if res.Redacted == 0 {
		t.Error("the augmented finding was not counted")
	}
}

// TestCaptureRecordsTheAugmenterGap: a configured augmenter that is not
// installed does not stop the capture, which writes on the native scanner and
// says so in its receipt.
func TestCaptureRecordsTheAugmenterGap(t *testing.T) {
	augmenttest.Install(t, augmenttest.NotFound())
	res, _ := captureWithValue(t, "augmenter-gap")
	if !strings.Contains(res.Degraded, "fake augmenter not on PATH") {
		t.Fatalf("the receipt does not record the gap: %q", res.Degraded)
	}
}

// TestCaptureNotesAFailedAugmenterRun: a run that failed after the scanner was
// built is a degraded scan, and the receipt says so.
func TestCaptureNotesAFailedAugmenterRun(t *testing.T) {
	augmenttest.Install(t, &augmenttest.Func{F: func(string, string) ([]scanner.Finding, error) {
		return nil, errors.New("run failed")
	}})
	res, _ := captureWithValue(t, "augmenter-failed")
	if !strings.Contains(res.Degraded, "run failed") {
		t.Fatalf("the receipt does not record the failed run: %q", res.Degraded)
	}
}
