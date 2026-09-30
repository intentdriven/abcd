package intent

import (
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/adapter/scanner/augmenttest"
)

// failingAugmenter installs a repository opt-in augmenter whose run fails, so
// the scanner degrades DURING the scan rather than before it
// (iss-2608291814575788): the write must refuse, never persist text redacted
// without the coverage the repository asked for.
func failingAugmenter(t *testing.T) {
	t.Helper()
	augmenttest.Install(t, &augmenttest.Func{F: func(string, string) ([]scanner.Finding, error) {
		return nil, errors.New("run failed")
	}})
}

func wantDegradedRefusal(t *testing.T, err error) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), "run failed") {
		t.Fatalf("err = %v, want a refusal naming the failed augmenter run", err)
	}
}

func TestIntentRedactionRefusesAFailedAugmenterRun(t *testing.T) {
	failingAugmenter(t)
	_, _, err := redactIntentText(t.TempDir(), "some intent text")
	wantDegradedRefusal(t, err)
}

func TestVerdictProseReportsAFailedAugmenterRun(t *testing.T) {
	failingAugmenter(t)
	free, degraded, err := newVerdictProse(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := degraded(); err != nil {
		t.Fatalf("degraded before any rendering: %v", err)
	}
	free("some verdict prose")
	wantDegradedRefusal(t, degraded())
}
