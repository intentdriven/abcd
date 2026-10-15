package landing

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// m7: a preflight past its bound is stopped with every process it started,
// its whole process group, not make alone.
func TestAPreflightPastItsBoundIsKilledWithItsGroup(t *testing.T) {
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make is not installed")
	}
	dir := t.TempDir()
	mk := "preflight:\n\t@sleep 20 & echo $$! > child.pid; wait\n"
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(mk), 0o644); err != nil {
		t.Fatal(err)
	}
	prev := gateTimeout
	gateTimeout = 2 * time.Second
	t.Cleanup(func() { gateTimeout = prev })

	var out bytes.Buffer
	start := time.Now()
	err := makePreflight(&out)(dir)
	if err == nil {
		t.Fatal("a preflight past its bound passed")
	}
	if took := time.Since(start); took > 10*time.Second {
		t.Fatalf("the gate returned after %s: the bound did not stop what the preflight started", took)
	}
	raw, rerr := os.ReadFile(filepath.Join(dir, "child.pid"))
	if rerr != nil {
		t.Fatal(rerr)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
	if pid <= 0 {
		t.Fatalf("child.pid %q", raw)
	}
	deadline := time.Now().Add(3 * time.Second)
	for syscall.Kill(pid, 0) == nil {
		if time.Now().After(deadline) {
			_ = syscall.Kill(pid, syscall.SIGKILL)
			t.Fatalf("the preflight's child %d outlived the bound", pid)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
