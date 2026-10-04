//go:build unix

package history

import (
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// TestLocalTranscriptRootsBehindASymlinkedAbcdHomePullNothingIn: the
// declaration that moves a checkout's transcripts into its own tree is a
// home-scoped declaration like trusted-roots, and was read through a symlinked
// ~/.abcd.noindex the rules loader refuses (iss-2609281017573862). Behind the link it
// pulls nothing in, and says why; in a real ~/.abcd.noindex it is honoured.
func TestLocalTranscriptRootsBehindASymlinkedAbcdHomePullNothingIn(t *testing.T) {
	repo := t.TempDir()
	dotfiles := t.TempDir()
	declareLocal(t, dotfiles, repo, 0o600)
	home := t.TempDir()
	if err := os.Symlink(abcdhome.Path(dotfiles), abcdhome.Path(home)); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	ok, note := localDeclared(repo)
	if ok {
		t.Fatal("a local-transcript-roots declaration behind a symlinked ~/.abcd.noindex pulled the checkout in")
	}
	if !strings.Contains(note, LocalRootsDisplay) || !strings.Contains(note, "~/.abcd.noindex is a symlink") {
		t.Errorf("the ignored declaration must say it was refused for the link: %q", note)
	}

	t.Setenv("HOME", dotfiles)
	if ok, note := localDeclared(repo); !ok {
		t.Fatalf("the same declaration in a real ~/.abcd.noindex must be honoured; note %q", note)
	}
}
