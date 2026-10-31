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
// reservation.github_username there names no holder, and one stderr line says
// why; nor is the holder id read or kept there (iss-2610290809526033), so the
// query exits 2 saying so and writes nothing under that home.
func TestReserveSaysWhenAHomeInsideAWorkingTreeIsNotRead(t *testing.T) {
	w := newReserveWorld(t)
	pat := w.clone(t, "pat", "pat@example.com")
	fakeForge(t, "octo-person", 4242, nil)
	plantHome(t)
	code, stdout, stderr := reserveRun(t, pat, "reserve", "--holder")
	lines := strings.Split(strings.TrimSuffix(stderr, "\n"), "\n")
	if code != 2 || stdout != "" || len(lines) != 2 || !strings.HasPrefix(lines[0], "abcd reserve: HOME lies inside the git working tree") ||
		!strings.Contains(lines[1], "holder-id is neither read nor written there") || strings.Contains(stderr, "someone-else") {
		t.Fatalf("exit %d, stdout %q, stderr %q; want the home's line and the holder id refused", code, stdout, stderr)
	}
	home, _ := os.UserHomeDir()
	if _, err := os.Lstat(abcdhome.Path(home, "holder-id")); !os.IsNotExist(err) {
		t.Fatalf("a holder id was written under the HOME inside a working tree: %v", err)
	}
}

// TestReserveAndPeersSayWhenAHomeInsideAWorkingTreeIsNotRead: the act and
// abcd peers, which reads the holder for the release holds, each put the one
// line on stderr, and only under such a home; the act then refuses, pushing
// nothing, since no holder id is read or kept there.
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
		want, notes := 0, 1
		if verb == "reserve" {
			want, notes = 2, 2 // the home's line, then the refusal naming the holder id
		}
		if code != want || strings.Count(stderr, "HOME lies inside the git working tree") != notes || !strings.HasPrefix(stderr, "abcd "+verb+": HOME lies inside") {
			t.Fatalf("%s under a HOME inside a working tree: exit %d, stdout %q, stderr %q; want exit %d and the line saying why", verb, code, stdout, stderr, want)
		}
	}
	if tip := w.remoteTip(t, "iss-2"); tip != "" {
		t.Fatalf("the refused act reached origin: %s", tip)
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
