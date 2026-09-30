package capture

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/recordid"
)

// setMinter swaps the package mint seam for the test and restores it after.
func setMinter(t *testing.T, m recordid.Minter) {
	t.Helper()
	orig := minter
	minter = m
	t.Cleanup(func() { minter = orig })
}

var reNativeIssID = regexp.MustCompile(`^iss-[0-9]{16}$`)

// setSeqMinter installs a minter whose clock advances one second per mint (real
// entropy): successive mints in one test are strictly time-ordered, so tests
// that assert newest-first views stay deterministic without scripting entropy.
func setSeqMinter(t *testing.T) {
	t.Helper()
	var mu sync.Mutex
	base := time.Date(2026, 8, 20, 11, 0, 0, 0, time.UTC)
	n := 0
	setMinter(t, recordid.Minter{Now: func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		n++
		return base.Add(time.Duration(n) * time.Second)
	}})
}

// TestCaptureMintsTimestampID (spc-33): a capture mints a timestamp-numeric id
// — the injected clock's UTC stamp plus a 4-digit suffix — and never a
// sequential max+1 id.
func TestCaptureMintsTimestampID(t *testing.T) {
	repo, ir := ledger(t)
	setMinter(t, recordid.Minter{
		Now:     func() time.Time { return time.Date(2026, 8, 20, 11, 42, 7, 0, time.UTC) },
		Entropy: bytes.NewReader([]byte{0x03, 0x15}), // 789 -> "0789"
	})
	res, err := testCapture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "x", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "note",
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := "iss-2608201142070789"; res.ID != want {
		t.Fatalf("Capture minted %q, want %q", res.ID, want)
	}
	if _, status, err := findIssue(ir, res.ID); err != nil || status != StateOpen {
		t.Fatalf("minted issue not readable back from open/: status=%s err=%v", status, err)
	}
}

// TestCaptureSameInstantSameLedgerRedraws (spc-33 ruling 2): when a second mint
// in the same ledger draws the id an earlier mint already took — same second,
// same suffix — the tiebreak redraws a fresh suffix rather than bumping or
// failing. The shared entropy reader scripts the clash deterministically.
func TestCaptureSameInstantSameLedgerRedraws(t *testing.T) {
	repo, ir := ledger(t)
	instant := time.Date(2026, 8, 20, 11, 42, 7, 0, time.UTC)
	// One shared stream: draw1=0x002A (42), draw2=0x002A (clash -> redraw),
	// draw3=0x0007 (7).
	setMinter(t, recordid.Minter{
		Now:     func() time.Time { return instant },
		Entropy: bytes.NewReader([]byte{0x00, 0x2A, 0x00, 0x2A, 0x00, 0x07}),
	})
	first, err := testCapture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "a", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "one",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := testCapture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "two",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != "iss-2608201142070042" {
		t.Fatalf("first mint = %q, want iss-2608201142070042", first.ID)
	}
	if second.ID != "iss-2608201142070007" {
		t.Fatalf("second mint = %q, want the redrawn iss-2608201142070007 (never a bump of the taken id)", second.ID)
	}
}

// TestCaptureConcurrentSameInstantAllDistinct races many concurrent captures
// against one ledger with their clocks pinned to a single instant (go test
// -race covers the memory side): every mint must succeed and every id must be
// unique. With real entropy a same-suffix clash is rare, so this is a
// smoke/race check of the concurrent path — the redraw itself is pinned
// deterministically by TestCaptureSameInstantSameLedgerRedraws.
func TestCaptureConcurrentSameInstantAllDistinct(t *testing.T) {
	repo, ir := ledger(t)
	instant := time.Date(2026, 8, 20, 11, 42, 7, 0, time.UTC)
	// Real crypto entropy; only the clock is pinned.
	setMinter(t, recordid.Minter{Now: func() time.Time { return instant }})

	const n = 12
	var wg sync.WaitGroup
	ids := make([]string, n)
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res, err := testCapture(CaptureRequest{
				RepoRoot: repo, IssuesRoot: ir, Text: "x", Severity: SeverityMinor,
				Category: "bug", Source: "user-observation", FoundDuring: "t",
				Slug: fmt.Sprintf("race-%d", i),
			})
			ids[i], errs[i] = res.ID, err
		}(i)
	}
	wg.Wait()
	seen := map[string]int{}
	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("capture %d: %v", i, errs[i])
		}
		if !reNativeIssID.MatchString(ids[i]) {
			t.Fatalf("capture %d minted %q, want the 16-digit native shape", i, ids[i])
		}
		if prev, dup := seen[ids[i]]; dup {
			t.Fatalf("captures %d and %d minted the same id %q", prev, i, ids[i])
		}
		seen[ids[i]] = i
	}
}

// TestCaptureMintIgnoresLedgerMaximum (adr-45 ruling 2, the mechanism kill): a
// ledger already holding a HIGHER id than the mint would produce must not drag
// the new id upward — the mint never looks at any maximum. A max+1 allocator
// fails this by minting above the planted id.
func TestCaptureMintIgnoresLedgerMaximum(t *testing.T) {
	repo, ir := ledger(t)
	// Plant a committed issue whose numeric id is far above the injected clock's
	// stamp (year 2099).
	planted, err := testCapture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "planted", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "planted",
		ForceID: "iss-9912312359590000",
	})
	if err != nil {
		t.Fatal(err)
	}
	if planted.ID != "iss-9912312359590000" {
		t.Fatalf("ForceID mint = %q", planted.ID)
	}
	setMinter(t, recordid.Minter{
		Now:     func() time.Time { return time.Date(2026, 8, 20, 11, 42, 7, 0, time.UTC) },
		Entropy: bytes.NewReader([]byte{0x00, 0x2A}),
	})
	res, err := testCapture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "fresh", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "fresh",
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := "iss-2608201142070042"; res.ID != want {
		t.Fatalf("mint under a higher planted id = %q, want %q (the mint must not look at the maximum)", res.ID, want)
	}
}

// TestCaptureMintEntropyFailureIsLoud: a broken entropy source fails the
// capture with an error — never a fallback id, never a partial write left in
// the ledger.
func TestCaptureMintEntropyFailureIsLoud(t *testing.T) {
	repo, ir := ledger(t)
	setMinter(t, recordid.Minter{
		Now:     func() time.Time { return time.Date(2026, 8, 20, 11, 42, 7, 0, time.UTC) },
		Entropy: bytes.NewReader(nil), // immediate EOF
	})
	_, err := testCapture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "x", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "note",
	})
	if err == nil {
		t.Fatal("capture with failing entropy must error")
	}
	res, lerr := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateOpen})
	if lerr != nil {
		t.Fatal(lerr)
	}
	if len(res.Issues) != 0 || len(res.Skipped) != 0 {
		t.Fatalf("a failed mint must leave no ledger entry, found %d issues, %d skipped", len(res.Issues), len(res.Skipped))
	}
	// Skipped covers unparseable files, but a zero-byte placeholder is the exact
	// artefact a mis-ordered failure would leak — read open/ directly.
	entries, derr := os.ReadDir(filepath.Join(ir, "open"))
	if derr == nil && len(entries) != 0 {
		t.Fatalf("a failed mint left %d file(s) in open/", len(entries))
	}
}
