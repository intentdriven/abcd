package loop

// remoteid_start_test.go holds a drive's start-time comparison of an in-tree
// hooks copy with origin's tip and the branches beside it that fail toward
// stopping (iss-2611131308266006, fix round 1, rulings 2 to 5): which
// difference a refusal names, a committed hook linked elsewhere in the tree or
// outside it, an origin out of reach, a tip not held, a hooks directory git
// cannot name or read, and the walk's and the pin's own refusals. A test whose
// comment says it was watched failing failed at the round's base; the others
// hold branches that already refused there and that no test held.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// startRefused fails the test unless err is the drive's refusal to start
// naming want, with nothing held.
func startRefused(t *testing.T, tm *terminal, err error, want string) *Refusal {
	t.Helper()
	r := mustRefusal(t, err)
	if !strings.Contains(r.Reason, "the drive cannot start") || !strings.Contains(r.Reason, want) {
		t.Fatalf("the drive refuses to start naming %q: %+v", want, r)
	}
	if tm.o.Options.remote != nil {
		t.Fatal("nothing is held")
	}
	return r
}

// pushLinkedHook commits a pre-push hook in repo's .githooks that is a link
// to target, and lands it on origin.
func pushLinkedHook(t *testing.T, repo *gittest.Repo, target string) {
	t.Helper()
	relinkHook(t, filepath.Join(repo.Root(), ".githooks", "pre-push"), target)
	repo.Commit("a hook linked elsewhere")
	repo.Git("push", "-q", "origin", "main")
	repo.Git("fetch", "-q", "origin")
}

// TestADriveStartSaysWhichDifferenceItMet: a checkout behind origin's tip, a
// file git does not track in the hooks directory, and committed hooks that
// link outside the repository each refuse the start with a reason and a
// remedy of their own; a committed hook linked to a file elsewhere in the
// repository starts. Watched failing: the first three with one reason for
// all, the fourth refused.
func TestADriveStartSaysWhichDifferenceItMet(t *testing.T) {
	t.Run("behind the tip", func(t *testing.T) {
		repo, _ := behindCheckout(t)
		tm, err := startDrive(t, repo)
		r := startRefused(t, tm, err, "the checkout is behind origin's tip")
		if !strings.Contains(r.Remedy, "git pull --ff-only") {
			t.Fatalf("the remedy brings the checkout up to the tip: %+v", r)
		}
	})
	t.Run("a file git does not track", func(t *testing.T) {
		repo, _, _ := hookedOrigin(t)
		writeMode(t, filepath.Join(repo.Root(), ".githooks", "notes.txt"), "x\n", 0o644)
		tm, err := startDrive(t, repo)
		r := startRefused(t, tm, err, "files git does not track")
		if !strings.Contains(r.Reason, "notes.txt") || !strings.Contains(r.Remedy, "move the files git does not track") {
			t.Fatalf("the refusal names the file and the remedy moves it: %+v", r)
		}
	})
	t.Run("committed hooks that link outside the repository", func(t *testing.T) {
		repo, _, _ := hookedOrigin(t)
		outside := filepath.Join(t.TempDir(), "hook.sh")
		writeMode(t, outside, "#!/bin/sh\nexit 0\n", 0o755)
		pushLinkedHook(t, repo, outside)
		tm, err := startDrive(t, repo)
		r := startRefused(t, tm, err, "committed hooks in .githooks link outside it")
		if !strings.Contains(r.Reason, "a drive runs no lane there") || !strings.Contains(r.Remedy, "the repository's own") ||
			strings.Contains(r.Remedy, "commit") && strings.Contains(r.Remedy, "put the hooks back") {
			t.Fatalf("the refusal says the link is the repository's and no lane runs there: %+v", r)
		}
	})
	t.Run("a committed hook linked elsewhere in the repository", func(t *testing.T) {
		repo, _, _ := hookedOrigin(t)
		repo.Write("scripts/hook.sh", "#!/bin/sh\nexit 0\n")
		pushLinkedHook(t, repo, "../scripts/hook.sh")
		tm, err := startDrive(t, repo)
		if err != nil || tm.o.Options.remote == nil {
			t.Fatalf("the link is proved by the tip's file and the drive starts: %v", err)
		}
	})
	t.Run("the file a committed link points at differs", func(t *testing.T) {
		repo, _, _ := hookedOrigin(t)
		repo.Write("scripts/hook.sh", "#!/bin/sh\nexit 0\n")
		pushLinkedHook(t, repo, "../scripts/hook.sh")
		repo.Write("scripts/hook.sh", "#!/bin/sh\ntouch x\n")
		tm, err := startDrive(t, repo)
		startRefused(t, tm, err, "points at scripts/hook.sh")
	})
}

