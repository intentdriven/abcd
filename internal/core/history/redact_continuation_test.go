package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCaptureRedactsHomeWithANameSuffix pins the transcript store to the three
// continuation shapes: the caller's home followed by ".zip", "-old" and
// "_snapshot", behind a URL host and as a plain path, must land with the name
// gone — neither verbatim nor refused. The home here is a temp directory, not
// a /Users or /home root, so the nested shape exercises the multi-segment
// waiver of the leading anchor rather than the /Users backstop
// (iss-2608292036125100).
func TestCaptureRedactsHomeWithANameSuffix(t *testing.T) {
	base := t.TempDir()
	user := "zzhomeuser42"
	home := filepath.Join(base, user)
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".abcd", "history", testRootSHA, "transcripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	transcript := "the archive is at https://ci.example.com" + home + ".zip for review\n" +
		"old copy under " + home + "-old/x here\n" +
		"snapshot under /Volumes/T7" + home + "_snapshot/x here\n"
	res, err := Capture(t.TempDir(), testRootSHA, []byte(transcript), CaptureMeta{SessionID: "sess-suffix", Kind: "native"})
	if err != nil {
		t.Fatalf("Capture refused: %v", err)
	}
	onDisk, err := os.ReadFile(res.Record.Path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(onDisk), user) {
		t.Errorf("the caller's name survived on disk:\n%s", onDisk)
	}
}
