package site

// The output-directory gate (GHSA-fpf2-pg82-72rj, CWE-59).
//
// `site build` empties and rewrites the directory named by --out, and `site
// check` reads it. Both took the path at its word: a symlink at the leaf or at
// any ancestor was followed, so a committed link (git mode 120000) named `site`
// — the default --out — redirected the purge and the writes to wherever the
// link pointed; and the `.abcd-site-build` marker was recognised by NAME, so a
// committed marker made any directory later passed to --out "ours" and had its
// other entries removed. These tests are the attack paths, each asserting the
// build refuses AND that nothing outside the lexical destination was touched.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
)

func fsutilCaseFolds() bool { return fsutil.CaseFoldingFS() }

// symlinkOrSkip plants a symlink, skipping on a filesystem that cannot hold
// one: the subject is abcd's refusal, not the platform's symlink support.
func symlinkOrSkip(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
}

// genuineMarker is the marker a build of f writes — the forger's best case,
// byte-identical to the real thing.
func genuineMarker(t *testing.T, f *fixture) []byte {
	t.Helper()
	out := t.TempDir()
	buildFixture(t, f, out)
	data, err := os.ReadFile(filepath.Join(out, siteMarkerName))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// TestBuildRefusesASymlinkedOutputLeaf is attack path 1: a committed symlink
// named `site` pointing outside the checkout. The target is EMPTY, which is the
// case a name-only inspection reads as "nothing here, go ahead" and writes the
// whole tree through the link.
func TestBuildRefusesASymlinkedOutputLeaf(t *testing.T) {
	f := newFixture(t)
	outside := t.TempDir()
	link := filepath.Join(f.Root(), DefaultOutDir)
	symlinkOrSkip(t, outside, link)

	// The gate itself, not merely the handle behind it, refuses the leaf.
	if _, gerr := resolveOutDir(f.Root(), link); gerr == nil || !strings.Contains(gerr.Error(), "symlink") {
		t.Errorf("resolveOutDir accepted a symlinked leaf: %v", gerr)
	}
	_, err := Build(Request{RepoRoot: f.Root(), OutDir: link, Stamp: fixtureStamp})
	if err == nil {
		t.Fatal("the build wrote through a symlinked output directory")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Errorf("the refusal does not say why: %v", err)
	}
	entries, rerr := os.ReadDir(outside)
	if rerr != nil {
		t.Fatal(rerr)
	}
	if len(entries) != 0 {
		t.Errorf("the refused build wrote %d entries through the link", len(entries))
	}
}

// TestBuildRefusesASymlinkedAncestorOfAnAbsentLeaf is attack path 2: the leaf
// does not exist, so a leaf-only lstat sees nothing, and MkdirAll walks the
// committed ancestor link to create the tree outside the checkout.
func TestBuildRefusesASymlinkedAncestorOfAnAbsentLeaf(t *testing.T) {
	f := newFixture(t)
	outside := t.TempDir()
	symlinkOrSkip(t, outside, filepath.Join(f.Root(), "parent-link"))
	out := filepath.Join(f.Root(), "parent-link", DefaultOutDir)

	_, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: fixtureStamp})
	if err == nil {
		t.Fatal("the build created its tree through a symlinked ancestor")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Errorf("the refusal does not say why: %v", err)
	}
	if _, serr := os.Lstat(filepath.Join(outside, DefaultOutDir)); serr == nil {
		t.Error("the refused build still created the output tree outside the checkout")
	}
}

// TestBuildRefusesToPurgeThroughACommittedMarker is attack path 3: a marker
// committed into a tracked directory, byte-identical to a genuine one, next to
// a file that is not the build's. A name-only check reads the directory as
// "ours" and removes the file; the purge must never delete what abcd did not
// write, and a directory git tracks anything in is never abcd's.
func TestBuildRefusesToPurgeThroughACommittedMarker(t *testing.T) {
	f := newFixture(t)
	marker := genuineMarker(t, f)
	f.writeBytes("public/"+siteMarkerName, marker)
	f.write("public/precious.txt", "someone else's work")
	f.commitAt("2026-02-12T09:00:00+00:00", "chore: plant a marker", "None")

	out := filepath.Join(f.Root(), "public")
	_, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: fixtureStamp})
	if err == nil {
		t.Fatal("the build purged a tracked directory on the strength of a committed marker")
	}
	got, rerr := os.ReadFile(filepath.Join(out, "precious.txt"))
	if rerr != nil || string(got) != "someone else's work" {
		t.Errorf("the refused build removed the file that was not its own: %q, %v", got, rerr)
	}
}

