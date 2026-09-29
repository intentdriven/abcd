//go:build unix

package oracle

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
			if err := os.Symlink(dotfiles, filepath.Join(f.roots.Home, ".abcd")); err != nil {
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
