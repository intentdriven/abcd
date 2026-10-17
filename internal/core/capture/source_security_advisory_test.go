package capture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// source_security_advisory_test.go: `security-advisory` is the surfacing
// channel of a record minted from a private security advisory
// (spc-2610120557432880, step 2). The ledger's source enum names how a finding
// surfaced, and a private report is a channel the enum did not name.

// TestCaptureAcceptsSecurityAdvisorySource: a capture with the new source is
// written, and the record carries it.
func TestCaptureAcceptsSecurityAdvisorySource(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	res, err := Capture(CaptureRequest{
		RepoRoot:    repo,
		Text:        "a private report, held where only this machine reads it",
		Severity:    SeverityMajor,
		Category:    "security",
		Source:      "security-advisory",
		Slug:        "private-report",
		FoundDuring: "embargo intake",
		Remedy:      issueschema.MachineRemedy,
	})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(repo, res.Path))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `source: "security-advisory"`) {
		t.Fatalf("the record does not carry its source:\n%s", data)
	}
}

// TestCaptureStillRefusesAnUnknownSource: adding one value opens nothing else.
func TestCaptureStillRefusesAnUnknownSource(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	_, err := Capture(CaptureRequest{
		RepoRoot:    repo,
		Text:        "a finding",
		Severity:    SeverityMinor,
		Category:    "security",
		Source:      "security-advisories",
		Slug:        "a-finding",
		FoundDuring: "a test",
		Remedy:      "fix it",
	})
	var fv *FieldValueError
	if !errors.As(err, &fv) || fv.Field != "source" {
		t.Fatalf("err = %v, want the source refused", err)
	}
	if entries, _ := os.ReadDir(filepath.Join(repo, ".abcd", "work", "issues", "open")); len(entries) != 0 {
		t.Fatalf("a refused capture wrote %d file(s)", len(entries))
	}
}
