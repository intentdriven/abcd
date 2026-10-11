package cli

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The second half of "the status line must never do damage"
// (iss-2610050556383525): the harness waits on this verb on every refresh, so
// a hung git, a slow filesystem or a slow previous command must not freeze
// the person's status line. abcd's own work runs under one overall deadline,
// statuslineBudget, and the previous command under its own,
// previousCommandBudget; on either, the verb prints what it has — a partial
// row, the previous command's output so far, or nothing — and exits 0.

// These tests prove that a bound STOPS slow work, never how soon: a fixed
// wall-clock budget asserted on a race-instrumented, shared runner fails on
// scheduling jitter, not on a verb that waits (iss-2610080243367006). Each
// slow fixture would run for slowWork; the verb must return under
// stopCeiling, far above every budget and far below slowWork, so a verb that
// waited for the work still fails while a loaded runner a few seconds late
// does not. Which bound did the stopping is proven by the note the verb
// prints, which names that budget, and the production values are pinned by
// TestStatuslineBudgetIsUnderASecond.
const (
	slowWork    = 30 * time.Second
	stopCeiling = slowWork / 3
)

// slowSleep is slowWork as a sleep(1) argument, for the shell fixtures.
var slowSleep = "sleep " + strconv.Itoa(int(slowWork/time.Second))

// loadedGitLatency is how long the fake git takes before it hands a question
// it does not hang on to the real git: longer than the production git budget,
// as a loaded runner's git can be. A test whose row needs git's answer widens
// the budgets past it (widenBudgets), so its claim holds on a loaded machine
// too.
const loadedGitLatency = "sleep 0.4"

// widenBudgets resizes the verb's budgets for one test to sizes a loaded
// machine's real git cannot outrun, keeping their order (git's share inside
// abcd's ceiling, with room after it) and keeping every slow fixture far
// beyond them. The production values return when the test ends.
func widenBudgets(t *testing.T) {
	t.Helper()
	overall, git, previous := statuslineBudget, statuslineGitBudget, previousCommandBudget
	statuslineBudget, statuslineGitBudget, previousCommandBudget = 4*time.Second, 2*time.Second, 2*time.Second
	t.Cleanup(func() { statuslineBudget, statuslineGitBudget, previousCommandBudget = overall, git, previous })
}

// hasNote reports whether stderr carries a note of the verb's own, from
// "abcd statusline:" to the end of its line, that contains every one of want.
// The note need not open stderr, nor a line: a previous command's own stderr
// passes through ahead of it, with or without a final newline.
func hasNote(stderr string, want ...string) bool {
	for _, line := range strings.Split(stderr, "\n") {
		at := strings.Index(line, "abcd statusline:")
		if at < 0 {
			continue
		}
		all := true
		for _, w := range want {
			all = all && strings.Contains(line[at:], w)
		}
		if all {
			return true
		}
	}
	return false
}

