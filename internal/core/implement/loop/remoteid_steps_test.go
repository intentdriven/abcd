package loop

// remoteid_steps_test.go holds fix round 3 of iss-2611131308266006: every read
// the per-run check makes stops the step when it fails, but for git's own
// exact answer that no repository stands at a lane's path (ruling 1); a drive
// compares what it holds before each step and before every git command it
// runs that runs hooks or reaches the network (ruling 2); and a read that does
// not finish is asked once more before it stops a step (ruling 3). A test
// whose comment says it was watched failing failed at the round's base,
// 39d5b0fb; the others say what they hold.

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestAnIncludeReadableInOneEnvironmentOnlyStopsTheStep: a lane's settings
// name a home-form include that git refuses under the home abcd's isolated
// reads run in and that is absent under the stage's home, where the lane's
// add and commit run. The check's read of the lane fails where the lane's
// own git would not, and git there says a repository stands, so the step
// stops at the config stage naming the read, never passing the lane over as
// "git cannot read it either". Watched failing: the lane was passed over and
// the check went on. (At the checkout itself the same include stops the step
// earlier, in the read of the run's state, at the base too.)
func TestAnIncludeReadableInOneEnvironmentOnlyStopsTheStep(t *testing.T) {
	stagedGit(t)
	repo, _ := recordedRun(t)
	lane := laneRepo(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeMode(t, filepath.Join(home, "inc.gitconfig"), "[core\n", 0o644)
	appendFile(t, filepath.Join(lane, ".git", "config"), "[include]\n\tpath = ~/inc.gitconfig\n")
	err := fingerprintOf(t, repo.Root(), lane)
	var rf *readFailure
	if !errors.As(err, &rf) {
		t.Fatalf("the lane's read stops the check: %v", err)
	}
	configRefused(t, readRefusal(repo.Root(), err), "git config --get-regexp", "abcd's isolated reads", "at a lane's path")
}

// laneRepo is a repository of its own at a lane's path, with settings.
func laneRepo(t *testing.T, settings ...[2]string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "lane")
	mkdirs(t, dir)
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = gitutil.IsolatedEnv()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	for _, kv := range settings {
		run("config", kv[0], kv[1])
	}
	return dir
}

// fingerprintOf takes the run's fingerprint of repo with lane as a live lane.
func fingerprintOf(t *testing.T, root, lane string) error {
	t.Helper()
	_, err := takeFingerprint(root, []laneDir{{Path: lane, Live: true}}, "salt", time.Now())
	return err
}

// TestGitsWordsInsideAPathAreNotItsAnswerOfNone: git's "not a git
// repository" counts as none only as the first words of its error at exit
// status 128. The same words inside a path git repeats in another error
// stop the step, naming the read; a lane's path where git says exactly that,
// or where nothing stands, holds nothing and the check goes on. Watched
// failing: the error that repeated the words in a path was taken as none.
func TestGitsWordsInsideAPathAreNotItsAnswerOfNone(t *testing.T) {
	repo, _ := recordedRun(t)
	t.Run("the words inside a path", func(t *testing.T) {
		lane := laneRepo(t)
		standInGit(t, `*"`+lane+`"*"rev-parse"*) echo "fatal: bad config line 1 in file `+lane+`/fatal: not a git repository/config" >&2; exit 128 ;;`)
		err := fingerprintOf(t, repo.Root(), lane)
		var rf *readFailure
		if !errors.As(err, &rf) || !strings.Contains(err.Error(), "git rev-parse --git-path hooks") {
			t.Fatalf("the read stops the check, named: %v", err)
		}
		configRefused(t, readRefusal(repo.Root(), err), "git rev-parse --git-path hooks")
	})
	t.Run("git's own answer", func(t *testing.T) {
		if err := fingerprintOf(t, repo.Root(), t.TempDir()); err != nil {
			t.Fatalf("a path git says holds no repository is passed over: %v", err)
		}
	})
	t.Run("nothing at the path", func(t *testing.T) {
		if err := fingerprintOf(t, repo.Root(), filepath.Join(t.TempDir(), "gone")); err != nil {
			t.Fatalf("a path where nothing stands is passed over: %v", err)
		}
	})
	t.Run("another refusal at status 128", func(t *testing.T) {
		lane := laneRepo(t)
		standInGit(t, `*"`+lane+`"*"config"*|*"`+lane+`"*"rev-parse"*) echo "fatal: detected dubious ownership in repository at '`+lane+`'" >&2; exit 128 ;;`)
		err := fingerprintOf(t, repo.Root(), lane)
		var rf *readFailure
		if !errors.As(err, &rf) || !strings.Contains(err.Error(), "git config --get-regexp") {
			t.Fatalf("a lane whose settings git refuses to read stops the check, named: %v", err)
		}
	})
}

