package capture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/intent"
)

// TestRecordProbeReportsAnUnreadableBucketAsItself is iss-260: the probe that
// resolve's --intent/--spec and promote's --intent share treated every bucket
// read failure as an empty bucket, so an unreadable bucket — here a symlinked
// one — was answered "not found in the store", which lies about the cause. An
// absent bucket stays soft; an unreadable one is reported as what it is.
func TestRecordProbeReportsAnUnreadableBucketAsItself(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "a finding",
		Severity: SeverityMinor, Category: "bug", Source: "manual-test", Slug: "probe", FoundDuring: "t"})
	if err != nil {
		t.Fatal(err)
	}
	elsewhere := t.TempDir()
	if err := os.WriteFile(filepath.Join(elsewhere, "itd-7-x.md"), []byte("---\nid: itd-7\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	intents := filepath.Join(repo, intent.IntentsRelDir)
	if err := os.MkdirAll(intents, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(intents, intent.BucketDrafts)); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, err = Resolve(ResolveRequest{RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Resolution: "fixed",
		Impact: "fix", ByIntent: "itd-7"})
	if err == nil {
		t.Fatal("a resolve naming an intent behind a symlinked bucket succeeded")
	}
	if strings.Contains(err.Error(), "not found") {
		t.Fatalf("an unreadable bucket was reported as not found: %v", err)
	}
	if !strings.Contains(err.Error(), intent.BucketDrafts) {
		t.Fatalf("the refusal does not name the bucket it could not read: %v", err)
	}
}
