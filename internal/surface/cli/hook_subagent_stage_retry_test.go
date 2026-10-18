package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// iss-2610051434156079: a burst of sub-agent completions lost transcripts on a
// loaded runner while every hook exited 0. The cause is a git call that fails
// once: RootCommit reads it as no root commit, the cwd then reads as outside
// any repository, a burst's first hooks run before any sibling has noted the
// session, and the transcript was dropped with only a stderr line no one sees.

// failRootCommitGit puts a `git` first on PATH that fails the root-commit
// lookup (`rev-list … --max-parents=0`) for its first n calls, as a fork
// refused or a git killed early on a loaded machine does, and runs the real
// git for everything else. The count is shared across processes, so a burst
// of hooks takes the failures between them.
func failRootCommitGit(t *testing.T, n int) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git unavailable")
	}
	dir := t.TempDir()
	count := filepath.Join(t.TempDir(), "calls")
	lock := count + ".lock"
	body := "#!/bin/sh\n" +
		"case \"$*\" in\n" +
		"*max-parents*)\n" +
		"  while ! mkdir '" + lock + "' 2>/dev/null; do :; done\n" +
		"  k=$(cat '" + count + "' 2>/dev/null || echo 0); k=$((k+1)); echo $k > '" + count + "'\n" +
		"  rmdir '" + lock + "'\n" +
		"  if [ $k -le " + strconv.Itoa(n) + " ]; then echo 'fatal: simulated transient failure' >&2; exit 128; fi;;\n" +
		"esac\n" +
		"exec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// TestHookSubagentStopRetriesATransientRootFailure: one failed root-commit
