package layered

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/rules"
	"github.com/intentdriven/abcd/internal/gittest"
)

func rootsGitInit(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "-C", dir, "init", "--initial-branch=main")
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("git init unavailable: %v (%s)", err, out)
	}
}

func physical(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// TestRootsForResolvesTheRepoLayerTheWayTheRulesLoaderDoes is review-tier1 F2:
// every consumer of a layered file builds its Roots through one helper, and the
// repository layer is read from the root the rules loader resolves for the same
// working directory, so a session's rules, its guard and its layered
// configuration can never come from two different directories. A nested .abcd/
// inside the working tree (a monorepo member) is the case where git's toplevel
// and that root differ.
func TestRootsForResolvesTheRepoLayerTheWayTheRulesLoaderDoes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := physical(t, t.TempDir())
	rootsGitInit(t, repo)
	member := filepath.Join(repo, "pkg")
	sub := filepath.Join(member, "deep")
	if err := os.MkdirAll(filepath.Join(member, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, cwd := range []string{repo, member, sub} {
		r, notes := RootsFor(cwd)
		if want := rules.ResolveRoot(cwd); r.Repo != want {
			t.Errorf("RootsFor(%s).Repo = %q, want the rules root %q", cwd, r.Repo, want)
		}
		if r.Home != home {
			t.Errorf("RootsFor(%s).Home = %q, want %q", cwd, r.Home, home)
		}
		if len(notes) != 0 {
			t.Errorf("RootsFor(%s) notes = %q, want none for an owned checkout", cwd, notes)
		}
	}
	if r, _ := RootsFor(sub); r.Repo != member {
		t.Errorf("from inside the member, the repo layer is read at %q, want the member %q", r.Repo, member)
	}
}

// TestRootsForOutsideARepositoryIsTheWorkingDirectory: with no repository to
// bound at, the rules loader reads .abcd/ at the working directory and walks
// nowhere above it, and the layered files are read from the same place.
func TestRootsForOutsideARepositoryIsTheWorkingDirectory(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	plain := physical(t, t.TempDir())
	r, _ := RootsFor(plain)
	if r.Repo != plain {
		t.Fatalf("RootsFor(plain dir).Repo = %q, want the working directory %q", r.Repo, plain)
	}
}
