package loop

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// laneScopeRepo is hookedOrigin with extensions.worktreeConfig on in the
// repository's own settings and one lane worktree, named as the loop names a
// lane's, that sets gc.recentObjectsHook at its own scope.
func laneScopeRepo(t *testing.T) (repo *gittest.Repo, lane string) {
	t.Helper()
	repo, _, _ = hookedOrigin(t)
	repo.Git("config", "extensions.worktreeConfig", "true")
	lane = filepath.Join(t.TempDir(), "run-2611140000000001-lane-1")
	repo.Git("worktree", "add", "-q", "--detach", lane, "HEAD")
	repo.Git("-C", lane, "config", "--worktree", "gc.recentObjectsHook", "true")
	return repo, lane
}

// laneScopeNamed fails the test unless err is the drive's refusal at the
// config stage naming gc.recentObjectsHook in the lane worktree at lane.
func laneScopeNamed(t *testing.T, err error, lane string) {
	t.Helper()
	r := mustRefusal(t, err)
	if r.Stage != StageConfig || !strings.Contains(r.Reason, "gc.recentobjectshook in the worktree") ||
		!strings.Contains(r.Reason, filepath.Base(lane)) {
		t.Fatalf("the drive's comparison names the lane's key and worktree: %+v", r)
	}
}

// TestAHeldKeyGoneFromALaneWorktreesOwnScopeEndsTheDrive: the comparison of
// a lane worktree's own scope runs both ways, so a key the lane set as the
// drive started and later unset, changed, or lost with its config.worktree
// emptied, ends the drive naming the key and the worktree. A lane whose
// directory is gone is skipped. Watched failing on the branch before, which
// walked only the keys read now: the unset and emptied cases read
// unchanged.
func TestAHeldKeyGoneFromALaneWorktreesOwnScopeEndsTheDrive(t *testing.T) {
	for name, change := range map[string]func(t *testing.T, repo *gittest.Repo, lane string){
		"unset": func(t *testing.T, repo *gittest.Repo, lane string) {
			repo.Git("-C", lane, "config", "--worktree", "--unset", "gc.recentObjectsHook")
		},
		"changed": func(t *testing.T, repo *gittest.Repo, lane string) {
			repo.Git("-C", lane, "config", "--worktree", "gc.recentObjectsHook", "false")
		},
		"emptied": func(t *testing.T, repo *gittest.Repo, lane string) {
			file := repo.Git("-C", lane, "rev-parse", "--path-format=absolute", "--git-path", "config.worktree")
			if err := os.WriteFile(file, nil, 0o644); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			repo, lane := laneScopeRepo(t)
			id, err := readRemoteIdentity(repo.Root())
			if err != nil {
				t.Fatal(err)
			}
			if err := heldBeforeGit(repo.Root(), id); err != nil {
				t.Fatalf("nothing changed, and the comparison stopped the drive: %v", err)
			}
			change(t, repo, lane)
			laneScopeNamed(t, heldBeforeGit(repo.Root(), id), lane)
			laneScopeNamed(t, heldBeforeGitIn(repo.Root(), id, lane, nil), lane)
			worktreeRefused(t, remoteUnchanged(Context{RepoRoot: repo.Root(), remote: id}, "lane-2"), "gc.recentobjectshook in the worktree")
		})
	}
	t.Run("a lane whose directory is gone", func(t *testing.T) {
		repo, lane := laneScopeRepo(t)
		id, err := readRemoteIdentity(repo.Root())
		if err != nil {
			t.Fatal(err)
		}
		repo.Git("worktree", "remove", "--force", lane)
		if err := heldBeforeGit(repo.Root(), id); err != nil {
			t.Fatalf("a lane whose directory is gone stopped the drive: %v", err)
		}
	})
}

