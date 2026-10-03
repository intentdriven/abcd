//go:build unix

package oracle

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// TestConnectRefusesASymlinkedAbcdHome: the provider block (and, in the abcd
// home, the key beside it) is written into ~/.abcd, so through a ~/.abcd
// symlinked into a dotfiles checkout it would land in that repository — and
// the machine layer that reads it back refuses it there (iss-2609260958587561's
// shape, in the setup's other writer). Refused loudly, and nothing is left
// behind the link, in either key home the setup can write.
func TestConnectRefusesASymlinkedAbcdHome(t *testing.T) {
	for _, keyHome := range []string{KeyHomeNone, KeyHomeABCD} {
		t.Run(keyHome, func(t *testing.T) {
			p := newProvFake(t, 200, chat("local-model", "ok"))
			f := newFx(t)
			dotfiles := t.TempDir()
			if err := os.Symlink(dotfiles, abcdhome.Path(f.roots.Home)); err != nil {
				t.Fatal(err)
			}
			req := connectReq(f, p.base())
			req.Provider, req.Home = "desk", keyHome
			if keyHome == KeyHomeNone {
				req.Key = ""
			}
			_, err := Connect(context.Background(), req)
			if err == nil || !strings.Contains(err.Error(), "~/.abcd is a symlink") {
				t.Fatalf("err = %v, want a refusal naming the symlinked ~/.abcd", err)
			}
			if entries, _ := os.ReadDir(dotfiles); len(entries) != 0 {
				t.Fatalf("Connect left %d file(s) behind the link, first %q", len(entries), entries[0].Name())
			}
		})
	}
}

// TestProviderBlockIsNotWrittenThroughAnAbcdHomeSwappedForALink is
// iss-2609281310017733: ~/.abcd is a real directory when the provider block's
// writer judges it and a symlink into a dotfiles checkout by the time it
// writes. The lock and the file are reached through the descriptor of the
// directory that was judged, so the write is refused, names the link, and
// leaves the checkout as it was.
func TestProviderBlockIsNotWrittenThroughAnAbcdHomeSwappedForALink(t *testing.T) {
	home := t.TempDir()
	abcd := abcdhome.Path(home)
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
	err := writeProviderBlock(home, "desk", map[string]any{"base_url": "http://127.0.0.1:1"})
	if !swapped {
		t.Fatal("the writer never judged ~/.abcd, so the race was not staged")
	}
	if err == nil || !strings.Contains(err.Error(), "~/.abcd is a symlink") {
		t.Errorf("err = %v, want a refusal naming the symlinked ~/.abcd", err)
	}
	if entries, _ := os.ReadDir(dotfiles); len(entries) != 0 {
		t.Fatalf("the provider block went through the swapped link: the checkout holds %q", entries[0].Name())
	}
}
