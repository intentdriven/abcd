package dogfood

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// dogfood_routes_test.go — the routes by which an agent in a listed checkout
// moved the build that judges it (iss-2610070547057816, security review). Each
// one rewrites something in the checkout's own .git that the guard lets an
// agent touch: a remote-tracking ref, origin/HEAD, the fetch refspec, the
// remote's URL. None of them may select the build: the reviewed tip is read
// from the remote the person declares in their home.

// settle runs what a hook asks for until it asks for nothing more: Resolve,
// a build when one is asked for, and Resolve again.
func settle(t *testing.T, home, cwd string) Plan {
	t.Helper()
	p := Resolve(home, cwd)
	if p.StartBuild && p.Checkout != nil {
		_, _ = Build(home, *p.Checkout)
		p = Resolve(home, cwd)
	}
	return p
}

// agentCommit makes a commit on a local branch "evil" in the checkout whose
// build would print agent-evil, and returns it.
func (f fixture) agentCommit(t *testing.T) string {
	t.Helper()
	git(t, f.checkout, "checkout", "-q", "-b", "evil")
	write(t, filepath.Join(f.checkout, "cmd", "abcd", "main.go"), mainSource("agent-evil"))
	git(t, f.checkout, "commit", "-q", "-am", "an agent's own guard")
	return git(t, f.checkout, "rev-parse", "HEAD")
}

// assertReviewedRuns fails unless the plan runs a build printing want, and no
// build of the agent's commit exists in the store.
func assertReviewedRuns(t *testing.T, f fixture, p Plan, want, evil string) {
	t.Helper()
	if p.Binary == "" {
		t.Fatalf("the plan names no build after the route: %+v", p)
	}
	if got := run(t, p.Binary); got != want {
		t.Fatalf("the build the plan names printed %q, want the reviewed tip's %q", got, want)
	}
	matches, _ := filepath.Glob(abcdhome.Path(f.home, "dogfood", "*", evil))
	if len(matches) != 0 {
		t.Fatalf("the agent's commit %s was built: %v", short(evil), matches)
	}
}