// TestBuildRefusesTheRepositoryRootAsOutput is attack path 3 at its worst: the
// marker committed at the root and `--out .`. The purge would remove every
// other entry — `.git` included.
func TestBuildRefusesTheRepositoryRootAsOutput(t *testing.T) {
	f := newFixture(t)
	marker := genuineMarker(t, f)
	f.writeBytes(siteMarkerName, marker)
	f.commitAt("2026-02-12T09:00:00+00:00", "chore: plant a marker at the root", "None")

	_, err := Build(Request{RepoRoot: f.Root(), OutDir: f.Root(), Stamp: fixtureStamp})
	if err == nil {
		t.Fatal("the build emptied the repository it was rendering")
	}
	for _, rel := range []string{".git", ManifestRelPath} {
		if _, serr := os.Lstat(filepath.Join(f.Root(), rel)); serr != nil {
			t.Errorf("the refused build removed %s: %v", rel, serr)
		}
	}
}

// TestBuildRefusesAnotherRepositorysOutput: the marker names the repository
// that wrote it, so a directory another repository's build filled is foreign —
// not a security boundary (a forger can copy the name), but the rule that keeps
// two projects sharing one --out from purging each other's site.
func TestBuildRefusesAnotherRepositorysOutput(t *testing.T) {
	a := newFixture(t)
	b := newFixtureRootedAt(t, "2026-01-04T09:00:00+00:00")
	rootOf := func(f *fixture) string { return f.gitOut("rev-list", "--max-parents=0", "HEAD") }
	if rootOf(a) == rootOf(b) {
		t.Fatal("the two fixtures share a root commit, so this test would prove nothing")
	}
	out := t.TempDir()
	buildFixture(t, a, out)

	_, err := Build(Request{RepoRoot: b.Root(), OutDir: out, Stamp: fixtureStamp})
	if err == nil {
		t.Fatal("a build of one repository purged another repository's site")
	}
	if _, serr := os.Stat(filepath.Join(out, "index.html")); serr != nil {
		t.Errorf("the refused build removed the other repository's page: %v", serr)
	}
}

// TestCheckRefusesASymlinkedOutputDir: the gate reads the directory it is
// handed, and a committed `site` link would have it grade — and, absent an
// index.html, build into — whatever the link points at.
func TestCheckRefusesASymlinkedOutputDir(t *testing.T) {
	f := newFixture(t)
	built := t.TempDir()
	buildFixture(t, f, built)
	link := filepath.Join(f.Root(), DefaultOutDir)
	symlinkOrSkip(t, built, link)

	_, err := Check(CheckRequest{RepoRoot: f.Root(), OutDir: link})
	if err == nil {
		t.Fatal("the check read through a symlinked output directory")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Errorf("the refusal does not say why: %v", err)
	}
}

// TestBuildRefusesAnIdentitylessOutputSharedByAnotherTree: a source tree with
// no root commit (no git, or none yet) has no identity a marker could name, so
// two such trees would otherwise write the same marker and purge each other.
// The rule fails closed: with no identity, a non-empty directory is foreign.
func TestBuildRefusesAnIdentitylessOutputSharedByAnotherTree(t *testing.T) {
	gitless := func() *fixture {
		f := newFixture(t)
		if err := os.RemoveAll(filepath.Join(f.Root(), ".git")); err != nil {
			t.Fatal(err)
		}
		return f
	}
	a, b := gitless(), gitless()
	out := t.TempDir()
	buildFixture(t, a, out)

	_, err := Build(Request{RepoRoot: b.Root(), OutDir: out, Stamp: fixtureStamp})
	if err == nil {
		t.Fatal("a build with no repository identity purged another identity-less tree's output")
	}
	if _, serr := os.Stat(filepath.Join(out, "index.html")); serr != nil {
		t.Errorf("the refused build removed the first tree's page: %v", serr)
	}
}

