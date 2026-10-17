//go:build unix

package loop

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// waitPid waits for the fake harness to log the pid in file, under e's log.
func waitPid(t *testing.T, e driveEnv, file string) int {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if raw, err := os.ReadFile(filepath.Join(e.log, file)); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil {
				t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
				return pid
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("the fake harness logged no %s", file)
	return 0
}

// gone reports whether pid is no longer running within wait: a process the
// group kill reached and its reaper has collected.
func gone(pid int, wait time.Duration) bool {
	deadline := time.Now().Add(wait)
	for {
		if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestAnInterruptedDrainStopsItsJudge: the drain's context ending while its
// routed judge runs (an interrupt or a termination of `abcd drain`) kills the
// runner's process group, its own child included, before Drain returns; the
// move says it was stopped, records no fallback for the stop, and keeps the
// request it awaits, and the next move asks the judge again.
func TestAnInterruptedDrainStopsItsJudge(t *testing.T) {
	repo := issueRepo(t)
	if first, err := Drain(repo.Root(), Options{}, DrainOptions{}); err != nil || first.Judging == nil {
		t.Fatalf("a drain with its judge on the host awaits the host's answer: %+v %v", first.Judging, err)
	}
	e := judgeEnv(t, "hang")
	routed := Options{Runners: runnerConfig(t, judgeRoutedConfig, "")}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := Drain(repo.Root(), routed, DrainOptions{Transcripts: &memTranscripts{}, JudgeTimeout: 40 * time.Second, Context: ctx})
		done <- err
	}()
	child := waitPid(t, e, "claude.child.pid")
	leader := waitPid(t, e, "claude.pid")
	cancel()
	var err error
	select {
	case err = <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("the drain did not return once its context ended")
	}
	if !gone(leader, 0) {
		t.Fatalf("the judge's runner %d outlived the drain", leader)
	}
	if !gone(child, 5*time.Second) {
		t.Fatalf("the judge's child %d outlived the drain", child)
	}
	r, ok := AsRefusal(err)
	if !ok || !strings.Contains(r.Reason, "stopped") {
		t.Fatalf("the move says the drain was stopped while its judge ran: %v", err)
	}
	st, live, rerr := readDrain(repo.Root())
	if rerr != nil || !live || st.Judging == nil || st.Judging.Issue != eligibleIssue {
		t.Fatalf("the request is kept for the next move: %+v %v %v", st.Judging, live, rerr)
	}
	if len(st.JudgeFallbacks) != 0 {
		t.Fatalf("a stop is no fallback: %+v", st.JudgeFallbacks)
	}

	t.Setenv(loopFakeEnv, "judge-no")
	next, err := Drain(repo.Root(), routed, DrainOptions{Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	if next.Judged == nil || next.Judged.Answer != JudgementNo || next.Start == nil {
		t.Fatalf("the next move asks the judge again, and its no opens the lane: %+v", next)
	}
}
