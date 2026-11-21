package interview

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// withOrigin gives r's checkout an origin holding HEAD on main, fetched, so a
// remote-tracking ref holds HEAD and the pre-push gate reads open before the
// first turn: a worktree cut from origin's default branch, or a branch just
// pushed.
func withOrigin(t *testing.T, r *writtenRun) {
	t.Helper()
	origin := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, origin)
	r.git.Git("remote", "add", "origin", origin)
	r.git.Git("push", "-q", "origin", "HEAD:refs/heads/main")
	r.git.Git("fetch", "-q", "origin")
}

// containing is what git's for-each-ref, as the pre-push hook's on_a_remote
// runs it, lists of the remote-tracking refs holding commit in dir.
func containing(r *writtenRun, dir, commit string) []string {
	return strings.Fields(r.git.Git("-C", dir, "for-each-ref", "--format=%(refname)", "--contains", commit, "refs/remotes"))
}

// TestASymbolicRefOutsideItsRemoteStopsTheInterview (iss-2611012144422634,
// iss-2610040739124513): the pre-push hook takes a commit any remote-tracking
// ref holds as already on a remote, and git resolves a symbolic ref there at
// push time, so one pointing outside its remote follows a target the
// person's own next commit, or the next stash, moves. A role that makes one
// during a turn is stopped after that dispatch, naming it, whether the gate
// already reads open (where its verdict on HEAD does not change) or reads
// shut (where the target does not exist yet, which git skips): a loose
// symbolic ref at the checkout's own branch from an open gate, from the main
// worktree and from a linked one; one at refs/stash from a shut gate; a link
// under refs/ leading to the own branch's loose ref; and, where the own
// branch is itself a symbolic ref, its target moved. After the run each is
// shown to be one git honours: the next commit, or stash, is held by a
// remote-tracking ref.
func TestASymbolicRefOutsideItsRemoteStopsTheInterview(t *testing.T) {
	for _, c := range []struct {
		name string
		// origin starts the gate open; linked runs the interview in a linked
		// worktree on its own branch.
		origin, linked bool
		// before readies the checkout, given the run and its common
		// directory, before the interview runs.
		before func(r *writtenRun, common string)
		// write is what the role writes or runs in turn 2, given the run, the
		// checkout, its branch, HEAD and the common directory; named is what
		// the refusal must name (a suffix of one of its paths).
		write func(t *testing.T, r *writtenRun, script, mine, branch, head, common string) (named string)
		// prove shows, after the run, that git honours what was written: it
		// returns the commit a push would carry next and the ref that must
		// then hold it.
		prove func(t *testing.T, r *writtenRun, mine string) (commit, ref string)
		// said are words the refusal must hold, and unsaid words it must not.
		said, unsaid []string
	}{
		{name: "a symbolic ref at the own branch, the gate open", origin: true, write: plantSymref("x", "refs/heads/"), prove: nextCommit("refs/remotes/origin/x")},
		{name: "a symbolic ref at the own branch, the gate open, the interview in a linked worktree", origin: true, linked: true, write: plantSymref("x", "refs/heads/"), prove: nextCommit("refs/remotes/origin/x")},
		{name: "a symbolic ref at refs/stash, the gate shut", write: func(t *testing.T, r *writtenRun, script, mine, branch, head, common string) string {
			stubAlso(t, script, 2, map[string]string{filepath.Join(common, "refs", "remotes", "origin", "x"): "ref: refs/stash\n"})
			return "refs/remotes/origin/x"
		}, prove: func(t *testing.T, r *writtenRun, mine string) (string, string) {
			if err := os.WriteFile(filepath.Join(mine, "stashed"), []byte("work\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			r.git.Git("-C", mine, "stash", "-q", "-u")
			return r.git.Git("-C", mine, "rev-parse", "HEAD"), "refs/remotes/origin/x"
		}},
		{name: "a link under refs/ leading to the own branch's loose ref, the gate open", origin: true, write: func(t *testing.T, r *writtenRun, script, mine, branch, head, common string) string {
			stubPeer(t, script, 2, r.git.Env(), stubCmd{Dir: mine, Script: "ln -s ../../heads/" + branch + " " + filepath.Join(common, "refs", "remotes", "origin", "l")})
			return "refs/remotes/origin/l"
		}, prove: nextCommit("refs/remotes/origin/l")},
		{name: "a remote's HEAD naming a receipt-less branch of its own remote", write: func(t *testing.T, r *writtenRun, script, mine, branch, head, common string) string {
			stubAlso(t, script, 2, map[string]string{filepath.Join(common, "refs", "remotes", "origin", "HEAD"): "ref: refs/remotes/origin/abcd-wip/x\n"})
			return "refs/remotes/origin/HEAD"
		}, prove: func(t *testing.T, r *writtenRun, mine string) (string, string) {
			commit, _ := nextCommit("")(t, r, mine)
			r.git.Git("-C", mine, "update-ref", "refs/remotes/origin/abcd-wip/x", commit)
			return commit, "refs/remotes/origin/HEAD"
		}, said: []string{"naming \"refs/remotes/origin/abcd-wip/x\"", "receipt-less"}, unsaid: []string{"outside its remote"}},
		{name: "a remote's HEAD naming a reserved branch of its own remote, the gate open", origin: true, write: func(t *testing.T, r *writtenRun, script, mine, branch, head, common string) string {
			stubAlso(t, script, 2, map[string]string{filepath.Join(common, "refs", "remotes", "origin", "HEAD"): "ref: refs/remotes/origin/abcd-reserved/x\n"})
			return "refs/remotes/origin/HEAD"
		}, said: []string{"receipt-less"}},
		{name: "a symbolic ref outside refs/remotes made", write: plantSymrefAt("refs/heads/alias", "refs/heads/"), said: []string{"refs/heads/alias, naming", "outside refs/remotes/, was made"}, unsaid: []string{"outside its remote", "the pre-push gate takes"}},
		{name: "a symbolic ref outside its remote changed", origin: true, before: func(r *writtenRun, common string) {
			r.git.Git("symbolic-ref", "refs/remotes/origin/x", "refs/heads/main")
		}, write: func(t *testing.T, r *writtenRun, script, mine, branch, head, common string) string {
			stubAlso(t, script, 2, map[string]string{filepath.Join(common, "refs", "remotes", "origin", "x"): "ref: refs/stash\n"})
			return "refs/remotes/origin/x"
		}, said: []string{"refs/remotes/origin/x, naming \"refs/stash\", outside its remote, was changed"}},
		{name: "a symbolic ref outside its remote removed", origin: true, before: func(r *writtenRun, common string) {
			r.git.Git("symbolic-ref", "refs/remotes/origin/x", "refs/heads/main")
		}, write: func(t *testing.T, r *writtenRun, script, mine, branch, head, common string) string {
			stubPeer(t, script, 2, r.git.Env(), stubCmd{Dir: mine, Script: "rm " + filepath.Join(common, "refs", "remotes", "origin", "x")})
			return "refs/remotes/origin/x"
		}, said: []string{"refs/remotes/origin/x, naming \"refs/heads/main\", outside its remote, was removed"}},
		{name: "the own branch a symbolic ref, its target moved", before: func(r *writtenRun, common string) {
			r.git.Git("branch", "other")
			r.git.Git("symbolic-ref", "refs/heads/main", "refs/heads/other")
		}, write: func(t *testing.T, r *writtenRun, script, mine, branch, head, common string) string {
			above := r.git.Git("commit-tree", r.git.Git("rev-parse", "HEAD^{tree}"), "-p", head, "-m", "above")
			stubAlso(t, script, 2, map[string]string{filepath.Join(common, "refs", "heads", "other"): above + "\n"})
			return "refs/heads/main"
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
			r := newWrittenRun(t, routedToClaude)
			ignoring(".abcd/.work.local/\n")(r)
			r.git.Commit("base")
			if c.origin {
				withOrigin(t, r)
			}
			common := filepath.Join(r.repo, ".git")
			if c.before != nil {
				c.before(r, common)
			}
			mine := r.repo
			if c.linked {
				mine = laneFor(t, r, "mine")
				r.w.Repo = mine
			}
			branch := strings.TrimPrefix(r.git.Git("-C", mine, "symbolic-ref", "HEAD"), "refs/heads/")
			head := r.git.Git("-C", mine, "rev-parse", "HEAD")
			if open := len(containing(r, mine, head)) > 0; open != c.origin {
				t.Fatalf("the gate reads open %v before the run, want %v", open, c.origin)
			}
			named := c.write(t, r, script, mine, branch, head, common)
			res, err := r.w.Run(context.Background())
			var uc *UnexpectedChangesError
			if !errors.As(err, &uc) {
				t.Fatalf("err = %v, want the interview stopped on %s", err, named)
			}
			if !slices.ContainsFunc(uc.Paths, func(got string) bool { return strings.HasSuffix(got, named) }) || !strings.Contains(err.Error(), named) {
				t.Fatalf("the refusal does not name %s: %v", named, err)
			}
			for _, w := range c.said {
				if !strings.Contains(err.Error(), w) {
					t.Fatalf("the refusal does not say %q: %v", w, err)
				}
			}
			for _, w := range c.unsaid {
				if strings.Contains(err.Error(), w) {
					t.Fatalf("the refusal says %q, which is not what was found: %v", w, err)
				}
			}
			if len(r.asked) != 1 || len(r.finished) != 0 {
				t.Fatalf("asked %d, finished %d; nothing is drawn or finished after the change", len(r.asked), len(r.finished))
			}
			if rec := readRecord(t, res.Record); len(rec.Answers) != 1 {
				t.Fatalf("record %+v", rec)
			}
			if c.prove != nil {
				commit, ref := c.prove(t, r, mine)
				if got := containing(r, mine, commit); !slices.Contains(got, ref) {
					t.Fatalf("git lists %v as holding %s, not %s, so the probe is not one the pre-push gate honours", got, commit[:12], ref)
				}
			}
		})
	}
}

// plantSymref is a turn-2 write of a loose symbolic ref
// refs/remotes/origin/<name> pointing at prefix plus the checkout's branch.
func plantSymref(name, prefix string) func(t *testing.T, r *writtenRun, script, mine, branch, head, common string) string {
	return func(t *testing.T, r *writtenRun, script, mine, branch, head, common string) string {
		stubAlso(t, script, 2, map[string]string{filepath.Join(common, "refs", "remotes", "origin", name): "ref: " + prefix + branch + "\n"})
		return "refs/remotes/origin/" + name
	}
}

// plantSymrefAt is a turn-2 write of a loose symbolic ref name pointing at
// prefix plus the checkout's branch.
func plantSymrefAt(name, prefix string) func(t *testing.T, r *writtenRun, script, mine, branch, head, common string) string {
	return func(t *testing.T, r *writtenRun, script, mine, branch, head, common string) string {
		stubAlso(t, script, 2, map[string]string{filepath.Join(common, filepath.FromSlash(name)): "ref: " + prefix + branch + "\n"})
		return name
	}
}

// nextCommit is a prove that commits in the checkout, as the person's next
// commit of the interview's edits does, and wants ref to hold it.
func nextCommit(ref string) func(t *testing.T, r *writtenRun, mine string) (string, string) {
	return func(_ *testing.T, r *writtenRun, mine string) (string, string) {
		r.git.Git("-C", mine, "commit", "-q", "--allow-empty", "-m", "the interview's edits")
		return r.git.Git("-C", mine, "rev-parse", "HEAD"), ref
	}
}

// TestAPeersSetHeadDoesNotStopTheInterview (iss-2611012144422634): a peer's
// `git remote set-head`, or a fetch that follows the remote's HEAD, writes
// refs/remotes/origin/HEAD as a symbolic ref to a ref of the same remote,
// which only a fetch moves. Made, changed or removed while the role writes the
// first question, it leaves the interview running to its outcome, from the
// main worktree and from a linked one.
func TestAPeersSetHeadDoesNotStopTheInterview(t *testing.T) {
	for _, c := range []struct {
		name string
		// linked runs the interview in a linked worktree; standing makes
		// refs/remotes/origin/HEAD before the run, and removed has the peer
		// remove it.
		linked, standing, removed bool
		// peer is what the peer runs while the role runs.
		peer string
	}{
		{name: "set-head made", peer: "git remote set-head origin main"},
		{name: "set-head made, the interview in a linked worktree", linked: true, peer: "git remote set-head origin main"},
		{name: "set-head changed to another branch of the remote", standing: true, peer: "git push -q origin HEAD:refs/heads/dev && git fetch -q origin && git remote set-head origin dev"},
		{name: "set-head removed", standing: true, removed: true, peer: "git remote set-head -d origin"},
	} {
		t.Run(c.name, func(t *testing.T) {
			script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubDone)
			r := newWrittenRun(t, routedToClaude)
			ignoring(".abcd/.work.local/\n")(r)
			r.git.Commit("base")
			withOrigin(t, r)
			common := filepath.Join(r.repo, ".git")
			if err := os.Remove(filepath.Join(common, "refs", "remotes", "origin", "HEAD")); err != nil && !errors.Is(err, os.ErrNotExist) {
				t.Fatal(err)
			}
			if c.standing {
				r.git.Git("remote", "set-head", "origin", "main")
			}
			peer := laneFor(t, r, "lane")
			if c.linked {
				r.w.Repo = laneFor(t, r, "mine")
			}
			stubPeer(t, script, 1, r.git.Env(), stubCmd{Dir: peer, Script: c.peer})
			res, err := r.w.Run(context.Background())
			if err != nil {
				t.Fatalf("err = %v, want the interview run to its outcome past a peer's set-head", err)
			}
			if len(res.Changed) != 0 || len(r.asked) != 1 || len(r.finished) != 1 {
				t.Fatalf("changed %v, asked %d, finished %d", res.Changed, len(r.asked), len(r.finished))
			}
			b, err := os.ReadFile(filepath.Join(common, "refs", "remotes", "origin", "HEAD"))
			if !c.removed && (err != nil || !strings.HasPrefix(string(b), "ref: refs/remotes/origin/")) {
				t.Fatalf("refs/remotes/origin/HEAD = %q, %v; the peer wrote no loose symbolic ref, so the case proves nothing", b, err)
			}
		})
	}
}

// TestARepositoryKeepingItsRefsInAReftableIsRefused (iss-2611012144422634): a
// reftable keeps every ref, symbolic ones included, in binary tables, and
// git's listings drop a symbolic ref whose target does not exist, so the
// reading cannot tell a symbolic ref made outside its remote: it refuses,
// naming the reftable, rather than read the refs as if they were loose.
func TestARepositoryKeepingItsRefsInAReftableIsRefused(t *testing.T) {
	dir := t.TempDir()
	init := exec.Command("git", "-C", dir, "init", "-q", "--ref-format=reftable")
	init.Env = gittest.Env(t)
	if out, err := init.CombinedOutput(); err != nil {
		t.Skipf("this git keeps no reftable: %v (%s)", err, out)
	}
	_, err := readTree(dir, "")
	if err == nil || !strings.Contains(err.Error(), "reftable") {
		t.Fatalf("err = %v, want the reading refused naming the reftable", err)
	}
}

// TestARemotesHeadStaysInsideOnlyNamingARefTheGateCounts: a remote's HEAD
// naming a branch of its own remote is left out only when the pre-push hook
// would itself count that branch, outside every receipt-less prefix, in any
// case: the hook judges the symbolic ref by its own name, so one naming a
// receipt-less branch would count a commit pushed there first with no
// preflight as already on a remote.
func TestARemotesHeadStaysInsideOnlyNamingARefTheGateCounts(t *testing.T) {
	for target, want := range map[string]bool{
		"refs/remotes/origin/main":                true,
		"refs/remotes/origin/feat/x":              true,
		"refs/remotes/origin/abcd-wip/x":          false,
		"refs/remotes/origin/feat/ABCD-WIP/x":     false,
		"refs/remotes/origin/abcd-reserved/itd-1": false,
		"refs/remotes/origin/Abcd-Status/someone": false,
		"refs/remotes/upstream/main":              false,
		"refs/heads/main":                         false,
	} {
		if got := symrefInsideRemote("refs/remotes/origin/HEAD", target); got != want {
			t.Errorf("symrefInsideRemote(refs/remotes/origin/HEAD, %s) = %v, want %v", target, got, want)
		}
	}
}

// TestAReftableKeptInTheCommonDirectoryIsRefused: the refusal of a
// repository keeping its refs in a reftable is reached wherever git can make
// one or not (a reftable/ in the common directory is what git reads one by),
// and names no remedy git refuses: `git refs migrate` refuses a repository
// with worktrees, abcd's usual layout.
func TestAReftableKeptInTheCommonDirectoryIsRefused(t *testing.T) {
	r := newWrittenRun(t, routedToClaude)
	r.git.Commit("base")
	if err := os.Mkdir(filepath.Join(r.repo, ".git", "reftable"), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := readTree(r.repo, "")
	if err == nil || !strings.Contains(err.Error(), "reftable") || !strings.Contains(err.Error(), "does not run") {
		t.Fatalf("err = %v, want the reading refused, saying the interview does not run with a reftable", err)
	}
	if strings.Contains(err.Error(), "refs migrate") {
		t.Fatalf("the refusal names git refs migrate, which refuses a repository with worktrees: %v", err)
	}
}

// TestRefsGitReadsElsewhereAreRefused: what git reads of the refs through a
// link, which no reading holds, is refused before the first dispatch, naming
// the link: the common directory's refs/ a link to a directory elsewhere, a
// link under refs/ leading to a directory, and a link on the way from the
// checkout's own branch.
func TestRefsGitReadsElsewhereAreRefused(t *testing.T) {
	for _, c := range []struct {
		name string
		// plant makes the link in the repository's git directory.
		plant func(t *testing.T, r *writtenRun, gitDir string)
		said  string
	}{
		{name: "refs/ a link to a directory elsewhere", plant: func(t *testing.T, r *writtenRun, gitDir string) {
			// A detached HEAD names no branch, so the refusal is refs/'s own.
			r.git.Git("checkout", "-q", "--detach")
			elsewhere := filepath.Join(t.TempDir(), "refs")
			if err := os.Rename(filepath.Join(gitDir, "refs"), elsewhere); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(elsewhere, filepath.Join(gitDir, "refs")); err != nil {
				t.Fatal(err)
			}
		}, said: ".git/refs is not a directory"},
		{name: "a link under refs/ leading to a directory", plant: func(t *testing.T, r *writtenRun, gitDir string) {
			if err := os.Symlink("heads", filepath.Join(gitDir, "refs", "aliased")); err != nil {
				t.Fatal(err)
			}
		}, said: ".git/refs/aliased is a link to a directory"},
		{name: "the own branch's loose ref a link", plant: func(t *testing.T, r *writtenRun, gitDir string) {
			head := r.git.Git("rev-parse", "HEAD")
			real := filepath.Join(gitDir, "main-elsewhere")
			if err := os.WriteFile(real, []byte(head+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			own := filepath.Join(gitDir, "refs", "heads", "main")
			if err := os.Remove(own); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("../../main-elsewhere", own); err != nil {
				t.Fatal(err)
			}
			if got := r.git.Git("rev-parse", "refs/heads/main"); got != head {
				t.Fatalf("git reads the linked branch as %s, not %s", got, head)
			}
		}, said: ".git/refs/heads/main, which the branch this checkout is on leads to, is a link"},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := newWrittenRun(t, routedToClaude)
			r.git.Commit("base")
			if b := r.git.Git("symbolic-ref", "--short", "HEAD"); b != "main" {
				t.Fatalf("the fixture is on %s, not main", b)
			}
			c.plant(t, r, filepath.Join(r.repo, ".git"))
			_, err := readTree(r.repo, "")
			if err == nil || !strings.Contains(err.Error(), c.said) {
				t.Fatalf("err = %v, want the reading refused saying %q", err, c.said)
			}
		})
	}
}

// TestRefsPastTheEntryBoundAreRefused: a common directory's refs/ holding
// more entries than one reading walks is refused, naming the bound and the
// packing that clears it, rather than read in part; one at the bound is
// read.
func TestRefsPastTheEntryBoundAreRefused(t *testing.T) {
	keep := maxLooseRefs
	t.Cleanup(func() { maxLooseRefs = keep })
	r := newWrittenRun(t, routedToClaude)
	r.git.Commit("base")
	for _, b := range []string{"one", "two", "three"} {
		r.git.Git("branch", b)
	}
	n := 0
	if err := filepath.WalkDir(filepath.Join(r.repo, ".git", "refs"), func(string, fs.DirEntry, error) error {
		n++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	maxLooseRefs = n
	if _, err := readTree(r.repo, ""); err != nil {
		t.Fatalf("refs/ at the bound of %d entries is refused: %v", n, err)
	}
	maxLooseRefs = n - 1
	_, err := readTree(r.repo, "")
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("more than %d entries", n-1)) || !strings.Contains(err.Error(), "git pack-refs --all") {
		t.Fatalf("err = %v, want refs/ past the bound of %d refused", err, n-1)
	}
}
