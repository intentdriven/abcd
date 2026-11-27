package capture

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/adapter/scanner/augmenttest"
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
	if out, err := r.Redact(in); err != nil || strings.Contains(out, token) || strings.Contains(out, home) {
		t.Fatalf("Redact left the token or the home path in place: %q (%v)", out, err)
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

// TestTheTextRedactorReportsAScanThatDegrades holds the redactor to the
// scanner's contract that Unavailable is asked after ScanText as well as
// before: an augmenter whose run fails part way degrades the scanner during
// the scan, and the text it handed back is not cleared. An augmenter the
// repository configured whose tool is not installed is a scan that cannot run
// whole either.
func TestTheTextRedactorReportsAScanThatDegrades(t *testing.T) {
	repo := t.TempDir()
	a := &augmenttest.Func{F: func(string, string) ([]scanner.Finding, error) {
		return nil, errors.New("the run timed out")
	}}
	orig := newContextScanner
	t.Cleanup(func() { newContextScanner = orig })
	newContextScanner = func(ctx context.Context, root string) (*scanner.Scanner, error) {
		return scanner.NewContext(ctx, root, scanner.WithAugmenter(a))
	}
	r, err := NewTextRedactor(context.Background(), repo)
	if err != nil {
		t.Fatalf("NewTextRedactor before any scan: %v", err)
	}
	if out, err := r.Redact("a text the augmenter never cleared"); !errors.Is(err, ErrScannerUnavailable) {
		t.Fatalf("Redact under a scan that degraded = %q, %v; want ErrScannerUnavailable", out, err)
	}

	newContextScanner = func(ctx context.Context, root string) (*scanner.Scanner, error) {
		return scanner.NewContext(ctx, root, scanner.WithAugmenter(augmenttest.NotFound()))
	}
	if _, err := NewTextRedactor(context.Background(), repo); !errors.Is(err, ErrScannerUnavailable) {
		t.Fatalf("NewTextRedactor with a configured augmenter not installed = %v; want ErrScannerUnavailable", err)
	}
}

// TestTheTextRedactorReScansWhatItRedacted holds Redact to the scanner's
// two-stage contract: what survives redaction (here, an occurrence the
// augmenter did not report) is not handed back as cleared.
func TestTheTextRedactorReScansWhatItRedacted(t *testing.T) {
	repo := t.TempDir()
	firstOnly := &augmenttest.Func{F: func(text, file string) ([]scanner.Finding, error) {
		all := augmenttest.Locate(text, file, augmenttest.Value, augmenttest.Kind)
		if len(all) > 1 {
			all = all[:1]
		}
		return all, nil
	}}
	orig := newContextScanner
	t.Cleanup(func() { newContextScanner = orig })
	newContextScanner = func(ctx context.Context, root string) (*scanner.Scanner, error) {
		return scanner.NewContext(ctx, root, scanner.WithAugmenter(firstOnly))
	}
	r, err := NewTextRedactor(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	in := "once " + augmenttest.Value + " and twice " + augmenttest.Value
	if out, err := r.Redact(in); !errors.Is(err, ErrRedactionResidual) || errors.Is(err, ErrScannerUnavailable) {
		t.Fatalf("Redact left %q with %v; want this text refused for its residual, the scanner still good", out, err)
	}
}

// TestTheLedgerRedactorsReScanWhatTheyRedacted holds the issue ledger's two
// redactors to scanner.Redact's contract: the caller re-scans the result and
// fails closed if a span survived (an occurrence the augmenter did not
// report, here).
func TestTheLedgerRedactorsReScanWhatTheyRedacted(t *testing.T) {
	repo := t.TempDir()
	firstOnly := &augmenttest.Func{F: func(text, file string) ([]scanner.Finding, error) {
		all := augmenttest.Locate(text, file, augmenttest.Value, augmenttest.Kind)
		if len(all) > 1 {
			all = all[:1]
		}
		return all, nil
	}}
	orig := newLedgerScanner
	t.Cleanup(func() { newLedgerScanner = orig })
	newLedgerScanner = func(root string) (*scanner.Scanner, error) {
		return scanner.NewContext(context.Background(), root, scanner.WithAugmenter(firstOnly))
	}
	in := "once " + augmenttest.Value + " and twice " + augmenttest.Value
	if out, _, _, err := redactLedgerText(repo, in); strings.Contains(out, augmenttest.Value) || !errors.Is(err, ErrRedactionResidual) {
		t.Errorf("redactLedgerText handed back %q, %v; want the text refused, a span surviving", out, err)
	}
	r := newLedgerRedactor(repo)
	if out, _ := r.redact(in); strings.Contains(out, augmenttest.Value) || !errors.Is(r.Err(), ErrRedactionResidual) {
		t.Errorf("ledgerRedactor.redact handed back %q, %v; want the text refused, a span surviving", out, r.Err())
	}
}