// TestADriveStartWaitsOnAnOriginItCannotReach: an origin the start cannot
// reach for a network reason is waited on as a lane's ask of origin is, and
// asked again; the start is not refused. Watched failing: the start was
// refused.
func TestADriveStartWaitsOnAnOriginItCannotReach(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	was := remoteHeadTip
	t.Cleanup(func() { remoteHeadTip = was })
	asked := 0
	remoteHeadTip = func(ctx context.Context, git func(context.Context, string, ...string) (string, error), root, remote string) (string, string, error) {
		asked++
		return "", "", errors.New("fatal: unable to access 'https://origin.invalid/x.git/': Could not resolve host: origin.invalid")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var said []string
	tm := newTerminal(repo.Root(), TerminalOptions{Stages: DefaultStages(), Say: func(s string) { said = append(said, s) },
		Sleep: func(context.Context, time.Duration) { cancel() }})
	res, err := tm.run(ctx)
	if err != nil {
		t.Fatalf("the start waits rather than refuses: %v", err)
	}
	if res.Ended != TerminalKilled || asked != 1 || tm.o.Options.remote != nil {
		t.Fatalf("the start waited once and was ended while waiting: %+v asked %d", res, asked)
	}
	if !strings.Contains(strings.Join(said, "\n"), "lost the network") {
		t.Fatalf("the wait says why: %q", said)
	}
}

// TestADriveStartRefusesWhatItCannotCompare: a tip origin reports that this
// repository does not hold, an origin that answers with an error that is not
// the network's, a hooks directory git cannot name, and an in-tree hooks
// directory that cannot be read each refuse the start, with nothing held.
// Watched failing: a hooks directory git cannot name started the drive, and a
// tip not held was refused with a remedy that did not name the fetch. The
// other two refused at the round's base too.
func TestADriveStartRefusesWhatItCannotCompare(t *testing.T) {
	ask := func(t *testing.T, tip string, err error) {
		t.Helper()
		was := remoteHeadTip
		t.Cleanup(func() { remoteHeadTip = was })
		remoteHeadTip = func(context.Context, func(context.Context, string, ...string) (string, error), string, string) (string, string, error) {
			return "main", tip, err
		}
	}
	t.Run("a tip not held", func(t *testing.T) {
		repo, _, _ := hookedOrigin(t)
		ask(t, strings.Repeat("1", 40), nil)
		tm, err := startDrive(t, repo)
		r := startRefused(t, tm, err, "does not hold yet")
		if !strings.Contains(r.Remedy, "git fetch origin") {
			t.Fatalf("the remedy fetches origin: %+v", r)
		}
	})
	t.Run("an origin that refuses", func(t *testing.T) {
		repo, _, _ := hookedOrigin(t)
		ask(t, "", errors.New("fatal: 'origin' does not appear to be a git repository"))
		tm, err := startDrive(t, repo)
		startRefused(t, tm, err, "could not ask origin")
	})
	t.Run("a hooks directory git cannot name", func(t *testing.T) {
		repo, _, _ := hookedOrigin(t)
		repo.Git("config", "core.hooksPath", "~no-such-user-of-this-machine/hooks")
		tm, err := startDrive(t, repo)
		startRefused(t, tm, err, "")
	})
	t.Run("an in-tree hooks directory that cannot be read", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root reads a directory whatever its mode")
		}
		repo, _, _ := hookedOrigin(t)
		dir := filepath.Join(repo.Root(), ".githooks")
		chmod(t, dir, 0)
		t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
		tm, err := startDrive(t, repo)
		startRefused(t, tm, err, "cannot be read")
	})
}

