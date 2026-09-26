package lint

import (
	"os"
	"path/filepath"
	"testing"
)

// A tree that is present but cannot be read is a fault, not an absent tree
// (the doctrine ScanSpecLinks, scanIssueLedger and scanRecordStores hold): the
// intent-tree scan and the spec-store probes part ENOENT from every other error
// rather than swallowing all of them as "no tree" (iss-2608261533419897). The
// legs sit behind the roots walk, which reports an unreadable directory first,
// so they are driven directly.
func unreadableDir(t *testing.T, dir string) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("root reads a mode-000 directory")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
}

func TestScanIntentTreeReportsAnUnreadableSubtree(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/intents/planned/itd-1-a.md", "---\nid: itd-1\n---\n")
	unreadableDir(t, filepath.Join(root, "rec", "intents", "archive"))
	if _, err := scanIntentTree(root, filepath.Join(root, "rec"), "intents"); err == nil {
		t.Fatal("an unreadable directory under the intent tree was swallowed as absent")
	}
}

func TestScanIntentTreeReportsAnUnstattableTree(t *testing.T) {
	root := t.TempDir()
	unreadableDir(t, filepath.Join(root, "rec", "sealed"))
	if _, err := scanIntentTree(root, filepath.Join(root, "rec"), filepath.Join("sealed", "intents")); err == nil {
		t.Fatal("an intent tree that cannot be stat'd was read as absent")
	}
	if _, err := scanIntentTree(root, filepath.Join(root, "rec"), "missing"); err != nil {
		t.Fatalf("an absent intent tree is soft, got %v", err)
	}
}

func TestSpecStoreProbesReportAnUnstattableStore(t *testing.T) {
	root := t.TempDir()
	unreadableDir(t, filepath.Join(root, "rec", "sealed"))
	cfg := RuleConfig{Enabled: true, Severity: "blocker", SpecsDir: filepath.Join("sealed", "specs"), IntentsDir: "intents"}
	if _, err := checkSpecLifecycle(root, filepath.Join(root, "rec"), cfg, Config{}); err == nil {
		t.Error("spec_lifecycle read a store that cannot be stat'd as absent")
	}
	if _, err := checkSpecIDUnique(root, filepath.Join(root, "rec"), cfg, Config{}); err == nil {
		t.Error("spec_id_unique read a store that cannot be stat'd as absent")
	}
}
