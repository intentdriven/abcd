package loop

// lanepin_test.go is the lane's hooks pin (iss-2611071820538335): where the
// checkout has an origin, the worktree stage asks it for its default branch's
// tip, opens the lane only on that tip or an ancestor of it whose hooks are the
// tip's, proves the lane's hooks on disk are that commit's, and pins them, so a
// hook the reviewed default branch changed is no change in the lane, and a hook
// changed in the lane, or standing there other than the commit's, is refused.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/gittest"
)

const (
	hookV1 = "#!/bin/sh\n# the first reviewed pre-commit\nexit 0\n"
	hookV2 = "#!/bin/sh\n# the second reviewed pre-commit\nexit 0\n"
)

// writeHook writes the repository's .githooks/<name> as an executable file.
func writeHook(t *testing.T, repo *gittest.Repo, name, body string) {
	t.Helper()
	repo.Write(".githooks/"+name, body)
	if err := os.Chmod(filepath.Join(repo.Root(), ".githooks", name), 0o755); err != nil {
		t.Fatal(err)
	}
}

// hookedOrigin is a run's repository whose hooks are a relative
// core.hooksPath, .githooks (an executable pre-commit, a symlink beside it, a
// subdirectory holding a file the hooks could source),
// committed, on a local bare origin whose HEAD is main, as fetched here. It
// returns the repository, the bare origin and the commit pushed.
func hookedOrigin(t *testing.T) (*gittest.Repo, string, string) {
	t.Helper()
	repo := briefRepo(t, agentsMarked)
	writeHook(t, repo, "pre-commit", hookV1)
	if err := os.Symlink("pre-commit", filepath.Join(repo.Root(), ".githooks", "pre-merge-commit")); err != nil {
		t.Fatal(err)
	}
	repo.Write(".githooks/lib/common.sh", "shared=1\n")
	repo.Commit("hooks")
	repo.Git("config", "core.hooksPath", ".githooks")
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	repo.Git("remote", "add", "origin", bare)
	repo.Git("push", "-q", "origin", "main")
	repo.Git("fetch", "-q", "origin")
	return repo, bare, repo.Git("rev-parse", "main")
}

