package fsutil

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// The re-exec helper's environment: which primitive the child takes, on what
// path, and whether it leaves a grandchild running while it holds the lock.
const (
	lockHelperEnv  = "ABCD_FSUTIL_LOCK_HELPER"
	lockPathEnv    = "ABCD_FSUTIL_LOCK_PATH"
	lockSpawnEnv   = "ABCD_FSUTIL_LOCK_SPAWN"
	lockHelperDir  = "dir"
	lockHelperFile = "file"
)

// TestLockHelperProcess is not a test: it is the child process the lock tests
// re-exec. It takes the named lock, prints "held" once it has it, and holds it
// until its stdin closes. With lockSpawnEnv set it first starts a grandchild
// that outlives it, the shape of a lock holder that shells out to git.
func TestLockHelperProcess(t *testing.T) {
	kind := os.Getenv(lockHelperEnv)
	if kind == "" {
		t.Skip("the lock tests' child process; runs only when re-executed")
	}
	path := os.Getenv(lockPathEnv)
	hold := func() error {
		if os.Getenv(lockSpawnEnv) != "" {
			if err := exec.Command("sleep", "60").Start(); err != nil {
				return err
			}
		}
		fmt.Println("held")
		_, _ = io.Copy(io.Discard, os.Stdin)
		return nil
	}
	var err error
	switch kind {
	case lockHelperDir:
		err = WithDirLock(path, 5*time.Second, hold)
	case lockHelperFile:
		err = WithFileLock(path, 5*time.Second, hold)
	}
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// lockHolder is a child process holding a lock, in a process group of its own
// so the test can reap anything it left behind by that group and nothing else.
type lockHolder struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
}

// startLockHolder re-execs the test binary as a lock holder and returns once
// the child reports the lock held.
func startLockHolder(t *testing.T, kind, path string, spawn bool) *lockHolder {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestLockHelperProcess$", "-test.count=1")
	cmd.Env = append(os.Environ(), lockHelperEnv+"="+kind, lockPathEnv+"="+path)
	if spawn {
		cmd.Env = append(cmd.Env, lockSpawnEnv+"=1")
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	h := &lockHolder{cmd: cmd, stdin: stdin}
	pgid := cmd.Process.Pid
	t.Cleanup(func() {
		// The group this test created, and only it: the child if it still
		// runs, and any grandchild it left.
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		_ = stdin.Close()
		_ = cmd.Wait()
	})
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "held" {
		t.Fatalf("the lock holder did not report the lock held: %q, %v", line, err)
	}
	return h
}

// dirOrFileLock takes the lock kind names on path with timeout.
func dirOrFileLock(kind, path string, timeout time.Duration, fn func() error) error {
	if kind == lockHelperDir {
		return WithDirLock(path, timeout, fn)
	}
	return WithFileLock(path, timeout, fn)
}

// lockTarget is a fresh path for the lock kind names: a directory that exists
// for a directory lock, a file path (created by the lock) for a file lock.
func lockTarget(t *testing.T, kind string) string {
	t.Helper()
	if kind == lockHelperDir {
		return t.TempDir()
	}
	return filepath.Join(t.TempDir(), "k.lock")
}

// A lock one process holds excludes another process, for its whole budget, and
// the refusal is ErrLockContention naming the caller's budget; once the holder
// lets go, the same lock is granted. Two processes, not two goroutines: flock is
// held per open file description, and a same-process test cannot tell that
// apart from a lock that excludes nothing across processes.
func TestTheLockExcludesAnotherProcess(t *testing.T) {
	for _, kind := range []string{lockHelperDir, lockHelperFile} {
		t.Run(kind, func(t *testing.T) {
			path := lockTarget(t, kind)
			h := startLockHolder(t, kind, path, false)

			const timeout = 300 * time.Millisecond
			ran := false
			start := time.Now()
			err := dirOrFileLock(kind, path, timeout, func() error { ran = true; return nil })
			waited := time.Since(start)
			if !errors.Is(err, ErrLockContention) || ran {
				t.Fatalf("a lock another process holds: ran=%v err=%v; want ErrLockContention with fn not run", ran, err)
			}
			if !strings.Contains(err.Error(), "within "+timeout.String()) {
				t.Errorf("the contention error %q does not name the caller's %s budget", err, timeout)
			}
			if waited < timeout {
				t.Errorf("gave up after %s, inside its %s budget", waited, timeout)
			}

			_ = h.stdin.Close()
			if err := h.cmd.Wait(); err != nil {
				t.Fatalf("the holder: %v", err)
			}
			if err := dirOrFileLock(kind, path, timeout, func() error { ran = true; return nil }); err != nil || !ran {
				t.Fatalf("the lock after its holder let go: ran=%v err=%v", ran, err)
			}
		})
	}
}

// A holder that dies without letting go frees the lock: the kernel drops a
// flock when the last descriptor on it closes, so nothing it leaves behind — a
// lock file, a directory — wedges the next taker. That includes a grandchild the
// holder started and left running: a lock descriptor it inherited across exec
// would hold the lock for as long as the grandchild lives, so the descriptor is
// close-on-exec.
func TestTheLockIsReleasedWhenItsHolderDies(t *testing.T) {
	for _, kind := range []string{lockHelperDir, lockHelperFile} {
		for _, spawn := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/grandchild=%v", kind, spawn), func(t *testing.T) {
				path := lockTarget(t, kind)
				h := startLockHolder(t, kind, path, spawn)
				if err := h.cmd.Process.Kill(); err != nil {
					t.Fatal(err)
				}
				_ = h.cmd.Wait()
				ran := false
				if err := dirOrFileLock(kind, path, 2*time.Second, func() error { ran = true; return nil }); err != nil || !ran {
					t.Fatalf("the lock after its holder died (a grandchild it started still running: %v): ran=%v err=%v", spawn, ran, err)
				}
			})
		}
	}
}

// The lock-file primitive judges the file type on the opened descriptor under
// the S_IFMT mask, so a FIFO at the lock path — which the open itself admits —
// is refused as ErrLockPathUnsafe. Memory's store lock relied on a mask of its
// own for this until it moved onto WithFileLock (iss-2608261133210491, iss-129).
func TestWithFileLockRefusesANonRegularFileOnTheDescriptor(t *testing.T) {
	lock := filepath.Join(t.TempDir(), "k.lock")
	if err := syscall.Mkfifo(lock, 0o600); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}
	ran := false
	err := WithFileLock(lock, 0, func() error { ran = true; return nil })
	if !errors.Is(err, ErrLockPathUnsafe) || ran {
		t.Fatalf("a FIFO at the lock path: ran=%v err=%v; want ErrLockPathUnsafe", ran, err)
	}
}
