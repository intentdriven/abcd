package spec

import (
	"os"
	"path/filepath"
	"testing"
)

// The spec store's directory creator refused a symlinked LEAF and then called
// os.MkdirAll, which follows a symlinked ANCESTOR and creates the rest of the
// chain under its target — so a committed `.abcd/development -> <elsewhere>`
// redirected the mint and every later write out of the checkout
// (iss-2609012037137250, the sibling of GHSA-865x-5m7q-qm79). The intent store
// closed the same hole through fsutil.EnsureRealDirAll; these pin the spec side.

// plantAncestorLink makes root/.abcd/development a symlink to a fresh
// directory outside root and returns that directory.
func plantAncestorLink(t *testing.T, root string) string {
	t.Helper()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ".abcd", "development")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	return outside
}

// assertEmptyDir fails when dir holds anything: a refusal that still created a
// directory under the link's target has already written outside the checkout.
func assertEmptyDir(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("the refused write still created %v under the symlink's target %s", names, dir)
	}
}

func TestCreateRefusesASymlinkedAncestor(t *testing.T) {
	root := t.TempDir()
	outside := plantAncestorLink(t, root)
	if _, err := Create(root, "itd-9", "my-feature", ""); err == nil {
		t.Fatal("Create minted a spec through a symlinked .abcd/development")
	}
	assertEmptyDir(t, outside)
}

func TestCloseRefusesASymlinkedAncestor(t *testing.T) {
	// The store is minted for real in a directory that is later reached only
	// through the link, so Close meets an existing open/ spec behind a
	// symlinked ancestor and must not create closed/ under the target.
	staging := t.TempDir()
	sp, err := Create(staging, "itd-9", "my-feature", "")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(staging, ".abcd", "development")
	if err := os.Symlink(target, filepath.Join(root, ".abcd", "development")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	if _, err := Close(root, sp.ID); err == nil {
		t.Fatal("Close moved a spec through a symlinked .abcd/development")
	}
	if _, err := os.Lstat(filepath.Join(target, "specs", StatusClosed)); !os.IsNotExist(err) {
		t.Fatalf("the refused close still created specs/closed under the symlink's target (lstat err = %v)", err)
	}
}
