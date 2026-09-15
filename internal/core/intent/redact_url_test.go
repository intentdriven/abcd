package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCreateFromTextRedactsHomePathBehindAURLHost pins the intent record to
// the URL case: the caller's home behind a URL host must not reach the
// committed draft. Like the issue ledger, the intent store has no store-side
// backstop, so the detector is the only gate.
func TestCreateFromTextRedactsHomePathBehindAURLHost(t *testing.T) {
	root := t.TempDir()
	home := "/Users/zzhomeuser42" // abcd-audit:allow
	t.Setenv("HOME", home)

	it, err := CreateFromText(root, "I want the build log at https://ci.example.com"+home+"/build.log to be reviewable by the whole team", "", "")
	if err != nil {
		t.Fatalf("CreateFromText: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, it.Path))
	if err != nil {
		t.Fatalf("created file unreadable: %v", err)
	}
	if strings.Contains(string(data), home) {
		t.Fatalf("committed draft carries the caller's home behind a URL host:\n%s", data)
	}
}
