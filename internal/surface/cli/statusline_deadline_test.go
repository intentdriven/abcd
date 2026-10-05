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

// deadlineMargin is the slack a timing assertion allows over a budget: process
// start-up, a loaded CI machine. It is generous on purpose: what these tests
// catch is a verb that waits for a ten-second sleep, not one that is a few
// hundred milliseconds late.
const deadlineMargin = 2 * time.Second

// statuslineWithin runs the verb through the front door with stdin bound and
// fails the test, rather than hanging it, when the verb has not returned
// within limit.
func statuslineWithin(t *testing.T, stdin io.Reader, limit time.Duration) (stdout, stderr string, code int, took time.Duration) {
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
	case <-time.After(limit):
		t.Fatalf("the status verb had not returned after %s; the harness's status line would be frozen", limit)
	}
	return "", "", 0, 0
}

// fakeGit puts a `git` first on PATH that runs script (a POSIX sh body, with
// $REAL_GIT naming the real git) and returns nothing. The fixture's own git
// calls are made before it is installed.
func fakeGit(t *testing.T, script string) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	body := "#!/bin/sh\nREAL_GIT='" + real + "'\n" + script + "\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// TestStatuslineBudgetIsUnderASecond pins the documented value: abcd's own
// share of a refresh is bounded below one second, and the git share of it
// leaves room inside it for the rest of the row.
func TestStatuslineBudgetIsUnderASecond(t *testing.T) {
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
// for ten seconds. The verb returns within its budget, exits 0, and says on
// stderr why the row is empty.
func TestStatuslineHungGitDoesNotFreezeTheLine(t *testing.T) {
	root := managedCheckout(t)
	fakeGit(t, "exec sleep 10")

	stdout, stderr, code, took := statuslineWithin(t, strings.NewReader(payloadFor(root)), statuslineBudget+deadlineMargin)
	if code != 0 {
		t.Errorf("exit %d, want 0: a slow git is not an error the harness should see (stderr %q)", code, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want nothing: with no answer for the checkout there is no row to vouch for", stdout)
	}
	if !strings.HasPrefix(stderr, "abcd statusline:") || !strings.Contains(stderr, "git") {
		t.Errorf("stderr = %q, want one note naming git as what did not answer", stderr)
	}
	t.Logf("returned in %s (budget %s)", took, statuslineBudget)
}

// TestStatuslineSlowBranchStillRendersThePartialRow: git answers for the
// checkout at once but hangs on the branch. The row renders without the
// branch, badge first, within the budget — the best partial row.
func TestStatuslineSlowBranchStillRendersThePartialRow(t *testing.T) {
	root := managedCheckout(t)
	fakeGit(t, `for a in "$@"; do case "$a" in symbolic-ref|--short) exec sleep 10;; esac; done
exec "$REAL_GIT" "$@"`)

	stdout, stderr, code, _ := statuslineWithin(t, strings.NewReader(payloadFor(root)), statuslineBudget+deadlineMargin)
	if code != 0 {
		t.Fatalf("exit %d (stderr %q), want 0", code, stderr)
	}
	if !strings.Contains(stdout, "abcd-managed") || !strings.Contains(stdout, filepath.Base(root)) {
		t.Errorf("stdout = %q, want the badge and the repository", stdout)
	}
	if strings.Contains(stdout, "main") {
		t.Errorf("stdout = %q carries a branch git never gave", stdout)
	}
}

// TestStatuslineWithoutGitReturnsAtOnce: no git on PATH at all. The checkout
// cannot be resolved, so the verb behaves as outside a managed repository —
// nothing recorded, nothing printed — quickly and at exit 0.
func TestStatuslineWithoutGitReturnsAtOnce(t *testing.T) {
	root := managedCheckout(t)
	t.Setenv("PATH", t.TempDir())

	stdout, stderr, code, _ := statuslineWithin(t, strings.NewReader(payloadFor(root)), statuslineBudget+deadlineMargin)
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

	stdout, stderr, code, _ := statuslineWithin(t, r, statuslineBudget+deadlineMargin)
	if code != 0 || stdout != "" {
		t.Errorf("exit %d stdout %q, want nothing and exit 0", code, stdout)
	}
	if !strings.HasPrefix(stderr, "abcd statusline:") {
		t.Errorf("stderr = %q, want one note saying the budget ran out", stderr)
	}
}

// TestStatuslineSlowPreviousCommandIsStopped: outside a managed checkout the
// recorded previous command sleeps far past its budget, and starts a
// background child that would write a file late. The verb returns at the
// budget with the output the command had already printed, exits 0, and the
// whole process group is stopped — the late write never happens.
func TestStatuslineSlowPreviousCommandIsStopped(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(statuslineFallbackEnv, "")
	repo := t.TempDir()
	gitInitAt(t, repo)
	t.Chdir(repo)
	late := filepath.Join(t.TempDir(), "late")
	writeUserSettings(t, `{"schema_version":1,"previous_command":"printf partial; (sleep 1; printf x > '`+late+`') & sleep 10"}`)

	orig := previousCommandBudget
	previousCommandBudget = 300 * time.Millisecond
	t.Cleanup(func() { previousCommandBudget = orig })

	stdout, stderr, code, took := statuslineWithin(t, strings.NewReader(""), previousCommandBudget+deadlineMargin)
	if code != 0 {
		t.Errorf("exit %d, want 0 (stderr %q)", code, stderr)
	}
	if stdout != "partial" {
		t.Errorf("stdout = %q, want the output the command printed before it was stopped", stdout)
	}
	if !strings.HasPrefix(stderr, "abcd statusline:") || !strings.Contains(stderr, "previous status command") {
		t.Errorf("stderr = %q, want one note naming the stopped previous command", stderr)
	}
	t.Logf("returned in %s (budget %s)", took, previousCommandBudget)

	time.Sleep(1500 * time.Millisecond)
	if _, err := os.Lstat(late); err == nil {
		t.Errorf("a child of the previous command outlived the stop and wrote %s", late)
	}
}

// TestStatuslineKillsTheRootCommitGit: a checkout managed through the history
// index alone carries no marker block, so whether abcd manages it needs git's
// root commit. That git hangs. The verb returns within its budget with
// nothing printed — whether abcd manages the checkout is unknown, so neither
// abcd's row nor the previous command can be vouched for — and the git it
// started is gone when it returns: killed at the deadline, not left to run on
// after every refresh.
func TestStatuslineKillsTheRootCommitGit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv(statuslineFallbackEnv, "")
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
	if out, _, code, _ := statuslineWithin(t, strings.NewReader(payloadFor(repo)), statuslineBudget+deadlineMargin); code != 0 || !strings.Contains(out, "abcd-managed") {
		t.Fatalf("control: stdout %q exit %d, want abcd's row for a registered checkout", out, code)
	}

	pidFile := filepath.Join(t.TempDir(), "pid")
	fakeGit(t, `for a in "$@"; do case "$a" in rev-list) echo $$ > '`+pidFile+`'; exec sleep 10;; esac; done
exec "$REAL_GIT" "$@"`)

	stdout, stderr, code, _ := statuslineWithin(t, strings.NewReader(payloadFor(repo)), statuslineBudget+deadlineMargin)
	if code != 0 || stdout != "" {
		t.Errorf("exit %d stdout %q (stderr %q), want nothing and exit 0", code, stdout, stderr)
	}
	if !strings.HasPrefix(stderr, "abcd statusline:") || !strings.Contains(stderr, "whether abcd manages this checkout") {
		t.Errorf("stderr = %q, want one note saying git did not say whether abcd manages the checkout", stderr)
	}
	raw, err := os.ReadFile(pidFile)
	if os.IsNotExist(err) {
		// Killed before its first line ran (a loaded machine): nothing of it
		// is left to outlive the verb.
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	// Give an abandoned git no excuse: a killed one is already reaped.
	time.Sleep(100 * time.Millisecond)
	if err := syscall.Kill(pid, 0); err == nil {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Errorf("the root-commit git (pid %d) outlived the verb", pid)
	}
}
