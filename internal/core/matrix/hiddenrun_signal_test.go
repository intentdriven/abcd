//go:build unix

package matrix

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// A hidden-test run never outlives abcd's stop: a signal abcd catches kills the
// run's group whole before abcd returns, and a stop it cannot catch, or a run
// past its deadline that abcd no longer watches, is ended by the group's own
// supervisor, with no word from abcd.

// waitForFile waits up to 20s for p to be written.
func waitForFile(t *testing.T, p string) {
	t.Helper()
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(25 * time.Millisecond) {
		if data, err := os.ReadFile(p); err == nil && len(strings.TrimSpace(string(data))) > 0 {
			return
		}
	}
	t.Fatalf("nothing was written at %s", p)
}

// groupGone waits up to 10s for the process group pgid to hold no process.
func groupGone(pgid int) bool {
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		if err := syscall.Kill(-pgid, 0); errors.Is(err, syscall.ESRCH) {
			return true
		}
	}
	return false
}

// groupOf is the process group of the test command's background child, read
// while it lives.
func groupOf(t *testing.T, pidFile string) (int, int) {
	t.Helper()
	pid := hiddenChildPid(t, pidFile)
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		t.Fatalf("the background child %d has no group: %v", pid, err)
	}
	if pgid == syscall.Getpgrp() {
		t.Fatalf("the test command runs in the test's own process group %d", pgid)
	}
	return pid, pgid
}

// A termination abcd receives while the hidden tests run is a stop: the run's
// error, never the copy's score, with the whole group killed before the run
// returns.
func TestASignalledStopKillsTheTestGroupBeforeTheRunReturns(t *testing.T) {
	hr := newHiddenRun(t, nil)
	pidFile := childPidFile(t)
	hr.in.Command = sh(`sleep 300 & echo $! > "$1"; wait`, pidFile)
	hr.in.Deadline = 10 * time.Minute
	pgid := make(chan int, 1)
	go func() {
		// No t here: this goroutine only finds the group and sends the stop.
		defer close(pgid)
		for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(25 * time.Millisecond) {
			data, err := os.ReadFile(pidFile)
			if err != nil {
				continue
			}
			pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
			if err != nil || pid <= 1 {
				continue
			}
			if g, err := syscall.Getpgid(pid); err == nil && g != syscall.Getpgrp() {
				pgid <- g
				_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
			}
			return
		}
	}()
	out, err := runHiddenTests(context.Background(), hr.in)
	if err == nil || !strings.Contains(err.Error(), "stopped") || out.Score != nil {
		t.Fatalf("a run abcd was told to stop gave %+v, %v; want the run's error", out, err)
	}
	g, ok := <-pgid
	if !ok {
		t.Fatal("the test command's group was never found, so no stop was sent")
	}
	if !groupGone(g) {
		t.Fatalf("the test group %d outlived abcd's stop", g)
	}
}

// TestHiddenRunHelperProcess is the abcd that TestAnUncaughtStopLeavesNoTestGroup
// kills outright: it runs a copy's hidden tests and is killed while they run.
func TestHiddenRunHelperProcess(t *testing.T) {
	pidFile := os.Getenv("ABCD_HIDDEN_RUN_HELPER")
	if pidFile == "" {
		t.Skip("run only as TestAnUncaughtStopLeavesNoTestGroup's abcd")
	}
	hr := newHiddenRun(t, nil)
	hr.in.Command = sh(`sleep 300 & echo $! > "$1"; wait`, pidFile)
	hr.in.Deadline = 10 * time.Minute
	_, _ = runHiddenTests(context.Background(), hr.in)
}

// An abcd killed outright (SIGKILL, which no process catches) leaves no test
// group behind: the group's supervisor reads abcd gone and ends it whole.
func TestAnUncaughtStopLeavesNoTestGroup(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no /bin/sh")
	}
	pidFile := childPidFile(t)
	helper := exec.Command(os.Args[0], "-test.run=^TestHiddenRunHelperProcess$", "-test.count=1")
	helper.Env = append(os.Environ(), "ABCD_HIDDEN_RUN_HELPER="+pidFile, "TMPDIR="+t.TempDir())
	helper.Stdout, helper.Stderr = os.Stderr, os.Stderr
	if err := helper.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = helper.Process.Kill(); _ = helper.Wait() })
	waitForFile(t, pidFile)
	pid, pgid := groupOf(t, pidFile)
	if err := helper.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = helper.Wait()
	if !pidGone(pid) || !groupGone(pgid) {
		t.Fatalf("the test group %d outlived its abcd, killed outright", pgid)
	}
}

// Past its backstop the supervisor ends its group by itself, while abcd lives
// and does nothing: the wall-time bound holds with no kill from abcd.
func TestTheBackstopEndsTheGroupPastItsDeadlineWithoutAbcd(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no /bin/sh")
	}
	pidFile := filepath.Join(t.TempDir(), "pid")
	lr, alive, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer alive.Close()
	args, err := superviseArgs("/bin/sh", []string{"-c", `sleep 300 & echo $! > "$1"; wait`, "sh", pidFile}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(superviseShell, args...)
	cmd.ExtraFiles = []*os.File{lr}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	_ = lr.Close()
	t.Cleanup(func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) })
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("the supervisor did not end its group at its backstop")
	}
	waitForFile(t, pidFile)
	pid := hiddenChildPid(t, pidFile)
	if !pidGone(pid) || !groupGone(cmd.Process.Pid) {
		t.Fatalf("the group %d outlived its backstop", cmd.Process.Pid)
	}
}

