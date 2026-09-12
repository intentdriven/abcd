package rules

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

func gitInitAt(t *testing.T, dir string) {
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

func realPath(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", p, err)
	}
	return r
}

// ResolveRoot is bounded at the git working tree (GHSA-vvqc-3mv2-5p49): the
// nearest .abcd directory INSIDE the tree wins, a tree with none resolves to
// its own toplevel, an ancestor's .abcd is never reached, and outside git there
// is no walk at all.
func TestResolveRootIsBoundedByTheWorkingTree(t *testing.T) {
	outer := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outer, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	inner := filepath.Join(outer, "inner-repo")
	gitInitAt(t, inner)
	sub := filepath.Join(inner, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	wantInner := realPath(t, inner)

	if got := realPath(t, ResolveRoot(sub)); got != wantInner {
		t.Errorf("ResolveRoot(%q) = %q, escaped the working tree %q", sub, got, wantInner)
	}
	if err := os.MkdirAll(filepath.Join(inner, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := realPath(t, ResolveRoot(sub)); got != wantInner {
		t.Errorf("ResolveRoot(%q) = %q, want the tree's own .abcd at %q", sub, got, wantInner)
	}
	mid := filepath.Join(inner, "a")
	if err := os.MkdirAll(filepath.Join(mid, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, want := realPath(t, ResolveRoot(sub)), realPath(t, mid); got != want {
		t.Errorf("ResolveRoot(%q) = %q, want the nearest .abcd inside the tree %q", sub, got, want)
	}

	plain := filepath.Join(outer, "plain")
	if err := os.MkdirAll(plain, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := ResolveRoot(plain); got != plain {
		t.Errorf("ResolveRoot(%q) = %q, want cwd (no walk outside git)", plain, got)
	}
}

// mustDir creates dir (and its parents) or fails the test.
func mustDir(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// refuseOwnership stages, deterministically, the state abcd's own isolated git
// environment cannot be rescued from: a real repository git will not answer
// for. GIT_TEST_ASSUME_DIFFERENT_OWNER makes git's ownership check fire, and
// the normal escape — `safe.directory` in the developer's global config — is
// discarded by the GIT_CONFIG_GLOBAL=/dev/null, GIT_CONFIG_NOSYSTEM=1 that
// gitutil.Run runs every command under. It is the same shape as a checkout
// owned by another uid, a container bind mount, or a corrupt .git.
func refuseOwnership(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("GIT_TEST_ASSUME_DIFFERENT_OWNER", "1")
	if _, err := gitutil.Run(dir, "rev-parse", "--show-toplevel"); err == nil {
		t.Skip("this git ignores GIT_TEST_ASSUME_DIFFERENT_OWNER; the refusal could not be staged")
	}
}

// TestResolveRootFallsBackToTheGitMarkerWhenGitRefuses is the second half of the
// GHSA-vvqc-3mv2-5p49 bound. "Not a repository" and "a repository git will not
// answer for" are different states, and only the first is safe to resolve as
// cwd-with-no-walk: in the second, content here IS version-controlled and the
// repo's own .abcd is the governing configuration. Collapsing the two dropped
// it silently — from a subdirectory the root became the subdirectory, so the
// repo's rules.json, its kill switch and its guard.json all stopped applying
// and nothing said so. The .git marker bounds the resolution when git will not.
func TestResolveRootFallsBackToTheGitMarkerWhenGitRefuses(t *testing.T) {
	outer := mustDir(t, t.TempDir())
	mustDir(t, filepath.Join(outer, ".abcd"))
	inner := filepath.Join(outer, "inner-repo")
	gitInitAt(t, inner)
	sub := mustDir(t, filepath.Join(inner, "internal", "deep"))
	wantInner := realPath(t, inner)

	refuseOwnership(t, sub)

	// No .abcd inside the tree: the marker root, never cwd and never the
	// ancestor's plant.
	if got := realPath(t, ResolveRoot(sub)); got != wantInner {
		t.Errorf("ResolveRoot(%q) with git refusing = %q, want the .git-marker root %q", sub, got, wantInner)
	}
	// The repo's own .abcd is what a subdirectory session must read.
	mustDir(t, filepath.Join(inner, ".abcd"))
	if got := realPath(t, ResolveRoot(sub)); got != wantInner {
		t.Errorf("ResolveRoot(%q) with git refusing = %q, want the repo's own .abcd at %q", sub, got, wantInner)
	}
	// Nearest-first still holds inside the tree.
	mid := mustDir(t, filepath.Join(inner, "internal"))
	mustDir(t, filepath.Join(mid, ".abcd"))
	if got, want := realPath(t, ResolveRoot(sub)), realPath(t, mid); got != want {
		t.Errorf("ResolveRoot(%q) with git refusing = %q, want the nearest .abcd inside the tree %q", sub, got, want)
	}
	// And the bound is still a bound: a genuinely non-repo directory does not
	// walk to the .abcd planted above it.
	plain := mustDir(t, filepath.Join(outer, "plain"))
	if got := ResolveRoot(plain); got != plain {
		t.Errorf("ResolveRoot(%q) = %q, want cwd (a non-repo directory must not walk)", plain, got)
	}
}

// TestResolveRootFallsBackToTheGitMarkerWhenGitIsAbsent is the same bound with
// the other unanswerable cause: git off the PATH the host hook runs under. A
// hook launched from a GUI session or a stripped container has no git at all,
// and every repository on that machine would otherwise resolve its rules and
// its guard from whatever directory the session happened to start in.
func TestResolveRootFallsBackToTheGitMarkerWhenGitIsAbsent(t *testing.T) {
	outer := mustDir(t, t.TempDir())
	mustDir(t, filepath.Join(outer, ".abcd"))
	inner := filepath.Join(outer, "inner-repo")
	gitInitAt(t, inner) // needs git, so it happens before the PATH is emptied
	mustDir(t, filepath.Join(inner, ".abcd"))
	sub := mustDir(t, filepath.Join(inner, "pkg"))
	wantInner := realPath(t, inner)

	t.Setenv("PATH", "/nonexistent")
	if out, err := gitutil.Run(sub, "rev-parse", "--show-toplevel"); err == nil {
		t.Fatalf("git answered %q with an emptied PATH; the fixture did not stage the failure", out)
	}

	if got := realPath(t, ResolveRoot(sub)); got != wantInner {
		t.Errorf("ResolveRoot(%q) with git absent = %q, want the .git-marker root %q", sub, got, wantInner)
	}
	plain := mustDir(t, filepath.Join(outer, "plain"))
	if got := ResolveRoot(plain); got != plain {
		t.Errorf("ResolveRoot(%q) with git absent = %q, want cwd (a non-repo directory must not walk)", plain, got)
	}
}

// plantMarker creates a `.git`-NAMED entry of the given shape at dir — the
// three shapes an unprivileged process can create anywhere it can write, none
// of which is a repository: an empty file (`: > .git`), an empty directory
// (`mkdir .git`), and a dangling symlink.
func plantMarker(t *testing.T, dir, shape string) {
	t.Helper()
	marker := filepath.Join(dir, ".git")
	switch shape {
	case "empty file":
		if err := os.WriteFile(marker, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	case "empty directory":
		mustDir(t, marker)
	case "dangling symlink":
		if err := os.Symlink(filepath.Join(dir, "nowhere"), marker); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatalf("unknown marker shape %q", shape)
	}
}

// TestResolveRootRejectsAnImplausibleGitMarker is the bound on the git-refused
// fallback itself. gitutil.RepoShapedRoot is a NAME check — any `.git`-named
// entry, walked to the filesystem root — so on its own it hands the resolution
// to whoever can write a name into a shared ancestor: `: > /tmp/.git` beside a
// planted /tmp/.abcd would govern the rules and the guard of every session
// whose cwd is a plain directory under /tmp, on any host where git cannot
// answer for that directory (which it cannot, because it is not a repository).
//
// The fallback therefore accepts a marker only when it is a PLAUSIBLE
// repository — a .git directory carrying HEAD, or a .git file whose content
// begins "gitdir: " — which is what git itself reads before it will call a
// directory a repository. Anything else is the non-repo case: cwd, no walk,
// nothing above it consulted. This does not make the fallback a trust boundary
// against a writer who takes the trouble to lay out a real repository
// (iss-2609020259564193); it stops the one-command plant, and it stops a stray
// leftover `.git` from silently governing a session.
func TestResolveRootRejectsAnImplausibleGitMarker(t *testing.T) {
	for _, shape := range []string{"empty file", "empty directory", "dangling symlink"} {
		t.Run(shape, func(t *testing.T) {
			outer := mustDir(t, t.TempDir())
			mustDir(t, filepath.Join(outer, ".abcd"))
			plantMarker(t, outer, shape)
			plain := mustDir(t, filepath.Join(outer, "work"))

			if got, err := gitutil.Run(plain, "rev-parse", "--show-toplevel"); err == nil {
				t.Fatalf("git answered %q for a planted marker; the fixture did not stage the git-refused path", got)
			}
			if got := ResolveRoot(plain); got != plain {
				t.Errorf("ResolveRoot(%q) with a %s .git planted above = %q; a plant that is not a repository must not govern the session", plain, shape, got)
			}
		})
	}
}

// TestResolveRootAcceptsAPlausibleGitMarker pins the other side: the plausible
// shapes a real checkout has must still bound the resolution when git will not
// answer, or the fix would close the plant by reopening GHSA-vvqc-3mv2-5p49.
// A .git directory carrying HEAD is the ordinary checkout; a .git FILE reading
// "gitdir: …" is a linked worktree or a submodule, whose working-tree root is
// exactly the directory that file sits in.
func TestResolveRootAcceptsAPlausibleGitMarker(t *testing.T) {
	t.Run("directory with HEAD", func(t *testing.T) {
		outer := mustDir(t, t.TempDir())
		mustDir(t, filepath.Join(outer, ".abcd"))
		repo := mustDir(t, filepath.Join(outer, "checkout"))
		mustDir(t, filepath.Join(repo, ".git"))
		if err := os.WriteFile(filepath.Join(repo, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		sub := mustDir(t, filepath.Join(repo, "internal", "deep"))

		if got, want := realPath(t, ResolveRoot(sub)), realPath(t, repo); got != want {
			t.Errorf("ResolveRoot(%q) = %q, want the checkout root %q", sub, got, want)
		}
	})
	t.Run("gitdir file", func(t *testing.T) {
		outer := mustDir(t, t.TempDir())
		mustDir(t, filepath.Join(outer, ".abcd"))
		wt := mustDir(t, filepath.Join(outer, "linked-worktree"))
		if err := os.WriteFile(filepath.Join(wt, ".git"), []byte("gitdir: "+filepath.Join(outer, "main", ".git", "worktrees", "wt")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		sub := mustDir(t, filepath.Join(wt, "pkg"))

		if got, want := realPath(t, ResolveRoot(sub)), realPath(t, wt); got != want {
			t.Errorf("ResolveRoot(%q) = %q, want the linked worktree's root %q", sub, got, want)
		}
	})
}

// --- ownership of the git-refused fallback root (iss-2609020259564193) -------
//
// The fallback accepts a marker root only when it LOOKS like a repository, which
// stops the one-command plant but not a real one: `git init` in a shared
// directory owned by another uid produces a genuine repository, and git's
// refusal on ownership under abcd's isolated environment is the SAME signal in
// that attack as in the case the fallback exists for. /Users/Shared (root-owned,
// mode 1777) and a root-owned /tmp are both real instances an unprivileged local
// user can lay a repository in.
//
// The policy: a marker root whose owner is not the caller is REFUSED by default,
// loudly, with an explicit home-scoped opt-in that re-admits it.

// resolvedPath is EvalSymlinks with a lexical fallback — the same normalisation
// Resolve applies, so a fixture path and the path the resolver hands the owner
// lookup compare equal on a host whose temp root is a symlink.
func resolvedPath(p string) string {
	if real, err := filepath.EvalSymlinks(p); err == nil {
		return real
	}
	return filepath.Clean(p)
}

// foreignUID is a uid this process is not. A detector for an ownership refusal
// needs a second uid, which a test process cannot create; the repository's
// established answer to a branch the host cannot provoke is to substitute the
// predicate (fsutil.caseFoldingFS, and the launch/lifeboat copies its comment
// names), and that is what ownedByAnother does.
func foreignUID() uint32 { return uint32(os.Getuid()) + 1 }

// ownedByAnother makes the owner lookup report a foreign uid for each named
// path, delegating every other path to the real one — so a single fixture can
// hold a foreign-owned root AND a caller-owned home, and the caller-owned cases
// in the same file keep running against the real filesystem.
func ownedByAnother(t *testing.T, paths ...string) {
	t.Helper()
	real := ownerUID
	foreign := make(map[string]bool, len(paths))
	for _, p := range paths {
		foreign[resolvedPath(p)] = true
	}
	ownerUID = func(path string) (uint32, error) {
		if foreign[resolvedPath(path)] {
			return foreignUID(), nil
		}
		return real(path)
	}
	t.Cleanup(func() { ownerUID = real })
}

// layPlausibleRepository lays the shape `git init` leaves and plausibleRepository
// accepts: a .git directory carrying HEAD. The detectors below do not shell out
// to git for it, because the point under test is the fallback that runs when git
// will not answer at all.
func layPlausibleRepository(t *testing.T, root string) {
	t.Helper()
	mustDir(t, filepath.Join(root, ".git"))
	if err := os.WriteFile(filepath.Join(root, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// plantConfiguration writes the two files the resolved root governs: the rules
// kill switch and a guard registry. Both carry "disabled": true, so whether they
// were READ is observable rather than inferred from the resolved path alone.
func plantConfiguration(t *testing.T, root string) {
	t.Helper()
	mustDir(t, filepath.Join(root, ".abcd"))
	if err := os.WriteFile(filepath.Join(root, ".abcd", "rules.json"),
		[]byte(`{"schema_version":1,"disabled":true,"domains":{}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".abcd", "guard.json"),
		[]byte(`{"schema_version":1,"disabled":true,"entries":{}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// declareTrusted writes the home-scoped opt-in naming each root.
func declareTrusted(t *testing.T, home string, roots ...string) string {
	t.Helper()
	path := filepath.Join(home, filepath.FromSlash(TrustedRootsRelPath))
	mustDir(t, filepath.Dir(path))
	body := "# roots this machine's owner has declared trustworthy\n"
	for _, r := range roots {
		body += r + "\n"
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// foreignPlant is the whole fixture: a real-shaped repository at plant, owned by
// another uid, carrying a rules kill switch and a guard kill switch, with the
// session's working directory a plain directory beneath it. home is an isolated
// user scope the detector controls.
func foreignPlant(t *testing.T) (plant, victim, home string) {
	t.Helper()
	outer := mustDir(t, t.TempDir())
	home = mustDir(t, filepath.Join(outer, "home"))
	t.Setenv("HOME", home)
	plant = mustDir(t, filepath.Join(outer, "plant"))
	layPlausibleRepository(t, plant)
	plantConfiguration(t, plant)
	victim = mustDir(t, filepath.Join(plant, "victim"))
	return plant, victim, home
}

// assertPlantNotRead proves the refusal in the terms that matter: neither the
// planted rules.json nor the planted guard.json reached the loaders. Resolving
// somewhere else is the mechanism; not reading those two files is the property.
func assertPlantNotRead(t *testing.T, root string) {
	t.Helper()
	rs, err := Load(root)
	if err != nil {
		t.Fatalf("Load(%q): %v", root, err)
	}
	if rs.Disabled {
		t.Errorf("the planted rules.json was read: the loader kill switch is in force at %q", root)
	}
	reg, err := guard.Load(root)
	if err != nil {
		t.Fatalf("guard.Load(%q): %v", root, err)
	}
	if reg.Disabled {
		t.Errorf("the planted guard.json was read: the hazard registry is switched off at %q", root)
	}
}

// noteMentioning returns the first note containing want, or "".
func noteMentioning(notes []string, want string) string {
	for _, n := range notes {
		if strings.Contains(n, want) {
			return n
		}
	}
	return ""
}

// TestResolveRootRefusesAMarkerRootTheCallerDoesNotOwn: the ruling. A marker
// root owned by another uid, with no opt-in, does not govern the session — and
// the refusal is loud and names the remedy, because a resolution that silently
// declined to read a repository's own configuration is the failure mode the
// bound itself was written to avoid (loud-staging).
func TestResolveRootRefusesAMarkerRootTheCallerDoesNotOwn(t *testing.T) {
	plant, victim, _ := foreignPlant(t)
	ownedByAnother(t, plant)

	res := Resolve(victim)
	if res.Root != victim {
		t.Errorf("Resolve(%q).Root = %q; a marker root owned by another uid must not govern the session", victim, res.Root)
	}
	assertPlantNotRead(t, res.Root)

	if len(res.Notes) == 0 {
		t.Fatalf("the refusal is silent: Resolve(%q) produced no note", victim)
	}
	note := noteMentioning(res.Notes, TrustedRootsDisplay)
	if note == "" {
		t.Fatalf("no note names the remedy %q; notes = %q", TrustedRootsDisplay, res.Notes)
	}
	if !strings.Contains(note, filepath.Base(plant)) {
		t.Errorf("the refusal does not name the root it refused (%q): %s", plant, note)
	}
}

// TestResolveRootAdmitsAForeignRootDeclaredInTheUserHome: the opt-in exists so a
// container bind mount, a foreign-uid checkout and a shared CI checkout have a
// supported route. Declared, the same root governs the session exactly as it did
// before the ownership policy — kill switch and all — and says nothing.
func TestResolveRootAdmitsAForeignRootDeclaredInTheUserHome(t *testing.T) {
	plant, victim, home := foreignPlant(t)
	ownedByAnother(t, plant)
	declareTrusted(t, home, plant)

	res := Resolve(victim)
	if got, want := resolvedPath(res.Root), resolvedPath(plant); got != want {
		t.Fatalf("Resolve(%q).Root = %q, want the declared root %q", victim, got, want)
	}
	if len(res.Notes) != 0 {
		t.Errorf("a declared root must resolve quietly; notes = %q", res.Notes)
	}
	rs, err := Load(res.Root)
	if err != nil {
		t.Fatalf("Load(%q): %v", res.Root, err)
	}
	if !rs.Disabled {
		t.Errorf("the declared root's rules.json was not read: its kill switch is not in force")
	}
}

// TestResolveRootAdmitsAMarkerRootTheCallerOwns: the ordinary case, unchanged.
// The owner lookup is NOT substituted here, so this runs against the real
// filesystem — the fixture is genuinely this process's own.
func TestResolveRootAdmitsAMarkerRootTheCallerOwns(t *testing.T) {
	plant, victim, _ := foreignPlant(t)

	res := Resolve(victim)
	if got, want := resolvedPath(res.Root), resolvedPath(plant); got != want {
		t.Fatalf("Resolve(%q).Root = %q, want the caller's own checkout root %q", victim, got, want)
	}
	if len(res.Notes) != 0 {
		t.Errorf("a root the caller owns must resolve quietly; notes = %q", res.Notes)
	}
	rs, err := Load(res.Root)
	if err != nil {
		t.Fatalf("Load(%q): %v", res.Root, err)
	}
	if !rs.Disabled {
		t.Errorf("the caller's own rules.json was not read: its kill switch is not in force")
	}
}

// TestResolveRootIgnoresATrustDeclarationInsideTheRootItself: the circularity
// bar. A tree that could vouch for itself would be no bar at all, so the
// declaration is read ONLY from the user-scope home — never from the root under
// judgement, at any spelling.
func TestResolveRootIgnoresATrustDeclarationInsideTheRootItself(t *testing.T) {
	plant, victim, _ := foreignPlant(t)
	ownedByAnother(t, plant)
	// The same file the home scope would honour, written inside the foreign
	// root: at the repo-relative spelling, and bare at its top level.
	declareTrusted(t, plant, plant)
	if err := os.WriteFile(filepath.Join(plant, "trusted-roots"), []byte(plant+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := Resolve(victim)
	if res.Root != victim {
		t.Errorf("Resolve(%q).Root = %q; a root that declares ITSELF trusted must still be refused", victim, res.Root)
	}
	assertPlantNotRead(t, res.Root)
}

// TestResolveRootHonoursOnlyACallerOwnedDeclaration: the declaration is
// caller-controlled or it is nothing. A ~/.abcd/trusted-roots this process does
// not own, or one anyone can write, was not necessarily written by the caller —
// so it re-admits nothing, and says why.
func TestResolveRootHonoursOnlyACallerOwnedDeclaration(t *testing.T) {
	t.Run("owned by another uid", func(t *testing.T) {
		plant, victim, home := foreignPlant(t)
		decl := declareTrusted(t, home, plant)
		ownedByAnother(t, plant, decl)

		res := Resolve(victim)
		if res.Root != victim {
			t.Errorf("Resolve(%q).Root = %q; a declaration the caller does not own must re-admit nothing", victim, res.Root)
		}
		assertPlantNotRead(t, res.Root)
		if noteMentioning(res.Notes, TrustedRootsDisplay) == "" {
			t.Errorf("the ignored declaration is silent; notes = %q", res.Notes)
		}
	})
	t.Run("writable by anyone", func(t *testing.T) {
		plant, victim, home := foreignPlant(t)
		decl := declareTrusted(t, home, plant)
		if err := os.Chmod(decl, 0o666); err != nil {
			t.Fatal(err)
		}
		ownedByAnother(t, plant)

		res := Resolve(victim)
		if res.Root != victim {
			t.Errorf("Resolve(%q).Root = %q; a world-writable declaration must re-admit nothing", victim, res.Root)
		}
		assertPlantNotRead(t, res.Root)
	})
}

// TestResolveRootOwnershipLeavesTheGitAnsweringPathAlone: the policy is a bound
// on the FALLBACK, not a new condition on resolution. When git answers, the
// toplevel it names is the answer whoever owns it — that is a repository git
// itself vouched for, and narrowing it here would break every legitimate
// foreign-uid checkout on a host whose git is configured for it.
func TestResolveRootOwnershipLeavesTheGitAnsweringPathAlone(t *testing.T) {
	outer := mustDir(t, t.TempDir())
	t.Setenv("HOME", mustDir(t, filepath.Join(outer, "home")))
	repo := filepath.Join(outer, "checkout")
	gitInitAt(t, repo)
	plantConfiguration(t, repo)
	sub := mustDir(t, filepath.Join(repo, "internal", "deep"))
	if _, err := gitutil.Run(sub, "rev-parse", "--show-toplevel"); err != nil {
		t.Skipf("git does not answer for the fixture: %v", err)
	}
	ownedByAnother(t, repo)

	res := Resolve(sub)
	if got, want := resolvedPath(res.Root), resolvedPath(repo); got != want {
		t.Fatalf("Resolve(%q).Root = %q, want the toplevel git named, %q", sub, got, want)
	}
	if len(res.Notes) != 0 {
		t.Errorf("the git-answering path must be untouched by the ownership policy; notes = %q", res.Notes)
	}
	rs, err := Load(res.Root)
	if err != nil {
		t.Fatalf("Load(%q): %v", res.Root, err)
	}
	if !rs.Disabled {
		t.Errorf("the repo's own rules.json was not read: its kill switch is not in force")
	}
}

// ownerUnreadable makes the owner lookup FAIL for path — the condition a
// fail-closed gate must not spell the same way as "mine". It cannot be staged
// on a real filesystem here either, so it goes through the same seam.
func ownerUnreadable(t *testing.T, path string) {
	t.Helper()
	real := ownerUID
	target := resolvedPath(path)
	ownerUID = func(p string) (uint32, error) {
		if resolvedPath(p) == target {
			return 0, errors.New("owner lookup failed")
		}
		return real(p)
	}
	t.Cleanup(func() { ownerUID = real })
}

// TestResolveRootRefusesAMarkerRootWhoseOwnerCannotBeRead: "I could not learn
// who owns this" and "I own this" are different answers, and a gate that folds
// the first into the second admits exactly the root it exists to refuse.
func TestResolveRootRefusesAMarkerRootWhoseOwnerCannotBeRead(t *testing.T) {
	plant, victim, _ := foreignPlant(t)
	ownerUnreadable(t, plant)

	res := Resolve(victim)
	if res.Root != victim {
		t.Errorf("Resolve(%q).Root = %q; a root whose owner could not be read must not govern the session", victim, res.Root)
	}
	assertPlantNotRead(t, res.Root)
	if noteMentioning(res.Notes, "could not be read") == "" {
		t.Errorf("the refusal does not say the owner was unreadable; notes = %q", res.Notes)
	}
}

// TestResolveRootMatchesADeclaredRootExactly: the declaration names ROOTS, one
// per line, and admits those and nothing else. Reading it as a containment rule
// would silently widen every entry into a subtree — a single `/tmp` line would
// re-admit every plant beneath it, which is the defect the gate closes.
func TestResolveRootMatchesADeclaredRootExactly(t *testing.T) {
	plant, victim, home := foreignPlant(t)
	ownedByAnother(t, plant)
	declareTrusted(t, home, filepath.Dir(plant))

	res := Resolve(victim)
	if res.Root != victim {
		t.Errorf("Resolve(%q).Root = %q; declaring an ANCESTOR of a root must not admit the root", victim, res.Root)
	}
	assertPlantNotRead(t, res.Root)
}
