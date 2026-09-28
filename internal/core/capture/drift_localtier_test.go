package capture

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The issue-drift receipt is written into the checkout's local tier, and a
// checkout can carry any level of that tier as a committed symlink. A by-path
// MkdirAll followed such a link out of the checkout and wrote the receipt at its
// target — the sibling of iss-2609260948440803's memory lint write. Every level
// is proved real before the receipt directory is created under it.

// TestIssueDriftRefusesASymlinkedLocalTierAncestor plants the link at each level
// of the receipt chain and holds that the check refuses and writes nothing at
// the link's target.
func TestIssueDriftRefusesASymlinkedLocalTierAncestor(t *testing.T) {
	for _, level := range []string{".abcd/.work.local", ".abcd/.work.local/logs", ".abcd/.work.local/logs/audit"} {
		t.Run(level, func(t *testing.T) {
			repo, ir := driftFixture(t)
			outside := t.TempDir()
			link := filepath.Join(repo, filepath.FromSlash(level))
			if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, link); err != nil {
				t.Fatal(err)
			}

			_, err := IssueDrift(IssueDriftRequest{RepoRoot: repo, IssuesRoot: ir, Now: time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)})
			if err == nil {
				t.Fatalf("issue drift with %s symlinked out of the checkout returned nil; it must refuse", level)
			}
			if !errors.Is(err, fsutil.ErrNotRealDir) {
				t.Errorf("issue drift refused for the wrong reason: %v", err)
			}
			if entries, _ := os.ReadDir(outside); len(entries) != 0 {
				t.Errorf("issue drift wrote %d entr(y|ies) at the link's target outside the checkout", len(entries))
			}
		})
	}
}

// TestIssueDriftWritesItsReceiptUnderARealNestedTier is the control: a real
// tier, partly present, still receives the receipt where ReceiptPath says, and a
// second run in the same instant gets a directory of its own.
func TestIssueDriftWritesItsReceiptUnderARealNestedTier(t *testing.T) {
	repo, ir := driftFixture(t)
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", ".work.local", "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	seen := map[string]bool{}
	for i := 0; i < 2; i++ {
		res, err := IssueDrift(IssueDriftRequest{RepoRoot: repo, IssuesRoot: ir, Now: now})
		if err != nil {
			t.Fatalf("issue drift under a real tier: %v", err)
		}
		if seen[res.ReceiptPath] {
			t.Fatalf("two runs in one instant share the receipt %s", res.ReceiptPath)
		}
		seen[res.ReceiptPath] = true
		if fi, err := os.Lstat(filepath.Join(repo, filepath.FromSlash(res.ReceiptPath))); err != nil || !fi.Mode().IsRegular() {
			t.Errorf("the receipt %s is not a regular file: %v", res.ReceiptPath, err)
		}
	}
}
