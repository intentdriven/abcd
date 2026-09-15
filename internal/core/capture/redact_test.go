package capture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

	res, err := Capture(CaptureRequest{
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
	res, err := Capture(CaptureRequest{
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
	res, err := Capture(CaptureRequest{
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

	res, err := Capture(CaptureRequest{
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