// TestResolveOutDirFoldsTheGitComponent: every other location rule case-folds
// on a folding filesystem, and `.GIT` names `.git` there.
func TestResolveOutDirFoldsTheGitComponent(t *testing.T) {
	if !fsutilCaseFolds() {
		t.Skip("the filesystem does not fold case")
	}
	f := newFixture(t)
	if _, err := resolveOutDir(f.Root(), filepath.Join(f.Root(), ".GIT", "x")); err == nil {
		t.Fatal("a case-variant .git component was accepted as an output directory")
	}
}

// TestResolveOutDirRefusesAGitFile: a worktree or submodule checkout carries
// `.git` as a FILE, and is a checkout all the same.
func TestResolveOutDirRefusesAGitFile(t *testing.T) {
	f := newFixture(t)
	wt := filepath.Join(t.TempDir(), "wt")
	if err := os.MkdirAll(wt, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt, ".git"), []byte("gitdir: elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveOutDir(f.Root(), wt); err == nil || !strings.Contains(err.Error(), ".git") {
		t.Fatalf("a directory holding a .git file was accepted: %v", err)
	}
}

// TestResolveOutDirFollowsAnAncestorLinkOutsideTheRepository is the deliberate
// half of the symlink rule: a link the operating system or the operator keeps
// outside the checkout is followed, because an untrusted commit cannot have
// planted it.
func TestResolveOutDirFollowsAnAncestorLinkOutsideTheRepository(t *testing.T) {
	f := newFixture(t)
	base := t.TempDir()
	real := filepath.Join(base, "real")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	symlinkOrSkip(t, real, link)
	want := filepath.Join(link, DefaultOutDir)
	got, err := resolveOutDir(f.Root(), want)
	if err != nil {
		t.Fatalf("an ancestor link outside the repository was refused: %v", err)
	}
	if got != want {
		t.Errorf("resolveOutDir = %q, want the lexical %q", got, want)
	}
}

// TestResolveOutDirKeysTheBoundaryOnTheCheckout: the repository boundary is the
// git toplevel, not the directory the build was invoked from, so a committed
// link ABOVE a subdirectory cwd is still inside the checkout.
func TestResolveOutDirKeysTheBoundaryOnTheCheckout(t *testing.T) {
	f := newFixture(t)
	outside := t.TempDir()
	symlinkOrSkip(t, outside, filepath.Join(f.Root(), "above-link"))
	sub := filepath.Join(f.Root(), "docs")
	_, err := resolveOutDir(sub, filepath.Join(f.Root(), "above-link", DefaultOutDir))
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("a committed link above the invoking directory was followed: %v", err)
	}
}

// TestDescribeReportsARefusedOutputDirectory: the status board reads the
// output directory through the same gate, and reports a refusal rather than
// following a committed link to count what lies beyond it.
func TestDescribeReportsARefusedOutputDirectory(t *testing.T) {
	f := newFixture(t)
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "index.html"), []byte("<p>"), 0o644); err != nil {
		t.Fatal(err)
	}
	symlinkOrSkip(t, outside, filepath.Join(f.Root(), DefaultOutDir))
	st, err := Describe(f.Root(), DefaultOutDir)
	if err != nil {
		t.Fatal(err)
	}
	if st.OutExists || st.OutFiles != 0 {
		t.Errorf("the board counted entries through a symlinked output directory: %+v", st)
	}
	if st.OutRefused == "" {
		t.Error("the board does not say the output directory was refused")
	}
}

