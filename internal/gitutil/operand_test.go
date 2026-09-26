package gitutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

func mkdirs(t *testing.T, dirs ...string) {
	t.Helper()
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

// TestProveOperandDirRefusesALinkInsideACheckout: a symlinked ancestor inside
// the checkout the operand sits in is refused, named relative to the checkout,
// with no absolute path in the message.
func TestProveOperandDirRefusesALinkInsideACheckout(t *testing.T) {
	repo, elsewhere := t.TempDir(), t.TempDir()
	mkdirs(t, filepath.Join(repo, ".git"), filepath.Join(elsewhere, "out"))
	if err := os.Symlink(elsewhere, filepath.Join(repo, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	err := ProveOperandDir(filepath.Join(repo, "link", "out"))
	var oe *OperandError
	if !errors.As(err, &oe) || !errors.Is(err, fsutil.ErrNotRealDir) {
		t.Fatalf("ProveOperandDir = %v; want an *OperandError wrapping ErrNotRealDir", err)
	}
	if oe.Level != "link" {
		t.Errorf("refused level = %q, want link", oe.Level)
	}
	if strings.Contains(err.Error(), repo) || strings.Contains(err.Error(), elsewhere) {
		t.Errorf("the refusal carries an absolute path: %v", err)
	}
}

// TestProveOperandDirRefusesALinkIntoAnotherCheckout: a link whose target is a
// checkout of its own would make that target the nearest .git; the proof moves
// out to the enclosing checkout and refuses the link there.
func TestProveOperandDirRefusesALinkIntoAnotherCheckout(t *testing.T) {
	repo, other := t.TempDir(), t.TempDir()
	mkdirs(t, filepath.Join(repo, ".git"), filepath.Join(other, ".git"), filepath.Join(other, "out"))
	if err := os.Symlink(other, filepath.Join(repo, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := ProveOperandDir(filepath.Join(repo, "link", "out")); !errors.Is(err, fsutil.ErrNotRealDir) {
		t.Fatalf("ProveOperandDir through a link into another checkout = %v; want ErrNotRealDir", err)
	}
}

// TestProveOperandDirAcceptsPlainAndOutsidePaths is the control: real nested
// levels inside a checkout, an absent tail, and a link outside every checkout
// all pass.
func TestProveOperandDirAcceptsPlainAndOutsidePaths(t *testing.T) {
	repo := t.TempDir()
	mkdirs(t, filepath.Join(repo, ".git"), filepath.Join(repo, "a", "b"))
	for _, p := range []string{filepath.Join(repo, "a", "b"), filepath.Join(repo, "a", "b", "absent", "deeper"), repo} {
		if err := ProveOperandDir(p); err != nil {
			t.Errorf("ProveOperandDir(%s) = %v; want nil", p, err)
		}
	}
	plain, elsewhere := t.TempDir(), t.TempDir()
	if err := os.Symlink(elsewhere, filepath.Join(plain, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := ProveOperandDir(filepath.Join(plain, "link")); err != nil {
		t.Errorf("a link outside every checkout was refused: %v", err)
	}
}