// goneWithin reports whether no process holds pid within stopCeiling: a
// process the verb stopped is gone at once, and one it left running would
// still be sleeping slowWork. A stopped process is reaped by its parent
// (the verb's Wait, or the system for an orphan), so a short wait is not an
// excuse.
func goneWithin(pid int) bool {
	deadline := time.Now().Add(stopCeiling)
	for time.Now().Before(deadline) {
		if syscall.Kill(pid, 0) != nil {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

// readPid reads the pid a fixture wrote, or reports none when the fixture was
// stopped before its first line ran (a loaded machine): nothing of it is left
// to outlive the verb.
func readPid(t *testing.T, path string) (int, bool) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0, false
	}
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	return pid, true
}

// statuslineWithin runs the verb through the front door with stdin bound and
// fails the test, rather than hanging it, when the verb has not returned
// within stopCeiling: it waited for the slow work rather than stopping it.
func statuslineWithin(t *testing.T, stdin io.Reader) (stdout, stderr string, code int, took time.Duration) {
	t.Helper()
	type result struct {
		so, se string
		code   int
	}
	done := make(chan result, 1)
	start := time.Now()
	go func() {
		var so, se bytes.Buffer
		c := run([]string{"statusline"}, stdin, &so, &se)
		done <- result{so.String(), se.String(), c}
	}()
	select {
	case r := <-done:
		return r.so, r.se, r.code, time.Since(start)
	case <-time.After(stopCeiling):
		t.Fatalf("the status verb had not returned after %s; it waited for work it should have stopped, and the harness's status line would be frozen", stopCeiling)
	}
	return "", "", 0, 0
}

// fakeGit puts a `git` first on PATH that runs script (a POSIX sh body, with
// $REAL_GIT naming the real git) and returns nothing. The fixture's own git
// calls are made before it is installed.
//
// So is the process's one `git version` probe: gitutil asks it once, before
// the first git it runs, outside any deadline and under a lock, and caches
// the answer. Left to the verb, the probe would run the fake, and a fake that
// hangs would hold every git in the process for slowWork, so the test that
// happened to run first in the process (go test -run TestStatusline) failed
// at abcd's overall ceiling instead of git's budget, and so did each one
// after it (iss-2610090642407850).
func fakeGit(t *testing.T, script string) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not on PATH")
	}
	if _, err := gitutil.Run(t.TempDir(), "version"); err != nil {
		t.Fatalf("priming the git version probe with the real git: %v", err)
	}
	dir := t.TempDir()
	body := "#!/bin/sh\nREAL_GIT='" + real + "'\n" + script + "\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// TestStatuslineBudgetIsUnderASecond pins the documented values: abcd's own
// share of a refresh is bounded below one second, and the git share of it
// leaves room inside it for the rest of the row. The budgets are variables so
// the tests below can widen them; this is what keeps a changed production
// value from passing.
func TestStatuslineBudgetIsUnderASecond(t *testing.T) {
	if statuslineBudget != 500*time.Millisecond || statuslineGitBudget != 300*time.Millisecond || previousCommandBudget != 5*time.Second {
		t.Errorf("budgets = %s / %s / %s, want the documented 500ms / 300ms / 5s", statuslineBudget, statuslineGitBudget, previousCommandBudget)
	}
	if statuslineBudget <= 0 || statuslineBudget >= time.Second {
		t.Errorf("statuslineBudget = %s, want a positive bound under one second", statuslineBudget)
	}
	if statuslineGitBudget <= 0 || statuslineGitBudget >= statuslineBudget {
		t.Errorf("statuslineGitBudget = %s, want a positive bound inside statuslineBudget (%s)", statuslineGitBudget, statuslineBudget)
	}
	if previousCommandBudget <= 0 {
		t.Errorf("previousCommandBudget = %s, want a positive bound", previousCommandBudget)
	}
}

