//go:build unix

package layered

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMachineLayerBehindASymlinkedAbcdHomeIsRefused: what the machine layer
// says decides which model a step reaches, and it was read through a symlinked
// ~/.abcd the rules loader refuses (iss-2609281017573862). The file itself is
// well-formed, owned and owner-only; the refusal is the link's, and loud. A
// symlinked ~/.abcd holding no such file is an absent layer.
func TestMachineLayerBehindASymlinkedAbcdHomeIsRefused(t *testing.T) {
	f := newFixture(t)
	dotfiles := t.TempDir()
	if err := os.Symlink(dotfiles, filepath.Join(f.roots.Home, ".abcd")); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(Config, f.roots); err != nil {
		t.Fatalf("a symlinked ~/.abcd holding no config.json must read as an absent layer: %v", err)
	}
	f.write(filepath.Join(dotfiles, filepath.FromSlash(Config.MachineRel)), `{"pace":{"work_minutes":5}}`)
	_, err := Load(Config, f.roots)
	if err == nil || !strings.Contains(err.Error(), "~/.abcd is a symlink") {
		t.Fatalf("err = %v, want a refusal naming the symlinked ~/.abcd", err)
	}
}