// TestALanesHooksDirectoryGitCannotNameStopsTheStep: the checkout reads
// whole, and a live lane's own core.hooksPath holds a newline, which git
// answers on two lines: the lane's branch of the check stops, naming the read
// and why. It held at the base too (the lane's branch had no test of its own).
func TestALanesHooksDirectoryGitCannotNameStopsTheStep(t *testing.T) {
	repo, _ := recordedRun(t)
	lane := laneRepo(t, [2]string{"core.hooksPath", "hooks\nelsewhere"})
	err := fingerprintOf(t, repo.Root(), lane)
	var rf *readFailure
	if !errors.As(err, &rf) || !strings.Contains(err.Error(), "git rev-parse --git-path hooks") || !strings.Contains(err.Error(), "newline") {
		t.Fatalf("the lane's hooks read stops the check, named: %v", err)
	}
}

// TestAReadSlowOnceIsAskedAgainAndTheStepGoesOn: git asked for the hooks
// directory does not finish the first time and answers the second: the step
// goes on. Watched failing: the step stopped on the first ask.
func TestAReadSlowOnceIsAskedAgainAndTheStepGoesOn(t *testing.T) {
	repo, runID := recordedRun(t)
	quickRetry(t)
	mark := filepath.Join(t.TempDir(), "asked")
	standInGit(t, `*"--git-path hooks"*) if [ ! -e '`+mark+`' ]; then : > '`+mark+`'; exec sleep 30; fi ;;`)
	if err := configTripwire(repo.Root(), runID); err != nil {
		t.Fatalf("a read slow once is asked again and the step goes on: %v", err)
	}
	if _, err := os.Stat(mark); err != nil {
		t.Fatalf("the first ask was the slow one: %v", err)
	}
}

// quickRetry shortens a read's bound and its wait before the second ask for
// the rest of the test.
func quickRetry(t *testing.T) {
	t.Helper()
	bound, wait := heldReadTimeout, heldReadRetryWait
	heldReadTimeout, heldReadRetryWait = 4*time.Second, 200*time.Millisecond
	t.Cleanup(func() { heldReadTimeout, heldReadRetryWait = bound, wait })
}

// globalHooks sets a global settings file for the person's own environment
// that names a hooks directory holding a pre-push, and returns the
// directory; the stage's environment reads a global file of its own
// (stagedGit), which names none.
func globalHooks(t *testing.T) string {
	t.Helper()
	stagedGit(t)
	hooks := filepath.Join(t.TempDir(), "global-hooks")
	mkdirs(t, hooks)
	writeMode(t, filepath.Join(hooks, "pre-push"), "#!/bin/sh\nexit 0\n", 0o755)
	global := filepath.Join(t.TempDir(), "gitconfig")
	writeMode(t, global, "[core]\n\thooksPath = "+hooks+"\n", 0o644)
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	return hooks
}

// changedDuringTheLane has the world's validate stage, which runs after the
// lane's agent, make change, standing in for that agent; its land stage
// records that it ran. It returns where the landing is recorded and what
// origin held just before the change.
func (w *terminalWorld) changedDuringTheLane(t *testing.T, change func()) (landed *bool, before *string) {
	t.Helper()
	landed, before = new(bool), new(string)
	for i := range w.stages {
		switch w.stages[i].Name {
		case StageValidate:
			w.stages[i].Run = func(Context, *Lane) (Outcome, error) {
				*before = w.repo.Git("ls-remote", w.bare)
				change()
				return Outcome{Note: "validate done"}, nil
			}
		case StageLand:
			w.stages[i].Run = func(Context, *Lane) (Outcome, error) {
				*landed = true
				return Outcome{Note: "land done"}, nil
			}
		}
	}
	return landed, before
}