// A test command named by a path relative to the copy, ./gradlew's shape, runs
// the copy's own program in the export, never one beside abcd.
func TestARelativeTestCommandRunsTheCopysOwnProgram(t *testing.T) {
	hr := newHiddenRun(t, func(clone string) {
		if err := os.WriteFile(filepath.Join(clone, "check.sh"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	})
	hr.in.Command = []string{"./check.sh"}
	out, err := runHiddenTests(context.Background(), hr.in)
	if err != nil || out.Status != StatusPassed {
		t.Fatalf("the copy's ./check.sh gave %+v, %v; want it passed", out, err)
	}
}

// The test command keeps the default action of an interrupt and a quit: the
// supervisor never runs it as a job of its own, which a shell without job
// control starts with both ignored. A command that interrupts itself ends.
func TestATestCommandKeepsTheDefaultInterrupt(t *testing.T) {
	for _, sig := range []string{"INT", "QUIT"} {
		hr := newHiddenRun(t, nil)
		hr.in.Command = sh(`kill -` + sig + ` $$; exit 0`)
		out, err := runHiddenTests(context.Background(), hr.in)
		if err != nil {
			t.Fatal(err)
		}
		if out.Status != StatusFailed || out.Score == nil || out.Score.Passed != 0 {
			t.Errorf("a command sending itself SIG%s gave %+v; want it ended, so failed", sig, out)
		}
	}
}

// A status of 128 and a stop signal's number is the copy's code's own choice
// of status, its failure: the run returns at once, never stalled to its
// deadline, and the kept output carries nothing of the supervisor's.
func TestAStopSignalStatusIsTheCopysFailureNotAStall(t *testing.T) {
	for _, code := range []string{"145", "146", "147", "149", "150", "200"} {
		hr := newHiddenRun(t, nil)
		hr.in.Command = sh(`exit ` + code)
		hr.in.Deadline = 5 * time.Second
		start := time.Now()
		out, err := runHiddenTests(context.Background(), hr.in)
		if err != nil {
			t.Fatal(err)
		}
		if took := time.Since(start); took > 4*time.Second {
			t.Errorf("exit %s returned after %s, stalled toward its deadline", code, took)
		}
		if out.Status != StatusFailed || !strings.Contains(out.Reason, "exited "+code) || strings.Contains(out.Output, "kill") {
			t.Errorf("exit %s gave %+v; want failed, naming the status, with nothing of the supervisor's output", code, out)
		}
	}
}

// A group ended before abcd saw its test command end, by its own watcher or
// by a process in it, while abcd lives and before abcd's deadline, is a run
// abcd did not see through: unexamined, never failed.
func TestAGroupEndedBeforeItsCommandsEndIsUnexamined(t *testing.T) {
	hr := newHiddenRun(t, nil)
	hr.in.Command = sh(`kill -9 0`)
	out, err := runHiddenTests(context.Background(), hr.in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != StatusUnexamined || out.Score != nil {
		t.Fatalf("a group killed whole from inside gave %+v; want unexamined with no score", out)
	}
}

// The backstop's timer is the system's own sleep, named by its path: a sleep
// earlier on the PATH the run carries is never the one it starts.
func TestTheBackstopsSleepIsNeverFoundThroughThePath(t *testing.T) {
	bin := t.TempDir()
	marker := filepath.Join(t.TempDir(), "ran")
	fake := "#!/bin/sh\necho ran > '" + marker + "'\nexec /bin/sleep \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "sleep"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	hr := newHiddenRun(t, nil)
	hr.in.Command = []string{"/bin/sleep", "1"}
	out, err := runHiddenTests(context.Background(), hr.in)
	if err != nil || out.Status != StatusPassed {
		t.Fatalf("the run gave %+v, %v; want it passed", out, err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the backstop started the sleep it found on the PATH")
	}
}

// openFiles counts this process's open descriptors.
func openFiles(t *testing.T) int {
	t.Helper()
	ents, err := os.ReadDir("/dev/fd")
	if err != nil {
		t.Skipf("no /dev/fd: %v", err)
	}
	return len(ents)
}

// A system with no sleep at a path the backstop knows refuses the run, which
// names the missing sleep and not the test command, and leaves no descriptor
// of the run's pipes open.
func TestARunWithNoBackstopSleepNamesItAndLeavesNoPipeOpen(t *testing.T) {
	hr := newHiddenRun(t, nil)
	hr.in.Command = sh(`exit 0`)
	saved := backstopSleeps
	backstopSleeps = []string{filepath.Join(t.TempDir(), "no-sleep")}
	t.Cleanup(func() { backstopSleeps = saved })
	// A first refused run settles what the process opens once, on first use
	// (its poller among them); the second is the one counted.
	_, _ = runHiddenTests(context.Background(), hr.in)
	before := openFiles(t)
	out, err := runHiddenTests(context.Background(), hr.in)
	after := openFiles(t)
	if err == nil || out.Score != nil {
		t.Fatalf("a run with no backstop gave %+v, %v; want the run's error", out, err)
	}
	if !strings.Contains(err.Error(), "sleep") || strings.Contains(err.Error(), `test command "sh"`) {
		t.Errorf("the refusal names the test command, not the missing sleep: %v", err)
	}
	if after != before {
		t.Errorf("the refused run left %d descriptors open", after-before)
	}
}