// TestADriveRefusesItsNextLaneOnAHooksTreeGrownPastTheBound: a hooks tree
// grown past maxHookEntries entries between two lanes refuses the second
// lane, naming the bound.
func TestADriveRefusesItsNextLaneOnAHooksTreeGrownPastTheBound(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	lane, err := driveSecondLane(t, repo, func() {
		many := filepath.Join(repo.Root(), ".githooks", "many")
		mkdirs(t, many)
		for i := 0; i <= maxHookEntries; i++ {
			writeMode(t, filepath.Join(many, fmt.Sprintf("h%05d", i)), "x\n", 0o644)
		}
	})
	worktreeRefused(t, err, fmt.Sprintf("more than %d entries", maxHookEntries))
	absent(t, lane, "no second lane worktree is made")
}

// TestALinksPlaceIsJudgedAsWrittenAndAsResolved: a link whose absolute target
// is written in resolved form stays in a root reached through a linked parent
// (watched failing: it was refused); one whose text stays in the root but
// that resolves outside it, through a directory in the root that is a link,
// does not.
func TestALinksPlaceIsJudgedAsWrittenAndAsResolved(t *testing.T) {
	real, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	hooks := filepath.Join(real, "repo", ".githooks")
	mkdirs(t, hooks)
	writeMode(t, filepath.Join(hooks, "y"), "y\n", 0o755)
	parent := filepath.Join(t.TempDir(), "linked")
	if err := os.Symlink(real, parent); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(parent, "repo", ".githooks")
	if !linkStaysIn(root, filepath.Join(root, "x"), filepath.Join(hooks, "y")) {
		t.Fatal("a target written in resolved form stays in the root reached through a linked parent")
	}
	outside := t.TempDir()
	writeMode(t, filepath.Join(outside, "y"), "y\n", 0o755)
	if err := os.Symlink(outside, filepath.Join(hooks, "sub")); err != nil {
		t.Fatal(err)
	}
	relinkHook(t, filepath.Join(hooks, "x"), "sub/y")
	if linkStaysIn(root, filepath.Join(root, "x"), "sub/y") {
		t.Fatal("a link that resolves outside the root through a linked directory leaves it")
	}
}

// TestAPinRefusesASubmoduleInItsHooksDirectory: a submodule entry the commit
// holds in its hooks directory is not proved; it is refused, naming its mode.
func TestAPinRefusesASubmoduleInItsHooksDirectory(t *testing.T) {
	dir := t.TempDir()
	mkdirs(t, filepath.Join(dir, "sub"))
	_, err := provenHooks("salt", dir, map[string]hookTreeEntry{"sub": {mode: "160000", oid: strings.Repeat("a", 40), size: -1}}, nil)
	if err == nil || !strings.Contains(err.Error(), "mode 160000") {
		t.Fatalf("the submodule is refused naming its mode: %v", err)
	}
}

// TestTheHooksWalkFollowsEachLinkedDirectoryOnce: in the walk a link loop
// ends, each link recorded as one that resolves to nothing; a link to a link
// to a directory outside the hooks directory has that directory's files
// recorded under the first link's path; and a second link to the same
// directory records the link alone.
func TestTheHooksWalkFollowsEachLinkedDirectoryOnce(t *testing.T) {
	dir, outside := t.TempDir(), t.TempDir()
	writeMode(t, filepath.Join(outside, "f"), "f\n", 0o644)
	for name, target := range map[string]string{"a": "b", "b": "a", "l1": "l2", "l2": outside, "l3": outside} {
		if err := os.Symlink(target, filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	m := map[string]string{}
	if _, over := hashDir(m, "salt", dir); over {
		t.Fatal("the walk ends under the bound")
	}
	for _, name := range []string{"a", "b"} {
		if m[filepath.Join(dir, name)] != saltedHash("salt", "symlink", map[string]string{"a": "b", "b": "a"}[name], "unresolved") {
			t.Fatalf("the loop's link %s is recorded as resolving to nothing", name)
		}
	}
	if _, ok := m[filepath.Join(dir, "l1", "f")]; !ok {
		t.Fatalf("the directory reached through a link to a link is recorded under the first link: %v", m)
	}
	for _, at := range []string{filepath.Join(dir, "l2", "f"), filepath.Join(dir, "l3", "f")} {
		if _, ok := m[at]; ok {
			t.Fatalf("a directory already walked is not walked again under %s", at)
		}
	}
}