// lookup no longer drops the transcript; the hook asks again and stages.
func TestHookSubagentStopRetriesATransientRootFailure(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	tp := writeAgentTranscript(t, t.TempDir(), "r1", `{"agent":"r1"}`+"\n")
	failRootCommitGit(t, 1)

	_, stderr, failed := runHookAllowingFailure(subagentPayload(t, "sess-retry", repo, "r1", tp, "general-purpose"), "hook", "subagent-stop")
	if failed {
		t.Fatalf("the hook exited non-zero: %s", stderr)
	}
	staged, err := history.ListStaged(repo, rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(staged) != 1 {
		t.Fatalf("a transient git failure dropped the transcript: %d staged\nstderr: %s", len(staged), stderr)
	}
	if !strings.Contains(stderr, "resolved on try") {
		t.Errorf("stderr does not say the repository resolved on a retry: %q", stderr)
	}
}

// TestHookSubagentStopBurstSurvivesTransientGitFailures is the record's burst
// on a loaded machine made deterministic: twelve completions at once, with the
// first six root-commit lookups failing, and every transcript staged.
func TestHookSubagentStopBurstSurvivesTransientGitFailures(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	dir := t.TempDir()
	const n = 12
	payloads := make([]string, n)
	for i := range n {
		id := "t" + string(rune('a'+i))
		tp := writeAgentTranscript(t, dir, id, `{"agent":"`+id+`"}`+"\n")
		payloads[i] = subagentPayload(t, "sess-burst-fail", repo, id, tp, "general-purpose")
	}
	failRootCommitGit(t, n/2)

	var wg sync.WaitGroup
	stderrs := make([]string, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, stderrs[i], _ = runHookAllowingFailure(payloads[i], "hook", "subagent-stop")
		}()
	}
	wg.Wait()
	staged, err := history.ListStaged(repo, rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(staged) != n {
		t.Fatalf("want %d staged from %d completions with transient git failures, got %d\n%s", n, n, len(staged), notStaged(stderrs))
	}
}

// stagingLockFor is the path of the staging lock one agent's stage takes:
// history keys it on the agent id under the staging directory's locks folder.
func stagingLockFor(t *testing.T, repo, rootSHA, agentID string) string {
	t.Helper()
	store, err := history.Resolve(repo, rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	locks := filepath.Join(store.Staging, "locks")
	if err := fsutil.EnsureRealDir(store.Staging, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := fsutil.EnsureRealDir(locks, 0o700); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(locks, agentID+".lock")
}

// holdLock takes path's lock and holds it for d, signalling once it is held.
func holdLock(t *testing.T, path string, d time.Duration) (held <-chan struct{}, done <-chan error) {
	t.Helper()
	h, out := make(chan struct{}), make(chan error, 1)
	go func() {
		out <- fsutil.WithFileLock(path, time.Second, func() error {
			close(h)
			time.Sleep(d)
			return nil
		})
	}()
	return h, out
}

// TestHookSubagentStopWaitsOutAHeldStagingLock: a staging lock held past its
// own five-second timeout (the record's deliberately slowed lock) is waited
// out by a retry, not reported as a lost transcript.
func TestHookSubagentStopWaitsOutAHeldStagingLock(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	tp := writeAgentTranscript(t, t.TempDir(), "w1", `{"agent":"w1"}`+"\n")
	held, done := holdLock(t, stagingLockFor(t, repo, rootSHA, "w1"), 7*time.Second)
	<-held

	_, stderr, failed := runHookAllowingFailure(subagentPayload(t, "sess-held", repo, "w1", tp, "general-purpose"), "hook", "subagent-stop")
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if failed {
		t.Fatalf("the hook exited non-zero: %s", stderr)
	}
	staged, err := history.ListStaged(repo, rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(staged) != 1 {
		t.Fatalf("a held staging lock dropped the transcript: %d staged\nstderr: %s", len(staged), stderr)
	}
	if !strings.Contains(stderr, "trying again") {
		t.Errorf("stderr does not say the stage was tried again: %q", stderr)
	}
	if _, ok, _ := history.StageMiss(repo, rootSHA); ok {
		t.Error("a stage that succeeded on a retry was recorded as missed")
	}
}

// TestHookSubagentStopMarksAStageThatStillFails: a stage that fails for good,
// with the store known, is counted in a marker that `abcd history staged`
// reports, so the loss is visible to the person and not only in a debug log.
func TestHookSubagentStopMarksAStageThatStillFails(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	tp := writeAgentTranscript(t, t.TempDir(), "m1", `{"agent":"m1"}`+"\n")
	prev := subagentStageRetries
	subagentStageRetries = 0
	t.Cleanup(func() { subagentStageRetries = prev })
	held, done := holdLock(t, stagingLockFor(t, repo, rootSHA, "m1"), 7*time.Second)
	<-held

	_, stderr, failed := runHookAllowingFailure(subagentPayload(t, "sess-miss", repo, "m1", tp, "general-purpose"), "hook", "subagent-stop")
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if failed {
		t.Fatalf("the hook exited non-zero, which blocks the sub-agent: %s", stderr)
	}
	miss, ok, err := history.StageMiss(repo, rootSHA)
	if err != nil || !ok {
		t.Fatalf("the missed stage left no marker (%v)\nstderr: %s", err, stderr)
	}
	if miss.Count != 1 || miss.LastWho != "sub-agent m1" || !strings.Contains(miss.LastReason, "lock") {
		t.Errorf("marker = %+v; want one miss by m1 naming the lock", miss)
	}
	t.Chdir(repo)
	if out := string(runCLI(t, "history", "staged")); !strings.Contains(out, "1 transcript(s) were read but could not be staged") {
		t.Errorf("history staged does not report the missed stage:\n%s", out)
	}
	// Under --json the array stays an array, and the note reaches the agent
	// reading it on stderr.
	jsonOut, jsonErr, _ := runHookAllowingFailure("", "history", "staged", "--json")
	if !strings.HasPrefix(strings.TrimSpace(jsonOut), "[") || strings.Contains(jsonOut, "NOTE") {
		t.Errorf("history staged --json stdout is not the plain array:\n%s", jsonOut)
	}
	if !strings.Contains(jsonErr, "1 transcript(s) were read but could not be staged") {
		t.Errorf("history staged --json does not report the missed stage on stderr:\n%s", jsonErr)
	}
}

// notStaged lists the stderr of every hook in a burst that did not report
// staging, so a failing burst names its cause instead of only a count.
func notStaged(stderrs []string) string {
	var b strings.Builder
	for i, s := range stderrs {
		if !strings.Contains(s, "staged sub-agent") {
			b.WriteString("hook " + strconv.Itoa(i) + ": " + strings.TrimSpace(s) + "\n")
		}
	}
	return b.String()
}

// TestHookSessionEndRetriesATransientRootFailure: the session's own transcript
// had the same silent loss as a sub-agent's. One failed root-commit lookup at
// session end no longer gives the session up; the hook asks again and stages.
func TestHookSessionEndRetriesATransientRootFailure(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	tp := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(tp, []byte(`{"role":"user","text":"hello"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	failRootCommitGit(t, 1)

	_, stderr, failed := runHookAllowingFailure(endPayload(t, "sess-end-retry", repo, tp), "hook", "session-end")
	if failed {
		t.Fatalf("the hook exited non-zero: %s", stderr)
	}
	staged, err := history.ListStaged(repo, rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(staged) != 1 {
		t.Fatalf("a transient git failure at session end dropped the session: %d staged\nstderr: %s", len(staged), stderr)
	}
	if !strings.Contains(stderr, "resolved on try") {
		t.Errorf("stderr does not say the repository resolved on a retry: %q", stderr)
	}
}
