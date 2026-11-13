package loop

// remoteid_reads_test.go holds fix round 2 of iss-2611131308266006: a read the
// hold depends on never gives way to "unchanged" (ruling 1), every
// environment a drive's git runs in is held (ruling 2), the start's wait on
// origin is the drive's own (ruling 3), the attributes files are found on a
// git that does not know their names (ruling 4), and the smaller findings
// (ruling 5). A test whose comment says it was watched failing failed at the
// round's base, dd5615ef; the others say what they hold.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// standInGit puts a git first on PATH, for the rest of the test, that runs
// script's case arms over its arguments and otherwise the real git.
func standInGit(t *testing.T, arms string) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeMode(t, filepath.Join(dir, "git"), "#!/bin/sh\ncase \"$*\" in\n"+arms+"\nesac\nexec '"+real+"' \"$@\"\n", 0o755)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// recordedRun is a run started and stepped once in a repository of its own,
// so its fingerprint is recorded.
func recordedRun(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{noReserve: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := advance(repo.Root(), start.RunID, DefaultStages(), Options{}); err != nil {
		t.Fatal(err)
	}
	return repo, start.RunID
}

// configRefused fails unless err stops the step at the config stage naming
// each of want.
func configRefused(t *testing.T, err error, want ...string) *Refusal {
	t.Helper()
	r := mustRefusal(t, err)
	if r.Stage != StageConfig {
		t.Fatalf("the read stops the step at the config stage: %+v", r)
	}
	for _, w := range want {
		if !strings.Contains(r.Reason, w) {
			t.Fatalf("the refusal names %q: %+v", w, r)
		}
	}
	return r
}

// TestAHooksPathGitAnswersWithANewlineStopsTheStep: a core.hooksPath holding
// a newline is answered by git on two lines, which abcd does not take; the
// step's check, and a drive's start, stop at the config stage naming the read
// and why, never read as nothing changed. Watched failing: the step passed.
func TestAHooksPathGitAnswersWithANewlineStopsTheStep(t *testing.T) {
	t.Run("the step", func(t *testing.T) {
		repo, runID := recordedRun(t)
		repo.Git("config", "core.hooksPath", "hooks\nelsewhere")
		configRefused(t, configTripwire(repo.Root(), runID), "git rev-parse --git-path hooks", "newline")
	})
	t.Run("the drive's start", func(t *testing.T) {
		repo, _, _ := hookedOrigin(t)
		repo.Git("config", "core.hooksPath", ".githooks\nelsewhere")
		tm, err := startDrive(t, repo)
		r := startRefused(t, tm, err, "git rev-parse --git-path hooks")
		if r.Stage != StageConfig || !strings.Contains(r.Reason, "newline") {
			t.Fatalf("the start stops at the config stage naming why: %+v", r)
		}
	})
}

// TestAReadThatDoesNotFinishStopsTheStep: git asked for the hooks directory
// and not answering in time stops the step at the config stage, as a read
// that did not finish; it is never read as nothing changed, nor as a change.
// Watched failing: the step passed.
func TestAReadThatDoesNotFinishStopsTheStep(t *testing.T) {
	repo, runID := recordedRun(t)
	was := heldReadTimeout
	heldReadTimeout = 2 * time.Second
	t.Cleanup(func() { heldReadTimeout = was })
	standInGit(t, `*"--git-path hooks"*) exec sleep 30 ;;`)
	r := configRefused(t, configTripwire(repo.Root(), runID), "git rev-parse --git-path hooks", "did not finish within")
	if strings.Contains(r.Reason, "changed during the run:") {
		t.Fatalf("a read that did not finish is not a change: %+v", r)
	}
}

// TestAPushsHooksDirectoryNamedOnlyInThePersonsGlobalSettingsIsHeld: with
// core.hooksPath set only in the person's global settings, which the lane's
// add and commit do not read under the stage's home and the push does, a
// pre-push there changed between two lanes refuses the second. Watched
// failing: the second lane opened.
func TestAPushsHooksDirectoryNamedOnlyInThePersonsGlobalSettingsIsHeld(t *testing.T) {
	stagedGit(t)
	repo, _, _ := hookedOrigin(t)
	repo.Git("config", "--unset", "core.hooksPath")
	hooks := filepath.Join(t.TempDir(), "global-hooks")
	mkdirs(t, hooks)
	writeMode(t, filepath.Join(hooks, "pre-push"), "#!/bin/sh\nexit 0\n", 0o755)
	global := filepath.Join(t.TempDir(), "gitconfig")
	writeMode(t, global, "[core]\n\thooksPath = "+hooks+"\n", 0o644)
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	lane, err := driveSecondLane(t, repo, func() {
		writeMode(t, filepath.Join(hooks, "pre-push"), "#!/bin/sh\ntouch planted\n", 0o755)
	})
	worktreeRefused(t, err, "pre-push (changed)")
	absent(t, lane, "no second lane worktree is made")
}

// TestASettingChangedNamesTheEnvironmentToListItIn: the remedy of a held
// setting that changed says how to list the settings in each environment,
// the stage's home among them. Watched failing: it named the person's own
// listing only.
func TestASettingChangedNamesTheEnvironmentToListItIn(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	_, err := driveSecondLane(t, repo, func() { repo.Git("config", "core.editor", "/bin/true") })
	r := worktreeRefused(t, err, "core.editor")
	if !strings.Contains(r.Remedy, "HOME set to the stage's home") {
		t.Fatalf("the remedy names the stage's environment: %+v", r)
	}
}

// TestADriveStartAsksAgainAfterAWaitAndProceeds: an origin the start cannot
// reach is waited on, asked again, and once it answers the drive starts,
// holding what it read. Watched failing: the wait was not followed by a
// second ask (the test's kill came first).
func TestADriveStartAsksAgainAfterAWaitAndProceeds(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	was := remoteHeadTip
	t.Cleanup(func() { remoteHeadTip = was })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	asked := 0
	remoteHeadTip = func(c context.Context, git func(context.Context, string, ...string) (string, error), root, remote string) (string, string, error) {
		if asked++; asked == 1 {
			return "", "", errors.New("fatal: unable to access 'https://origin.invalid/x.git/': Could not resolve host: origin.invalid")
		}
		cancel()
		return was(c, git, root, remote)
	}
	var said []string
	tm := newTerminal(repo.Root(), TerminalOptions{Stages: DefaultStages(), Say: func(s string) { said = append(said, s) },
		Sleep: func(context.Context, time.Duration) {}})
	res, err := tm.run(ctx)
	if err != nil || asked != 2 || tm.o.Options.remote == nil || res.Ended != TerminalKilled {
		t.Fatalf("the start waited, asked again and started: %+v %v asked %d\n%s", res, err, asked, strings.Join(said, "\n"))
	}
}

// hookedTerminalWorld is a terminal world whose checkout runs a committed
// in-tree hooks directory, so the drive's start asks origin for its tip.
func hookedTerminalWorld(t *testing.T) *terminalWorld {
	t.Helper()
	w := newTerminalWorld(t)
	writeHook(t, w.repo, "pre-commit", hookV1)
	w.repo.Commit("hooks")
	w.repo.Git("push", "-q", "origin", "main")
	w.repo.Git("fetch", "-q", "origin")
	w.repo.Git("config", "core.hooksPath", ".githooks")
	return w
}

// originDown has the start's ask of origin lose the network for the rest of
// the test.
func originDown(t *testing.T) *int {
	t.Helper()
	was := remoteHeadTip
	t.Cleanup(func() { remoteHeadTip = was })
	asked := 0
	remoteHeadTip = func(context.Context, func(context.Context, string, ...string) (string, error), string, string) (string, string, error) {
		asked++
		return "", "", errors.New("fatal: unable to access 'https://origin.invalid/x.git/': Could not resolve host: origin.invalid")
	}
	return &asked
}

// TestADriveStartWaitingOnOriginEndsAtTheDrivesBound: a drive with an until
// bound whose origin stays out of reach ends at the bound with the bound's
// own outcome, its lost network reported to the run's shared outage, whose
// probe it then waits on as a lane does (asking origin no more while the
// probe finds the network down), and one held by the check before a stage
// ends as that check says. Watched failing:
// the start waited past the bound until the test's guard on its waits.
func TestADriveStartWaitingOnOriginEndsAtTheDrivesBound(t *testing.T) {
	t.Run("the until bound", func(t *testing.T) {
		w := hookedTerminalWorld(t)
		asked := originDown(t)
		o := w.bounded(t, 3*time.Hour)
		o.Options.NetworkProbe = func() (bool, string) { return false, "the network is still down" }
		sha := w.repo.Git("rev-list", "--max-parents=0", "HEAD")
		down := false
		w.onSleep = func(time.Duration) {
			if run, err := implement.Peek(sha); err == nil {
				if cur, err := run.Outage().Current(); err == nil && cur != nil && slices.Contains(cur.Down, implement.ServiceNetwork) {
					down = true
				}
			}
		}
		res, err := RunTerminal(context.Background(), w.repo.Root(), o)
		if err != nil || res.Ended != TerminalUntil || *asked != 1 {
			t.Fatalf("the start's wait ends at the bound: %+v %v asked %d\n%s", res, err, *asked, w.said())
		}
		if !strings.Contains(w.said(), "lost the network") || !strings.Contains(w.said(), "reaches the network, which is down; it waits on the shared network probe") {
			t.Fatalf("the wait reports the lost network to the shared outage, and then waits on its probe rather than ask again:\n%s", w.said())
		}
		if !down {
			t.Fatalf("the shared outage held the network down while the start waited:\n%s", w.said())
		}
	})
	t.Run("the check before a stage", func(t *testing.T) {
		w := hookedTerminalWorld(t)
		originDown(t)
		o := w.bounded(t, 30*24*time.Hour)
		checks := 0
		o.BeforeStage = func() error {
			if checks++; checks > 2 {
				return &WallEnded{Ended: TerminalOutage}
			}
			return nil
		}
		res, err := RunTerminal(context.Background(), w.repo.Root(), o)
		if err != nil || res.Ended != TerminalOutage {
			t.Fatalf("the start's wait ends as the check says: %+v %v\n%s", res, err, w.said())
		}
	})
}

// TestTheAttributesFilesAreFoundWhereGitDoesNotKnowTheirNames: with a git
// that answers `git var GIT_ATTR_*` with its usage, as one older than 2.42
// does, the global attributes file is core.attributesFile where set, else
// the XDG default, else the home's; the system one is the gitattributes
// beside the system settings file, and none under GIT_ATTR_NOSYSTEM. The
// drive starts there and holds the global file. Watched failing: the start
// was refused for git's version.
func TestTheAttributesFilesAreFoundWhereGitDoesNotKnowTheirNames(t *testing.T) {
	standInGit(t, `*"var GIT_ATTR_GLOBAL"|*"var GIT_ATTR_SYSTEM") echo 'usage: git var (-l | <variable>)' >&2; exit 129 ;;`)
	repo := briefRepo(t, agentsMarked)
	home, xdg := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("GIT_ATTR_NOSYSTEM", "")
	got, err := attributesFiles(personEnv, repo.Root(), "/etc/gitconfig")
	if err != nil || !slices.Equal(got, []string{home + "/.config/git/attributes", "/etc/gitattributes"}) {
		t.Fatalf("the home's default and the system file beside its settings: %v %v", got, err)
	}
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("GIT_ATTR_NOSYSTEM", "true")
	got, err = attributesFiles(personEnv, repo.Root(), "/etc/gitconfig")
	if err != nil || !slices.Equal(got, []string{xdg + "/git/attributes"}) {
		t.Fatalf("the XDG default, and no system file under GIT_ATTR_NOSYSTEM: %v %v", got, err)
	}
	repo.Git("config", "core.attributesFile", "~/attrs")
	got, err = attributesFiles(personEnv, repo.Root(), "")
	if err != nil || !slices.Equal(got, []string{home + "/attrs"}) {
		t.Fatalf("core.attributesFile as git expands it, and no system file where none is named: %v %v", got, err)
	}
	t.Setenv("GIT_ATTR_NOSYSTEM", "maybe")
	if _, err := attributesFiles(personEnv, repo.Root(), "/etc/gitconfig"); err == nil || !strings.Contains(err.Error(), "GIT_ATTR_NOSYSTEM") {
		t.Fatalf("a value git refuses is not read as either: %v", err)
	}
	t.Setenv("GIT_ATTR_NOSYSTEM", "")
	repo.Git("config", "--unset", "core.attributesFile")

	hooked, _, _ := hookedOrigin(t)
	lane, err := driveSecondLane(t, hooked, func() {
		mkdirs(t, filepath.Join(xdg, "git"))
		writeMode(t, filepath.Join(xdg, "git", "attributes"), "* filter=planted\n", 0o644)
	})
	worktreeRefused(t, err, "attributes (added)")
	absent(t, lane, "no second lane worktree is made")
}

// TestAGitThatSaysThereIsNoSystemAttributesFileHoldsNone: git 2.42 and later
// answer status 1 and nothing for a system attributes file switched off;
// that is held as none. Not watched failing: the round's base asked the same.
func TestAGitThatSaysThereIsNoSystemAttributesFileHoldsNone(t *testing.T) {
	if ok, _ := gitutil.GitAtLeast(2, 42); !ok {
		t.Skip("git on PATH is older than 2.42")
	}
	repo := briefRepo(t, agentsMarked)
	t.Setenv("GIT_ATTR_NOSYSTEM", "1")
	got, err := attributesFiles(personEnv, repo.Root(), "")
	if err != nil || len(got) != 1 {
		t.Fatalf("the global file only: %v %v", got, err)
	}
}

// TestAnIncludePathGitCannotResolveIsNamedByItsKey: a conditional include
// whose path git cannot expand stops the read, naming the setting by its key
// and scope. Watched failing: it was held under a salted marker naming no
// setting.
func TestAnIncludePathGitCannotResolveIsNamedByItsKey(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	repo.Git("config", "includeIf.onbranch:no-such-branch.path", "~no-such-user-of-this-machine/inc")
	_, err := readHeldSettings(personEnv, repo.Root(), "salt", map[string]string{})
	var rf *readFailure
	if !errors.As(err, &rf) || !strings.Contains(err.Error(), "includeif.") || !strings.Contains(err.Error(), ".path (local scope)") {
		t.Fatalf("the read stops naming the setting: %v", err)
	}
}

// TestALinkWhoseTextLeavesALinkedDirectoryIsNotProvedAsCleanedText: a
// committed hook whose link text is ../scripts/x/../y, where x stands as a
// link on disk, resolves where the system resolves it, not at scripts/y, so
// the start does not prove it. Watched failing: the start proved scripts/y
// and the drive started.
func TestALinkWhoseTextLeavesALinkedDirectoryIsNotProvedAsCleanedText(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	repo.Write("scripts/x/keep", "x\n")
	repo.Write("scripts/y", "#!/bin/sh\nexit 0\n")
	pushLinkedHook(t, repo, "../scripts/x/../y")
	elsewhere := filepath.Join(t.TempDir(), "a")
	mkdirs(t, filepath.Join(elsewhere, "b"))
	writeMode(t, filepath.Join(elsewhere, "y"), "#!/bin/sh\ntouch planted\n", 0o755)
	removeAll(t, filepath.Join(repo.Root(), "scripts", "x"))
	if err := os.Symlink(filepath.Join(elsewhere, "b"), filepath.Join(repo.Root(), "scripts", "x")); err != nil {
		t.Fatal(err)
	}
	tm, err := startDrive(t, repo)
	startRefused(t, tm, err, "itself a symlink")
}

// TestAHooksPathInsideTheGitDirectoryIsHeldAcrossLanes: a hooks directory
// under the git directory that is not git's default is not compared with
// origin's tip (no commit can hold it), and is held across lanes. Not
// watched failing: the round's base held it too; it holds that recognising
// the git directory by git's own answer keeps it so.
func TestAHooksPathInsideTheGitDirectoryIsHeldAcrossLanes(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	own := filepath.Join(repo.Root(), ".git", "my-hooks")
	mkdirs(t, own)
	writeMode(t, filepath.Join(own, "post-checkout"), "#!/bin/sh\nexit 0\n", 0o755)
	repo.Git("config", "core.hooksPath", ".git/my-hooks")
	if rel, err := ownHooksDir(repo.Root(), own); err != nil || rel != "" {
		t.Fatalf("the git directory is recognised by git's own answer: %q %v", rel, err)
	}
	lane, err := driveSecondLane(t, repo, func() {
		writeMode(t, filepath.Join(own, "post-checkout"), "#!/bin/sh\ntouch planted\n", 0o755)
	})
	worktreeRefused(t, err, "post-checkout (changed)")
	absent(t, lane, "no second lane worktree is made")
}

// TestAHooksDirectoryThatIsALinkIsCountedPastTheBound: the marker of a tree
// past the bound carries its count where the hooks directory is itself a
// link, so one more entry changes it. Watched failing: the count was 0.
func TestAHooksDirectoryThatIsALinkIsCountedPastTheBound(t *testing.T) {
	real := filepath.Join(t.TempDir(), "hooks")
	mkdirs(t, real)
	for i := 0; i <= maxHookEntries; i++ {
		writeMode(t, filepath.Join(real, fmt.Sprintf("h%05d", i)), "x\n", 0o644)
	}
	link := filepath.Join(t.TempDir(), "hooks")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if n := countEntries(link); n != maxHookEntries+1 {
		t.Fatalf("the linked directory is counted: %d", n)
	}
	before := map[string]string{}
	hashDir(before, "salt", link)
	writeMode(t, filepath.Join(real, "one-more"), "x\n", 0o644)
	after := map[string]string{}
	hashDir(after, "salt", link)
	if before[link] == "" || before[link] == after[link] {
		t.Fatal("one more entry past the bound changes the marker")
	}
}
