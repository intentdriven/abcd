package cli

// homeignore_test.go pins the front door's half of itd-2610140858048004: every
// command, before and after it runs, makes sure abcd's home ignores itself, and
// a lay that fails never fails the command.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"

	"github.com/intentdriven/abcd/internal/core/homeignore"
)

// TestFrontDoorLaysTheIgnoreFileInAnExistingHome proves a home from before the
// change is laid by the next command.
func TestFrontDoorLaysTheIgnoreFileInAnExistingHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := abcdhome.Path(home)
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, stderr, code := runFrontDoor("--version"); code != 0 {
		t.Fatalf("--version exit %d\n%s", code, stderr)
	}
	for _, name := range []string{homeignore.IgnoreName, homeignore.MarkerName} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s was not laid: %v", name, err)
		}
	}
}

// TestFrontDoorLaysAfterTheCommandThatCreatesTheHome proves the home a command
// creates is laid before the command returns: a private ideate verdict is the
// first write into a fresh home, and its ignore file stands when the run ends.
func TestFrontDoorLaysAfterTheCommandThatCreatesTheHome(t *testing.T) {
	repo := ideateRepo(t)
	t.Chdir(repo)
	dir := abcdhome.Path(os.Getenv("HOME"))
	if _, err := os.Lstat(dir); !os.IsNotExist(err) {
		t.Fatalf("the fixture home already holds abcd's home: %v", err)
	}
	_, stderr, code := runFrontDoor("ideate", "record", "a-first-private-idea", "--private",
		"--verdict-json", writeVerdict(t, ideateVerdictJSON("itd-104")))
	if code != 0 {
		t.Fatalf("ideate record exit %d\n%s", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, homeignore.IgnoreName)); err != nil {
		t.Errorf("the home the command created was not laid: %v", err)
	}
}

// TestFrontDoorSaysWhenTheLayFails proves a lay that cannot write never fails
// the command, and says so in one line on stderr.
func TestFrontDoorSaysWhenTheLayFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes through a read-only directory")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := abcdhome.Path(home)
	if err := os.Mkdir(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	_, stderr, code := runFrontDoor("--version")
	if code != 0 {
		t.Fatalf("a failed lay failed the command: exit %d\n%s", code, stderr)
	}
	if n := strings.Count(stderr, "was not laid"); n != 1 {
		t.Errorf("stderr names the failed lay %d times, want once:\n%s", n, stderr)
	}
	if strings.Contains(stderr, home) {
		t.Errorf("the line echoes the absolute home:\n%s", stderr)
	}
}

// TestFrontDoorLaysNothingWhileTheOldHomeStands proves the stop still writes
// nothing: with the old ~/.abcd standing, no ignore file is laid in the new home.
func TestFrontDoorLaysNothingWhileTheOldHomeStands(t *testing.T) {
	home, _, _ := stoppedHome(t, true)
	runFrontDoor("--version")
	if _, err := os.Lstat(abcdhome.Path(home, homeignore.IgnoreName)); !os.IsNotExist(err) {
		t.Errorf("an ignore file was laid while the old home stands: %v", err)
	}
}

// TestFrontDoorIsSilentForASymlinkedHome proves a dotfiles-symlinked abcd home,
// which no lay can ever serve, costs no stderr line on any command.
func TestFrontDoorIsSilentForASymlinkedHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.Symlink(t.TempDir(), abcdhome.Path(home)); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, stderr, code := runFrontDoor("--version")
	if code != 0 || strings.Contains(stderr, "was not laid") {
		t.Errorf("--version exit %d with stderr %q; want 0 and no lay line", code, stderr)
	}
}

// TestFrontDoorKeepsAFailedLayOffMachineStderr proves a failed lay adds nothing
// to stderr where a machine reads it: under --json, where nothing is written to
// stderr by design (iss-2609100519128005), and in a hook call, whose stderr the
// host hands the agent as the hook's own words.
func TestFrontDoorKeepsAFailedLayOffMachineStderr(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes through a read-only directory")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := abcdhome.Path(home)
	if err := os.Mkdir(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	for _, args := range [][]string{
		{"--json", "--version"},
		{"guard", "hook"},
		{"hook", "prompt-router"},
	} {
		_, stderr, _ := runFrontDoor(args...)
		if strings.Contains(stderr, "was not laid") {
			t.Errorf("%v: a failed lay reached machine-read stderr:\n%s", args, stderr)
		}
	}
}
