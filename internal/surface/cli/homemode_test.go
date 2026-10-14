package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// homemode_test.go — abcd keeps its home readable only by the account it
// belongs to: the front door narrows an existing home that its group or every
// account can open, and leaves a symlinked home, and every folder inside the
// home, as they stand.

func homeModeRun(t *testing.T) {
	t.Helper()
	var out, errOut bytes.Buffer
	if code := Run([]string{"--version"}, &out, &errOut); code != 0 {
		t.Fatalf("abcd --version: exit %d, stderr %q", code, errOut.String())
	}
}

// TestTheFrontDoorNarrowsAnOpenHome: a home made 0755 (an earlier abcd, a
// hand-run mkdir) is 0700 after any command runs, and the folders inside it
// keep their own modes.
func TestTheFrontDoorNarrowsAnOpenHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	inside := abcdhome.Path(home, "worktrees")
	for _, d := range []string{abcdhome.Path(home), inside} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	homeModeRun(t)
	if fi, err := os.Lstat(abcdhome.Path(home)); err != nil || fi.Mode().Perm() != 0o700 {
		t.Fatalf("the home after a command: %v %v; want 0700", fi.Mode(), err)
	}
	if fi, err := os.Lstat(inside); err != nil || fi.Mode().Perm() != 0o755 {
		t.Fatalf("a folder inside the home was re-moded: %v %v", fi.Mode(), err)
	}
}

// TestTheFrontDoorLeavesASymlinkedHome: a home that is a symlink is never
// followed, so the folder it points at keeps its mode; and no home is made
// where there was none.
func TestTheFrontDoorLeavesASymlinkedHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	target := filepath.Join(t.TempDir(), "dotfiles-abcd")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, abcdhome.Path(home)); err != nil {
		t.Fatal(err)
	}
	homeModeRun(t)
	if fi, err := os.Lstat(target); err != nil || fi.Mode().Perm() != 0o755 {
		t.Fatalf("the symlinked home's target was re-moded: %v %v", fi.Mode(), err)
	}

	bare := t.TempDir()
	t.Setenv("HOME", bare)
	homeModeRun(t)
	if _, err := os.Lstat(abcdhome.Path(bare)); !os.IsNotExist(err) {
		t.Fatalf("the front door made a home where there was none: %v", err)
	}
}