// TestLocalRoutesNeverSelectTheAgentsCommit: every demonstrated route that
// moves the checkout's idea of origin's tip to an agent's commit leaves the
// plan running the reviewed tip, and builds nothing of the agent's.
func TestLocalRoutesNeverSelectTheAgentsCommit(t *testing.T) {
	routes := []struct {
		name string
		move func(t *testing.T, f fixture, evil string)
	}{
		{"fetch into the remote-tracking ref", func(t *testing.T, f fixture, _ string) {
			git(t, f.checkout, "fetch", "-q", ".", "+refs/heads/evil:refs/remotes/origin/main")
		}},
		{"push a branch and set origin/HEAD to it", func(t *testing.T, f fixture, _ string) {
			git(t, f.checkout, "push", "-q", "origin", "evil")
			git(t, f.checkout, "remote", "set-head", "origin", "evil")
		}},
		{"rewrite the fetch refspec", func(t *testing.T, f fixture, _ string) {
			git(t, f.checkout, "push", "-q", "origin", "evil")
			git(t, f.checkout, "config", "remote.origin.fetch", "+refs/heads/evil:refs/remotes/origin/main")
			git(t, f.checkout, "fetch", "-q", "origin")
		}},
		{"update-ref", func(t *testing.T, f fixture, evil string) {
			git(t, f.checkout, "update-ref", "refs/remotes/origin/main", evil)
		}},
		{"symbolic-ref", func(t *testing.T, f fixture, evil string) {
			git(t, f.checkout, "update-ref", "refs/remotes/origin/evil", evil)
			git(t, f.checkout, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/evil")
		}},
		{"push a branch whose name ends in the reviewed ref", func(t *testing.T, f fixture, _ string) {
			git(t, f.checkout, "push", "-q", "origin", "evil:refs/heads/x/refs/heads/main")
			git(t, f.checkout, "fetch", "-q", "origin")
		}},
		{"set-url to another repository", func(t *testing.T, f fixture, _ string) {
			other := filepath.Join(realDir(t), "other.git")
			git(t, f.checkout, "init", "-q", "--bare", other)
			git(t, f.checkout, "push", "-q", other, "evil:main")
			git(t, f.checkout, "remote", "set-url", "origin", other)
			git(t, f.checkout, "fetch", "-q", "origin")
		}},
	}
	for _, r := range routes {
		t.Run(r.name, func(t *testing.T) {
			f := newFixture(t, "github.com/intentdriven/abcd")
			f.declare(t)
			if p := settle(t, f.home, f.checkout); p.Binary == "" {
				t.Fatalf("premise: the reviewed tip builds and runs: %+v", p)
			}
			evil := f.agentCommit(t)
			r.move(t, f, evil)
			p := settle(t, f.home, f.checkout)
			assertReviewedRuns(t, f, p, "reviewed-1", evil)
			if again := Resolve(f.home, f.checkout); again.StartBuild {
				t.Fatalf("the moved ref asks for a check on every hook, not once: %+v", again)
			}
		})
	}
}

// TestALocalRefCannotSelectAnOlderKeptBuild: the store keeps the builds of
// earlier reviewed tips, and pointing the remote-tracking ref back at one of
// them must not run it in place of the current reviewed tip's.
func TestALocalRefCannotSelectAnOlderKeptBuild(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	f.declare(t)
	first := git(t, f.checkout, "rev-parse", "origin/main")
	if p := settle(t, f.home, f.checkout); p.Binary == "" || run(t, p.Binary) != "reviewed-1" {
		t.Fatalf("premise: the first reviewed tip builds: %+v", p)
	}
	f.land(t, mainSource("reviewed-2"))
	if p := settle(t, f.home, f.checkout); p.Binary == "" || run(t, p.Binary) != "reviewed-2" {
		t.Fatalf("premise: the second reviewed tip builds: %+v", p)
	}
	git(t, f.checkout, "update-ref", "refs/remotes/origin/main", first)
	p := Resolve(f.home, f.checkout)
	if p.Binary == "" || run(t, p.Binary) != "reviewed-2" {
		t.Fatalf("a local ref pointed at an older kept build selected it: %+v", p)
	}
	if p = settle(t, f.home, f.checkout); p.Binary == "" || run(t, p.Binary) != "reviewed-2" {
		t.Fatalf("after the check, the older kept build runs: %+v", p)
	}
	if _, err := os.Lstat(abcdhome.Path(f.home, "dogfood", p.Checkout.RootSHA, first, BinaryName)); err != nil {
		t.Fatalf("premise: the older build is still kept: %v", err)
	}
}

// TestReplaceObjectsNeverReachTheBuild: `git replace` stands another object
// in for a commit everywhere git reads it, archive and ls-tree included. A
// replacement the agent made for the reviewed tip, before its first build,
// must not be what is built.
func TestReplaceObjectsNeverReachTheBuild(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	f.declare(t)
	// A tip that is not the root commit, replaced by a sibling, so the
	// replacement leaves the repository's root, which keys the store, where
	// it was.
	tip := f.land(t, mainSource("reviewed-2"))
	git(t, f.checkout, "checkout", "-q", "--detach", tip+"~1")
	evil := f.agentCommit(t)
	git(t, f.checkout, "replace", tip, evil)
	assertReviewedRuns(t, f, settle(t, f.home, f.checkout), "reviewed-2", evil)
}

// TestAlteredObjectStoreFailsTheBuild: git checks the hash of a commit and its
// root tree when it reads them, but not of a subtree, so a loose object written
// over the reviewed commit's cmd/abcd tree once fed archive and ls-tree a
// different tree. The build refuses instead, and nothing is built.
func TestAlteredObjectStoreFailsTheBuild(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	f.declare(t)
	sub := git(t, f.checkout, "rev-parse", "origin/main:cmd/abcd")
	evil := f.agentCommit(t)
	evilSub := git(t, f.checkout, "rev-parse", evil+":cmd/abcd")
	loose := func(oid string) string { return filepath.Join(f.checkout, ".git", "objects", oid[:2], oid[2:]) }
	body, err := os.ReadFile(loose(evilSub))
	if err != nil {
		t.Fatalf("premise: the agent's tree is a loose object: %v", err)
	}
	if _, err := os.Lstat(loose(sub)); err != nil {
		t.Fatalf("premise: the reviewed subtree is a loose object: %v", err)
	}
	// Unlinked first, so a clone's hard link to the remote's copy is broken
	// rather than written through.
	if err := os.Remove(loose(sub)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(loose(sub), body, 0o444); err != nil {
		t.Fatal(err)
	}
	p := Resolve(f.home, f.checkout)
	if p.Checkout == nil || !p.StartBuild {
		t.Fatalf("Resolve = %+v, want a build asked for", p)
	}
	bin, err := Build(f.home, *p.Checkout)
	if err == nil {
		t.Fatalf("Build over an altered object store succeeded; the build at %s prints %q", bin, run(t, bin))
	}
	if !strings.Contains(err.Error(), "does not hash to its name") {
		t.Fatalf("Build error %q does not say an object failed its hash", err)
	}
	if after := Resolve(f.home, f.checkout); after.Binary != "" {
		t.Fatalf("Resolve after the refused build = %+v, want the installed release", after)
	}
}

// TestUnreachableRemoteRunsTheRelease: a declared remote that cannot be read
// records no tip, the installed release runs, and the note says why; no
// check is asked for again until the re-check bound or a moved ref.
func TestUnreachableRemoteRunsTheRelease(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	f.list(t, f.checkout+"\thttps://absent.example.invalid/abcd.git")
	p := Resolve(f.home, f.checkout)
	if p.Checkout == nil || !p.StartBuild {
		t.Fatalf("Resolve = %+v, want a check asked for", p)
	}
	if _, err := Build(f.home, *p.Checkout); err == nil {
		t.Fatal("Build against an unreachable remote succeeded")
	}
	after := Resolve(f.home, f.checkout)
	if after.Binary != "" || after.StartBuild {
		t.Fatalf("Resolve after the failed check = %+v, want the installed release and no new check yet", after)
	}
	if n := notes(after); !strings.Contains(n, "could not be read") || !strings.Contains(n, "installed release runs") {
		t.Fatalf("the note does not say the remote could not be read and what runs: %q", n)
	}
}

// TestUnreachableRemoteDropsTheEarlierBuild: a check that fails clears the
// verified tip, so the build of an earlier tip stops running and the
// installed release judges until the remote answers again.
func TestUnreachableRemoteDropsTheEarlierBuild(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	f.declare(t)
	if p := settle(t, f.home, f.checkout); p.Binary == "" {
		t.Fatalf("premise: the reviewed tip builds: %+v", p)
	}
	prev := lsRemote
	t.Cleanup(func() { lsRemote = prev })
	lsRemote = func(context.Context, string, string, string) (string, error) {
		return "", errors.New("could not resolve host")
	}
	t.Cleanup(func() { now = time.Now })
	now = func() time.Time { return time.Now().Add(recheckAfter + time.Minute) }
	p := settle(t, f.home, f.checkout)
	if p.Binary != "" || !strings.Contains(notes(p), "could not resolve host") {
		t.Fatalf("Resolve after a failed check = %+v, want the installed release and the reason", p)
	}
}

// TestUnfetchedTipRunsTheRelease: a reviewed tip the checkout does not hold
// yet is recorded and not built; the installed release runs with a note to
// fetch, and the fetch that moves origin/main prompts the build.
func TestUnfetchedTipRunsTheRelease(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	f.declare(t)
	if p := settle(t, f.home, f.checkout); p.Binary == "" {
		t.Fatalf("premise: the reviewed tip builds: %+v", p)
	}
	write(t, filepath.Join(f.upstream, "cmd", "abcd", "main.go"), mainSource("reviewed-2"))
	git(t, f.upstream, "commit", "-q", "-am", "landed, not fetched")
	git(t, f.upstream, "push", "-q", f.remote, "main")
	t.Cleanup(func() { now = time.Now })
	now = func() time.Time { return time.Now().Add(recheckAfter + time.Minute) }
	p := settle(t, f.home, f.checkout)
	if p.Binary != "" || p.StartBuild {
		t.Fatalf("Resolve with an unfetched reviewed tip = %+v, want the installed release and no new check", p)
	}
	if n := notes(p); !strings.Contains(n, "git fetch") || !strings.Contains(n, "installed release runs") {
		t.Fatalf("the note does not say to fetch and what runs: %q", n)
	}
	git(t, f.checkout, "fetch", "-q", "origin")
	if p = settle(t, f.home, f.checkout); p.Binary == "" || run(t, p.Binary) != "reviewed-2" {
		t.Fatalf("after the fetch, Resolve = %+v, want the new tip's build", p)
	}
}

// TestResolveNeverReadsTheNetwork: the hook's half asks nothing of the remote,
// in any state the store can be in; only Build does.
func TestResolveNeverReadsTheNetwork(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	f.declare(t)
	calls := 0
	forge := lsRemote
	t.Cleanup(func() { lsRemote = forge })
	lsRemote = func(ctx context.Context, dir, url, branch string) (string, error) {
		calls++
		return forge(ctx, dir, url, branch)
	}
	resolve := func(state string) Plan {
		t.Helper()
		before := calls
		p := Resolve(f.home, f.checkout)
		if calls != before {
			t.Fatalf("Resolve %s read the remote", state)
		}
		return p
	}
	p := resolve("with no record")
	if _, err := Build(f.home, *p.Checkout); err != nil || calls != 1 {
		t.Fatalf("premise: Build reads the remote once: %v, %d calls", err, calls)
	}
	resolve("with a verified build")
	f.land(t, mainSource("reviewed-2"))
	resolve("after origin/main moved")
	t.Cleanup(func() { now = time.Now })
	now = func() time.Time { return time.Now().Add(recheckAfter + time.Minute) }
	resolve("with a stale record")
	f.list(t, f.checkout+"\tssh://192.0.2.1/unreachable.git")
	start := time.Now()
	resolve("with an unreachable remote declared")
	if took := time.Since(start); took > 10*time.Second {
		t.Fatalf("Resolve with an unreachable remote declared took %s", took)
	}
}
