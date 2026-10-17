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
	plantHome(t)
	code, stdout, stderr := reserveRun(t, pat, "reserve", "--holder")
	if code != 0 || stdout != "would reserve as octo-person (from token)\n" {
		t.Fatalf("exit %d, stdout %q, stderr %q; want the token's holder", code, stdout, stderr)
	}
	if strings.Count(stderr, "\n") != 1 || !strings.HasPrefix(stderr, "abcd reserve: ") || !strings.Contains(stderr, "HOME lies inside the git working tree") {
		t.Fatalf("stderr %q; want one line saying HOME lies inside a working tree", stderr)
	}
}

// TestReserveAndPeersSayWhenAHomeInsideAWorkingTreeIsNotRead: the act and
// abcd peers, which reads the holder for the release holds, each put the one
// line on stderr, and only under such a home.
func TestReserveAndPeersSayWhenAHomeInsideAWorkingTreeIsNotRead(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	fakeForge(t, "octo-person", 4242, nil)
	for _, verb := range []string{"reserve", "peers"} {
		args := []string{verb}
		if verb == "reserve" {
			args = append(args, "iss-1", "--footprint", "a.go")
		}
		if code, _, stderr := reserveRun(t, p, args...); code != 0 || strings.Contains(stderr, "HOME lies inside") {
			t.Fatalf("%s under the world's own HOME: exit %d, stderr %q; want no line", verb, code, stderr)
		}
	}
	plantHome(t)
	for _, verb := range []string{"reserve", "peers"} {
		args := []string{verb}
		if verb == "reserve" {
			args = append(args, "iss-2", "--footprint", "b.go")
		}
		code, stdout, stderr := reserveRun(t, p, args...)
		if code != 0 || strings.Count(stderr, "HOME lies inside the git working tree") != 1 || !strings.Contains(stderr, "abcd "+verb+": HOME lies inside") {
			t.Fatalf("%s under a HOME inside a working tree: exit %d, stdout %q, stderr %q; want one line saying why", verb, code, stdout, stderr)
		}
	}
}

// plantHome points HOME at a directory inside a git working tree whose
// machine config.json names mallory as the reservation holder.
func plantHome(t *testing.T) {
	t.Helper()
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
}
