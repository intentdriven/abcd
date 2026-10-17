package homeignore

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"

	"github.com/intentdriven/abcd/internal/gittest"
)

// abcdHome lays an empty ~/.abcd.noindex under a fresh home and returns both.
func abcdHome(t *testing.T) (home, dir string) {
	t.Helper()
	home = t.TempDir()
	dir = abcdhome.Path(home)
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return home, dir
}

// TestLayIntoAnExistingHome proves the first lay writes the ignore file holding
// `*` and the marker beside it, both owner-only.
func TestLayIntoAnExistingHome(t *testing.T) {
	home, dir := abcdHome(t)
	got, err := Lay(home)
	if err != nil {
		t.Fatalf("Lay: %v", err)
	}
	if got != Laid {
		t.Errorf("outcome = %v, want Laid", got)
	}
	body, err := os.ReadFile(filepath.Join(dir, IgnoreName))
	if err != nil {
		t.Fatalf("the ignore file is not on disk: %v", err)
	}
	if !strings.Contains("\n"+string(body), "\n*\n") {
		t.Errorf("the ignore file has no `*` line:\n%s", body)
	}
	for _, name := range []string{IgnoreName, MarkerName} {
		fi, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if fi.Mode().Perm() != 0o600 {
			t.Errorf("%s mode = %o, want 600", name, fi.Mode().Perm())
		}
	}
}

// TestLayLeavesAMissingHomeAlone proves the lay never creates abcd's home: a
// home abcd has not made yet is left exactly as it is.
func TestLayLeavesAMissingHomeAlone(t *testing.T) {
	home := t.TempDir()
	got, err := Lay(home)
	if err != nil || got != NoHome {
		t.Fatalf("Lay = %v, %v; want NoHome, nil", got, err)
	}
	if _, err := os.Lstat(abcdhome.Path(home)); !os.IsNotExist(err) {
		t.Errorf("the lay created abcd's home: %v", err)
	}
}

// TestLayNeverLaysAgain proves a deleted ignore file stays deleted: deleting it
// is the person's deliberate opt-in, and the marker remembers the first lay.
func TestLayNeverLaysAgain(t *testing.T) {
	home, dir := abcdHome(t)
	if _, err := Lay(home); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, IgnoreName)); err != nil {
		t.Fatal(err)
	}
	got, err := Lay(home)
	if err != nil || got != AlreadyLaid {
		t.Fatalf("second Lay = %v, %v; want AlreadyLaid, nil", got, err)
	}
	if _, err := os.Lstat(filepath.Join(dir, IgnoreName)); !os.IsNotExist(err) {
		t.Errorf("the deleted ignore file was laid again: %v", err)
	}
}

// TestLayKeepsAnExistingIgnoreFile proves a .gitignore the person already keeps
// in abcd's home is never overwritten; the marker is written beside it.
func TestLayKeepsAnExistingIgnoreFile(t *testing.T) {
	home, dir := abcdHome(t)
	mine := "transcripts/\n"
	if err := os.WriteFile(filepath.Join(dir, IgnoreName), []byte(mine), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Lay(home)
	if err != nil || got != KeptExisting {
		t.Fatalf("Lay = %v, %v; want KeptExisting, nil", got, err)
	}
	body, err := os.ReadFile(filepath.Join(dir, IgnoreName))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != mine {
		t.Errorf("the person's ignore file was rewritten:\n%s", body)
	}
	if _, err := os.Stat(filepath.Join(dir, MarkerName)); err != nil {
		t.Errorf("no marker beside the kept file: %v", err)
	}
}

// TestLayLeavesASymlinkedHomeAlone proves the lay never writes through a link,
// and reads a symlinked abcd home as absent rather than as a fault: a
// dotfiles-symlinked ~/.abcd.noindex is a layout the configuration layer
// already reads as absent, and an error would repeat on every command, since no
// later run can lay it either.
func TestLayLeavesASymlinkedHomeAlone(t *testing.T) {
	home := t.TempDir()
	elsewhere := t.TempDir()
	if err := os.Symlink(elsewhere, abcdhome.Path(home)); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	got, err := Lay(home)
	if err != nil || got != NoHome {
		t.Fatalf("Lay = %v, %v; want NoHome, nil for a symlinked home", got, err)
	}
	entries, err := os.ReadDir(elsewhere)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		t.Errorf("the refused lay wrote %s at the link's target", e.Name())
	}
}

// TestLayReportsAFailedWrite proves a home the lay cannot write into returns an
// error rather than a false Laid, so the caller can say the file was not laid.
func TestLayReportsAFailedWrite(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes through a read-only directory")
	}
	home, dir := abcdHome(t)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	got, err := Lay(home)
	if err == nil {
		t.Fatalf("Lay into a read-only home = %v, nil; want an error", got)
	}
	if !errors.Is(err, os.ErrPermission) {
		t.Errorf("err = %v, want a permission error", err)
	}
}

// TestGitAddSkipsTheLaidHome is the intent's mechanism, run: in a home that is
// itself a git working tree, `git add -A` after the lay stages nothing that
// abcd keeps in its home, while the person's own files are staged as ever.
func TestGitAddSkipsTheLaidHome(t *testing.T) {
	home, dir := abcdHome(t)
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", home}, args...)...)
		cmd.Env = gittest.Env(t)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
		return string(out)
	}
	git("init", "-q")
	if err := os.MkdirAll(filepath.Join(dir, "transcripts"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "transcripts", "session.md"), []byte("private\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export X=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Lay(home); err != nil {
		t.Fatal(err)
	}
	git("add", "-A")
	staged := git("diff", "--cached", "--name-only")
	if strings.Contains(staged, abcdhome.Rel()) {
		t.Errorf("git add -A staged abcd's home:\n%s", staged)
	}
	if !strings.Contains(staged, ".zshrc") {
		t.Errorf("the person's own file was not staged:\n%s", staged)
	}
}