// TestTheHoldAsksGitWhetherItReadsAWorktreesOwnSettings: git decides whether
// it reads a worktree's own settings from extensions.worktreeConfig in the
// repository's own settings file alone, following no include there, so a
// value in an included file, in config.worktree, in the global settings or on
// the command line changes nothing, as a real git shows here; the hold asks
// git the same way. With true in the repository's settings and false
// elsewhere, the lane's own scope is still read, and a key set there ends the
// drive; with false there and true elsewhere, git reads no worktree's own
// settings, and neither does the hold. Watched failing on the branch before,
// which took the last record of the whole listing: the include and
// config.worktree cases with true in the repository read no lane, and the
// include case with false there read one. The global and command-line cases
// were added after the fix and pass on it; they were not watched failing.
func TestTheHoldAsksGitWhetherItReadsAWorktreesOwnSettings(t *testing.T) {
	for name, tc := range map[string]struct {
		repoValue string
		later     func(t *testing.T, repo *gittest.Repo)
		gitReads  bool
	}{
		"true, then false in an include": {"true", func(t *testing.T, repo *gittest.Repo) {
			inc := filepath.Join(t.TempDir(), "ext.gitconfig")
			writeMode(t, inc, "[extensions]\n\tworktreeConfig = false\n", 0o644)
			repo.Git("config", "include.path", inc)
		}, true},
		"true, then false in config.worktree": {"true", func(t *testing.T, repo *gittest.Repo) {
			repo.Git("config", "--worktree", "extensions.worktreeConfig", "false")
		}, true},
		"false, then true in an include": {"false", func(t *testing.T, repo *gittest.Repo) {
			inc := filepath.Join(t.TempDir(), "ext.gitconfig")
			writeMode(t, inc, "[extensions]\n\tworktreeConfig = true\n", 0o644)
			repo.Git("config", "include.path", inc)
		}, false},
		"true, then false in the global settings": {"true", func(t *testing.T, repo *gittest.Repo) {
			writeMode(t, filepath.Join(os.Getenv("HOME"), ".gitconfig"), "[extensions]\n\tworktreeConfig = false\n", 0o644)
		}, true},
		"false, then true in the global settings": {"false", func(t *testing.T, repo *gittest.Repo) {
			writeMode(t, filepath.Join(os.Getenv("HOME"), ".gitconfig"), "[extensions]\n\tworktreeConfig = true\n", 0o644)
		}, false},
		"true, then false on the command line": {"true", func(t *testing.T, repo *gittest.Repo) {
			t.Setenv("GIT_CONFIG_PARAMETERS", "'extensions.worktreeconfig'='false'")
		}, true},
		"false, then true on the command line": {"false", func(t *testing.T, repo *gittest.Repo) {
			t.Setenv("GIT_CONFIG_PARAMETERS", "'extensions.worktreeconfig'='true'")
		}, false},
	} {
		t.Run(name, func(t *testing.T) {
			repo, _, _ := hookedOrigin(t)
			lane := filepath.Join(t.TempDir(), "run-2611140000000001-lane-1")
			repo.Git("config", "extensions.worktreeConfig", "true")
			repo.Git("worktree", "add", "-q", "--detach", lane, "HEAD")
			tc.later(t, repo)
			if tc.repoValue == "false" {
				repo.Git("config", "--file", filepath.Join(repo.Root(), ".git", "config"), "extensions.worktreeConfig", "false")
			}
			// Git itself: a key written to the lane's config.worktree is read
			// there only where git reads a worktree's own settings.
			file := filepath.Join(repo.Root(), ".git", "worktrees", filepath.Base(lane), "config.worktree")
			writeMode(t, file, "[gc]\n\trecentObjectsHook = true\n", 0o644)
			read := exec.Command("git", "-C", lane, "config", "--show-scope", "--get", "gc.recentObjectsHook")
			// The person's own environment: their global settings, and the
			// command line's settings as git hands them to a git it runs.
			read.Env = append(gitutil.ScrubbedEnv(), gitutil.CommandLineConfig()...)
			got, err := read.Output()
			if reads := err == nil && strings.HasPrefix(string(got), "worktree"); reads != tc.gitReads {
				t.Fatalf("fixture: git reads the lane's own settings = %v (%q, %v), want %v", reads, got, err, tc.gitReads)
			}
			id, err := readRemoteIdentity(repo.Root())
			if err != nil {
				t.Fatal(err)
			}
			writeMode(t, file, "[gc]\n\trecentObjectsHook = false\n", 0o644)
			err = heldBeforeGit(repo.Root(), id)
			if tc.gitReads {
				laneScopeNamed(t, err, lane)
				return
			}
			if err != nil {
				t.Fatalf("git reads no worktree's own settings here, and a change there stopped the drive: %v", err)
			}
		})
	}
}

