package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// versionControlledHome is the dotfiles-in-home fixture (iss-2609020219198779):
// a HOME that is itself a git working tree, carrying a .abcd whose guard.json
// switches the hazard registry off, and a plain directory beneath it that is
// not a repository of its own. It returns the home and that directory. The
// .abcd is the repository tier's spelling, which is also the home's old name:
// the walk looks for exactly that name, so it is the shape that would stop the
// walk at the home. The renamed home, ~/.abcd.noindex, is a name the walk never
// looks for (TestResolveRootNeverAdoptsTheRenamedHome).
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
// not a project. A .abcd at the home must not govern a session as the repo
// root — which is what adopting the home's git toplevel did, handing a plain
// directory beneath it the home's guard.json and config.json as though they
// were that directory's repository's own.
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

// homeSpellings are the spellings of one home directory the exclusion has to
// see through (iss-2609261753285273): HOME is the caller's string, the walk
// climbs the physical path git reports, and the two name the same directory
// without being the same bytes. Each returns the HOME to set and the working
// directory to resolve from, given the home as created and a plain directory
// beneath it; link is a symlink to the home beside it.
var homeSpellings = []struct {
	name  string
	spell func(t *testing.T, home, plain, link string) (homeEnv, cwd string)
}{
	{"trailing slash", func(_ *testing.T, home, plain, _ string) (string, string) {
		return home + string(filepath.Separator), plain
	}},
	{"symlinked HOME", func(_ *testing.T, _, plain, link string) (string, string) {
		return link, plain
	}},
	{"cwd through a symlinked HOME", func(t *testing.T, home, plain, link string) (string, string) {
		rel, err := filepath.Rel(home, plain)
		if err != nil {
			t.Fatal(err)
		}
		return link, filepath.Join(link, rel)
	}},
	{"case variant", func(t *testing.T, home, plain, _ string) (string, string) {
		variant := filepath.Join(filepath.Dir(home), strings.ToUpper(filepath.Base(home)))
		hi, herr := os.Stat(home)
		vi, verr := os.Stat(variant)
		if herr != nil || verr != nil || !os.SameFile(hi, vi) {
			t.Skipf("the test filesystem is case-sensitive: %q names no directory, so a case-variant HOME cannot be staged here", variant)
		}
		return variant, plain
	}},
}

// TestResolveRootNeverAdoptsTheHomeAtAnySpelling (iss-2609261753285273): the
// exclusion compares the home by file IDENTITY, not by spelling. A HOME with a
// trailing slash, reached through a symlink, or spelled as a case variant of
// the on-disk path on a case-insensitive volume names the same directory the
// walk arrives at, and a string comparison missed the last of those — the
// version-controlled home became the repo root and its .abcd was read a second
// time as the repo layer. Both sites are exercised: with a ~/.abcd the walk
// would stop at the home, and without one the toplevel IS the home.
func TestResolveRootNeverAdoptsTheHomeAtAnySpelling(t *testing.T) {
	for _, planted := range []bool{true, false} {
		site := "the walk passes over a .abcd at the home"
		if !planted {
			site = "a toplevel that is the home takes the non-repo route"
		}
		for _, shape := range homeSpellings {
			t.Run(site+"/"+shape.name, func(t *testing.T) {
				outer := mustDir(t, t.TempDir())
				home := filepath.Join(outer, "home")
				gitInitAt(t, home)
				if planted {
					plantConfiguration(t, home)
				}
				plain := mustDir(t, filepath.Join(home, "scratch", "notes"))
				if top, err := gitutil.Run(plain, "rev-parse", "--show-toplevel"); err != nil || resolvedPath(top) != resolvedPath(home) {
					t.Skipf("git does not name the home as the toplevel for the fixture (%q, %v)", top, err)
				}
				link := filepath.Join(outer, "link")
				if err := os.Symlink(home, link); err != nil {
					t.Fatal(err)
				}
				homeEnv, cwd := shape.spell(t, home, plain, link)
				t.Setenv("HOME", homeEnv)

				res := Resolve(cwd)
				if got := resolvedPath(res.Root); got == resolvedPath(home) {
					t.Fatalf("HOME=%q: Resolve(%q).Root = the home directory %q; the home must not be a repo root at any spelling", homeEnv, cwd, got)
				}
				if res.Root != cwd {
					t.Errorf("HOME=%q: Resolve(%q).Root = %q, want cwd with no walk (the non-repo route)", homeEnv, cwd, res.Root)
				}
				if len(res.Notes) != 0 {
					t.Errorf("declining the home as a repo root declines nothing the session should read; notes = %q", res.Notes)
				}
			})
		}
	}
}

// TestResolveRootNeverAdoptsTheRenamedHome: the home renamed to
// ~/.abcd.noindex (itd-2610030720038073) holds the user layer under a name the
// walk never looks for, so a version-controlled home carrying it, whatever its
// guard.json says, is still not a session's repo root: the walk takes the
// non-repo route and the home's registry governs nothing beneath it.
func TestResolveRootNeverAdoptsTheRenamedHome(t *testing.T) {
	outer := mustDir(t, t.TempDir())
	home := filepath.Join(outer, "home")
	gitInitAt(t, home)
	t.Setenv("HOME", home)
	renamed := mustDir(t, abcdhome.Path(home))
	if err := os.WriteFile(filepath.Join(renamed, "guard.json"),
		[]byte(`{"schema_version":1,"disabled":true,"entries":{}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plain := mustDir(t, filepath.Join(home, "scratch", "notes"))
	if top, err := gitutil.Run(plain, "rev-parse", "--show-toplevel"); err != nil || resolvedPath(top) != resolvedPath(home) {
		t.Skipf("git does not name the home as the toplevel for the fixture (%q, %v)", top, err)
	}

	res := Resolve(plain)
	if res.Root != plain {
		t.Fatalf("Resolve(%q).Root = %q, want cwd (the non-repo route): the renamed home is not a project", plain, res.Root)
	}
	reg, err := guard.Load(res.Root)
	if err != nil {
		t.Fatalf("guard.Load(%q): %v", res.Root, err)
	}
	if reg.Disabled {
		t.Errorf("the renamed home's guard.json governs a session beneath it at %q", res.Root)
	}
}
