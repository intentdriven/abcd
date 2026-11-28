//go:build unix

package matrix

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// pidGone waits up to 10s for pid to be no process at all, reaped.
func pidGone(pid int) bool {
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// childPidFile is where a test command writes its background child's pid;
// the child is killed by that pid when the test ends, should the run under
// test have left it, so a failing run never leaves a process behind.
func childPidFile(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "pid")
	t.Cleanup(func() {
		data, err := os.ReadFile(p)
		if err != nil {
			return
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil || pid <= 1 {
			return
		}
		if comm, err := exec.Command("ps", "-o", "comm=", "-p", strconv.Itoa(pid)).Output(); err == nil && strings.TrimSpace(string(comm)) == "sleep" {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	})
	return p
}

// hiddenChildPid reads the pid a test command wrote to p.
func hiddenChildPid(t *testing.T, p string) int {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 1 {
		t.Fatalf("the command wrote %q for its child's pid", data)
	}
	return pid
}

// A run past its deadline is killed as one process group through the handle
// the run holds: a child the command started in the background, holding its
// output open, is gone with it.
func TestATimedOutTestRunIsKilledAsOneProcessGroup(t *testing.T) {
	hr := newHiddenRun(t, nil)
	pidFile := childPidFile(t)
	hr.in.Command = sh(`sleep 300 & echo $! > "$1"; wait`, pidFile)
	hr.in.Deadline = 5 * time.Second
	out, err := runHiddenTests(context.Background(), hr.in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != StatusUnexamined || out.Score != nil {
		t.Fatalf("outcome %+v; want unexamined", out)
	}
	pid := hiddenChildPid(t, pidFile)
	if !pidGone(pid) {
		t.Fatalf("the background child %d outlived its group's deadline", pid)
	}
}

// A passing run's leader exits, and what it left running in its group is
// killed: no process left in the run's group outlives the run. (A process
// that leaves the group, by setsid, is out of the group's reach.)
func TestWhatAFinishedTestRunLeftInItsGroupIsKilled(t *testing.T) {
	hr := newHiddenRun(t, nil)
	pidFile := childPidFile(t)
	hr.in.Command = sh(`sleep 300 >/dev/null 2>&1 </dev/null & echo $! > "$1"; exit 0`, pidFile)
	out, err := runHiddenTests(context.Background(), hr.in)
	if err != nil || out.Status != StatusPassed {
		t.Fatalf("outcome %+v, %v", out, err)
	}
	pid := hiddenChildPid(t, pidFile)
	if !pidGone(pid) {
		t.Fatalf("the background child %d outlived its finished run", pid)
	}
}

// A leader that exits while a child it left in its group holds its output
// open is judged on its exit, at once: the child is killed with the rest of
// the group and the output drained, never waited on to the deadline.
func TestALeaderThatExitsWhileItsChildHoldsItsOutputIsJudgedAtOnce(t *testing.T) {
	hr := newHiddenRun(t, nil)
	pidFile := childPidFile(t)
	hr.in.Command = sh(`echo before; sleep 300 & echo $! > "$1"; exit 0`, pidFile)
	hr.in.Deadline = 20 * time.Second
	start := time.Now()
	out, err := runHiddenTests(context.Background(), hr.in)
	took := time.Since(start)
	if err != nil || out.Status != StatusPassed || out.Score == nil || out.Score.Passed != 1 || out.Score.Total != 1 {
		t.Fatalf("outcome %+v, %v; want passed 1 of 1", out, err)
	}
	if took > 10*time.Second {
		t.Errorf("returned after %s; the leader exited at once", took)
	}
	if !strings.Contains(out.Output, "before") {
		t.Errorf("the output was not drained: %q", out.Output)
	}
	pid := hiddenChildPid(t, pidFile)
	if !pidGone(pid) {
		t.Fatalf("the background child %d outlived its finished run", pid)
	}
}

// A test that made a file or directory of its export immutable (chflags
// uchg) does not keep the export: its flags are cleared and it is removed.
func TestAnExportATestMadeImmutableIsRemoved(t *testing.T) {
	if _, err := exec.LookPath("chflags"); err != nil || runtime.GOOS == "linux" {
		t.Skip("no chflags here")
	}
	hr := newHiddenRun(t, nil)
	hr.in.Command = sh(`touch stuck && chflags uchg stuck && mkdir held && touch held/f && chflags uchg held/f held`)
	out, err := runHiddenTests(context.Background(), hr.in)
	if err != nil || out.Status != StatusPassed {
		t.Fatalf("outcome %+v, %v; want passed with the export removed", out, err)
	}
	if ents, _ := os.ReadDir(filepath.Join(hr.store, exportsDir, cloneRun)); len(ents) > 0 {
		t.Errorf("the export is left: %v", ents)
	}
}

// A stopped run kills its group too.
func TestAStoppedTestRunIsKilledAsOneProcessGroup(t *testing.T) {
	hr := newHiddenRun(t, nil)
	pidFile := childPidFile(t)
	hr.in.Command = sh(`sleep 300 & echo $! > "$1"; wait`, pidFile)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for i := 0; i < 200; i++ {
			if _, err := os.Stat(pidFile); err == nil {
				break
			}
			time.Sleep(25 * time.Millisecond)
		}
		cancel()
	}()
	if _, err := runHiddenTests(ctx, hr.in); err == nil {
		t.Fatal("a stopped run gave an outcome")
	}
	pid := hiddenChildPid(t, pidFile)
	if !pidGone(pid) {
		t.Fatalf("the background child %d outlived its stopped run", pid)
	}
}

// A copy holding a FIFO is unexamined, and the export never waits on it.
func TestACopyHoldingAFIFOIsUnexaminedAndNeverWaitedOn(t *testing.T) {
	hr := newHiddenRun(t, func(clone string) {
		if err := syscall.Mkfifo(filepath.Join(clone, "pipe"), 0o600); err != nil {
			t.Fatal(err)
		}
	})
	hr.in.Command = sh(`exit 0`)
	hr.in.Deadline = 30 * time.Second
	start := time.Now()
	out, err := runHiddenTests(context.Background(), hr.in)
	if err != nil || out.Status != StatusUnexamined || !strings.Contains(out.Reason, "pipe") {
		t.Fatalf("outcome %+v, %v; want unexamined naming the FIFO", out, err)
	}
	if took := time.Since(start); took > 15*time.Second {
		t.Errorf("the export waited %s on the FIFO", took)
	}
}

// An immutable file under a directory a test closed to everyone (mode 000)
// does not keep the export: the first clearing of flags cannot list that
// directory, so it is cleared again once every directory is opened.
func TestAnImmutableFileUnderAClosedDirectoryIsRemoved(t *testing.T) {
	if _, err := exec.LookPath("chflags"); err != nil || runtime.GOOS == "linux" {
		t.Skip("no chflags here")
	}
	hr := newHiddenRun(t, nil)
	hr.in.Command = sh(`mkdir -p deep/inner && touch deep/inner/f && chflags uchg deep/inner/f && chmod 000 deep`)
	out, err := runHiddenTests(context.Background(), hr.in)
	if err != nil || out.Status != StatusPassed {
		t.Fatalf("outcome %+v, %v; want passed with the export removed", out, err)
	}
	if ents, _ := os.ReadDir(filepath.Join(hr.store, exportsDir, cloneRun)); len(ents) > 0 {
		t.Errorf("the export is left: %v", ents)
	}
}
