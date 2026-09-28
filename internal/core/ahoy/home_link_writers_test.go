//go:build unix

package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// TestMachineWritesRefuseASymlinkedAbcdHome: the model-tier routing table and
// the status-line setting are written into ~/.abcd, and each is read back
// through a guard that refuses a symlinked ~/.abcd — so a write through the
// link would land in a dotfiles repository AND be a file its own reader
// refuses. Each writer refuses loudly, names the link, and leaves nothing
// behind it (iss-2609281017573862, iss-2609260958587561's shape).
func TestMachineWritesRefuseASymlinkedAbcdHome(t *testing.T) {
	writers := map[string]func(a *applyCtx){
		"oracle routing": func(a *applyCtx) { a.writeMachineRouting([]byte("{}\n")) },
		"status line":    func(a *applyCtx) { a.wireStatusLine(harnessSettings{}, "/example/abcd", nil, "") },
	}
	for name, write := range writers {
		t.Run(name, func(t *testing.T) {
			home, _ := setupHermetic(t)
			dotfiles := t.TempDir()
			if err := os.RemoveAll(filepath.Join(home, ".abcd")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(dotfiles, filepath.Join(home, ".abcd")); err != nil {
				t.Fatal(err)
			}
			a := &applyCtx{}
			write(a)
			joined := strings.Join(a.notes, "\n")
			if !strings.Contains(joined, "~/.abcd is a symlink") {
				t.Errorf("the %s write must refuse naming the symlinked ~/.abcd; notes = %q", name, a.notes)
			}
			if entries, _ := os.ReadDir(dotfiles); len(entries) != 0 {
				t.Fatalf("the %s write left %d file(s) behind the link, first %q", name, len(entries), entries[0].Name())
			}
		})
	}
}

// TestMachineWritesRefuseAnAbcdHomeSwappedForALink is iss-2609281310017733:
// ~/.abcd is a real directory when each writer judges it and a symlink into a
// dotfiles checkout by the time it writes (a same-uid race, staged through
// the vetting hook). A check by path followed by a create by path writes
// through the link; each writer here reaches the file through the descriptor
// of the directory that was judged, so the checkout is left as it was.
func TestMachineWritesRefuseAnAbcdHomeSwappedForALink(t *testing.T) {
	writers := map[string]func(a *applyCtx){
		"path entry":       func(a *applyCtx) { _ = writePathEntry("/example/bin/abcd", strings.Repeat("0", 64), "") },
		"oracle routing":   func(a *applyCtx) { a.writeMachineRouting([]byte("{}\n")) },
		"history registry": func(a *applyCtx) { _, _ = bootstrapHistory() },
		"history lock":     func(a *applyCtx) { _ = withHistoryLock(func() error { return nil }) },
	}
	for name, write := range writers {
		t.Run(name, func(t *testing.T) {
			home, _ := setupHermetic(t)
			abcd := filepath.Join(home, ".abcd")
			dotfiles := filepath.Join(home, "dotfiles", "abcd")
			for _, dir := range []string{abcd, dotfiles} {
				if err := os.MkdirAll(dir, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			swapped := false
			t.Cleanup(fsutil.SwapHomeScopeVettedForTest(func(dir string) {
				if swapped || dir != abcd {
					return
				}
				swapped = true
				if err := os.Rename(abcd, filepath.Join(home, "moved-aside")); err != nil {
					t.Fatalf("swap: %v", err)
				}
				if err := os.Symlink(dotfiles, abcd); err != nil {
					t.Fatalf("swap: %v", err)
				}
			}))
			write(&applyCtx{})
			if !swapped {
				t.Fatalf("the %s writer never judged ~/.abcd, so the race was not staged", name)
			}
			if entries, _ := os.ReadDir(dotfiles); len(entries) != 0 {
				t.Fatalf("the %s write went through the swapped link: the checkout holds %q", name, entries[0].Name())
			}
		})
	}
}

// TestPathEntryRemovalRemovesNothingBehindAnAbcdHomeSwappedForALink is the
// remove half of iss-2609281310017733: ~/.abcd is a real directory when the
// provenance record's removal judges it and a symlink into a dotfiles checkout
// by the time it removes (staged through the vetting hook). A remove by path
// after a check by path unlinks the checkout's copy of path-entry; the remove
// through the descriptor of the directory that was judged leaves it alone.
func TestPathEntryRemovalRemovesNothingBehindAnAbcdHomeSwappedForALink(t *testing.T) {
	home, _ := setupHermetic(t)
	abcd := filepath.Join(home, ".abcd")
	dotfiles := filepath.Join(home, "dotfiles", "abcd")
	for _, dir := range []string{abcd, dotfiles} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for _, dir := range []string{abcd, dotfiles} {
		if err := os.WriteFile(filepath.Join(dir, "path-entry"), []byte("path=/example/bin/abcd\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	swapped := false
	t.Cleanup(fsutil.SwapHomeScopeVettedForTest(func(dir string) {
		if swapped || dir != abcd {
			return
		}
		swapped = true
		if err := os.Rename(abcd, filepath.Join(home, "moved-aside")); err != nil {
			t.Fatalf("swap: %v", err)
		}
		if err := os.Symlink(dotfiles, abcd); err != nil {
			t.Fatalf("swap: %v", err)
		}
	}))
	removePathEntry()
	if !swapped {
		t.Fatal("the removal never judged ~/.abcd, so the race was not staged")
	}
	if _, err := os.Lstat(filepath.Join(dotfiles, "path-entry")); err != nil {
		t.Fatalf("the removal went through the swapped link: the checkout's path-entry is gone (%v)", err)
	}
}
