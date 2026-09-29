package capture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// swapAbcdForSymlink replaces repo/.abcd with a symlink to target, moving the
// real directory aside — the move a local racer makes in the window between a
// ledger check and the write it licensed.
func swapAbcdForSymlink(t *testing.T, repo, target string) {
	t.Helper()
	if err := os.Rename(filepath.Join(repo, ".abcd"), filepath.Join(repo, ".abcd-moved")); err != nil {
		t.Fatalf("move .abcd aside: %v", err)
	}
	if err := os.Symlink(target, filepath.Join(repo, ".abcd")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
}

// walkFiles lists every regular file under dir, for asserting a write never
// landed there.
func walkFiles(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	_ = filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
		if err == nil && fi.Mode().IsRegular() {
			out = append(out, p)
		}
		return nil
	})
	return out
}

// TestLedgerMkdirCannotBeRedirectedOutsideTheCheckout is iss-2609012037143368's
// mkdir half. The directory walk checks each segment with Lstat and then creates
// the next; a racer who swaps an already-checked ancestor for a symlink in
// between used to have the next os.Mkdir create the ledger outside the checkout.
// Created through an os.Root on the checkout, the swap cannot escape it.
func TestLedgerMkdirCannotBeRedirectedOutsideTheCheckout(t *testing.T) {
	repo, ir := ledger(t)
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	swapped := false
	ledgerRaceHook = func(stage, rel string) {
		if stage == "mkdir" && !swapped && filepath.Base(rel) == "work" {
			swapped = true
			swapAbcdForSymlink(t, repo, outside)
		}
	}
	t.Cleanup(func() { ledgerRaceHook = nil })

	_, err := testCapture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "a finding",
		Severity: SeverityMinor, Category: "bug", Source: "manual-test", Slug: "raced", FoundDuring: "t"})
	if !swapped {
		t.Fatal("the race hook never fired at the .abcd/work mkdir")
	}
	if err == nil {
		t.Fatal("a capture whose ledger ancestor was swapped for an outside symlink succeeded")
	}
	// The refusal is the ledger's own path-unsafe sentinel, not merely an error:
	// mapEscape classifies os.Root's escape by its message, and this pins the
	// match against the Go release in use (iss-2609251823559111).
	if !errors.Is(err, ErrPathUnsafe) {
		t.Fatalf("the swapped-ancestor mkdir was refused with %v, want ErrPathUnsafe", err)
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Fatalf("the ledger walk created %d entr(ies) outside the checkout: %v", len(entries), entries)
	}
}

// TestLedgerRecordWriteCannotBeRedirectedOutsideTheCheckout is the write half:
// between the commit's last re-read of its placeholder and the write, a swapped
// ancestor used to carry the record out of the checkout.
func TestLedgerRecordWriteCannotBeRedirectedOutsideTheCheckout(t *testing.T) {
	repo, ir := ledger(t)
	if _, err := testCapture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "a first finding",
		Severity: SeverityMinor, Category: "bug", Source: "manual-test", Slug: "first", FoundDuring: "t"}); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outside, "work", "issues", "open"), 0o755); err != nil {
		t.Fatal(err)
	}
	swapped := false
	ledgerRaceHook = func(stage, rel string) {
		if stage == "write" && !swapped {
			swapped = true
			swapAbcdForSymlink(t, repo, outside)
		}
	}
	t.Cleanup(func() { ledgerRaceHook = nil })

	_, err := testCapture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "a second finding",
		Severity: SeverityMinor, Category: "bug", Source: "manual-test", Slug: "second", FoundDuring: "t"})
	if !swapped {
		t.Fatal("the race hook never fired before the record write")
	}
	if err == nil {
		t.Fatal("a capture whose ledger ancestor was swapped at the write succeeded")
	}
	if !errors.Is(err, ErrPathUnsafe) {
		t.Fatalf("the swapped-ancestor write was refused with %v, want ErrPathUnsafe", err)
	}
	for _, f := range walkFiles(t, outside) {
		if strings.HasSuffix(f, ".md") || strings.Contains(filepath.Base(f), "abcd-tmp") {
			t.Fatalf("the record write landed outside the checkout: %s", f)
		}
	}
}

// TestMapEscapeClassifiesTheRealOSRootEscape pins mapEscape's message match to
// the error os.Root actually returns on the Go release in use. The os package
// does not export its escape error, so the match is by text; a release that
// rewords it would still refuse the write but degrade ErrPathUnsafe to a
// generic error, and this test is what notices (iss-2609251823559111).
func TestMapEscapeClassifiesTheRealOSRootEscape(t *testing.T) {
	base := t.TempDir()
	root, err := os.OpenRoot(base)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	_, escErr := root.Open("../outside")
	if escErr == nil {
		t.Fatal("os.Root opened a path outside its root")
	}
	if got := mapEscape(escErr, filepath.Join(base, "..", "outside")); !errors.Is(got, ErrPathUnsafe) {
		t.Fatalf("mapEscape(%v) = %v, want ErrPathUnsafe", escErr, got)
	}
}
