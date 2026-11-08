package matrix

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Local cells are served by one inference server at a time (decision 12,
// spc-2610161949559050 step 3), started as one owned process group and
// stopped as one when its cells end.

// serverArgv is a stand-in inference server that starts a child of its own
// and writes the child's pid, so the test can see the whole group go.
func serverArgv(t *testing.T) ([]string, string) {
	t.Helper()
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	return []string{"/bin/sh", "-c", `sleep 300 & echo $! > "$1"; wait`, "server", pidFile}, pidFile
}

func childPid(t *testing.T, pidFile string) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(pidFile); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && pid > 0 {
				return pid
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("the stand-in server never started its child")
	return 0
}

func gone(pid int) bool {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

// TestOneLocalServerAtATimeAsAnOwnedProcessGroup: a second cell's server is
// refused while the first's runs; the first is its own process group, and
// stopping it stops the group, the child it started included; the next cell's
// server then starts.
func TestOneLocalServerAtATimeAsAnOwnedProcessGroup(t *testing.T) {
	var s localServer
	t.Cleanup(func() { _ = s.stop() })
	argv, pidFile := serverArgv(t)
	if err := s.start("local-a", argv); err != nil {
		t.Fatal(err)
	}
	child := childPid(t, pidFile)
	lead := s.cmd.Process.Pid
	if pgid, err := syscall.Getpgid(lead); err != nil || pgid != lead {
		t.Fatalf("the server leads its own process group: pgid %d, pid %d, %v", pgid, lead, err)
	}
	if pgid, err := syscall.Getpgid(child); err != nil || pgid != lead {
		t.Fatalf("the server's child is in its group: pgid %d, %v", pgid, err)
	}
	if s.serving() != "local-a" {
		t.Fatalf("the server names the cell it serves: %q", s.serving())
	}

	other, _ := serverArgv(t)
	if err := s.start("local-b", other); err == nil || !strings.Contains(err.Error(), "local-a") {
		t.Fatalf("a second cell's server is refused, naming the one running: %v", err)
	}
	if err := s.start("local-a", argv); err != nil {
		t.Fatalf("the running cell's server is the one it already has: %v", err)
	}

	if err := s.stop(); err != nil {
		t.Fatal(err)
	}
	if !gone(lead) || !gone(child) {
		t.Fatal("stopping the server stops its whole group")
	}
	if s.serving() != "" {
		t.Fatalf("a stopped server serves nothing: %q", s.serving())
	}
	next, nextPid := serverArgv(t)
	if err := s.start("local-b", next); err != nil {
		t.Fatalf("the next cell's server starts once the first is stopped: %v", err)
	}
	childPid(t, nextPid)
}

// TestAServerThatEndsByItselfLeavesNothing: a server that exits on its own is
// noticed, what it left of its group is stopped, and the slot is free.
func TestAServerThatEndsByItselfLeavesNothing(t *testing.T) {
	var s localServer
	t.Cleanup(func() { _ = s.stop() })
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	if err := s.start("local-a", []string{"/bin/sh", "-c", `sleep 300 & echo $! > "$1"; exit 0`, "server", pidFile}); err != nil {
		t.Fatal(err)
	}
	child := childPid(t, pidFile)
	deadline := time.Now().Add(5 * time.Second)
	for s.serving() != "" && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if s.serving() != "" {
		t.Fatal("a server that ended is not noticed")
	}
	if !gone(child) {
		t.Fatal("what an ended server left of its group is stopped")
	}
}

// TestALocalServerNeedsACommand: an empty command starts nothing.
func TestALocalServerNeedsACommand(t *testing.T) {
	var s localServer
	if err := s.start("local-a", nil); err == nil {
		t.Fatal("an empty command started a server")
	}
}
