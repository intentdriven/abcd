package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
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
// machine config.json names someone-else as the reservation holder.
func plantHome(t *testing.T) {
	t.Helper()
	tree := t.TempDir()
	if err := os.Mkdir(filepath.Join(tree, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(tree, "home")
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abcdhome.Path(home, "config.json"), []byte(`{"reservation": {"github_username": "someone-else"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
}

// TestImplementLoadRefusesAHomeInsideAWorkingTree: at either gate the load
// gate exits 1 naming the flag given, says once that it keeps no hold there,
// and creates nothing under that home.
func TestImplementLoadRefusesAHomeInsideAWorkingTree(t *testing.T) {
	waitFixture(t, machine(3, 3, 3))
	plantHome(t)
	home, _ := os.UserHomeDir()
	if err := os.Remove(abcdhome.Path(home, "config.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(abcdhome.Path(home)); err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"--wait", "--race-gate"} {
		code, _, errOut := implementCLI(t, "implement", "load", flag, "--timeout", "1m")
		line := "abcd implement load " + flag + ": the home directory lies inside a working tree, so the load gate keeps no hold there; run with your own home"
		if code != 1 || strings.Count(errOut, line) != 1 {
			t.Fatalf("%s: exit %d, stderr %q; want exit 1 with %q once", flag, code, errOut, line)
		}
		if entries, _ := os.ReadDir(home); len(entries) != 0 {
			t.Fatalf("%s: created under the home: %v", flag, entries)
		}
	}
}