// TestOpenOutDirRefusesASymlink pins the handle the purge and the writes go
// through: it is opened only over a real directory, so a leaf that became a
// link after the gate ran is never opened at its target.
func TestOpenOutDirRefusesASymlink(t *testing.T) {
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	symlinkOrSkip(t, real, link)
	if root, err := openOutDir(link); err == nil {
		root.Close()
		t.Fatal("the output handle was opened through a symlink")
	}
	root, err := openOutDir(real)
	if err != nil {
		t.Fatal(err)
	}
	root.Close()
}

// shimGit puts a `git` on PATH that runs the real one, altered by script (a
// shell fragment that sees "$@"; it must exec "$REAL_GIT" itself or exit).
func shimGit(t *testing.T, script string) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git unavailable")
	}
	dir := t.TempDir()
	body := "#!/bin/sh\nREAL_GIT=" + real + "\n" + script + "\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// TestCheckoutRootIgnoresAnEchoingGit: `git rev-parse` echoes an option it does
// not know to stdout and exits 0, so a git too old for a flag answers with the
// flag's text and the path on two lines. That answer must not become the
// boundary — a boundary that matches no real path widens the symlink rule to
// nothing.
func TestCheckoutRootIgnoresAnEchoingGit(t *testing.T) {
	f := newFixture(t)
	shimGit(t, `case "$*" in *rev-parse*--path-format*) echo "--path-format=absolute";; esac
exec "$REAL_GIT" "$@"`)
	outside := t.TempDir()
	symlinkOrSkip(t, outside, filepath.Join(f.Root(), "above-link"))
	sub := filepath.Join(f.Root(), "docs")

	top, err := checkoutRoot(sub)
	if err != nil {
		t.Fatalf("checkoutRoot: %v", err)
	}
	if !filepath.IsAbs(top) || strings.Contains(top, "\n") || strings.Contains(top, "--path-format") {
		t.Errorf("checkoutRoot took an echoed option as the boundary: %q", top)
	}
	if _, err := resolveOutDir(sub, filepath.Join(f.Root(), "above-link", DefaultOutDir)); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("an echoing git widened the checkout boundary: %v", err)
	}
}

// TestResolveOutDirRefusesWhenGitCannotNameTheCheckout: inside something
// repo-shaped, a git that cannot answer is a refusal, not a silent fallback to
// the invoking directory — the same fail-closed branch refuseTrackedOutDir has.
func TestResolveOutDirRefusesWhenGitCannotNameTheCheckout(t *testing.T) {
	f := newFixture(t)
	shimGit(t, `exit 128`)
	_, err := resolveOutDir(f.Root(), filepath.Join(f.Root(), DefaultOutDir))
	if err == nil || !strings.Contains(err.Error(), "git") {
		t.Fatalf("a checkout git cannot answer for was accepted on the invoking directory alone: %v", err)
	}
}

// TestRegateRefusesATrackedDirectoryAtTheWriteInstant pins the write-instant
// ownership rule on its own: the re-gate, called directly, refuses a directory
// git tracks a file in even when it carries a genuine marker. Deleting the
// tracked-files check from the re-gate fails this test.
func TestRegateRefusesATrackedDirectoryAtTheWriteInstant(t *testing.T) {
	f := newFixture(t)
	f.writeBytes("public/"+siteMarkerName, genuineMarker(t, f))
	f.write("public/precious.txt", "someone else's work")
	f.commitAt("2026-02-12T09:00:00+00:00", "chore: plant a marker", "None")
	out := filepath.Join(f.Root(), "public")

	root, err := regateOutDir(f.Root(), out, f.gitOut("rev-list", "--max-parents=0", "HEAD"))
	if err == nil {
		root.Close()
		t.Fatal("the re-gate purged a tracked directory")
	}
	if got, rerr := os.ReadFile(filepath.Join(out, "precious.txt")); rerr != nil || string(got) != "someone else's work" {
		t.Errorf("the re-gate removed the file that was not its own: %q, %v", got, rerr)
	}
}

