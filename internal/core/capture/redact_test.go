package capture

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
)

// TestCaptureRedactsHomePathOnWrite is the regression for
// iss-2608231025198888: `abcd capture` wrote free text straight to the
// committed ledger with no detector between, so an absolute home path reached
// the repository with every lint gate green. The committed file must never
// carry the caller's own home root.
func TestCaptureRedactsHomePathOnWrite(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	res, err := testCapture(CaptureRequest{
		RepoRoot:    repo,
		Text:        "the PATH entry is " + filepath.Join(home, ".local", "bin", "abcd") + " and it moved",
		Severity:    SeverityMinor,
		Category:    "process",
		Source:      "user-observation",
		Slug:        "home-path-in-body",
		FoundDuring: "unit-test",
	})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(repo, res.Path))
	if err != nil {
		// Path is repo-relative on the result; fall back to an absolute join.
		t.Fatalf("read committed issue: %v", err)
	}
	if strings.Contains(string(data), home) {
		t.Fatalf("committed issue carries the caller's home root %q:\n%s", home, data)
	}
	if res.Redacted == 0 {
		t.Fatalf("Capture reported no redaction, want a non-zero count so the surface can say so")
	}
}

// TestCaptureLeavesCleanTextAlone guards the other direction: a body with
// nothing to redact must be written byte-for-byte and report zero, so the
// notice never fires spuriously.
func TestCaptureLeavesCleanTextAlone(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("HOME", t.TempDir())

	body := "the enumeration is four items long and misses the artefact class"
	res, err := testCapture(CaptureRequest{
		RepoRoot:    repo,
		Text:        body,
		Severity:    SeverityMinor,
		Category:    "process",
		Source:      "user-observation",
		Slug:        "nothing-to-redact",
		FoundDuring: "unit-test",
	})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if res.Redacted != 0 {
		t.Fatalf("Redacted = %d, want 0 for a clean body", res.Redacted)
	}
	data, err := os.ReadFile(filepath.Join(repo, res.Path))
	if err != nil {
		t.Fatalf("read committed issue: %v", err)
	}
	if !strings.Contains(string(data), body) {
		t.Fatalf("clean body was altered:\n%s", data)
	}
}

// TestCaptureWithHomePathInDerivedSlugStillFiles is the regression for the
// refusal the first redaction attempt introduced. A caller who passes a slug
// derived from the issue text — which is what the CLI does — put the home path
// into the slug too. Redacting the RENDERED record rewrote it to a bracketed
// placeholder, the kebab-case check rejected it, and the capture failed
// outright: a leak turned into a lost finding.
//
// The earlier tests missed this because they passed a hand-written slug with no
// path in it, so the derived-slug path was never exercised.
func TestCaptureWithHomePathInDerivedSlugStillFiles(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	body := "the PATH entry is " + filepath.Join(home, ".local", "bin", "abcd") + " and it moved"
	res, err := testCapture(CaptureRequest{
		RepoRoot: repo,
		Text:     body,
		// The CLI derives the slug from the text, so the path lands here too.
		Slug:        body,
		Severity:    SeverityMinor,
		Category:    "process",
		Source:      "user-observation",
		FoundDuring: "unit-test",
	})
	if err != nil {
		t.Fatalf("Capture refused a body containing a home path: %v", err)
	}
	if res.Redacted == 0 {
		t.Fatalf("Redacted = 0, want the home path counted")
	}
	if strings.Contains(res.Slug, "[") || strings.Contains(res.Slug, "]") {
		t.Fatalf("slug %q carries a redaction placeholder's brackets", res.Slug)
	}
	data, err := os.ReadFile(filepath.Join(repo, res.Path))
	if err != nil {
		t.Fatalf("read committed issue: %v", err)
	}
	if strings.Contains(string(data), home) {
		t.Fatalf("committed issue carries the caller's home root:\n%s", data)
	}
}

// TestResolveRedactsTheNote covers the sibling write path: a resolution note is
// free text landing in the same committed ledger.
func TestResolveRedactsTheNote(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	res, err := testCapture(CaptureRequest{
		RepoRoot: repo, Text: "a finding", Slug: "a-finding",
		Severity: SeverityMinor, Category: "process",
		Source: "user-observation", FoundDuring: "unit-test",
	})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	tr, err := Resolve(ResolveRequest{
		Grounds:    testGrounds,
		RepoRoot:   repo,
		ID:         res.ID,
		Resolution: "fixed by the script at " + filepath.Join(home, "bin", "fix.sh"),
		Impact:     "internal",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(repo, tr.Path))
	if err != nil {
		data, err = os.ReadFile(tr.Path)
		if err != nil {
			t.Fatalf("read resolved issue: %v", err)
		}
	}
	if strings.Contains(string(data), home) {
		t.Fatalf("resolved issue carries the caller's home root:\n%s", data)
	}
	if tr.Redacted == 0 {
		t.Fatalf("Redacted = 0, want the home path in the note counted")
	}
}

// TestTheTextRedactorIsTheLedgerRedactor holds the exported form a store
// outside the ledger builds once and calls for each text (the findings list,
// spc-2611252115253843): the same redactor, so a home path and a planted token
// come back rewritten; and a scanner that cannot be made, or one that runs
// degraded, is an error rather than text handed back as cleared, so the caller
// can withhold the text instead.
func TestTheTextRedactorIsTheLedgerRedactor(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	token := "ghp_" + strings.Repeat("A1", 18)
	in := "the token " + token + " sat in " + filepath.Join(home, "notes.txt")
	r, err := NewTextRedactor(context.Background(), repo)
	if err != nil {
		t.Fatalf("NewTextRedactor: %v", err)
	}
	if out := r.Redact(in); strings.Contains(out, token) || strings.Contains(out, home) {
		t.Fatalf("Redact left the token or the home path in place: %q", out)
	}

	degraded := t.TempDir()
	if err := os.MkdirAll(filepath.Join(degraded, ".abcd", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(degraded, ".abcd", "config", "pii.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewTextRedactor(context.Background(), degraded); !errors.Is(err, ErrScannerUnavailable) {
		t.Fatalf("NewTextRedactor over a broken pii.json = %v; want ErrScannerUnavailable", err)
	}

	orig := newContextScanner
	newContextScanner = func(context.Context, string) (*scanner.Scanner, error) { return nil, errors.New("no scanner") }
	t.Cleanup(func() { newContextScanner = orig })
	if _, err := NewTextRedactor(context.Background(), repo); !errors.Is(err, ErrScannerUnavailable) {
		t.Fatalf("NewTextRedactor with no scanner = %v; want ErrScannerUnavailable", err)
	}
}