// heldStop fails unless the drive ended refused at the config stage naming
// want, ran no landing, and pushed nothing to origin after the change.
func (w *terminalWorld) heldStop(t *testing.T, res TerminalResult, err error, landed bool, before, want string) {
	t.Helper()
	r, ok := AsRefusal(err)
	if !ok || r.Stage != StageConfig || !strings.Contains(r.Reason, want) || res.Ended != TerminalRefused {
		t.Fatalf("the drive stops at the config stage naming %q: %+v %v\n%s", want, res, err, w.said())
	}
	if landed {
		t.Fatalf("the lane's landing does not run under what changed:\n%s", w.said())
	}
	if before == "" || w.repo.Git("ls-remote", w.bare) != before {
		t.Fatalf("no reservation push or release reaches origin after the change:\n%s", w.said())
	}
}

// TestALanesOwnPushDoesNotRunUnderAPrePushChangedDuringTheLane: a pre-push in
// the directory named only in the person's global settings, which the
// lane's add and commit do not read and the push does, changed by the lane's
// agent: that lane's landing does not run, no push of the drive's runs the
// changed hook (the hook records whether it ran), and the drive ends at the
// config stage naming it. Watched failing: the landing ran.
func TestALanesOwnPushDoesNotRunUnderAPrePushChangedDuringTheLane(t *testing.T) {
	w := newTerminalWorld(t)
	hooks := globalHooks(t)
	ran := filepath.Join(t.TempDir(), "pre-push-ran")
	landed, before := w.changedDuringTheLane(t, func() {
		writeMode(t, filepath.Join(hooks, "pre-push"), "#!/bin/sh\n: > '"+ran+"'\n", 0o755)
	})
	res, err := RunTerminal(context.Background(), w.repo.Root(), w.bounded(t, 3*time.Hour))
	w.heldStop(t, res, err, *landed, *before, "pre-push (changed)")
	absent(t, ran, "the changed pre-push never runs")
}

// TestAReservationPushDoesNotRunUnderAnSSHCommandChangedDuringTheLane: a
// core.sshCommand set in the person's global settings, changed by the lane's
// agent: no reservation push or release reaches origin after it, and the
// drive ends at the config stage naming the key. Watched failing: the lane
// landed and its release was pushed.
func TestAReservationPushDoesNotRunUnderAnSSHCommandChangedDuringTheLane(t *testing.T) {
	w := newTerminalWorld(t)
	stagedGit(t)
	global := filepath.Join(t.TempDir(), "gitconfig")
	writeMode(t, global, "[core]\n\tsshCommand = ssh\n", 0o644)
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	landed, before := w.changedDuringTheLane(t, func() {
		writeMode(t, global, "[core]\n\tsshCommand = /bin/true\n", 0o644)
	})
	res, err := RunTerminal(context.Background(), w.repo.Root(), w.bounded(t, 3*time.Hour))
	w.heldStop(t, res, err, *landed, *before, "core.sshcommand")
}

// TestAHookedGitOutsideAStepIsNotRunUnderAChangedHold: with a drive's check
// set, a reservation's fetch, a git command outside any step, fails naming
// the difference and does not run; with it cleared it runs.
func TestAHookedGitOutsideAStepIsNotRunUnderAChangedHold(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	gitutil.SetBeforeHookedGit(func() error { return heldBeforeGit(repo.Root(), id) })
	t.Cleanup(func() { gitutil.SetBeforeHookedGit(nil) })
	if _, err := gitutil.Net(repo.Root(), "fetch", "-q", "origin"); err != nil {
		t.Fatalf("an unchanged hold lets the fetch run: %v", err)
	}
	repo.Git("config", "core.editor", "/bin/true")
	_, err = gitutil.Net(repo.Root(), "fetch", "-q", "origin")
	if r, ok := AsRefusal(err); !ok || r.Stage != StageConfig || !strings.Contains(r.Reason, "core.editor") || !strings.Contains(err.Error(), "was not run") {
		t.Fatalf("the fetch is not run under a changed hold: %v", err)
	}
	gitutil.SetBeforeHookedGit(nil)
	if _, err := gitutil.Net(repo.Root(), "fetch", "-q", "origin"); err != nil {
		t.Fatalf("with no check set the fetch runs: %v", err)
	}
}