// TestStatuslineHungGitDoesNotFreezeTheLine: every git the verb starts sleeps
// for slowWork. The verb returns at git's budget, not at the work's end,
// exits 0, and says on stderr why the row is empty.
func TestStatuslineHungGitDoesNotFreezeTheLine(t *testing.T) {
	root := managedCheckout(t)
	widenBudgets(t)
	fakeGit(t, "exec "+slowSleep)

	stdout, stderr, code, took := statuslineWithin(t, strings.NewReader(payloadFor(root)))
	if code != 0 {
		t.Errorf("exit %d, want 0: a slow git is not an error the harness should see (stderr %q)", code, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want nothing: with no answer for the checkout there is no row to vouch for", stdout)
	}
	if !hasNote(stderr, "git", "within "+statuslineGitBudget.String()) {
		t.Errorf("stderr = %q, want one note naming git and its budget as what did not answer", stderr)
	}
	t.Logf("returned in %s (git budget %s)", took, statuslineGitBudget)
}

// TestStatuslineSlowBranchStillRendersThePartialRow: git answers for the
// checkout, as slowly as a loaded machine's git, but hangs on the branch. The
// row renders without the branch, badge first, once git's budget stops the
// branch — the best partial row — and the branch git is gone: killed at the
// budget, not abandoned to run on after the refresh. Like its siblings it
// proves the bound stops the work under the widened budgets and stopCeiling,
// never how soon (iss-2610090642407850).
func TestStatuslineSlowBranchStillRendersThePartialRow(t *testing.T) {
	root := managedCheckout(t)
	widenBudgets(t)
	pidFile := filepath.Join(t.TempDir(), "pids")
	fakeGit(t, `for a in "$@"; do case "$a" in symbolic-ref|--short) echo $$ >> '`+pidFile+`'; exec `+slowSleep+`;; esac; done
`+loadedGitLatency+`; exec "$REAL_GIT" "$@"`)

	stdout, stderr, code, took := statuslineWithin(t, strings.NewReader(payloadFor(root)))
	if code != 0 {
		t.Fatalf("exit %d (stderr %q), want 0", code, stderr)
	}
	if !strings.Contains(stdout, "abcd-managed") || !strings.Contains(stdout, filepath.Base(root)) {
		t.Errorf("stdout = %q, want the badge and the repository", stdout)
	}
	if strings.Contains(stdout, "main") {
		t.Errorf("stdout = %q carries a branch git never gave", stdout)
	}
	t.Logf("returned in %s (git budget %s)", took, statuslineGitBudget)

	// Every branch git that started must be gone. One stopped before its
	// first line ran (a loaded machine) wrote no pid and left nothing behind.
	raw, err := os.ReadFile(pidFile)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range strings.Fields(string(raw)) {
		pid, err := strconv.Atoi(field)
		if err != nil {
			t.Fatalf("pid file %q: %v", raw, err)
		}
		if !goneWithin(pid) {
			_ = syscall.Kill(pid, syscall.SIGKILL)
			t.Errorf("the branch git (pid %d) outlived the verb", pid)
		}
	}
}

// TestStatuslineWithoutGitReturnsAtOnce: no git on PATH at all. The checkout
// cannot be resolved, so the verb behaves as outside a managed repository —
// nothing recorded, nothing printed — quickly and at exit 0.
func TestStatuslineWithoutGitReturnsAtOnce(t *testing.T) {
	root := managedCheckout(t)
	t.Setenv("PATH", t.TempDir())

	stdout, stderr, code, _ := statuslineWithin(t, strings.NewReader(payloadFor(root)))
	if code != 0 || stdout != "" {
		t.Errorf("exit %d stdout %q stderr %q, want a quiet exit 0", code, stdout, stderr)
	}
}

// TestStatuslineStdinThatNeverClosesDoesNotFreezeTheLine is the hard ceiling:
// a step no git deadline reaches (here, a stdin the harness never closes)
// still ends at the overall budget, with nothing printed and exit 0.
func TestStatuslineStdinThatNeverClosesDoesNotFreezeTheLine(t *testing.T) {
	managedCheckout(t)
	r, w := io.Pipe()
	t.Cleanup(func() { _ = w.Close() })

	stdout, stderr, code, _ := statuslineWithin(t, r)
	if code != 0 || stdout != "" {
		t.Errorf("exit %d stdout %q, want nothing and exit 0", code, stdout)
	}
	if !hasNote(stderr, "within "+statuslineBudget.String()) {
		t.Errorf("stderr = %q, want one note saying abcd's budget ran out", stderr)
	}
}

// TestStatuslineSlowPreviousCommandIsStopped: outside a managed checkout the
// recorded previous command prints to both streams, starts a background child
// that records its pid, and sleeps for slowWork. The verb returns at the
// command's budget, not at the work's end, with the output the command had
// already printed, exits 0, and says why in a note of its own — after the
// command's own stderr, which passes through first (a shell that reports its
// killed jobs writes there too). The whole process group is stopped: the
// background child is gone too.
func TestStatuslineSlowPreviousCommandIsStopped(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(statuslineFallbackEnv, "")
	repo := t.TempDir()
	gitInitAt(t, repo)
	t.Chdir(repo)
	widenBudgets(t)
	pidFile := filepath.Join(t.TempDir(), "pid")
	writeUserSettings(t, `{"schema_version":1,"previous_command":"printf partial; printf own-stderr >&2; sh -c 'echo $$ > \"$0\"; exec `+slowSleep+`' '`+pidFile+`' & `+slowSleep+`"}`)

	stdout, stderr, code, took := statuslineWithin(t, strings.NewReader(""))
	if code != 0 {
		t.Errorf("exit %d, want 0 (stderr %q)", code, stderr)
	}
	if stdout != "partial" {
		t.Errorf("stdout = %q, want the output the command printed before it was stopped", stdout)
	}
	if !strings.HasPrefix(stderr, "own-stderr") {
		t.Errorf("stderr = %q, want the command's own stderr passed through first", stderr)
	}
	if !hasNote(stderr, "previous status command", "within "+previousCommandBudget.String()) {
		t.Errorf("stderr = %q, want one note naming the previous command and its budget as what was stopped", stderr)
	}
	t.Logf("returned in %s (budget %s)", took, previousCommandBudget)

	pid, ok := readPid(t, pidFile)
	if !ok {
		return
	}
	if !goneWithin(pid) {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Errorf("a background child of the previous command (pid %d) outlived the stop", pid)
	}
}

// TestStatuslineKillsTheRootCommitGit: a checkout managed through the history
// index alone carries no marker block, so whether abcd manages it needs git's
// root commit. That git hangs. The verb returns at git's budget with
// nothing printed — whether abcd manages the checkout is unknown, so neither
// abcd's row nor the previous command can be vouched for — and the git it
// started is gone: killed at the deadline, not left to run on after every
// refresh. The checkout's own git answers as slowly as a loaded machine's.
func TestStatuslineKillsTheRootCommitGit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv(statuslineFallbackEnv, "")
	widenBudgets(t)
	repo := t.TempDir()
	gitInitAt(t, repo)
	gitCommitAt(t, repo, "one")
	repo = realPath(t, repo)
	t.Chdir(repo)
	sha, err := gitutil.Run(repo, "rev-list", "-n", "1", "--max-parents=0", "HEAD")
	if err != nil || !gitutil.IsFullSHA(sha) {
		t.Fatalf("fixture root commit = %q, %v", sha, err)
	}
	histDir := abcdhome.Path(home, "history")
	if err := os.MkdirAll(histDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	index := `{"schema":1,"repos":[{"root_commit":"` + sha + `","path":"` + repo + `","status":"active"}]}`
	if err := os.WriteFile(filepath.Join(histDir, "index.json"), []byte(index), 0o600); err != nil {
		t.Fatal(err)
	}
	writeUserSettings(t, `{"schema_version":1,"previous_command":"printf previous"}`)
	// Control: with a git that answers, the checkout is abcd's.
	if out, _, code, _ := statuslineWithin(t, strings.NewReader(payloadFor(repo))); code != 0 || !strings.Contains(out, "abcd-managed") {
		t.Fatalf("control: stdout %q exit %d, want abcd's row for a registered checkout", out, code)
	}

	pidFile := filepath.Join(t.TempDir(), "pid")
	fakeGit(t, `for a in "$@"; do case "$a" in rev-list) echo $$ > '`+pidFile+`'; exec `+slowSleep+`;; esac; done
`+loadedGitLatency+`; exec "$REAL_GIT" "$@"`)

	stdout, stderr, code, _ := statuslineWithin(t, strings.NewReader(payloadFor(repo)))
	if code != 0 || stdout != "" {
		t.Errorf("exit %d stdout %q (stderr %q), want nothing and exit 0", code, stdout, stderr)
	}
	if !hasNote(stderr, "whether abcd manages this checkout", "within "+statuslineGitBudget.String()) {
		t.Errorf("stderr = %q, want one note saying git did not say whether abcd manages the checkout", stderr)
	}
	pid, ok := readPid(t, pidFile)
	if !ok {
		return
	}
	if !goneWithin(pid) {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Errorf("the root-commit git (pid %d) outlived the verb", pid)
	}
}

// TestStatuslineHelpStatesItWritesNothingAndIsBounded: the verb's help says
// the two things a person deciding whether to wire it needs — it writes
// nothing, and it is time-bounded — with the bounds the code enforces.
func TestStatuslineHelpStatesItWritesNothingAndIsBounded(t *testing.T) {
	var asJSON bool
	long := newStatuslineCommand(&asJSON).Long
	for _, want := range []string{"writes nothing", statuslineBudget.String(), previousCommandBudget.String()} {
		if !strings.Contains(long, want) {
			t.Errorf("the statusline help does not say %q:\n%s", want, long)
		}
	}
}
