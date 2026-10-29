//go:build unix

package gitutil

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

// TestAStoppedCommitLeavesNoChildGitStartsAfterTheSnapshot is the reproduced
// half of iss-2610292307181518: the stop reads git's descendants, and git
// went on running until the signals reached it, so a hook git started after
// the read escaped the stop, held git's output for the whole WaitDelay and
// was left running. Here the stop is held open after the read while the
// pre-commit hook ends and git would start prepare-commit-msg, whose program
// waits far past any deadline.
func TestAStoppedCommitLeavesNoChildGitStartsAfterTheSnapshot(t *testing.T) {
	dir, pidFile := lateChildRepo(t)
	var held time.Time
	afterSnapshot = func() {
		time.Sleep(2 * time.Second)
		held = time.Now()
	}
	t.Cleanup(func() { afterSnapshot = nil })
	stalled := errors.New("did not finish")

	_, err := RunCommitCommandBounded(dir, 500*time.Millisecond, stalled, "commit", "-q", "-m", "late", "--", "a.md")
	took := time.Since(held)
	if !errors.Is(err, stalled) {
		t.Fatalf("the stopped commit was not refused as stalled: %v", err)
	}
	time.Sleep(300 * time.Millisecond)
	if pid, ok := lateChildAlive(t, pidFile); ok {
		t.Errorf("prepare-commit-msg's program (pid %d) is still running after the stop", pid)
	}
	if took >= stopGrace {
		t.Errorf("the commit came back %s after the stop was let go: a late child held git's output for the WaitDelay", took)
	}
}

// lateChildRepo is a repository with a change to a.md ready to commit, whose
// pre-commit hook takes a second and whose prepare-commit-msg hook records
// its pid in the file returned and then waits a minute.
func lateChildRepo(t *testing.T) (dir, pidFile string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	dir = t.TempDir()
	git := func(args ...string) {
		t.Helper()
		c := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	git("init", "-q", "--initial-branch=main")
	git("config", "user.name", "Fixture")
	git("config", "user.email", "fixture@example.invalid")
	git("config", "commit.gpgsign", "false")
	write := func(rel, content string, mode os.FileMode) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), mode); err != nil {
			t.Fatal(err)
		}
	}
	write("a.md", "a\n", 0o644)
	git("add", "a.md")
	git("commit", "-q", "-m", "seed")
	write("a.md", "b\n", 0o644)
	pidFile = filepath.Join(t.TempDir(), "late.pid")
	write(".git/hooks/pre-commit", "#!/bin/sh\nsleep 1\n", 0o755)
	write(".git/hooks/prepare-commit-msg", "#!/bin/sh\necho $$ > '"+pidFile+"'\nexec sleep 60\n", 0o755)
	t.Cleanup(func() {
		if pid, ok := lateChildAlive(t, pidFile); ok {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	})
	return dir, pidFile
}

// lateChildAlive is the pid prepare-commit-msg recorded, and whether that
// process is still the hook's sleep.
func lateChildAlive(t *testing.T, pidFile string) (int, bool) {
	t.Helper()
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 1 {
		return 0, false
	}
	out, err := exec.Command("/bin/ps", "-o", "command=", "-p", strconv.Itoa(pid)).Output()
	return pid, err == nil && strings.Contains(string(out), "sleep 60")
}

