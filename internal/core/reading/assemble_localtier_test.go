package reading

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The default run directory lives in the checkout's local tier, and a checkout
// can carry any level of that tier as a committed symlink (a committed link beats
// .gitignore). A by-path MkdirAll followed such a link out of the checkout and
// wrote the assembled input and its manifest at the link's target — the sibling
// of iss-2609260948440803's memory lint write. Every level of a run directory
// named inside the repository is proved real before it is created.

// TestAssembleRefusesASymlinkedLocalTierAncestor plants the link at each level
// of the default run directory's chain and holds that the assembly refuses and
// writes nothing at the link's target.
func TestAssembleRefusesASymlinkedLocalTierAncestor(t *testing.T) {
	for _, level := range []string{".abcd/.work.local", ".abcd/.work.local/scratch", DefaultRunDir} {
		t.Run(level, func(t *testing.T) {
			root := fixtureRepo(t)
			outside := t.TempDir()
			link := filepath.Join(root, filepath.FromSlash(level))
			if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, link); err != nil {
				t.Fatal(err)
			}
			_, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD"})
			if err == nil {
				t.Fatalf("assemble with %s symlinked out of the checkout returned nil; it must refuse", level)
			}
			if !errors.Is(err, fsutil.ErrNotRealDir) {
				t.Errorf("assemble refused for the wrong reason: %v", err)
			}
			if entries, _ := os.ReadDir(outside); len(entries) != 0 {
				t.Errorf("assemble wrote %d entr(y|ies) at the link's target outside the checkout", len(entries))
			}
		})
	}
}

// TestAssembleRefusesAnOutSpelledIntoTheCheckoutThroughALink: an --out that is
// absolute, or that climbs out of the checkout and back in, is not therefore
// outside it. Either spelling of a directory below a committed link in the
// local tier is refused, and nothing lands at the link's target
// (iss-2609262231500173). A plain absolute directory outside every checkout is
// the operator's own and is still written.
func TestAssembleRefusesAnOutSpelledIntoTheCheckoutThroughALink(t *testing.T) {
	root := fixtureRepo(t)
	outside := t.TempDir()
	link := filepath.Join(root, ".abcd", ".work.local", "scratch")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	for _, sp := range []struct{ name, out string }{
		{"absolute", filepath.Join(link, "run")},
		{"climbing-in", filepath.Join("..", filepath.Base(root), ".abcd", ".work.local", "scratch", "run")},
	} {
		out := sp.out
		t.Run(sp.name, func(t *testing.T) {
			_, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", OutDir: out})
			if err == nil {
				t.Fatalf("assemble --out %s reached through a symlink inside the checkout returned nil; it must refuse", out)
			}
			if !errors.Is(err, fsutil.ErrNotRealDir) {
				t.Errorf("assemble refused for the wrong reason: %v", err)
			}
			if entries, _ := os.ReadDir(outside); len(entries) != 0 {
				t.Errorf("assemble wrote %d entr(y|ies) at the link's target outside the checkout", len(entries))
			}
		})
	}
	t.Run("outside every checkout", func(t *testing.T) {
		out := filepath.Join(t.TempDir(), "nested", "run")
		if _, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", OutDir: out}); err != nil {
			t.Fatalf("an absolute --out outside every checkout was refused: %v", err)
		}
		if _, err := os.Stat(filepath.Join(out, ManifestFileName)); err != nil {
			t.Errorf("the manifest was not written: %v", err)
		}
	})
}
