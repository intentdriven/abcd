package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// The re-group fixture runs this test binary itself as the "installer", so no
// package manager and no shell is involved: regroupEnv selects the role, and
// regroupPIDEnv names the file the holder's pid is written to, so the test
// stops the holder through that pid and never by pattern.
const (
	regroupEnv    = "ABCD_TOOLS_TEST_REGROUP"
	regroupPIDEnv = "ABCD_TOOLS_TEST_REGROUP_PIDFILE"
	holderLife    = 60 * time.Second
)

func TestMain(m *testing.M) {
	switch role := os.Getenv(regroupEnv); role {
	case "parent-hang", "parent-exit":
		os.Exit(regroupParent(role))
	case "holder":
		// Holds the inherited stdout/stderr pipe open, in a process group of
		// its own, for far longer than any bound under test.
		time.Sleep(holderLife)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// regroupParent is the step: it starts a holder that leaves the step's process
// group (Setpgid) while keeping the step's output pipe, records the holder's
// pid, and then either hangs until killed or exits cleanly.
func regroupParent(role string) int {
	self, err := os.Executable()
	if err != nil {
		return 2
	}
	holder := exec.Command(self)
	holder.Env = append(os.Environ(), regroupEnv+"=holder")
	holder.Stdout = os.Stdout
	holder.Stderr = os.Stderr
	holder.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := holder.Start(); err != nil {
		return 2
	}
	if err := os.WriteFile(os.Getenv(regroupPIDEnv), []byte(strconv.Itoa(holder.Process.Pid)), 0o600); err != nil {
		return 2
	}
	if role == "parent-hang" {
		time.Sleep(holderLife)
	}
	return 0
}

// TestRunArgvBoundHoldsAgainstAReGroupedDescendant is iss-2609261604485703: a
// descendant that re-groups out of the step's process group survives the group
// kill and holds the output pipe, and runArgv still returns within its bound,
// both when the step is killed on timeout and when it exits cleanly first.
func TestRunArgvBoundHoldsAgainstAReGroupedDescendant(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		role    string
		timeout time.Duration
		want    string
	}{
		{"parent-hang", 500 * time.Millisecond, "did not finish"},
		{"parent-exit", 30 * time.Second, "left its process group"},
	} {
		t.Run(tc.role, func(t *testing.T) {
			pidfile := filepath.Join(t.TempDir(), "holder.pid")
			t.Setenv(regroupEnv, tc.role)
			t.Setenv(regroupPIDEnv, pidfile)
			t.Cleanup(func() { stopHolder(t, pidfile) })
			old := pipeGrace
			pipeGrace = 200 * time.Millisecond
			t.Cleanup(func() { pipeGrace = old })

			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()
			done := make(chan error, 1)
			start := time.Now()
			go func() {
				_, err := runArgv(ctx, []string{self})
				done <- err
			}()
			select {
			case err := <-done:
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("err = %v, want it to say %q", err, tc.want)
				}
				if d := time.Since(start); d > 8*time.Second {
					t.Fatalf("runArgv returned after %s", d)
				}
			case <-time.After(8 * time.Second):
				t.Fatalf("runArgv still waiting after 8s: a re-grouped descendant holding the pipe made the bound unbounded")
			}
		})
	}
}

// stopHolder kills the holder through the pid its parent recorded.
func stopHolder(t *testing.T, pidfile string) {
	t.Helper()
	b, err := os.ReadFile(pidfile)
	if err != nil {
		t.Logf("no holder pid recorded: %v", err)
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid <= 1 {
		t.Logf("holder pid %q unusable", b)
		return
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
}
