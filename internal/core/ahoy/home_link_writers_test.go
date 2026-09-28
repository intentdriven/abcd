//go:build unix

package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