// openHookedLane starts a run on repo and steps it once: the fingerprint is
// recorded and, unless the step is refused, the lane's worktree made. It
// returns the run, the lane's derived worktree, and the step's error.
func openHookedLane(t *testing.T, repo *gittest.Repo) (string, string, error) {
	t.Helper()
	start, err := Start(repo.Root(), "itd-10", Options{noReserve: true})
	if err != nil {
		t.Fatal(err)
	}
	lw, err := laneWorktree(repo.Root(), start.RunID, "lane-1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = advance(repo.Root(), start.RunID, DefaultStages(), Options{})
	return start.RunID, lw.Path, err
}

// laneAt reads the run's first lane.
func laneAt(t *testing.T, repo *gittest.Repo, runID string) Lane {
	t.Helper()
	st, err := ReadState(repo.Root(), runID)
	if err != nil || len(st.Lanes) == 0 {
		t.Fatalf("the run has a lane: %+v %v", st.Lanes, err)
	}
	return st.Lanes[0]
}

// pinned reads the run's fingerprint entry for the lane worktree at path.
func pinned(t *testing.T, repo *gittest.Repo, runID, path string) (worktreeFingerprint, bool) {
	t.Helper()
	root, err := os.OpenRoot(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	fp, ok, err := readFingerprint(root, runID)
	if err != nil || !ok {
		t.Fatalf("the run has a fingerprint: %v %v", ok, err)
	}
	wt, ok := fp.Worktrees[filepath.Clean(path)]
	return wt, ok
}

// worktreeRefused fails the test unless err is the worktree stage's refusal
// naming want.
func worktreeRefused(t *testing.T, err error, want string) *Refusal {
	t.Helper()
	r := mustRefusal(t, err)
	if r.Stage != string(StageWorktree) || !strings.Contains(r.Reason, want) {
		t.Fatalf("the worktree stage refuses naming %q: %+v", want, r)
	}
	return r
}

// behindCheckout is hookedOrigin with a reviewed change to the pre-commit
// hook merged on origin's main after it, fetched here, while the checkout's
// own working tree stays on the commit before it: what a drive's checkout is
// when a lane is cut from origin's main after a hook change merged. It returns
// the repository and the commit carrying the change.
func behindCheckout(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	repo, _, first := hookedOrigin(t)
	writeHook(t, repo, "pre-commit", hookV2)
	repo.Commit("a reviewed hook change")
	merged := repo.Git("rev-parse", "main")
	repo.Git("push", "-q", "origin", "main")
	repo.Git("fetch", "-q", "origin")
	repo.Git("reset", "-q", "--hard", first)
	return repo, merged
}

// TestAHookChangeMergedOnTheDefaultBranchIsNoChangeInTheLane is the record's
// fault: a lane cut from the default branch after a reviewed hook change
// merged there, while the checkout's own copy of the hooks is older, pins the
// commit's hooks as it opens and takes its next step; it is not compared with
// the checkout's copy.
func TestAHookChangeMergedOnTheDefaultBranchIsNoChangeInTheLane(t *testing.T) {
	repo, merged := behindCheckout(t)
	runID, path, err := openHookedLane(t, repo)
	if err != nil {
		t.Fatal(err)
	}
	if l := laneAt(t, repo, runID); l.BaseSHA != merged || l.Worktree != path {
		t.Fatalf("the lane is cut from origin's main at %s: %+v", merged, l)
	}
	res, err := advance(repo.Root(), runID, DefaultStages(), Options{})
	notConfigRefused(t, err, "the default branch's reviewed hook change is no change in the lane")
	if err != nil || res.PerformedStage != StageBrief {
		t.Fatalf("the lane takes its next step: %+v %v", res, err)
	}
	if wt, ok := pinned(t, repo, runID, path); !ok || len(wt.Hooks) != 3 {
		t.Fatalf("the lane's three hook entries are pinned as it opens: %+v %v", wt, ok)
	}
}

// TestAHookChangedInAPinnedLaneIsRefused: once pinned, a hook changed in the
// lane stops the next step, as before the pin. That holds for a hook put back
// to the checkout's older copy too: the lane is held to the commit it opened
// on, never to the checkout.
func TestAHookChangedInAPinnedLaneIsRefused(t *testing.T) {
	for name, tc := range map[string]struct {
		behind bool
		body   string
	}{
		"changed to a new body":               {body: "#!/bin/sh\ntouch hook-ran\n"},
		"changed to the checkout's older one": {behind: true, body: hookV1},
	} {
		t.Run(name, func(t *testing.T) {
			var repo *gittest.Repo
			if tc.behind {
				repo, _ = behindCheckout(t)
			} else {
				repo, _, _ = hookedOrigin(t)
			}
			runID, path, err := openHookedLane(t, repo)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(path, ".githooks", "pre-commit"), []byte(tc.body), 0o755); err != nil {
				t.Fatal(err)
			}
			before := stateBytes(t, repo.Root(), runID)
			_, err = advance(repo.Root(), runID, DefaultStages(), Options{})
			r := mustRefusal(t, err)
			if r.Stage != StageConfig || !strings.Contains(r.Reason, "pre-commit") {
				t.Fatalf("the step is refused naming the lane's hook: %+v", r)
			}
			if after := stateBytes(t, repo.Root(), runID); string(after) != string(before) {
				t.Fatal("the refused step took no stage")
			}
			if _, ok := pinned(t, repo, runID, path); !ok {
				t.Fatal("the lane is pinned as it opens")
			}
		})
	}
}

// TestALaneWhoseHooksAreNotItsCommitsIsRefusedAsItOpens: what stands in the
// lane's hooks directory once git has made the worktree must be the commit's
// entries exactly: the same names, each file's bytes (its blob recomputed),
// its exec bit, each symlink's target, and each subdirectory's kind. Anything
// else is refused at the
// worktree stage, before any commit in the lane, and nothing is pinned. Put
// back as the commit has it, the next step adopts the lane and pins it.
func TestALaneWhoseHooksAreNotItsCommitsIsRefusedAsItOpens(t *testing.T) {
	for name, tc := range map[string]struct {
		alter func(t *testing.T, hooks string)
		undo  func(t *testing.T, hooks string)
		want  string
	}{
		"a file's bytes differ": {
			alter: func(t *testing.T, hooks string) {
				writeMode(t, filepath.Join(hooks, "pre-commit"), "#!/bin/sh\ntouch ran\n", 0o755)
			},
			undo: func(t *testing.T, hooks string) { writeMode(t, filepath.Join(hooks, "pre-commit"), hookV1, 0o755) },
			want: "pre-commit",
		},
		"an exec bit differs": {
			alter: func(t *testing.T, hooks string) { chmod(t, filepath.Join(hooks, "pre-commit"), 0o644) },
			undo:  func(t *testing.T, hooks string) { chmod(t, filepath.Join(hooks, "pre-commit"), 0o755) },
			want:  "pre-commit",
		},
		"a symlink's target differs": {
			alter: func(t *testing.T, hooks string) { relinkHook(t, filepath.Join(hooks, "pre-merge-commit"), "/bin/true") },
			undo: func(t *testing.T, hooks string) {
				relinkHook(t, filepath.Join(hooks, "pre-merge-commit"), "pre-commit")
			},
			want: "pre-merge-commit",
		},
		"a subdirectory replaced by a file": {
			alter: func(t *testing.T, hooks string) {
				removeAll(t, filepath.Join(hooks, "lib"))
				writeMode(t, filepath.Join(hooks, "lib"), "shared=1\n", 0o644)
			},
			undo: func(t *testing.T, hooks string) {
				removeAll(t, filepath.Join(hooks, "lib"))
				if err := os.Mkdir(filepath.Join(hooks, "lib"), 0o755); err != nil {
					t.Fatal(err)
				}
				writeMode(t, filepath.Join(hooks, "lib", "common.sh"), "shared=1\n", 0o644)
			},
			want: "lib",
		},
		"an entry the commit lacks": {
			alter: func(t *testing.T, hooks string) {
				writeMode(t, filepath.Join(hooks, "post-commit"), "#!/bin/sh\n", 0o755)
			},
			undo: func(t *testing.T, hooks string) { removeAll(t, filepath.Join(hooks, "post-commit")) },
			want: "post-commit",
		},
		"a file whose size is not the commit's": {
			alter: func(t *testing.T, hooks string) {
				writeMode(t, filepath.Join(hooks, "pre-commit"), strings.Repeat("#", 2<<20), 0o755)
			},
			undo: func(t *testing.T, hooks string) { writeMode(t, filepath.Join(hooks, "pre-commit"), hookV1, 0o755) },
			want: fmt.Sprintf("is %d bytes, and the commit's file %d", 2<<20, len(hookV1)),
		},
		"an entry the commit holds is missing": {
			alter: func(t *testing.T, hooks string) { removeAll(t, filepath.Join(hooks, "pre-merge-commit")) },
			undo: func(t *testing.T, hooks string) {
				relinkHook(t, filepath.Join(hooks, "pre-merge-commit"), "pre-commit")
			},
			want: "pre-merge-commit",
		},
	} {
		t.Run(name, func(t *testing.T) {
			repo, _, _ := hookedOrigin(t)
			was := addWorktree
			t.Cleanup(func() { addWorktree = was })
			addWorktree = func(root string, o worktree.AddOptions) (worktree.Added, error) {
				added, err := was(root, o)
				if err == nil {
					tc.alter(t, filepath.Join(added.Path, ".githooks"))
				}
				return added, err
			}
			runID, path, err := openHookedLane(t, repo)
			worktreeRefused(t, err, tc.want)
			if l := laneAt(t, repo, runID); l.Stage != StageWorktree || l.Worktree != "" {
				t.Fatalf("the lane did not leave its worktree stage: %+v", l)
			}
			if _, ok := pinned(t, repo, runID, path); ok {
				t.Fatal("nothing is pinned for a lane refused as it opens")
			}
			if log := repo.Git("-C", path, "log", "--format=%H", "-1"); log != repo.Git("rev-parse", "main") {
				t.Fatalf("no commit is made in the lane: its head is %s", log)
			}

			addWorktree = was
			tc.undo(t, filepath.Join(path, ".githooks"))
			res, err := advance(repo.Root(), runID, DefaultStages(), Options{})
			if err != nil || res.PerformedStage != StageWorktree {
				t.Fatalf("the lane put back is adopted: %+v %v", res, err)
			}
			if _, ok := pinned(t, repo, runID, path); !ok {
				t.Fatal("the adopted lane is pinned")
			}
		})
	}
}

// TestALaneCannotChooseWhatItOpensOn: a default branch here that the remote
// never had (its tracking refs gone, so the local main is read) is refused
// before the worktree is made; so is a tip the remote reports that was never
// fetched here, which nothing here can show the default branch descends to.
func TestALaneCannotChooseWhatItOpensOn(t *testing.T) {
	for name, tc := range map[string]struct {
		move func(t *testing.T, repo *gittest.Repo, bare string)
		want string
	}{
		"a local main the remote never had": {
			move: func(t *testing.T, repo *gittest.Repo, _ string) {
				writeHook(t, repo, "pre-commit", "#!/bin/sh\ntouch ran\n")
				repo.Commit("an unreviewed hook")
				repo.Git("update-ref", "-d", "refs/remotes/origin/main")
			},
			want: "not the remote's",
		},
		"a tip never fetched here": {
			move: func(t *testing.T, repo *gittest.Repo, bare string) {
				other := filepath.Join(t.TempDir(), "other")
				repo.Git("clone", "-q", bare, other)
				peer := gittest.OpenRepo(t, other)
				peer.Write("notes.txt", "merged elsewhere\n")
				peer.Commit("merged elsewhere")
				peer.Git("push", "-q", "origin", "main")
			},
			want: "does not hold yet",
		},
		"a remote whose default branch is not the one here": {
			move: func(t *testing.T, repo *gittest.Repo, bare string) {
				repo.Git("checkout", "-q", "-b", "develop")
				repo.Write("notes.txt", "on develop\n")
				repo.Commit("on develop")
				repo.Git("push", "-q", "origin", "develop")
				repo.Git("checkout", "-q", "main")
				repo.Git("fetch", "-q", "origin")
				repo.Git("-C", bare, "symbolic-ref", "HEAD", "refs/heads/develop")
			},
			want: "origin's default branch is develop, and the default branch here is main",
		},
	} {
		t.Run(name, func(t *testing.T) {
			repo, bare, _ := hookedOrigin(t)
			tc.move(t, repo, bare)
			_, path, err := openHookedLane(t, repo)
			r := worktreeRefused(t, err, "")
			if !strings.Contains(r.Reason+r.Remedy, tc.want) {
				t.Fatalf("the refusal says %q: %+v", tc.want, r)
			}
			absent(t, path, "no lane worktree is made")
		})
	}
}

// plantedCommit commits a hook named name holding body on top of parent and
// sets the checkout back to parent, so the commit is in the repository and on
// no branch. It returns the commit.
func plantedCommit(t *testing.T, repo *gittest.Repo, parent, name, body string) string {
	t.Helper()
	writeHook(t, repo, name, body)
	repo.Commit("planted")
	planted := repo.Git("rev-parse", "HEAD")
	repo.Git("reset", "-q", "--hard", parent)
	return planted
}

// TestAReplacedObjectDoesNotChooseTheLanesHooks: the proof reads git's
// objects as stored, so a replace ref standing a planted tree in for the tip's
// hooks directory, or a planted commit in for the tip itself, is refused at the
// worktree stage: git's checkout writes the replacement, and the proof holds it
// to the tree the commit stores.
func TestAReplacedObjectDoesNotChooseTheLanesHooks(t *testing.T) {
	for name, plant := range map[string]func(t *testing.T, repo *gittest.Repo, tip, planted string){
		"on the hooks tree": func(t *testing.T, repo *gittest.Repo, tip, planted string) {
			repo.Git("replace", repo.Git("rev-parse", tip+":.githooks"), repo.Git("rev-parse", planted+":.githooks"))
		},
		"on the tip commit": func(t *testing.T, repo *gittest.Repo, tip, planted string) {
			stand := repo.Git("commit-tree", planted+"^{tree}", "-p", tip+"^", "-m", "planted")
			repo.Git("replace", tip, stand)
		},
	} {
		t.Run(name, func(t *testing.T) {
			repo, _, tip := hookedOrigin(t)
			planted := plantedCommit(t, repo, tip, "pre-commit", "#!/bin/sh\ntouch ran\n")
			plant(t, repo, tip, planted)
			runID, path, err := openHookedLane(t, repo)
			worktreeRefused(t, err, "pre-commit")
			if _, ok := pinned(t, repo, runID, path); ok {
				t.Fatal("nothing is pinned for a lane whose hooks a replace ref chose")
			}
		})
	}
}

// TestABranchStandingUnderTheLanesNameIsCheckedOutOnlyWithTheTipsHooks: a
// branch already standing under the lane's name is checked out (and its
// post-checkout hook run) only once its head's hooks directory, as stored, is
// the tip's: one holding other hooks is refused before the worktree is added,
// so none of its hooks runs. One holding the tip's hooks is the lane's.
func TestABranchStandingUnderTheLanesNameIsCheckedOutOnlyWithTheTipsHooks(t *testing.T) {
	for name, tc := range map[string]struct {
		plant   bool
		refused bool
	}{
		"other hooks": {plant: true, refused: true},
		"the tip's":   {},
	} {
		t.Run(name, func(t *testing.T) {
			repo, _, tip := hookedOrigin(t)
			marker := filepath.Join(t.TempDir(), "post-checkout-ran")
			head := tip
			if tc.plant {
				head = plantedCommit(t, repo, tip, "post-checkout", "#!/bin/sh\ntouch '"+marker+"'\n")
			}
			start, err := Start(repo.Root(), "itd-10", Options{noReserve: true})
			if err != nil {
				t.Fatal(err)
			}
			lw, err := laneWorktree(repo.Root(), start.RunID, "lane-1")
			if err != nil {
				t.Fatal(err)
			}
			repo.Git("branch", lw.Branch, head)
			_, err = advance(repo.Root(), start.RunID, DefaultStages(), Options{})
			if !tc.refused {
				if err != nil {
					t.Fatalf("the branch holding the tip's hooks is the lane's: %v", err)
				}
				return
			}
			worktreeRefused(t, err, lw.Branch)
			absent(t, lw.Path, "no lane worktree is made")
			absent(t, marker, "the branch's post-checkout hook does not run")
		})
	}
}

// TestALaneConfigTooLargeToReadIsRefused: a lane whose git configuration is
// past the read's cap once git has made it cannot have its hooks told, so the
// worktree stage refuses it and pins nothing.
func TestALaneConfigTooLargeToReadIsRefused(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	was := addWorktree
	t.Cleanup(func() { addWorktree = was })
	addWorktree = func(root string, o worktree.AddOptions) (worktree.Added, error) {
		added, err := was(root, o)
		if err == nil {
			f, ferr := os.OpenFile(filepath.Join(root, ".git", "config"), os.O_APPEND|os.O_WRONLY, 0)
			if ferr != nil {
				t.Fatal(ferr)
			}
			_, _ = f.WriteString("[filter \"big\"]\n\tclean = " + strings.Repeat("a", 1<<20) + "\n")
			_ = f.Close()
		}
		return added, err
	}
	runID, path, err := openHookedLane(t, repo)
	worktreeRefused(t, err, "too large to read whole")
	if _, ok := pinned(t, repo, runID, path); ok {
		t.Fatal("nothing is pinned for a lane whose configuration cannot be read")
	}
}

// TestAFilterThatRewritesTheHooksOnCheckoutRefusesTheLane: a content filter
// that rewrites the hook files as git checks them out (an end-of-line
// conversion, a smudge filter) leaves them other than the commit's, so the
// lane is refused as it opens.
func TestAFilterThatRewritesTheHooksOnCheckoutRefusesTheLane(t *testing.T) {
	for name, attr := range map[string]string{
		"an end-of-line conversion": ".githooks/pre-commit text eol=crlf\n",
		"a smudge filter":           ".githooks/pre-commit filter=upper\n",
	} {
		t.Run(name, func(t *testing.T) {
			repo, _, _ := hookedOrigin(t)
			repo.Git("config", "filter.upper.smudge", "tr a-z A-Z")
			repo.Git("config", "filter.upper.clean", "cat")
			repo.Write(".gitattributes", attr)
			repo.Commit("attributes")
			repo.Git("push", "-q", "origin", "main")
			repo.Git("fetch", "-q", "origin")
			runID, path, err := openHookedLane(t, repo)
			worktreeRefused(t, err, "pre-commit")
			if _, ok := pinned(t, repo, runID, path); ok {
				t.Fatal("nothing is pinned")
			}
		})
	}
}

// TestAChangedRemoteURLStopsTheStepBeforeTheLaneOpens: the remote the lane's
// tip is asked of is the checkout's origin, whose URL the run's fingerprint
// watches, so pointing it elsewhere after the fingerprint was recorded stops
// the step before the worktree stage asks it. A regression guard: the run's
// tripwire stopped this before the pin, and the test passes without it.
func TestAChangedRemoteURLStopsTheStepBeforeTheLaneOpens(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	start, err := Start(repo.Root(), "itd-10", Options{noReserve: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := configTripwire(repo.Root(), start.RunID); err != nil {
		t.Fatal(err)
	}
	elsewhere := filepath.Join(t.TempDir(), "elsewhere.git")
	gittest.CloneBare(t, repo.Root(), elsewhere)
	repo.Git("remote", "set-url", "origin", elsewhere)
	_, err = advance(repo.Root(), start.RunID, DefaultStages(), Options{})
	r := mustRefusal(t, err)
	if r.Stage != StageConfig || !strings.Contains(r.Reason, "remote.origin.url") {
		t.Fatalf("the step is refused naming the remote's URL: %+v", r)
	}
	lw, err := laneWorktree(repo.Root(), start.RunID, "lane-1")
	if err != nil {
		t.Fatal(err)
	}
	absent(t, lw.Path, "no lane worktree is made")
}

// TestALaneIsRefusedWhenTheRemoteCannotBeAsked: a remote that cannot say what
// its default branch's tip is never reads as one that agrees: the worktree
// stage is refused and makes nothing. A lost network, or no answer within the
// ask's bound, is the stage's network failure, a contention; a remote that is
// gone is a refusal.
func TestALaneIsRefusedWhenTheRemoteCannotBeAsked(t *testing.T) {
	for name, tc := range map[string]struct {
		cut        func(t *testing.T, repo *gittest.Repo, bare string)
		contention bool
	}{
		"the remote is gone": {cut: func(t *testing.T, _ *gittest.Repo, bare string) { removeAll(t, bare) }},
		"the remote does not answer in time": {contention: true, cut: func(t *testing.T, repo *gittest.Repo, _ string) {
			was := originAskTimeout
			originAskTimeout = time.Second
			t.Cleanup(func() { originAskTimeout = was })
			repo.Git("config", "protocol.ext.allow", "always")
			repo.Git("remote", "set-url", "origin", "ext::sh -c sleep% 30")
		}},
		"the network is lost": {contention: true, cut: func(t *testing.T, repo *gittest.Repo, _ string) {
			repo.Git("config", "protocol.ext.allow", "always")
			repo.Git("remote", "set-url", "origin",
				"ext::sh -c echo% fatal:% Failed% to% connect% to% 127.0.0.1% port% 1:% Connection% refused% 1>&2;% exit% 128")
		}},
	} {
		t.Run(name, func(t *testing.T) {
			repo, bare, _ := hookedOrigin(t)
			tc.cut(t, repo, bare)
			_, path, err := openHookedLane(t, repo)
			r := worktreeRefused(t, err, "origin")
			if r.Contention != tc.contention {
				t.Fatalf("contention %v, want %v: %+v", r.Contention, tc.contention, r)
			}
			absent(t, path, "no lane worktree is made")
		})
	}
}

// TestALaneWithNoOriginIsNotPinned: with no origin there is no reviewed
// branch to pin a lane from, so nothing is pinned and the lane is compared
// with the checkout as the run recorded it. A regression guard: nothing was
// pinned before the pin either, and the test passes without it.
func TestALaneWithNoOriginIsNotPinned(t *testing.T) {
	repo, runID, lane := laneRun(t, func(repo *gittest.Repo) {
		writeHook(t, repo, "pre-commit", hookV1)
		repo.Commit("hooks")
		repo.Git("config", "core.hooksPath", ".githooks")
	})
	if _, ok := pinned(t, repo, runID, lane); ok {
		t.Fatal("a lane with no origin is not pinned")
	}
}

func writeMode(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
	chmod(t, path, mode)
}

func chmod(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func relinkHook(t *testing.T, path, target string) {
	t.Helper()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
}

func removeAll(t *testing.T, path string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
}

// TestALaneOpensOnTheRemotesTipWhateverTheRefHereSays: the tracking ref set
// back here to an older commit, whose hooks the default branch has since
// changed, does not choose what the lane opens on: the lane is cut from the
// tip origin reports and pins that tip's hooks.
func TestALaneOpensOnTheRemotesTipWhateverTheRefHereSays(t *testing.T) {
	repo, _, first := hookedOrigin(t)
	writeHook(t, repo, "pre-commit", hookV2)
	repo.Commit("a reviewed hook change")
	merged := repo.Git("rev-parse", "main")
	repo.Git("push", "-q", "origin", "main")
	repo.Git("update-ref", "refs/remotes/origin/main", first)

	runID, path, err := openHookedLane(t, repo)
	if err != nil {
		t.Fatal(err)
	}
	if l := laneAt(t, repo, runID); l.BaseSHA != merged {
		t.Fatalf("the lane is cut from origin's tip %s, not the ref set back here: %+v", merged, l)
	}
	if got, err := os.ReadFile(filepath.Join(path, ".githooks", "pre-commit")); err != nil || string(got) != hookV2 {
		t.Fatalf("the lane holds the tip's hook: %q %v", got, err)
	}
	if _, ok := pinned(t, repo, runID, path); !ok {
		t.Fatal("the lane is pinned from the tip")
	}
}

// TestAnAdoptedLaneWhoseHooksTheTipChangedIsRefused: a lane the stage made
// and adopts again (a step killed before its state was written) opens on
// where its branch left the default branch; once origin's tip holds other
// hooks than that commit, the lane's are not the reviewed default branch's,
// and the adoption is refused, naming the hooks.
func TestAnAdoptedLaneWhoseHooksTheTipChangedIsRefused(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	start, err := Start(repo.Root(), "itd-10", Options{noReserve: true})
	if err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	c := Context{RepoRoot: repo.Root(), RunDir: runRel(st.RunID), State: st}
	first := st.Lanes[0]
	if _, err := worktreeStage(c, &first); err != nil {
		t.Fatal(err)
	}
	writeHook(t, repo, "pre-commit", hookV2)
	repo.Commit("a reviewed hook change")
	repo.Git("push", "-q", "origin", "main")

	again := st.Lanes[0]
	_, err = worktreeStage(c, &again)
	worktreeRefused(t, err, "other hooks")
}

// TestATrialLaneIsNotPinnedAndAsksNoRemote: a trial's lanes are cut from its
// frozen commit, never from the default branch, so the stage asks no remote
// (origin here is gone, and the lane still opens) and pins nothing. A
// regression guard: a trial asked no remote before the pin either, and the
// test passes without it.
func TestATrialLaneIsNotPinnedAndAsksNoRemote(t *testing.T) {
	repo, bare, head := hookedOrigin(t)
	removeAll(t, bare)
	start, err := Start(repo.Root(), "itd-10", Options{Trial: true, Frozen: head})
	if err != nil {
		t.Fatal(err)
	}
	res, err := advance(repo.Root(), start.RunID, DefaultStages(), Options{})
	if err != nil || res.PerformedStage != StageWorktree {
		t.Fatalf("the trial's lane opens with origin gone: %+v %v", res, err)
	}
	if _, ok := pinned(t, repo, start.RunID, laneAt(t, repo, start.RunID).Worktree); ok {
		t.Fatal("a trial's lane is not pinned")
	}
}
