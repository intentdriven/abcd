package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeAppFile(t *testing.T, root, body string) {
	t.Helper()
	dir := filepath.Join(root, ".abcd", "relay")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, AppFile), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestReadAppClientID: a local session reads its GitHub App's public client
// id from .abcd/relay/github-app.json, which repository setup writes; no file
// is no client id, and a file that is not exactly that is refused.
func TestReadAppClientID(t *testing.T) {
	root := t.TempDir()
	if id, err := ReadAppClientID(root); id != "" || err != nil {
		t.Fatalf("no file: %q %v, want none and no error", id, err)
	}
	writeAppFile(t, root, `{"client_id": "Iv23liExample01", "note": "written by abcd"}`)
	if id, err := ReadAppClientID(root); id != "Iv23liExample01" || err != nil {
		t.Fatalf("a client id: %q %v", id, err)
	}
	for _, bad := range []string{
		`not json`,
		`{"client_id": 7}`,
		`{"client_id": ""}`,
		`{"client_id": "has space"}`,
		`{"client_id": "` + strings.Repeat("a", 101) + `"}`,
		`{"client_id": "Iv1.ok", "client_secret": "never here"}`,
	} {
		writeAppFile(t, root, bad)
		if id, err := ReadAppClientID(root); err == nil {
			t.Errorf("%s: %q accepted", bad, id)
		}
	}
	writeAppFile(t, root, `{"client_id": "`+strings.Repeat("a", 70<<10)+`"}`)
	if _, err := ReadAppClientID(root); err == nil {
		t.Error("an oversized file was read")
	}
	// A symlink is not followed out of the checkout.
	outside := filepath.Join(t.TempDir(), "app.json")
	if err := os.WriteFile(outside, []byte(`{"client_id": "Iv1.elsewhere"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, ".abcd", "relay", AppFile)
	_ = os.Remove(link)
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if id, err := ReadAppClientID(root); err == nil {
		t.Errorf("a symlink was followed: %q", id)
	}
}