// TestAStopThatCannotReadTheTableStillContinuesGit: with no process table,
// git alone is stopped, and it is continued after the SIGTERM, so it acts on
// it and removes index.lock, which a git left frozen until the WaitDelay's
// SIGKILL would leave behind.
func TestAStopThatCannotReadTheTableStillContinuesGit(t *testing.T) {
	dir, _ := lateChildRepo(t)
	hookPid := filepath.Join(t.TempDir(), "pre-commit.pid")
	if err := os.WriteFile(filepath.Join(dir, ".git", "hooks", "pre-commit"), []byte("#!/bin/sh\necho $$ > '"+hookPid+"'\nexec sleep 60\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if pid, ok := lateChildAlive(t, hookPid); ok {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	})
	was := psPaths
	psPaths = []string{filepath.Join(t.TempDir(), "no-ps")}
	t.Cleanup(func() { psPaths = was })
	stalled := errors.New("did not finish")

	_, err := RunCommitCommandBounded(dir, 500*time.Millisecond, stalled, "commit", "-q", "-m", "x", "--", "a.md")
	if !errors.Is(err, stalled) {
		t.Fatalf("the stopped commit was not refused as stalled: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(dir, ".git", "index.lock")); err == nil {
		t.Fatal("index.lock was left behind: git was not continued after the SIGTERM")
	}
}

// TestAStopFreezesGitItself: with no process table nothing below git is
// stopped, so pre-commit runs on and ends while the stop is held open, and
// only git's own SIGSTOP keeps it from starting prepare-commit-msg, the
// window a deadline that lands between two hooks leaves.
func TestAStopFreezesGitItself(t *testing.T) {
	dir, pidFile := lateChildRepo(t)
	began := filepath.Join(t.TempDir(), "pre-commit.began")
	if err := os.WriteFile(filepath.Join(dir, ".git", "hooks", "pre-commit"), []byte("#!/bin/sh\n: > '"+began+"'\nsleep 2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	was := psPaths
	psPaths = []string{filepath.Join(t.TempDir(), "no-ps")}
	t.Cleanup(func() { psPaths = was })
	var inHook bool
	afterSnapshot = func() {
		_, err := os.Stat(began)
		inHook = err == nil
		time.Sleep(3 * time.Second)
	}
	t.Cleanup(func() { afterSnapshot = nil })
	stalled := errors.New("did not finish")

	_, err := RunCommitCommandBounded(dir, 500*time.Millisecond, stalled, "commit", "-q", "-m", "late", "--", "a.md")
	if !errors.Is(err, stalled) {
		t.Fatalf("the stopped commit was not refused as stalled: %v", err)
	}
	if !inHook {
		t.Skip("git had not started pre-commit when the stop began, so the stop was not held between two hooks")
	}
	if _, err := os.Stat(pidFile); err == nil {
		t.Fatal("git started prepare-commit-msg while the stop was under way")
	}
}

// TestTheSweepStopsASurvivorBeforeKillingIt: a process that outlives the
// stop's SIGTERM and starts a child between the sweep's read and its SIGKILL
// is stopped first, and the child, found by a read made once it is stopped,
// is killed with it rather than left running.
func TestTheSweepStopsASurvivorBeforeKillingIt(t *testing.T) {
	tmp := t.TempDir()
	survivorPid, childPid := filepath.Join(tmp, "survivor.pid"), filepath.Join(tmp, "child.pid")
	script := "trap '' TERM; trap 'sleep 60 & echo $! > \"" + childPid + "\"' USR1; echo $$ > '" + survivorPid + "'; while :; do sleep 0.1; done"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", "/bin/sh -c '"+strings.ReplaceAll(script, "'", `'\''`)+"' & wait")
	cmd.WaitDelay = time.Second
	sweep := stopWithDescendants(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, f := range []string{childPid, survivorPid} {
			if pid, ok := readPid(f); ok {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
		}
	})
	survivor := waitPid(t, survivorPid)
	beforeSweepKill = func([]procEntry) {
		_ = syscall.Kill(survivor, syscall.SIGUSR1)
		waitPid(t, childPid)
	}
	t.Cleanup(func() { beforeSweepKill = nil })

	cancel()
	_ = cmd.Wait()
	sweep()
	child, _ := readPid(childPid)
	time.Sleep(300 * time.Millisecond)
	if state := procState(t, child); state != "" && !strings.HasPrefix(state, "Z") {
		t.Errorf("the child the survivor started before the SIGKILL (pid %d) is still running, in state %q", child, state)
	}
}

// waitPid is the pid written to path, waiting up to five seconds for it.
func waitPid(t *testing.T, path string) int {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		if pid, ok := readPid(path); ok {
			return pid
		}
		if time.Now().After(deadline) {
			t.Fatalf("no pid was written to %s", path)
		}
	}
}

// TestADescendantThatIgnoresTheStopIsNotLeftFrozen: a process that ignores
// SIGTERM goes on running once the stop has signalled it, continued rather
// than left stopped, for the sweep to kill.
func TestADescendantThatIgnoresTheStopIsNotLeftFrozen(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", "trap '' TERM; sleep 60 & echo $! > '"+pidFile+"'; wait")
	_ = stopWithDescendants(cmd)
	stopped := make(chan struct{})
	cancelStop := cmd.Cancel
	cmd.Cancel = func() error {
		defer close(stopped)
		return cancelStop()
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if child, ok := readPid(pidFile); ok {
			_ = syscall.Kill(child, syscall.SIGKILL)
		}
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	var child int
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		if pid, ok := readPid(pidFile); ok {
			child = pid
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the child never started")
		}
	}
	cancel()
	<-stopped
	for _, pid := range []int{cmd.Process.Pid, child} {
		if state := procState(t, pid); strings.HasPrefix(state, "T") || state == "" {
			t.Errorf("pid %d is in state %q after the stop; want it running", pid, state)
		}
	}
}

// TestAStopOfAGitThatHasAlreadyExitedIsHarmless: a stop that races git's own
// exit finds git reaped, and refuses as done, or exited and not yet reaped,
// where its signals do nothing and git's exit is still read.
func TestAStopOfAGitThatHasAlreadyExitedIsHarmless(t *testing.T) {
	reaped := exec.CommandContext(context.Background(), "/bin/sh", "-c", "exit 0")
	_ = stopWithDescendants(reaped)
	if err := reaped.Run(); err != nil {
		t.Fatal(err)
	}
	if err := reaped.Cancel(); !errors.Is(err, os.ErrProcessDone) {
		t.Errorf("the stop of a reaped git returned %v, want os.ErrProcessDone", err)
	}

	zombie := exec.CommandContext(context.Background(), "/bin/sh", "-c", "exit 0")
	sweep := stopWithDescendants(zombie)
	if err := zombie.Start(); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(5 * time.Second); !strings.HasPrefix(procState(t, zombie.Process.Pid), "Z"); time.Sleep(20 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("the process never exited")
		}
	}
	if err := zombie.Cancel(); err != nil {
		t.Errorf("the stop of an exited, unreaped git returned %v", err)
	}
	if err := zombie.Wait(); err != nil {
		t.Errorf("the exit was not read after the stop: %v", err)
	}
	sweep()
}

func readPid(path string) (int, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	return pid, err == nil && pid > 1
}

// procState is ps's state column for pid, "" when ps does not list it.
func procState(t *testing.T, pid int) string {
	t.Helper()
	out, _ := exec.Command("/bin/ps", "-o", "stat=", "-p", strconv.Itoa(pid)).Output()
	return strings.TrimSpace(string(out))
}
