package banlist

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// worktreePair initialises a repo, seeds a commit, and adds a linked worktree,
// returning both roots. It is the shape itd-150 exists for: one repository, two
// working trees, and a local-ephemeral tier that belongs to whichever one you are
// standing in.
func worktreePair(t *testing.T) (primary, linked string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	primary = t.TempDir()
	env := gittest.Env(t)
	git := func(dir string, args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	for _, args := range [][]string{
		{"init"},
		{"config", "user.name", "Alice Example"},
		{"config", "user.email", "alice@example.com"},
	} {
		if out, err := git(primary, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(primary, ".gitignore"), []byte(PrivateDirRelPath+"/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := git(primary, "add", "-A"); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	if out, err := git(primary, "-c", "core.hooksPath=/dev/null", "commit", "-m", "seed"); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	linked = filepath.Join(t.TempDir(), "linked")
	if out, err := git(primary, "worktree", "add", "-b", "linked", linked); err != nil {
		t.Skipf("git worktree add unavailable: %v\n%s", err, out)
	}
	return primary, linked
}

// TestPrimaryWorktreeRootResolvesOnlyFromALinkedWorktree pins both directions of
// the resolution the guard makes: a linked worktree finds its primary checkout, and
// a standalone checkout finds nothing — the second is what keeps the fallback a
// fallback rather than a second store every repo suddenly acquires.
func TestPrimaryWorktreeRootResolvesOnlyFromALinkedWorktree(t *testing.T) {
	primary, linked := worktreePair(t)

	got, ok := PrimaryWorktreeRoot(linked)
	if !ok {
		t.Fatal("a linked worktree did not resolve its primary checkout")
	}
	// Compared through EvalSymlinks: git answers with the resolved path, and a temp
	// dir on macOS sits behind /private.
	if resolve(t, got) != resolve(t, primary) {
		t.Errorf("primary root = %q, want %q", got, primary)
	}

	if got, ok := PrimaryWorktreeRoot(primary); ok {
		t.Errorf("the primary checkout resolved a primary of its own (%q); it must inherit nothing", got)
	}
	if got, ok := PrimaryWorktreeRoot(t.TempDir()); ok {
		t.Errorf("a directory that is not a repository resolved a primary (%q)", got)
	}
}

func resolve(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		return p
	}
	return r
}

// TestInheritedPrivateReportsThePrimaryStore pins the read side of the parity the
// spec requires: `abcd banlist` in a linked worktree must render the layer the
// guard actually enforces there. Without it the board reports the worktree
// INACTIVE while every commit made in it is checked against the primary's list —
// a status surface that disagrees with the guard about what is protected.
func TestInheritedPrivateReportsThePrimaryStore(t *testing.T) {
	primary, linked := worktreePair(t)

	// Nothing to inherit yet: the primary has no store either.
	inh, err := InheritedPrivate(linked)
	if err != nil {
		t.Fatal(err)
	}
	if inh != nil && inh.Private.Present {
		t.Errorf("an absent primary store was reported present")
	}

	if _, err := AddPrivate(AddPrivateRequest{RepoRoot: primary, Key: "widget-partner", Pattern: "widgetworks"}); err != nil {
		t.Fatal(err)
	}
	inh, err = InheritedPrivate(linked)
	if err != nil {
		t.Fatal(err)
	}
	if inh == nil {
		t.Fatal("a linked worktree inherited nothing from a primary checkout that has a store")
	}
	if !inh.Private.Present {
		t.Error("the inherited layer is not reported present")
	}
	if len(inh.Private.Entries) != 1 || inh.Private.Entries[0].Key != "widget-partner" {
		t.Errorf("inherited entries = %+v, want the primary's one key", inh.Private.Entries)
	}
	// The REPORT names no checkout: the resolution is pinned on PrimaryWorktreeRoot
	// above, and the path is deliberately not carried across to a front door (a
	// checkout's directory name is very often a name its own store bans).
	if got, ok := PrimaryWorktreeRoot(linked); !ok || resolve(t, got) != resolve(t, primary) {
		t.Errorf("PrimaryWorktreeRoot = %q (ok=%v), want %q", got, ok, primary)
	}

	// A standalone checkout inherits nothing, whatever its own store holds.
	if inh, err := InheritedPrivate(primary); err != nil || inh != nil {
		t.Errorf("the primary checkout inherited %+v (err=%v); it must inherit nothing", inh, err)
	}
}

// TestListCarriesTheInheritedLayer pins that the one read every status surface
// makes carries the inherited layer, so no surface has to remember to ask a second
// question — the way a surface ends up disagreeing with the guard.
func TestListCarriesTheInheritedLayer(t *testing.T) {
	primary, linked := worktreePair(t)
	if _, err := AddPrivate(AddPrivateRequest{RepoRoot: primary, Key: "widget-partner", Pattern: "widgetworks"}); err != nil {
		t.Fatal(err)
	}
	rep, err := List(linked)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Inherited == nil {
		t.Fatal("List did not carry the inherited layer in a linked worktree")
	}
	if len(rep.Inherited.Private.Entries) != 1 {
		t.Errorf("inherited entries = %+v, want one", rep.Inherited.Private.Entries)
	}
	rep, err = List(primary)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Inherited != nil {
		t.Errorf("List carried an inherited layer in a standalone checkout: %+v", rep.Inherited)
	}
}

// TestPrimaryWorktreeRootRefusesABareRepoWorktree pins the lockstep the guard and
// the board must keep. A worktree of a BARE clone has a common dir whose parent is
// not a working tree — it is whatever directory holds the git dir. The committed
// hook refuses to inherit there; if this resolver did not, the status board would
// list an unrelated neighbouring repository's entries as "enforced here too" while
// the guard enforced nothing, which is a board announcing protection that is not
// running.
func TestPrimaryWorktreeRootRefusesABareRepoWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	src, _ := worktreePair(t)
	parent := t.TempDir()
	env := gittest.Env(t)
	git := func(dir string, args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	// A neighbouring repository's private store, in the directory that will hold the
	// bare git dir. Nothing entitles a worktree of that bare repo to this list.
	neighbour := filepath.Join(parent, filepath.FromSlash(PrivateDirRelPath))
	if err := os.MkdirAll(neighbour, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, filepath.FromSlash(PrivateRelPath)),
		[]byte("# abcd-banlist: keyed\nneighbour-secret  somethingelse\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	bare := filepath.Join(parent, "repo.git")
	if out, err := git(parent, "clone", "--bare", src, bare); err != nil {
		t.Skipf("bare clone unavailable: %v\n%s", err, out)
	}
	linked := filepath.Join(t.TempDir(), "bare-linked")
	if out, err := git(bare, "worktree", "add", linked, "HEAD"); err != nil {
		t.Skipf("git worktree add unavailable: %v\n%s", err, out)
	}
	if got, ok := PrimaryWorktreeRoot(linked); ok {
		t.Errorf("a worktree of a bare repo resolved %q as its primary checkout; that directory is not a working tree", got)
	}
	inh, err := InheritedPrivate(linked)
	if err != nil {
		t.Fatal(err)
	}
	if inh != nil {
		t.Errorf("a neighbouring repository's private store was reported as inherited: %+v", inh)
	}
}

// hermeticGit returns a git driver for tests that must build repository SHAPES
// rather than reuse worktreePair's single repo. Every command runs under
// gittest.Env, so an ambient GIT_DIR cannot redirect the fixture onto a real
// checkout.
func hermeticGit(t *testing.T) func(dir string, args ...string) (string, error) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	env := gittest.Env(t)
	return func(dir string, args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
}

// seedCheckout makes dir a REAL working tree with one commit — the thing the
// resolver is supposed to require, and the thing a `.git` existence test cannot
// distinguish from a directory that merely holds somebody's git dir.
func seedCheckout(t *testing.T, git func(string, ...string) (string, error), dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init"},
		{"config", "user.name", "Alice Example"},
		{"config", "user.email", "alice@example.com"},
		{"-c", "core.hooksPath=/dev/null", "commit", "--allow-empty", "-m", "seed"},
	} {
		if out, err := git(dir, args...); err != nil {
			t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
		}
	}
}

// writeNeighbourStore plants a private store in a checkout that has nothing to do
// with the repository under test. Reading it is the whole finding.
func writeNeighbourStore(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(PrivateDirRelPath)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(PrivateRelPath)),
		[]byte("# abcd-banlist: keyed\nneighbour-secret  somethingelse\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestPrimaryWorktreeRootRefusesAMirrorInsideAnotherCheckout is the escalation the
// `.git` existence test does not survive. Clone a repository BARE into a directory
// that sits inside somebody else's working tree, then `git worktree add` from it:
// the common dir is <victim>/mirror.git, its parent is <victim>, and <victim>/.git
// exists — so a guard that asks only "does this directory carry a .git entry?"
// accepts an unrelated repository as this worktree's primary checkout and reads its
// private store. That store's whole contract is that its pattern values never reach
// output; inheriting it hands over a match/no-match oracle on another repository's
// private names, discloses their KEYS into this repo's hook output, and lets one
// malformed line over there refuse every commit here.
func TestPrimaryWorktreeRootRefusesAMirrorInsideAnotherCheckout(t *testing.T) {
	git := hermeticGit(t)
	root := t.TempDir()

	victim := filepath.Join(root, "victim")
	seedCheckout(t, git, victim)
	writeNeighbourStore(t, victim)

	src := filepath.Join(root, "src")
	seedCheckout(t, git, src)

	mirror := filepath.Join(victim, "mirror.git")
	if out, err := git(root, "clone", "--bare", src, mirror); err != nil {
		t.Skipf("bare clone unavailable: %v\n%s", err, out)
	}
	linked := filepath.Join(t.TempDir(), "linked")
	if out, err := git(mirror, "worktree", "add", linked, "HEAD"); err != nil {
		t.Skipf("git worktree add unavailable: %v\n%s", err, out)
	}

	if got, ok := PrimaryWorktreeRoot(linked); ok {
		t.Errorf("a worktree of a bare mirror resolved %q as its primary checkout; that checkout belongs to a DIFFERENT repository", got)
	}
	inh, err := InheritedPrivate(linked)
	if err != nil {
		t.Fatalf("resolution must degrade to nothing, never to an error: %v", err)
	}
	if inh != nil {
		t.Errorf("an unrelated checkout's private store was reported as inherited: %+v", inh)
	}
}

// TestPrimaryWorktreeRootRefusesASeparateGitDirInsideAnotherCheckout is the same
// finding reached by the second documented route. `git init --separate-git-dir`
// puts the git dir wherever it is told, so the common dir's parent is again a
// directory that merely HOLDS a git dir — and when that directory is somebody's
// checkout it carries a `.git` of its own, which is all the old test asked for.
func TestPrimaryWorktreeRootRefusesASeparateGitDirInsideAnotherCheckout(t *testing.T) {
	git := hermeticGit(t)
	root := t.TempDir()

	victim := filepath.Join(root, "victim")
	seedCheckout(t, git, victim)
	writeNeighbourStore(t, victim)

	work := filepath.Join(root, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := git(work, "init", "--separate-git-dir="+filepath.Join(victim, "gitdir")); err != nil {
		t.Skipf("git init --separate-git-dir unavailable: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.name", "Alice Example"},
		{"config", "user.email", "alice@example.com"},
		{"-c", "core.hooksPath=/dev/null", "commit", "--allow-empty", "-m", "seed"},
	} {
		if out, err := git(work, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	linked := filepath.Join(t.TempDir(), "sgd-linked")
	if out, err := git(work, "worktree", "add", "-b", "linked", linked); err != nil {
		t.Skipf("git worktree add unavailable: %v\n%s", err, out)
	}

	if got, ok := PrimaryWorktreeRoot(linked); ok && resolve(t, got) != resolve(t, work) {
		t.Errorf("a --separate-git-dir worktree resolved %q as its primary checkout; that is not this repository's working tree", got)
	}
	inh, err := InheritedPrivate(linked)
	if err != nil {
		t.Fatalf("resolution must degrade to nothing, never to an error: %v", err)
	}
	if inh != nil {
		t.Errorf("an unrelated checkout's private store was reported as inherited: %+v", inh)
	}
}

// TestPrimaryWorktreeRootRefusesAnAnswerThatIsNotOnePath: git before 2.31 does
// not know --path-format, echoes the option to stdout and exits 0, so a
// rev-parse answer can carry the flag's text as well as a path. Every answer
// the resolution compares must be one absolute path or the resolution fails
// closed, as the pre-commit guard's does (iss-2608291924452604). The git on
// PATH here prefixes the echoed option to its --git-dir and --git-common-dir
// answers only, so the --show-toplevel confirmation alone cannot refuse it.
func TestPrimaryWorktreeRootRefusesAnAnswerThatIsNotOnePath(t *testing.T) {
	_, linked := worktreePair(t)
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git unavailable")
	}
	bin := t.TempDir()
	script := "#!/bin/sh\nfor a in \"$@\"; do case \"$a\" in --git-dir|--git-common-dir) echo \"--path-format=absolute\";; esac; done\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if got, ok := PrimaryWorktreeRoot(linked); ok {
		t.Fatalf("resolved %q from rev-parse answers that are not one absolute path", got)
	}
}