// TestRegatePurgesThroughTheHandleNotThePath: between the decision and the
// removal the directory is swapped for a link to somewhere precious. A purge
// through the handle opened over the real directory cannot reach the link's
// target; a purge by path would empty it. Moving the purge ahead of the handle
// fails this test.
func TestRegatePurgesThroughTheHandleNotThePath(t *testing.T) {
	const identity = "0123456789abcdef0123456789abcdef01234567"
	base := t.TempDir()
	out := filepath.Join(base, DefaultOutDir)
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string][]byte{siteMarkerName: siteMarker(identity), "index.html": []byte("<p>")} {
		if err := os.WriteFile(filepath.Join(out, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	outside := t.TempDir()
	precious := filepath.Join(outside, "precious.txt")
	if err := os.WriteFile(precious, []byte("someone else's work"), 0o644); err != nil {
		t.Fatal(err)
	}

	prev := beforeOutDirPurge
	defer func() { beforeOutDirPurge = prev }()
	beforeOutDirPurge = func(dir string) {
		if err := os.Rename(dir, dir+".moved"); err != nil {
			t.Fatal(err)
		}
		symlinkOrSkip(t, outside, dir)
	}

	if root, err := regateOutDir(t.TempDir(), out, identity); err == nil {
		root.Close()
	}
	if got, rerr := os.ReadFile(precious); rerr != nil || string(got) != "someone else's work" {
		t.Errorf("the purge reached through the swapped-in link: %q, %v", got, rerr)
	}
}

// TestDescribeJSONCarriesNoAbsolutePath: the board's refusal is machine output
// too, and machine output never carries a developer-identity path (iss-81).
func TestDescribeJSONCarriesNoAbsolutePath(t *testing.T) {
	f := newFixture(t)
	symlinkOrSkip(t, t.TempDir(), filepath.Join(f.Root(), DefaultOutDir))
	st, err := Describe(f.Root(), DefaultOutDir)
	if err != nil {
		t.Fatal(err)
	}
	js, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	if st.OutRefused == "" {
		t.Fatal("the board did not refuse the symlinked output directory")
	}
	if strings.Contains(string(js), f.Root()) {
		t.Errorf("the JSON board carries the absolute repository path: %s", js)
	}
}

// TestCheckoutRootRefusesADecoyWorktree: a repo-local `core.worktree` (an
// archive-shipped checkout can carry one) makes `rev-parse --show-toplevel`
// answer somewhere else — one absolute line, no error. A boundary that does
// not contain the invoking directory is no boundary; the internal link must
// still be refused.
func TestCheckoutRootRefusesADecoyWorktree(t *testing.T) {
	f := newFixture(t)
	decoy := t.TempDir()
	f.git("2026-02-12T09:00:00+00:00", "config", "core.worktree", decoy)
	outside := t.TempDir()
	symlinkOrSkip(t, outside, filepath.Join(f.Root(), "linkdir"))

	if top, err := checkoutRoot(f.Root()); err == nil && !strings.HasPrefix(fsutil.RealExistingPath(f.Root()), top) {
		t.Errorf("checkoutRoot took the decoy %q as the boundary of %q", top, f.Root())
	}
	if _, err := resolveOutDir(f.Root(), filepath.Join(f.Root(), "linkdir", DefaultOutDir)); err == nil {
		t.Fatal("a decoy core.worktree widened the checkout boundary past an internal link")
	}
}

// TestCheckoutRootRefusesATrimmedAnswer: the git runner trims stdout, so a
// checkout whose directory name ends in a space answers with a path naming
// nothing real. That answer does not contain the checkout and is refused.
func TestCheckoutRootRefusesATrimmedAnswer(t *testing.T) {
	base := t.TempDir()
	repo := filepath.Join(base, "repo ")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Skipf("the filesystem refuses a trailing space: %v", err)
	}
	init := exec.Command("git", "init", "-q", repo)
	init.Env = gittest.Env(t)
	if out, err := init.CombinedOutput(); err != nil {
		t.Skipf("git init: %v (%s)", err, out)
	}
	symlinkOrSkip(t, t.TempDir(), filepath.Join(repo, "linkdir"))

	if _, err := resolveOutDir(repo, filepath.Join(repo, "linkdir", DefaultOutDir)); err == nil {
		t.Fatal("a trimmed toplevel widened the checkout boundary past an internal link")
	}
}
