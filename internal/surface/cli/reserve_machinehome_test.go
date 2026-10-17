package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReserveSaysWhenAHomeInsideAWorkingTreeIsNotRead: a HOME that lies
// inside a git working tree is not read as the machine's layer, so
// reservation.github_username there names no holder, the token does, and one
// stderr line says why.
func TestReserveSaysWhenAHomeInsideAWorkingTreeIsNotRead(t *testing.T) {
	w := newReserveWorld(t)
	pat := w.clone(t, "pat", "pat@example.com")
	fakeForge(t, "octo-person", 4242, nil)
	tree := t.TempDir()
	if err := os.Mkdir(filepath.Join(tree, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(tree, "home")
	if err := os.MkdirAll(filepath.Join(home, ".abcd.noindex"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".abcd.noindex", "config.json"), []byte(`{"reservation": {"github_username": "mallory"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	code, stdout, stderr := reserveRun(t, pat, "reserve", "--holder")
	if code != 0 || stdout != "would reserve as octo-person (from token)\n" {
		t.Fatalf("exit %d, stdout %q, stderr %q; want the token's holder", code, stdout, stderr)
	}
	if strings.Count(stderr, "\n") != 1 || !strings.HasPrefix(stderr, "abcd reserve: ") || !strings.Contains(stderr, "HOME lies inside the git working tree") {
		t.Fatalf("stderr %q; want one line saying HOME lies inside a working tree", stderr)
	}
}