// TestALaneTheDriveMakesHoldsTheCheckoutsOwnScopeAsItsStart: git's add
// copies the checkout's own config.worktree into the worktree it makes (git
// 2.36 and later, which the listing of worktrees the hold reads needs), so a
// checkout that sets a drive-only key at its own worktree scope gives each
// new lane that key. The drive reads a lane it makes as it makes it (laneMade)
// and holds that: the lane opens, the next comparisons read it unchanged, and
// a later change there ends the drive. A lane made since the start by
// another hand, whose own scope is the checkout's, reads unchanged too; one
// whose scope differs from the checkout's as the drive started is a held
// change, refused as the drive makes it. Watched failing on the branch
// before, where the copied key read as set newly and ended the drive at its
// first lane.
func TestALaneTheDriveMakesHoldsTheCheckoutsOwnScopeAsItsStart(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	repo.Git("config", "extensions.worktreeConfig", "true")
	repo.Git("config", "--worktree", "gc.recentObjectsHook", "true")
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
		t.Fatalf("the drive's first lane opens where the checkout sets a drive-only key at its own worktree scope: %v", err)
	}
	lw, err := laneWorktree(repo.Root(), st.RunID, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := repo.Git("-C", lw.Path, "config", "--worktree", "--get", "gc.recentObjectsHook"); got != "true" {
		t.Fatalf("fixture: git's add copied the checkout's own scope into the lane: %q", got)
	}
	if err := heldBeforeGit(repo.Root(), id); err != nil {
		t.Fatalf("the lane the drive made reads unchanged: %v", err)
	}
	if err := heldBeforeGitIn(repo.Root(), id, lw.Path, nil); err != nil {
		t.Fatalf("the lane the drive made reads unchanged before a git there: %v", err)
	}

	other := filepath.Join(t.TempDir(), "run-2611140000000002-lane-1")
	repo.Git("worktree", "add", "-q", "--detach", other, "HEAD")
	if err := heldBeforeGit(repo.Root(), id); err != nil {
		t.Fatalf("a lane made since the start whose own scope is the checkout's reads unchanged: %v", err)
	}

	repo.Git("-C", lw.Path, "config", "--worktree", "gc.recentObjectsHook", "false")
	laneScopeNamed(t, heldBeforeGit(repo.Root(), id), lw.Path)
	laneScopeNamed(t, heldBeforeGitIn(repo.Root(), id, lw.Path, nil), lw.Path)
	repo.Git("-C", lw.Path, "config", "--worktree", "gc.recentObjectsHook", "true")

	made := filepath.Join(t.TempDir(), "run-2611140000000003-lane-1")
	repo.Git("worktree", "add", "-q", "--detach", made, "HEAD")
	repo.Git("-C", made, "config", "--worktree", "gc.recentObjectsHook", "false")
	worktreeRefused(t, laneMade(c, "lane-3", made), "gc.recentobjectshook in the worktree")
}

// TestAGitAboutToRunReadsOnlyItsOwnLanesScope: before a git that runs hooks
// or reaches the network, the drive's check is given the directory that git
// runs in and reads the own scope of that one lane worktree: a change in
// another lane's scope does not stop it, and the comparison at the start of
// a step, which reads every lane, does. Watched failing on the branch
// before, whose check was given no directory and read every lane.
func TestAGitAboutToRunReadsOnlyItsOwnLanesScope(t *testing.T) {
	repo, lane := laneScopeRepo(t)
	other := filepath.Join(t.TempDir(), "run-2611140000000001-lane-2")
	repo.Git("worktree", "add", "-q", "--detach", other, "HEAD")
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	unset, err := gitutil.SetBeforeHookedGitAt(func(dir string, args []string) error { return heldBeforeGitIn(repo.Root(), id, dir, args) })
	if err != nil {
		t.Fatal(err)
	}
	defer unset()
	repo.Git("-C", lane, "config", "--worktree", "gc.recentObjectsHook", "false")
	if out, err := gitutil.WorkingTreeCommand(other, "rev-parse", "HEAD").Output(); err != nil {
		t.Fatalf("a git in another lane was stopped by a change in this one's own scope: %v %s", err, out)
	}
	_, err = gitutil.WorkingTreeCommand(lane, "rev-parse", "HEAD").Output()
	var nr *gitutil.NotRunError
	if !errors.As(err, &nr) {
		t.Fatalf("a git in the changed lane runs only after the comparison, which stops it: %v", err)
	}
	laneScopeNamed(t, nr.Err, lane)
	laneScopeNamed(t, heldBeforeGit(repo.Root(), id), lane)
}

