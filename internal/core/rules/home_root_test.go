package rules

import (
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// versionControlledHome is the dotfiles-in-home fixture (iss-2609020219198779):
// a HOME that is itself a git working tree, carrying a user-scope .abcd whose
// guard.json switches the hazard registry off, and a plain directory beneath it
// that is not a repository of its own. It returns the home and that directory.
func versionControlledHome(t *testing.T) (home, plain string) {
	t.Helper()
	outer := mustDir(t, t.TempDir())
	home = filepath.Join(outer, "home")
	gitInitAt(t, home)
	t.Setenv("HOME", home)
	plantConfiguration(t, home)
	plain = mustDir(t, filepath.Join(home, "scratch", "notes"))
	if top, err := gitutil.Run(plain, "rev-parse", "--show-toplevel"); err != nil || resolvedPath(top) != resolvedPath(home) {
		t.Skipf("git does not name the home as the toplevel for the fixture (%q, %v)", top, err)
	}
	return home, plain
}

// TestResolveRootNeverAdoptsTheHomeDirectory: a home under version control is
// not a project. The user-scope ~/.abcd is the USER layer (rules.json there is
// read as such whatever the root), and it must not govern a session a second
// time as the repo root — which is what adopting the home's git toplevel did,
// handing a plain directory beneath it the home's guard.json and config.json
// as though they were that directory's repository's own.
func TestResolveRootNeverAdoptsTheHomeDirectory(t *testing.T) {
	home, plain := versionControlledHome(t)

	res := Resolve(plain)
	if got := resolvedPath(res.Root); got == resolvedPath(home) {
		t.Fatalf("Resolve(%q).Root = the home directory %q; a version-controlled home must not be a session's repo root", plain, got)
	}
	if res.Root != plain {
		t.Errorf("Resolve(%q).Root = %q, want cwd with no walk (the non-repo route)", plain, res.Root)
	}
	if len(res.Notes) != 0 {
		t.Errorf("declining the home as a repo root declines nothing the session should read; notes = %q", res.Notes)
	}
	reg, err := guard.Load(res.Root)
	if err != nil {
		t.Fatalf("guard.Load(%q): %v", res.Root, err)
	}
	if reg.Disabled {
		t.Errorf("the home's guard.json governs a session beneath it: the hazard registry is switched off at %q", res.Root)
	}
}

// TestResolveRootNeverAdoptsTheHomeThroughTheMarker is the same bound on the
// fallback: with git unable to answer (off the PATH a hook runs under), the .git
// marker walk would otherwise find the home's own repository and adopt it.
func TestResolveRootNeverAdoptsTheHomeThroughTheMarker(t *testing.T) {
	home, plain := versionControlledHome(t)
	t.Setenv("PATH", "/nonexistent")
	if out, err := gitutil.Run(plain, "rev-parse", "--show-toplevel"); err == nil {
		t.Fatalf("git answered %q with an emptied PATH; the fixture did not stage the failure", out)
	}

	res := Resolve(plain)
	if got := resolvedPath(res.Root); got == resolvedPath(home) {
		t.Fatalf("Resolve(%q).Root = the home directory %q through the .git marker", plain, got)
	}
	if res.Root != plain {
		t.Errorf("Resolve(%q).Root = %q, want cwd with no walk", plain, res.Root)
	}
}

// TestResolveRootNeverAdoptsTheHomeBeneathAnotherToplevel: a repository whose
// toplevel CONTAINS the home (a hermetic harness that points HOME inside its
// checkout, or a whole-disk checkout) is the same shape one level up — the walk
// from a directory beneath the home passes through the home and would stop at
// ~/.abcd before it reached the toplevel. The home is skipped as a stop; the
// toplevel git named is still the root, because a repository git vouched for
// that CONTAINS the home is not the home, and its own .abcd stays its own.
func TestResolveRootNeverAdoptsTheHomeBeneathAnotherToplevel(t *testing.T) {
	outer := mustDir(t, t.TempDir())
	top := filepath.Join(outer, "checkout")
	gitInitAt(t, top)
	home := mustDir(t, filepath.Join(top, ".home"))
	t.Setenv("HOME", home)
	plantConfiguration(t, home)
	plain := mustDir(t, filepath.Join(home, "scratch"))

	res := Resolve(plain)
	if got := resolvedPath(res.Root); got == resolvedPath(home) {
		t.Fatalf("Resolve(%q).Root = the home directory %q; the walk must not stop at ~/.abcd", plain, got)
	}
	if got, want := resolvedPath(res.Root), resolvedPath(top); got != want {
		t.Errorf("Resolve(%q).Root = %q, want the toplevel that contains the home, %q", plain, got, want)
	}
	reg, err := guard.Load(res.Root)
	if err != nil {
		t.Fatalf("guard.Load(%q): %v", res.Root, err)
	}
	if reg.Disabled {
		t.Errorf("the home's guard.json governs a session beneath it: the hazard registry is switched off at %q", res.Root)
	}

	// The checkout's own .abcd, above the home, is still the checkout's.
	mustDir(t, filepath.Join(top, ".abcd"))
	if got, want := resolvedPath(Resolve(plain).Root), resolvedPath(top); got != want {
		t.Errorf("Resolve(%q).Root = %q, want the checkout's own .abcd at %q", plain, got, want)
	}
}

// TestResolveRootStillAdoptsARepositoryBeneathTheHome: the bound is on the
// home, not on everything under it. A real checkout inside a version-controlled
// home resolves its own toplevel and reads its own .abcd, exactly as before.
func TestResolveRootStillAdoptsARepositoryBeneathTheHome(t *testing.T) {
	home, _ := versionControlledHome(t)
	repo := filepath.Join(home, "src", "project")
	gitInitAt(t, repo)
	mustDir(t, filepath.Join(repo, ".abcd"))
	sub := mustDir(t, filepath.Join(repo, "internal"))

	if got, want := resolvedPath(Resolve(sub).Root), resolvedPath(repo); got != want {
		t.Errorf("Resolve(%q).Root = %q, want the checkout's own root %q", sub, got, want)
	}

	// A nearer .abcd inside the home's working tree, below the home, still
	// governs the directories beneath it: only the home itself is excluded.
	project := mustDir(t, filepath.Join(home, "notes-project"))
	mustDir(t, filepath.Join(project, ".abcd"))
	deep := mustDir(t, filepath.Join(project, "drafts"))
	if got, want := resolvedPath(Resolve(deep).Root), resolvedPath(project); got != want {
		t.Errorf("Resolve(%q).Root = %q, want the nearest .abcd below the home, %q", deep, got, want)
	}
}
