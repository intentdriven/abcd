package loop

// remoteid_bounded_test.go holds fix round 4 of iss-2611131308266006: every
// gitutil runner of a git with hooks, merge drivers or content filters on, or
// that reaches the network, asks the drive's comparison before it builds the
// command, so a change made after a step's own comparison and before that
// step's merge, pick commit or worktree add stops it (ruling 1); a
// comparison is bounded as a whole and its reads run side by side (ruling 2);
// and the tests say what they hold (ruling 5). A test whose comment says it
// was watched failing failed at the round's base, 77bf9a6a, or with the code
// it names taken out; the others say what they hold.

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// subcommand is the git subcommand args run, past any -c settings before it.
func subcommand(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "-c" {
			i++
			continue
		}
		return args[i]
	}
	return ""
}

// heldCheckAt sets, for the rest of the test, the check a drive sets
// (heldBeforeGit on root against id), with change made once at the ask of
// the first git command whose subcommand is verb: before that command's
// comparison when before, so the command meets the change, or after it, so
// the change stands once the command has run. It reports whether that ask
// came.
func heldCheckAt(t *testing.T, root string, id *remoteIdentity, verb string, before bool, change func()) *bool {
	t.Helper()
	asked := new(bool)
	unset, err := gitutil.SetBeforeHookedGit(func(args []string) error {
		at := !*asked && subcommand(args) == verb
		if at {
			*asked = true
			if before {
				change()
			}
		}
		err := heldBeforeGit(root, id)
		if at && !before {
			change()
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(unset)
	return asked
}

// notRun fails unless err stops the step at the config stage, saying git's
// verb was not run and naming want.
func notRun(t *testing.T, err error, verb, want string) {
	t.Helper()
	r := mustRefusal(t, err)
	if r.Stage != StageConfig || !strings.Contains(r.Reason, "git "+verb+" was not run") || !strings.Contains(r.Reason, want) {
		t.Fatalf("the step stops at the config stage, git %s not run, naming %q: %+v", verb, want, r)
	}
}

// TestALandStepsMergeDoesNotRunUnderAChangeMadeAfterItsFetch: the land step
// of an issue lane compares what the drive holds, fetches origin's default
// branch, which moved, and merges it into the lane. A held setting changed
// after the fetch and before the merge stops the step at the config stage,
// the merge not run: the lane's head is the judged one. Watched failing with
// runCommitBounded's ask taken out: the step went on with no refusal.
func TestALandStepsMergeDoesNotRunUnderAChangeMadeAfterItsFetch(t *testing.T) {
	f := syncFixture(t)
	judged := f.passToLandOnce(t, "fix.txt").HeadSHA
	tip := f.moveMain(t, "another account's file\n", "other.txt")
	id, err := readRemoteIdentity(f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	asked := heldCheckAt(t, f.repo.Root(), id, "merge", true, func() {
		f.repo.Git("config", "core.editor", "/bin/true")
	})
	_, err = advance(f.repo.Root(), f.runID, f.stages, Options{remote: id})
	notRun(t, err, "merge", "core.editor")
	if !*asked {
		t.Fatal("the merge asked the comparison")
	}
	if got := strings.TrimSpace(f.repo.Git("rev-parse", "refs/remotes/origin/main")); got != tip {
		t.Fatalf("the fetch ran before the merge was stopped: %s, want %s", got, tip)
	}
	if l := currentLane(t, f.repo, f.runID); l.HeadSHA != judged || strings.TrimSpace(f.repo.Git("rev-parse", l.Branch)) != judged {
		t.Fatalf("no merge was made: %+v", l)
	}
}

// TestAPickCommitDoesNotRunUnderAChangeMadeBeforeIt: a picked run's worktree
// stage adds the lane and makes the pick's record commit; a held setting
// changed after the add and before the commit stops the step at the config
// stage, the commit not run. Watched failing with runCommitBounded's ask
// taken out: the step went on with no refusal.
func TestAPickCommitDoesNotRunUnderAChangeMadeBeforeIt(t *testing.T) {
	repo := pickRepo(t, map[string][2]string{"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", fpSmall)}})
	res, err := Next(repo.Root(), Options{}, NextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	commits := repo.Git("rev-list", "--all")
	asked := heldCheckAt(t, repo.Root(), id, "commit", true, func() {
		repo.Git("config", "core.editor", "/bin/true")
	})
	_, err = advance(repo.Root(), res.Start.RunID, DefaultStages(), Options{remote: id})
	notRun(t, err, "commit", "core.editor")
	if !*asked {
		t.Fatal("the pick commit asked the comparison")
	}
	if l := currentLane(t, repo, res.Start.RunID); l.PickSHA != "" || repo.Git("rev-list", "--all") != commits {
		t.Fatalf("no pick commit was made: %+v", l)
	}
}

// TestALanesWorktreeAddDoesNotRunUnderAChangeMadeBeforeIt: a held setting
// changed after the worktree stage's own comparisons and before git's add of
// the lane's worktree stops the step at the config stage, the add not run.
// Watched failing with runCommitBounded's ask taken out: the step went on
// with no refusal.
func TestALanesWorktreeAddDoesNotRunUnderAChangeMadeBeforeIt(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	start, err := Start(repo.Root(), "itd-10", Options{noReserve: true})
	if err != nil {
		t.Fatal(err)
	}
	lw, err := laneWorktree(repo.Root(), start.RunID, "lane-1")
	if err != nil {
		t.Fatal(err)
	}
	asked := heldCheckAt(t, repo.Root(), id, "worktree", true, func() {
		repo.Git("config", "core.editor", "/bin/true")
	})
	_, err = advance(repo.Root(), start.RunID, DefaultStages(), Options{remote: id})
	notRun(t, err, "worktree", "core.editor")
	if !*asked {
		t.Fatal("the add asked the comparison")
	}
	absent(t, lw.Path, "no lane worktree is added")
}

// TestAChangeMadeDuringTheAskOfOriginIsMetAtTheWorktreeStage: the step's own
// comparison passes, and a held setting changes while the worktree stage asks
// origin for its tip; the stage's comparison after the ask, made on the hold
// the step passed into the stage's Context, refuses the lane at the worktree
// stage before anything else runs. Watched failing with the hold not passed
// into the stage's Context (advanceWith's move given nil): the step was
// stopped later, at the config stage, by the add's own ask.
func TestAChangeMadeDuringTheAskOfOriginIsMetAtTheWorktreeStage(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	start, err := Start(repo.Root(), "itd-10", Options{noReserve: true})
	if err != nil {
		t.Fatal(err)
	}
	asked := heldCheckAt(t, repo.Root(), id, "ls-remote", false, func() {
		repo.Git("config", "core.editor", "/bin/true")
	})
	_, err = advance(repo.Root(), start.RunID, DefaultStages(), Options{remote: id})
	r := mustRefusal(t, err)
	if r.Stage != string(StageWorktree) || !strings.Contains(r.Reason, "settings the drive holds across its lanes") || !strings.Contains(r.Reason, "core.editor") {
		t.Fatalf("the change is met at the worktree stage's own comparison: %+v", r)
	}
	if !*asked {
		t.Fatal("origin was asked")
	}
}

// TestAPushUnderAPrePushChangedSinceTheDriveStartedDoesNotRun: the person's
// global settings name a hooks directory, which the push runs and the lane's
// add and commit do not; its pre-push changed since the drive read what it
// holds. The landing push's runner, under the drive's check, does not run:
// the changed hook records nothing and origin does not move. With no check
// set the same push runs the hook. Watched failing with runHookedContext's
// ask taken out: the push under the changed pre-push returned no error.
func TestAPushUnderAPrePushChangedSinceTheDriveStartedDoesNotRun(t *testing.T) {
	hooks := globalHooks(t)
	repo, bare, _ := hookedOrigin(t)
	repo.Git("config", "--unset", "core.hooksPath")
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	ran := filepath.Join(t.TempDir(), "pre-push-ran")
	writeMode(t, filepath.Join(hooks, "pre-push"), "#!/bin/sh\n: > '"+ran+"'\n", 0o755)
	repo.Git("commit", "-q", "--allow-empty", "-m", "to push")
	before := repo.Git("ls-remote", bare)
	unset, err := gitutil.SetBeforeHookedGit(func([]string) error { return heldBeforeGit(repo.Root(), id) })
	if err != nil {
		t.Fatal(err)
	}
	_, err = gitutil.Net(repo.Root(), "push", "-q", "origin", "HEAD:refs/heads/main")
	unset()
	if r, ok := AsRefusal(err); !ok || r.Stage != StageConfig || !strings.Contains(err.Error(), "git push was not run") || !strings.Contains(r.Reason, "pre-push (changed)") {
		t.Fatalf("the push is not run under the changed pre-push: %v", err)
	}
	absent(t, ran, "the changed pre-push does not run")
	if repo.Git("ls-remote", bare) != before {
		t.Fatal("origin did not move")
	}
	if _, err := gitutil.Net(repo.Root(), "push", "-q", "origin", "HEAD:refs/heads/main"); err != nil {
		t.Fatalf("with no check set the push runs: %v", err)
	}
	if _, err := os.Stat(ran); err != nil {
		t.Fatalf("with no check set the push runs the hook: %v", err)
	}
}

// slowGit makes every read the comparison makes in each environment (the
// settings listing, the attributes files, the hooks directory and the common
// git directory) wait far past any bound, and shortens the comparison's
// overall bound to whole while each read keeps a longer one of its own.
func slowGit(t *testing.T, whole time.Duration) {
	t.Helper()
	standInGit(t, `*"--show-scope --show-origin"*|*"var GIT_ATTR_"*|*"--git-path hooks"*|*"--git-common-dir"*) exec /bin/sleep 600 ;;`)
	bound, wait, overall := heldReadTimeout, heldReadRetryWait, heldCompareTimeout
	heldReadTimeout, heldReadRetryWait, heldCompareTimeout = 10*time.Minute, 200*time.Millisecond, whole
	t.Cleanup(func() { heldReadTimeout, heldReadRetryWait, heldCompareTimeout = bound, wait, overall })
}

// TestAComparisonSlowOnEveryReadEndsAtItsOverallBound: with git slow on every
// read and each read bounded at ten minutes, a comparison ends at its own
// overall bound, long before any read's: the step stops at the config stage
// saying the comparison did not finish, never that anything changed or that
// nothing did. Watched failing at the base, with each read bounded at 6s
// there: it ran its first read out to that read's own bound and named no
// bound of the comparison's.
func TestAComparisonSlowOnEveryReadEndsAtItsOverallBound(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	slowGit(t, 2*time.Second)
	began := time.Now()
	err = heldBeforeGit(repo.Root(), id)
	took := time.Since(began)
	r := configRefused(t, err, "did not finish within 2s", "neither as changed nor as unchanged")
	if strings.Contains(r.Reason, "changed since it started") || took >= heldReadTimeout/2 {
		t.Fatalf("the comparison ends at its own bound, not a read's (%s), and names no change: %+v", took, r)
	}
}

// readsRunning counts the goroutines of this process inside a read of the
// hold (gitEnv.readBy).
func readsRunning() int {
	buf := make([]byte, 1<<20)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return strings.Count(string(buf[:n]), "loop.(*gitEnv).readBy(")
		}
		buf = make([]byte, 2*len(buf))
	}
}

// TestNoReadOfAComparisonIsLeftRunningOnceItEndsAtItsDeadline: with every
// read of the comparison slow, each one's git leaving a program behind that
// holds its output open for four seconds after git is killed, a comparison
// bounded at one second ends saying it did not finish, and when it has
// returned no read of it is still running, and none starts afterwards. The
// stand-in records each read's start; it cannot record an end, since a read
// the deadline cuts has its git killed, so the end is counted in the test:
// no goroutine is inside a read once the comparison has returned. Watched
// failing at f2f1169a: the comparison returned at its bound with the reads
// in the person's own environment and the stage's still waiting on their
// git's output.
func TestNoReadOfAComparisonIsLeftRunningOnceItEndsAtItsDeadline(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(t.TempDir(), "reads")
	standInGit(t, `*"--show-scope --show-origin"*|*"var GIT_ATTR_"*|*"--git-path hooks"*|*"--git-common-dir"*) echo start >> '`+log+`'; /bin/sleep 4; exit 0 ;;`)
	const retry = 200 * time.Millisecond
	bound, wait, overall := heldReadTimeout, heldReadRetryWait, heldCompareTimeout
	heldReadTimeout, heldReadRetryWait, heldCompareTimeout = 10*time.Minute, retry, time.Second
	t.Cleanup(func() { heldReadTimeout, heldReadRetryWait, heldCompareTimeout = bound, wait, overall })
	starts := func() int {
		b, _ := os.ReadFile(log)
		return strings.Count(string(b), "start")
	}
	err = heldBeforeGit(repo.Root(), id)
	running, started := readsRunning(), starts()
	configRefused(t, err, "did not finish within 1s", "neither as changed nor as unchanged")
	if running != 0 {
		t.Fatalf("no read of the comparison is still running once it has returned: %d are", running)
	}
	if started < 1 {
		t.Fatalf("the comparison's reads started: none did")
	}
	time.Sleep(3 * retry)
	if n := starts(); n != started {
		t.Fatalf("no read of the comparison starts once it has returned: %d had started, then %d", started, n)
	}
}

// TestAComparisonsReadsRunSideBySide: the stand-in git records each read of
// the comparison as it starts and as it ends, and holds every read until at
// least two have started: reads made one after another would never get past
// the first, and the comparison would end at its bound. So the comparison
// finishing with nothing changed, the log showing two reads started before
// any ended, is the reads overlapping; no time is measured. Watched failing
// with the comparison's reads made one at a time (a lock round each read):
// the first read waited for a second that never started, and the step
// stopped at its bound. A read's own bound is set past the comparison's, so
// no read is cut and asked again: a retry's start would otherwise count as a
// second read's, and reads made one after another passed.
func TestAComparisonsReadsRunSideBySide(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	bound := heldReadTimeout
	heldReadTimeout = 10 * heldCompareTimeout
	t.Cleanup(func() { heldReadTimeout = bound })
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(t.TempDir(), "reads")
	standInGit(t, `*"--show-scope --show-origin"*|*"var GIT_ATTR_"*|*"--git-path hooks"*|*"--git-common-dir"*) echo start >> '`+log+`'; `+
		`while [ "$(grep -c start '`+log+`')" -lt 2 ]; do /bin/sleep 0.05; done; echo end >> '`+log+`' ;;`)
	if err := heldBeforeGit(repo.Root(), id); err != nil {
		t.Fatalf("the comparison finishes, its reads side by side: %v", err)
	}
	b, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	events := strings.Fields(string(b))
	first := slices.Index(events, "end")
	if first < 2 || strings.Count(string(b), "start") < 8 || strings.Count(string(b), "start") != strings.Count(string(b), "end") {
		t.Fatalf("two or more of the comparison's dozen reads started before any ended, and each ended: %q", events)
	}
}

// TestALetGoComparesBeforeItTakesTheDrainLock: a drive's let-go compares what
// it holds before it takes the drain's lock, whose wait other sessions keep
// short, so a comparison slow on every read ends at its own bound with the
// lock never asked for: here another session holds the lock throughout, and
// the let-go still ends saying nothing was released because the comparison
// did not finish, not that the lock was busy. Watched failing with the
// comparison before the lock taken out, so that the release's pushes asked it
// under the lock: the let-go waited on the lock for its 3s and said only that
// the lock could not be had.
func TestALetGoComparesBeforeItTakesTheDrainLock(t *testing.T) {
	w := newTerminalWorld(t)
	id, err := readRemoteIdentity(w.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	tm := newTerminal(w.repo.Root(), w.options(t))
	tm.o.Options.remote = id
	slowGit(t, 2*time.Second)
	lock := filepath.Join(w.repo.Root(), filepath.FromSlash(drainLockRel))
	if err := fsutil.EnsureRealDirAll(w.repo.Root(), RunRelDir, dirPerm); err != nil {
		t.Fatal(err)
	}
	if err := fsutil.WithFileLock(lock, lockTimeout, func() error { tm.letGo(); return nil }); err != nil {
		t.Fatal(err)
	}
	said := w.said()
	if !strings.Contains(said, "nothing this drive holds was released") || !strings.Contains(said, "did not finish within 2s") || strings.Contains(said, "lapses with its lease: ") {
		t.Fatalf("the let-go says nothing was released because the comparison did not finish, never that the lock was busy:\n%s", said)
	}
}
