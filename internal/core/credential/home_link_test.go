//go:build unix

package credential

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// dotfilesHome returns a home whose ~/.abcd is a symlink to a directory in a
// "dotfiles checkout", and that directory.
func dotfilesHome(t *testing.T) (home, dotfiles string) {
	t.Helper()
	home, dotfiles = t.TempDir(), t.TempDir()
	if err := os.Symlink(dotfiles, filepath.Join(home, ".abcd")); err != nil {
		t.Fatal(err)
	}
	return home, dotfiles
}

// TestSetMachineRefusesASymlinkedAbcdHome is iss-2609260958587561: a secret
// written through a ~/.abcd symlinked into a dotfiles checkout lands in that
// repository. The write is refused loudly, names the link and the repair, and
// leaves nothing behind the link — not the store and not its lock.
func TestSetMachineRefusesASymlinkedAbcdHome(t *testing.T) {
	home, dotfiles := dotfilesHome(t)
	changed, err := SetMachine(home, "openrouter", "sk-example-0123456789")
	if err == nil || changed {
		t.Fatalf("SetMachine wrote through a symlinked ~/.abcd: changed %v, err %v", changed, err)
	}
	for _, want := range []string{"~/.abcd is a symlink", "real directory"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal must say %q: %v", want, err)
		}
	}
	if strings.Contains(err.Error(), "sk-example") {
		t.Fatalf("the refusal echoed the value: %v", err)
	}
	entries, rerr := os.ReadDir(dotfiles)
	if rerr != nil {
		t.Fatal(rerr)
	}
	if len(entries) != 0 {
		t.Fatalf("SetMachine left %d file(s) behind the link, first %q", len(entries), entries[0].Name())
	}
}

// A store that is there behind a symlinked ~/.abcd is refused on read too,
// loudly; a symlinked ~/.abcd holding no store reads as no store.
func TestResolveRefusesAStoreBehindASymlinkedAbcdHome(t *testing.T) {
	home, dotfiles := dotfilesHome(t)
	if _, err := Machine(home).Resolve("openrouter"); err != ErrNotSet {
		t.Fatalf("a symlinked ~/.abcd with no store must resolve to ErrNotSet, got %v", err)
	}
	if err := os.WriteFile(filepath.Join(dotfiles, StoreFileName), []byte(`{"openrouter":"sk-example-0123456789"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	v, err := Machine(home).Resolve("openrouter")
	if err == nil || err == ErrNotSet || v != "" {
		t.Fatalf("a store behind a symlinked ~/.abcd must be refused loudly: value %q, err %v", v, err)
	}
	if !strings.Contains(err.Error(), "~/.abcd is a symlink") {
		t.Errorf("the refusal must name the link: %v", err)
	}
}

// The second half of iss-2609260958587561: the store's lock was created 0644,
// and a read-only descriptor holds LOCK_EX, so any local user could stall every
// write for its five-second wait. The lock is the owner's alone, and a lock an
// earlier version created 0644 is tightened on the next write.
func TestTheStoreLockIsOwnerOnly(t *testing.T) {
	home := t.TempDir()
	if _, err := SetMachine(home, "openrouter", "sk-example-0123456789"); err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(home, ".abcd", storeLockFileName)
	assertOwnerOnly(t, lock)

	if err := os.Chmod(lock, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := SetMachine(home, "other", "sk-example-9876543210"); err != nil {
		t.Fatal(err)
	}
	assertOwnerOnly(t, lock)
}

func assertOwnerOnly(t *testing.T, p string) {
	t.Helper()
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0o077 != 0 {
		t.Fatalf("%s is mode %04o; the lock must be the owner's alone", filepath.Base(p), fi.Mode().Perm())
	}
}

// TestSetMachineWritesNothingThroughAnAbcdHomeSwappedForALink is
// iss-2609281310017733: ~/.abcd is a real directory when SetMachine judges it
// and a symlink into a dotfiles checkout by the time it writes. A check by path
// followed by a create by path lands the secret (and its lock) in the checkout;
// the write through the descriptor of the directory that was judged is refused,
// names the link, and leaves the checkout as it was.
func TestSetMachineWritesNothingThroughAnAbcdHomeSwappedForALink(t *testing.T) {
	home := t.TempDir()
	dotfiles := filepath.Join(home, "dotfiles", "abcd")
	for _, dir := range []string{filepath.Join(home, ".abcd"), dotfiles} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	swapped := false
	t.Cleanup(fsutil.SwapHomeScopeVettedForTest(func(dir string) {
		if swapped || dir != filepath.Join(home, ".abcd") {
			return
		}
		swapped = true
		if err := os.Rename(dir, filepath.Join(home, "moved-aside")); err != nil {
			t.Fatalf("swap: %v", err)
		}
		if err := os.Symlink(dotfiles, dir); err != nil {
			t.Fatalf("swap: %v", err)
		}
	}))
	changed, err := SetMachine(home, "openrouter", "sk-example-0123456789")
	if !swapped {
		t.Fatal("the vetting hook never ran, so the race was not staged")
	}
	if err == nil || changed || !strings.Contains(err.Error(), "~/.abcd is a symlink") {
		t.Errorf("SetMachine must refuse a ~/.abcd swapped for a link, naming it: changed %v, err %v", changed, err)
	}
	entries, rerr := os.ReadDir(dotfiles)
	if rerr != nil {
		t.Fatal(rerr)
	}
	if len(entries) != 0 {
		t.Fatalf("SetMachine wrote %d file(s) through the swapped link, first %q", len(entries), entries[0].Name())
	}
}

// TestResolveRefusesAStoreSwappedForAGroupReadableOneAfterItsCheck is the
// mode half of iss-2609281310017733: the store is 0600 when a check by path
// judges it and a group- and other-readable file by the time it is opened (a
// same-uid race, staged through the vetting hook of the ~/.abcd walk). A mode
// judged by path vouches for a file other than the one read; the mode judged
// on the opened file's own fstat refuses it, and the value is never returned.
func TestResolveRefusesAStoreSwappedForAGroupReadableOneAfterItsCheck(t *testing.T) {
	home := t.TempDir()
	store := writeStore(t, home, `{"openrouter": "sk-example-owner-only"}`, 0o600)
	exposed := filepath.Join(home, "exposed.json")
	if err := os.WriteFile(exposed, []byte(`{"openrouter": "`+secretValue+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(exposed, 0o644); err != nil {
		t.Fatal(err)
	}
	swapped := false
	t.Cleanup(fsutil.SwapHomeScopeVettedForTest(func(dir string) {
		if swapped || dir != filepath.Join(home, ".abcd") {
			return
		}
		swapped = true
		if err := os.Rename(exposed, store); err != nil {
			t.Fatalf("swap: %v", err)
		}
	}))
	v, err := Machine(home).Resolve("openrouter")
	if !swapped {
		t.Fatal("the vetting hook never ran, so the race was not staged")
	}
	if err == nil || errors.Is(err, ErrNotSet) || v != "" {
		t.Fatalf("a store swapped for a 0644 file after its check must be refused: value %q, err %v", v, err)
	}
	if strings.Contains(err.Error(), secretValue) || !strings.Contains(err.Error(), "chmod 0600") {
		t.Fatalf("the refusal must name the remedy and never the value: %v", err)
	}
}
