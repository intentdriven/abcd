package loop

// remoteid_holds_test.go holds what a drive holds across its lanes and proves
// as it starts (iss-2611131308266006): the checkout's hooks tree at every
// depth, a symlinked hook by what it resolves to, the bound on the tree, every
// watched setting at every scope with info/attributes, and an in-tree hooks
// copy compared with origin's tip as the drive starts. Each test was watched
// failing on the default branch before this change, except where its comment
// or a subtest's says otherwise.

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/gittest"
)

// driveSecondLane reads what a drive holds in repo, opens the drive's first
// lane, applies change, and runs the second lane's worktree stage, returning
// the second lane's path and its error.
func driveSecondLane(t *testing.T, repo *gittest.Repo, change func()) (string, error) {
	t.Helper()
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	start, err := Start(repo.Root(), "itd-10", Options{noReserve: true})
	if err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	c := Context{RepoRoot: repo.Root(), RunDir: runRel(st.RunID), State: st, remote: id}
	first := st.Lanes[0]
	if _, err := worktreeStage(c, &first); err != nil {
		t.Fatalf("the drive's first lane opens: %v", err)
	}
	change()
	second := st.Lanes[0]
	second.ID = "lane-2"
	lw, err := laneWorktree(repo.Root(), st.RunID, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = worktreeStage(c, &second)
	return lw.Path, err
}

// mkdirs makes dir and its parents.
func mkdirs(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

// startDrive runs a drive's start in repo, its context already ended, so it
// returns once it has read and proved what it holds.
func startDrive(t *testing.T, repo *gittest.Repo) (*terminal, error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tm := newTerminal(repo.Root(), TerminalOptions{Stages: DefaultStages(), Say: func(string) {}})
	_, err := tm.run(ctx)
	return tm, err
}

// TestADriveHoldsTheWatchedSettingsAtEveryScopeAcrossLanes: a content filter
// and an attributes line set in the shared git directory during lane N, the
// attributes line alone, and a watched key set only in the person's global
// settings, each refuse lane N+1 at its worktree stage, naming what changed,
// before its add: no worktree is made and the filter writes no marker.
func TestADriveHoldsTheWatchedSettingsAtEveryScopeAcrossLanes(t *testing.T) {
	for name, tc := range map[string]struct {
		global bool
		change func(t *testing.T, repo *gittest.Repo, global, marker string)
		want   string
	}{
		"a filter and an attributes line in the shared git directory": {
			change: func(t *testing.T, repo *gittest.Repo, _, marker string) {
				repo.Git("config", "filter.planted.clean", "touch '"+marker+"'; cat")
				writeMode(t, filepath.Join(repo.Root(), ".git", "info", "attributes"), "* filter=planted\n", 0o644)
			},
			want: "filter.planted.clean",
		},
		"an attributes line alone": {
			change: func(t *testing.T, repo *gittest.Repo, _, _ string) {
				writeMode(t, filepath.Join(repo.Root(), ".git", "info", "attributes"), "* filter=lfs\n", 0o644)
			},
			want: "attributes (added)",
		},
		"a filter in the person's global settings": {
			global: true,
			change: func(t *testing.T, _ *gittest.Repo, global, marker string) {
				writeMode(t, global, "[filter \"planted\"]\n\tclean = touch '"+marker+"'; cat\n", 0o644)
			},
			want: "filter.planted.clean",
		},
		"an editor in the person's global settings": {
			global: true,
			change: func(t *testing.T, _ *gittest.Repo, global, _ string) {
				writeMode(t, global, "[core]\n\teditor = /bin/true\n", 0o644)
			},
			want: "core.editor",
		},
	} {
		t.Run(name, func(t *testing.T) {
			global := filepath.Join(t.TempDir(), "global.gitconfig")
			if tc.global {
				writeMode(t, global, "", 0o644)
				t.Setenv("GIT_CONFIG_GLOBAL", global)
			}
			repo, _, _ := hookedOrigin(t)
			marker := filepath.Join(t.TempDir(), "filter-ran")
			lane, err := driveSecondLane(t, repo, func() { tc.change(t, repo, global, marker) })
			worktreeRefused(t, err, tc.want)
			absent(t, lane, "no second lane worktree is made")
			absent(t, marker, "no filter ran")
		})
	}
}

// TestADriveHoldsAHookInASubdirectoryAndALinkedHookByWhatItResolvesTo: a file
// in a subdirectory of the hooks directory changed between two lanes, and the
// bytes of the file a hook's symlink resolves to outside the directory
// changed, each refuse the second lane naming the entry. A link to a
// directory outside the hooks directory has that directory's files held under
// the link's path.
func TestADriveHoldsAHookInASubdirectoryAndALinkedHookByWhatItResolvesTo(t *testing.T) {
	for name, tc := range map[string]struct {
		setup  func(t *testing.T, repo *gittest.Repo, outside string)
		change func(t *testing.T, repo *gittest.Repo, outside string)
		want   string
	}{
		"a file in a subdirectory": {
			change: func(t *testing.T, repo *gittest.Repo, _ string) {
				repo.Write(".githooks/lib/common.sh", "shared=2\n")
			},
			want: "lib/common.sh (changed)",
		},
		"the file a linked hook resolves to": {
			setup: func(t *testing.T, repo *gittest.Repo, outside string) {
				writeMode(t, filepath.Join(outside, "hook.sh"), "#!/bin/sh\nexit 0\n", 0o755)
				relinkHook(t, filepath.Join(repo.Root(), ".githooks", "post-checkout"), filepath.Join(outside, "hook.sh"))
			},
			change: func(t *testing.T, _ *gittest.Repo, outside string) {
				writeMode(t, filepath.Join(outside, "hook.sh"), "#!/bin/sh\ntouch x\n", 0o755)
			},
			want: "post-checkout (changed)",
		},
		"a file in a directory a link resolves to": {
			setup: func(t *testing.T, repo *gittest.Repo, outside string) {
				writeMode(t, filepath.Join(outside, "x.sh"), "x=1\n", 0o644)
				relinkHook(t, filepath.Join(repo.Root(), ".githooks", "shared"), outside)
			},
			change: func(t *testing.T, _ *gittest.Repo, outside string) {
				writeMode(t, filepath.Join(outside, "x.sh"), "x=2\n", 0o644)
			},
			want: "shared/x.sh (changed)",
		},
	} {
		t.Run(name, func(t *testing.T) {
			repo, _, _ := hookedOrigin(t)
			outside := t.TempDir()
			if tc.setup != nil {
				tc.setup(t, repo, outside)
			}
			lane, err := driveSecondLane(t, repo, func() { tc.change(t, repo, outside) })
			worktreeRefused(t, err, tc.want)
			absent(t, lane, "no second lane worktree is made")
		})
	}
}

// TestALanesPinHoldsItsHooksTreeAtEveryDepth: a file in a subdirectory of the
// lane's hooks directory that differs from the commit's as the lane opens is
// refused at its worktree stage; in a lane already pinned, one changed stops
// the next step at the stage config, naming it; and a committed hook that is
// a symlink leaving the hooks directory is refused as the lane opens, naming
// it.
func TestALanesPinHoldsItsHooksTreeAtEveryDepth(t *testing.T) {
	t.Run("a subdirectory's file differs as the lane opens", func(t *testing.T) {
		repo, _, _ := hookedOrigin(t)
		was := addWorktree
		t.Cleanup(func() { addWorktree = was })
		addWorktree = func(root string, o worktree.AddOptions) (worktree.Added, error) {
			added, err := was(root, o)
			if err == nil {
				writeMode(t, filepath.Join(added.Path, ".githooks", "lib", "common.sh"), "shared=2\n", 0o644)
			}
			return added, err
		}
		runID, path, err := openHookedLane(t, repo)
		worktreeRefused(t, err, filepath.Join("lib", "common.sh")+" differs from the commit's file")
		if _, ok := pinned(t, repo, runID, path); ok {
			t.Fatal("nothing is pinned")
		}
	})
	t.Run("a subdirectory's file changed in a pinned lane", func(t *testing.T) {
		repo, _, _ := hookedOrigin(t)
		runID, path, err := openHookedLane(t, repo)
		if err != nil {
			t.Fatal(err)
		}
		writeMode(t, filepath.Join(path, ".githooks", "lib", "common.sh"), "shared=2\n", 0o644)
		_, err = advance(repo.Root(), runID, DefaultStages(), Options{})
		r := mustRefusal(t, err)
		if r.Stage != StageConfig || !strings.Contains(r.Reason, filepath.Join("lib", "common.sh")) {
			t.Fatalf("the step is refused naming the lane's file: %+v", r)
		}
	})
	// The three cases below are ruling 2 of the fix round: the first two were
	// watched failing at the round's base, where every link leaving the hooks
	// directory was refused at the pin; the third, where the refusal's reason
	// and remedy changed.
	linked := func(t *testing.T, target string) *gittest.Repo {
		t.Helper()
		repo, _, _ := hookedOrigin(t)
		repo.Write("scripts/hook.sh", "#!/bin/sh\nexit 0\n")
		mkdirs(t, filepath.Join(repo.Root(), "scripts", "lib"))
		repo.Write("scripts/lib/x.sh", "x=1\n")
		relinkHook(t, filepath.Join(repo.Root(), ".githooks", "post-commit"), target)
		repo.Commit("a hook linked elsewhere")
		repo.Git("push", "-q", "origin", "main")
		repo.Git("fetch", "-q", "origin")
		return repo
	}
	t.Run("a hook linked to a file elsewhere in the tree is proved by the commit's file there", func(t *testing.T) {
		repo := linked(t, "../scripts/hook.sh")
		runID, path, err := openHookedLane(t, repo)
		if err != nil {
			t.Fatalf("the lane opens: %v", err)
		}
		wt, ok := pinned(t, repo, runID, path)
		if !ok || wt.Hooks[filepath.Join(path, ".githooks", "post-commit")] == "" {
			t.Fatalf("the linked hook is pinned: %+v %v", wt.Hooks, ok)
		}
		if err := configTripwire(repo.Root(), runID); err != nil {
			t.Fatalf("the pinned lane reads as unchanged at the next check: %v", err)
		}
	})
	t.Run("a committed link reached through a directory that is a link in the lane", func(t *testing.T) {
		repo := linked(t, "../scripts/hook.sh")
		elsewhere := t.TempDir()
		writeMode(t, filepath.Join(elsewhere, "hook.sh"), "#!/bin/sh\nexit 0\n", 0o755)
		was := addWorktree
		t.Cleanup(func() { addWorktree = was })
		addWorktree = func(root string, o worktree.AddOptions) (worktree.Added, error) {
			added, err := was(root, o)
			if err == nil {
				removeAll(t, filepath.Join(added.Path, "scripts"))
				if err := os.Symlink(elsewhere, filepath.Join(added.Path, "scripts")); err != nil {
					t.Fatal(err)
				}
			}
			return added, err
		}
		_, _, err := openHookedLane(t, repo)
		worktreeRefused(t, err, "through a directory that is not the commit's")
	})
	t.Run("the file a committed link points at differs in the lane", func(t *testing.T) {
		repo := linked(t, "../scripts/hook.sh")
		was := addWorktree
		t.Cleanup(func() { addWorktree = was })
		addWorktree = func(root string, o worktree.AddOptions) (worktree.Added, error) {
			added, err := was(root, o)
			if err == nil {
				writeMode(t, filepath.Join(added.Path, "scripts", "hook.sh"), "#!/bin/sh\nexit 1\n", 0o644)
			}
			return added, err
		}
		_, _, err := openHookedLane(t, repo)
		worktreeRefused(t, err, "points at scripts/hook.sh, which differs from the commit's file")
	})
	t.Run("a committed hook linked to a directory elsewhere in the tree", func(t *testing.T) {
		repo := linked(t, "../scripts/lib")
		runID, path, err := openHookedLane(t, repo)
		if err != nil {
			t.Fatalf("the lane opens: %v", err)
		}
		if wt, ok := pinned(t, repo, runID, path); !ok || wt.Hooks[filepath.Join(path, ".githooks", "post-commit", "x.sh")] == "" {
			t.Fatalf("the linked directory's file is pinned under the link: %+v", wt.Hooks)
		}
		if err := configTripwire(repo.Root(), runID); err != nil {
			t.Fatalf("the pin records the linked directory as the fingerprint's walk does: %v", err)
		}
	})
	t.Run("a committed hook linked outside the tree", func(t *testing.T) {
		outside := filepath.Join(t.TempDir(), "hook.sh")
		writeMode(t, outside, "#!/bin/sh\nexit 0\n", 0o755)
		repo := linked(t, outside)
		_, _, err := openHookedLane(t, repo)
		r := worktreeRefused(t, err, "post-commit is a symlink whose target leaves the working tree")
		if !strings.Contains(r.Reason, "committed hooks link outside") || strings.Contains(r.Remedy, "remove the lane's worktree") {
			t.Fatalf("the refusal says the commit holds the link and does not send the person to remake the lane: %+v", r)
		}
	})
}

// TestADriveDoesNotStartPastTheHooksBound: a hooks tree past maxHookEntries
// entries, here in a subdirectory, refuses the drive's start, as the lane's
// pin refuses one, and nothing is held.
func TestADriveDoesNotStartPastTheHooksBound(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	many := filepath.Join(repo.Root(), ".githooks", "many")
	mkdirs(t, many)
	for i := 0; i <= maxHookEntries; i++ {
		writeMode(t, filepath.Join(many, fmt.Sprintf("h%05d", i)), "x\n", 0o644)
	}
	tm, err := startDrive(t, repo)
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "the drive cannot start") || !strings.Contains(r.Reason, fmt.Sprint(maxHookEntries)) {
		t.Fatalf("the drive refuses to start past the bound: %+v", r)
	}
	if tm.o.Options.remote != nil {
		t.Fatal("nothing is held")
	}
}

// TestADriveProvesAnInTreeHooksCopyAgainstOriginsTipAsItStarts: where the
// hooks directory is the checkout's own in-tree copy and the repository has
// an origin, a copy that differs from origin's tip (a hook edited, a file
// added in a subdirectory) refuses the drive's start, naming each file. A
// hooks directory outside the tree has nothing to be proved against, and the
// drive starts; so does a repository with no origin.
func TestADriveProvesAnInTreeHooksCopyAgainstOriginsTipAsItStarts(t *testing.T) {
	t.Run("an in-tree copy that differs", func(t *testing.T) {
		repo, _, _ := hookedOrigin(t)
		writeHook(t, repo, "pre-commit", "#!/bin/sh\ntouch ran\n")
		repo.Write(".githooks/lib/extra.sh", "extra=1\n")
		tm, err := startDrive(t, repo)
		r := mustRefusal(t, err)
		if !strings.Contains(r.Reason, "the drive cannot start") || !strings.Contains(r.Reason, "pre-commit") ||
			!strings.Contains(r.Reason, filepath.Join("lib", "extra.sh")) {
			t.Fatalf("the drive refuses to start naming each file: %+v", r)
		}
		if tm.o.Options.remote != nil {
			t.Fatal("nothing is held")
		}
	})
	// This subtest and the next could not fail on the default branch: no
	// comparison existed there, and the directory the repository's own
	// settings named was already held. They hold what the comparison leaves
	// alone.
	t.Run("a hooks directory outside the tree", func(t *testing.T) {
		repo, _, _ := hookedOrigin(t)
		outside := t.TempDir()
		writeMode(t, filepath.Join(outside, "pre-commit"), "#!/bin/sh\ntouch ran\n", 0o755)
		repo.Git("config", "core.hooksPath", outside)
		tm, err := startDrive(t, repo)
		if err != nil || tm.o.Options.remote == nil {
			t.Fatalf("the drive starts and holds the outside directory: %v", err)
		}
		if _, ok := tm.o.Options.remote.hooks[filepath.Join(outside, "pre-commit")]; !ok {
			t.Fatalf("the outside directory is the one held: %d entries", len(tm.o.Options.remote.hooks))
		}
	})
	t.Run("no origin", func(t *testing.T) {
		repo, _, _ := hookedOrigin(t)
		repo.Git("remote", "remove", "origin")
		writeHook(t, repo, "pre-commit", "#!/bin/sh\ntouch ran\n")
		if _, err := startDrive(t, repo); err != nil {
			t.Fatalf("with no origin nothing is proved and the drive starts: %v", err)
		}
	})
}

// TestAnOlderRunsFingerprintStopsOnceAndAFreshRunStartsClean: a fingerprint
// an older build recorded (no entries below the hooks directory's top level,
// a symlink by its target's text alone) reads as changed at the run's next
// check, which stops it with the reason the page gives; once the person
// accepts it the run moves. A fresh run records the tree whole and its next
// checks pass.
func TestAnOlderRunsFingerprintStopsOnceAndAFreshRunStartsClean(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	start, err := Start(repo.Root(), "itd-10", Options{noReserve: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := configTripwire(repo.Root(), start.RunID); err != nil {
		t.Fatalf("a fresh run records its fingerprint: %v", err)
	}
	for range 2 {
		if err := configTripwire(repo.Root(), start.RunID); err != nil {
			t.Fatalf("a fresh run's next checks pass: %v", err)
		}
	}
	path := fingerprintPath(repo, start.RunID)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fp configFingerprint
	if err := json.Unmarshal(data, &fp); err != nil {
		t.Fatal(err)
	}
	hooks := filepath.Join(repo.Root(), ".githooks")
	for p := range fp.Files {
		if rel, err := filepath.Rel(hooks, p); err == nil && strings.Contains(rel, string(filepath.Separator)) {
			delete(fp.Files, p)
		}
	}
	fp.Files[filepath.Join(hooks, "pre-merge-commit")] = saltedHash(fp.Salt, "symlink", "pre-commit")
	older, err := json.Marshal(fp)
	if err != nil {
		t.Fatal(err)
	}
	writeMode(t, path, string(older), 0o600)

	r := mustRefusal(t, configTripwire(repo.Root(), start.RunID))
	if r.Stage != StageConfig || !strings.Contains(r.Reason, "the repository's git configuration changed during the run") ||
		!strings.Contains(r.Reason, filepath.Join("lib", "common.sh")+" (added)") || !strings.Contains(r.Reason, "pre-merge-commit (changed)") {
		t.Fatalf("the older run stops once with the page's reason: %+v", r)
	}
	if _, err := AcceptConfigChange(repo.Root(), start.RunID, Options{}); err != nil {
		t.Fatal(err)
	}
	if err := configTripwire(repo.Root(), start.RunID); err != nil {
		t.Fatalf("accepted, the run moves: %v", err)
	}
}

// TestALanePinnedByAnOlderBuildStopsOnceAndMovesWhenAccepted: a lane an older
// build pinned (its hooks recorded with no entry below the hooks directory's
// top level) reads as changed at the run's next check, which stops it with
// the reason the page gives, naming the lane's nested entry; once the person
// accepts it the run moves. Not watched failing: it passes at the round's
// base too; it holds what the page states for such a lane.
func TestALanePinnedByAnOlderBuildStopsOnceAndMovesWhenAccepted(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	runID, path, err := openHookedLane(t, repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := configTripwire(repo.Root(), runID); err != nil {
		t.Fatalf("the pinned lane passes its next check: %v", err)
	}
	file := fingerprintPath(repo, runID)
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var fp configFingerprint
	if err := json.Unmarshal(data, &fp); err != nil {
		t.Fatal(err)
	}
	wt, ok := fp.Worktrees[path]
	if !ok {
		t.Fatal("the lane is pinned")
	}
	hooks := filepath.Join(path, ".githooks")
	for p := range wt.Hooks {
		if rel, err := filepath.Rel(hooks, p); err == nil && strings.Contains(rel, string(filepath.Separator)) {
			delete(wt.Hooks, p)
		}
	}
	fp.Worktrees[path] = wt
	older, err := json.Marshal(fp)
	if err != nil {
		t.Fatal(err)
	}
	writeMode(t, file, string(older), 0o600)

	r := mustRefusal(t, configTripwire(repo.Root(), runID))
	if r.Stage != StageConfig || !strings.Contains(r.Reason, filepath.Join("lib", "common.sh")) {
		t.Fatalf("the older lane's pin stops the run once, naming the nested entry: %+v", r)
	}
	if _, err := AcceptConfigChange(repo.Root(), runID, Options{}); err != nil {
		t.Fatal(err)
	}
	if err := configTripwire(repo.Root(), runID); err != nil {
		t.Fatalf("accepted, the run moves: %v", err)
	}
}

// TestTheHooksWalkIsLinearInTheTree: the walk examines each entry once, so a
// tree four times larger takes about four times the steps; and however many
// links point at one directory outside it, that directory is walked once, so
// its entries count once: ten links to a directory of fifty files cost the
// ten links, one directory and fifty files, where walking it at each link
// would cost nine hundred more. Not watched failing: it is new with the walk;
// the second half fails with the once-only rule removed.
func TestTheHooksWalkIsLinearInTheTree(t *testing.T) {
	steps := func(n int) int {
		dir := t.TempDir()
		for i := range n {
			sub := filepath.Join(dir, fmt.Sprintf("d%02d", i%8))
			mkdirs(t, sub)
			writeMode(t, filepath.Join(sub, fmt.Sprintf("f%04d", i)), "f\n", 0o644)
		}
		got, over := hashDir(map[string]string{}, "salt", dir)
		if over {
			t.Fatalf("%d entries are under the bound", n)
		}
		return got
	}
	small, large := steps(100), steps(400)
	if ratio := float64(large) / float64(small); ratio > 5 || ratio < 3 {
		t.Fatalf("four times the tree takes about four times the steps: %d and %d (%.1f)", small, large, ratio)
	}

	dir, outside := t.TempDir(), t.TempDir()
	for i := range 50 {
		writeMode(t, filepath.Join(outside, fmt.Sprintf("o%02d", i)), "o\n", 0o644)
	}
	for i := range 3 {
		writeMode(t, filepath.Join(dir, fmt.Sprintf("f%d", i)), "f\n", 0o644)
	}
	for i := range 10 {
		if err := os.Symlink(outside, filepath.Join(dir, fmt.Sprintf("l%02d", i))); err != nil {
			t.Fatal(err)
		}
	}
	// One for the hooks directory, one for each of its thirteen entries, one
	// for the directory the links resolve to, one for each of its fifty.
	if got, _ := hashDir(map[string]string{}, "salt", dir); got != 1+13+1+50 {
		t.Fatalf("a directory ten links point at is walked once: %d steps, want %d", got, 1+13+1+50)
	}
}

// TestADrivesChecksLeaveThePersonsOwnCommitAsItWas: a drive's start, its
// proof and its check before a lane's add only read: the repository's
// settings and hooks are byte for byte as they were, and the person's own git
// commit runs the repository's pre-commit hook as before. A regression guard:
// it passes on the default branch too.
func TestADrivesChecksLeaveThePersonsOwnCommitAsItWas(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	marker := filepath.Join(t.TempDir(), "pre-commit-ran")
	writeHook(t, repo, "pre-commit", "#!/bin/sh\ntouch '"+marker+"'\n")
	repo.Commit("a pre-commit that leaves a marker")
	repo.Git("push", "-q", "origin", "main")
	repo.Git("fetch", "-q", "origin")
	_ = os.Remove(marker)
	cfg := filepath.Join(repo.Root(), ".git", "config")
	before, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	hooks := map[string]string{}
	hashDir(hooks, "salt", filepath.Join(repo.Root(), ".githooks"))

	tm, err := startDrive(t, repo)
	if err != nil {
		t.Fatalf("the drive starts: %v", err)
	}
	if err := checkoutHooksUnchanged(Context{RepoRoot: repo.Root(), remote: tm.o.Options.remote}, "lane-1", tm.o.Options.remote.hooksDir); err != nil {
		t.Fatalf("an unchanged checkout passes: %v", err)
	}
	after, err := os.ReadFile(cfg)
	if err != nil || string(after) != string(before) {
		t.Fatalf("the repository's settings are as they were: %v", err)
	}
	now := map[string]string{}
	hashDir(now, "salt", filepath.Join(repo.Root(), ".githooks"))
	if !maps.Equal(now, hooks) {
		t.Fatal("the repository's hooks are as they were")
	}
	repo.Write("notes.txt", "the person's own change\n")
	repo.Commit("the person's own commit")
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("the person's own commit runs the repository's pre-commit hook: %v", err)
	}
}
