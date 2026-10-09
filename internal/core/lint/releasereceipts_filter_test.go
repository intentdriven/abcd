package lint_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// TestCheckReleaseReceiptsRunsNoRepoFilter is iss-2610090821548169: the
// receipts check lists what is uncommitted under the reviews directory, and
// over receipts whose index stat no longer matches git re-hashes each one
// through the repository's clean filter. The check must compare bytes without
// running it, still pass byte-identical committed receipts, and still refuse a
// real edit.
func TestCheckReleaseReceiptsRunsNoRepoFilter(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sentinel script")
	}
	r, roll := releaseFixture(t)
	dir := ".abcd/work/reviews/" + roll + "/"
	receipts := []string{dir + "docs-currency-reviewer.json", dir + "iss35-brief-surface-crosscheck.json"}
	r.Write(receipts[0], promote("docs-currency-reviewer", roll))
	r.Write(receipts[1], promote("iss35-brief-surface-crosscheck", roll))
	r.Commit("receipts")

	mark := filepath.Join(t.TempDir(), "filter-ran")
	script := filepath.Join(t.TempDir(), "evil-filter.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho ran >> "+mark+"\ncat\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("config", "filter.evil.clean", script)
	if err := os.WriteFile(filepath.Join(r.Root(), ".git", "info", "attributes"), []byte("* filter=evil\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stale := func(age time.Duration) {
		old := time.Now().Add(-age)
		for _, p := range receipts {
			if err := os.Chtimes(filepath.Join(r.Root(), filepath.FromSlash(p)), old, old); err != nil {
				t.Fatal(err)
			}
		}
	}
	stale(48 * time.Hour)

	// The fixture is live: a plain status under the same environment runs it.
	cmd := exec.Command("git", "-C", r.Root(), "status", "--porcelain", "--", ".abcd/work/reviews")
	cmd.Env = r.Env()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture status: %v: %s", err, out)
	}
	if _, err := os.Stat(mark); err != nil {
		t.Fatal("fixture: a plain git status did not run the clean filter, so this test proves nothing")
	}
	if err := os.Remove(mark); err != nil {
		t.Fatal(err)
	}
	stale(24 * time.Hour)

	check, err := lint.CheckReleaseReceipts(r.Root())
	if err != nil {
		t.Fatalf("CheckReleaseReceipts: %v", err)
	}
	if _, err := os.Stat(mark); err == nil {
		t.Fatal("the receipts check ran the repository's clean filter")
	}
	if !check.Pass || len(check.Uncommitted) != 0 {
		t.Fatalf("byte-identical committed receipts must pass, got %+v", check)
	}

	r.Write(receipts[0], promote("docs-currency-reviewer", roll)+"\n")
	check, err = lint.CheckReleaseReceipts(r.Root())
	if err != nil {
		t.Fatalf("CheckReleaseReceipts after an edit: %v", err)
	}
	if _, err := os.Stat(mark); err == nil {
		t.Fatal("the receipts check ran the repository's clean filter over an edited receipt")
	}
	if check.Pass || len(check.Uncommitted) != 1 {
		t.Fatalf("an edited receipt must still refuse as uncommitted, got %+v", check)
	}
}
