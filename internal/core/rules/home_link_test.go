//go:build unix

package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTrustedRootsBehindASymlinkedAbcdHomeReAdmitNothing: the rules loader
// refuses a rules.json behind a symlinked ~/.abcd, and trusted-roots — which
// re-admits a root the loader would otherwise refuse — sat in the same home and
// was read through the link (iss-2609281017573862). A well-formed, owned,
// owner-only declaration naming the marker: the ONLY defect is the link. It
// re-admits nothing, and says why.
func TestTrustedRootsBehindASymlinkedAbcdHomeReAdmitNothing(t *testing.T) {
	marker := t.TempDir()
	dotfiles := t.TempDir()
	declareTrusted(t, dotfiles, marker)
	home := t.TempDir()
	if err := os.Symlink(filepath.Join(dotfiles, ".abcd"), filepath.Join(home, ".abcd")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	ok, note := trustedRootDeclared(marker)
	if ok {
		t.Fatal("a trusted-roots declaration behind a symlinked ~/.abcd re-admitted a root")
	}
	if !strings.Contains(note, TrustedRootsDisplay) || !strings.Contains(note, "~/.abcd is a symlink") {
		t.Errorf("the ignored declaration must say it was refused for the link: %q", note)
	}

	// The control: the same declaration in a real ~/.abcd is honoured.
	t.Setenv("HOME", dotfiles)
	if ok, note := trustedRootDeclared(marker); !ok {
		t.Fatalf("the same declaration in a real ~/.abcd must re-admit the root; note %q", note)
	}
}
