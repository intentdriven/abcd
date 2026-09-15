package capture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCaptureRedactsHomePathBehindAURLHost pins the ledger to the URL case:
// the caller's home behind a URL host ("https://ci.example.com<HOME>/x") must
// not reach the committed record. The ledger has no store-side backstop, so
// the detector is the only thing between this text and the repository.
func TestCaptureRedactsHomePathBehindAURLHost(t *testing.T) {
	repo := t.TempDir()
	home := "/Users/zzhomeuser42" // abcd-audit:allow
	t.Setenv("HOME", home)

	res, err := Capture(CaptureRequest{
		RepoRoot:    repo,
		Text:        "the artifact is at https://ci.example.com" + home + "/build.log and it failed",
		Severity:    SeverityMinor,
		Category:    "process",
		Source:      "user-observation",
		Slug:        "home-path-behind-url-host",
		FoundDuring: "unit-test",
	})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(repo, res.Path))
	if err != nil {
		t.Fatalf("read committed issue: %v", err)
	}
	if strings.Contains(string(data), home) {
		t.Fatalf("committed issue carries the caller's home behind a URL host:\n%s", data)
	}
}