// TestAnIncludeAtALaneWorktreesOwnScopeIsReadInEveryEnvironment: a lane
// worktree's own scope is listed once, in abcd's isolated environment, unless
// it holds an include, whose file is then read in every environment a drive's
// git runs in: a drive-only key changed in that file ends the drive.
func TestAnIncludeAtALaneWorktreesOwnScopeIsReadInEveryEnvironment(t *testing.T) {
	repo, lane := laneScopeRepo(t)
	inc := filepath.Join(t.TempDir(), "lane.gitconfig")
	writeMode(t, inc, "[gpg]\n\tprogram = /bin/true\n", 0o644)
	repo.Git("-C", lane, "config", "--worktree", "include.path", inc)
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if err := heldBeforeGit(repo.Root(), id); err != nil {
		t.Fatalf("nothing changed, and the comparison stopped the drive: %v", err)
	}
	writeMode(t, inc, "[gpg]\n\tprogram = /bin/false\n", 0o644)
	r := mustRefusal(t, heldBeforeGit(repo.Root(), id))
	if !strings.Contains(r.Reason, "gpg.program in the worktree") {
		t.Fatalf("the comparison names the key the include changed at the lane's own scope: %+v", r)
	}
}

// TestAGitThatNamesALaneReadsThatLanesScope: a git that runs in the checkout
// but acts on a lane worktree reads that lane's own scope: `git worktree
// remove` of a lane whose own scope changed (git's remove starts a status
// there) is stopped, and so is a git moved into the lane by -C; one given
// --git-dir or --work-tree, whose lane cannot be told, reads every lane. A
// git in the checkout that names no lane reads none. Watched failing on the
// branch before, whose check read only the lane of the directory the git
// was started in: the remove ran, and removed the lane.
func TestAGitThatNamesALaneReadsThatLanesScope(t *testing.T) {
	repo, lane := laneScopeRepo(t)
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	unset, err := gitutil.SetBeforeHookedGitAt(func(dir string, args []string) error { return heldBeforeGitIn(repo.Root(), id, dir, args) })
	if err != nil {
		t.Fatal(err)
	}
	defer unset()
	repo.Git("-C", lane, "config", "--worktree", "gc.recentObjectsHook", "false")
	if _, err := pickGit(repo.Root(), "rev-parse", "HEAD"); err != nil {
		t.Fatalf("a git in the checkout that names no lane was stopped: %v", err)
	}
	stopped := func(what string, args ...string) {
		t.Helper()
		_, err := pickGit(repo.Root(), args...)
		var nr *gitutil.NotRunError
		if !errors.As(err, &nr) {
			t.Fatalf("%s runs only after the comparison of the lane it names, which stops it: %v", what, err)
		}
		laneScopeNamed(t, nr.Err, lane)
	}
	stopped("the removal of the lane", "worktree", "remove", "--", lane)
	if _, err := os.Stat(lane); err != nil {
		t.Fatalf("the lane stands: %v", err)
	}
	stopped("a git moved into the lane", "-C", lane, "rev-parse", "HEAD")
	stopped("a git given the lane's work tree", "--work-tree="+lane, "rev-parse", "HEAD")
}

// TestALaneTheWorktreeStageAdoptsIsReadAsItIsAdopted: a lane worktree made by
// hand where the drive would make it is adopted by the worktree stage, which
// reads its own scope then, before any agent runs there (laneMade): one that
// sets a key the drive holds otherwise than the checkout's own scope did as
// the drive started is refused at the worktree stage, naming the key; one
// that matches opens and is held. Watched failing on the branch before, which
// did not hold the adopted lane; the differing case was refused there too,
// by the comparison the stage makes before it asks origin, which reads every
// lane.
func TestALaneTheWorktreeStageAdoptsIsReadAsItIsAdopted(t *testing.T) {
	for _, differs := range []bool{true, false} {
		repo, _, _ := hookedOrigin(t)
		repo.Git("config", "extensions.worktreeConfig", "true")
		repo.Git("config", "--worktree", "gc.recentObjectsHook", "true")
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
		first := st.Lanes[0]
		lw, err := laneWorktree(repo.Root(), st.RunID, first.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err := lw.lane.Ensure(); err != nil {
			t.Fatal(err)
		}
		repo.Git("worktree", "add", "-q", "-b", lw.Branch, lw.Path, "HEAD")
		if differs {
			repo.Git("-C", lw.Path, "config", "--worktree", "gc.recentObjectsHook", "false")
		}
		c := Context{RepoRoot: repo.Root(), RunDir: runRel(st.RunID), State: st, remote: id}
		_, err = worktreeStage(c, &first)
		if differs {
			worktreeRefused(t, err, "gc.recentobjectshook in the worktree")
			continue
		}
		if err != nil {
			t.Fatalf("an adopted lane whose own scope is the checkout's opens: %v", err)
		}
		id.mu.Lock()
		_, held := id.lanes[lanePath(lw.Path)]
		id.mu.Unlock()
		if !held {
			t.Fatal("the adopted lane is held as it is adopted")
		}
	}
}
