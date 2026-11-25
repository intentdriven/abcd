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

// alive reports whether pid names a process that has not been reaped.
func alive(pid int) bool { return syscall.Kill(pid, 0) == nil }

// readPid waits for a pid file to be written and returns its pid.
func readPid(t *testing.T, p string) int {
	t.Helper()
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		raw, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if pid, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil && pid > 0 {
			return pid
		}
	}
	t.Fatalf("no pid was written at %s", p)
	return 0
}

// TestAStopEndsTheLaunchAsTheRunsErrorAndKillsItsGroupWhole: a launch stopped
// while its copy's runner is at work kills that runner's whole process group,
// the process it left running in the background included, through the handle
// the launch holds; a process of the same name outside the group lives on, so
// nothing is killed by a pattern. The stop is the run's error, never the
// copy's score: no trial is written, the copy is left interrupted, and it is
// not launched again.
func TestAStopEndsTheLaunchAsTheRunsErrorAndKillsItsGroupWhole(t *testing.T) {
	r, home, m, rec, _ := checkLab(t)
	key := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 1}
	awaitRunner(t, r, home, m, rec, key, KindCode)
	logs := t.TempDir()
	q := func(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }
	// The fake runner leaves a sleeper in the background, then works until
	// it is killed, as a copy at work does.
	script := "#!/bin/sh\n" +
		"/bin/sleep 300 &\n" +
		"echo $! > " + q(filepath.Join(logs, "child")) + "\n" +
		"echo $$ > " + q(filepath.Join(logs, "leader")) + "\n" +
		"exec /bin/sleep 300\n"
	d := launchCell(t, home, script)
	// A process of the same name, in a group of its own, that the stop must
	// leave alone.
	decoy := exec.Command("/bin/sleep", "300")
	decoy.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := decoy.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = decoy.Process.Kill(); _ = decoy.Wait() })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	var res LaunchResult
	go func() {
		var err error
		res, err = testLauncher(r, home, m, d).launch(ctx, rec.RunID, &key, nil)
		done <- err
	}()
	leader, child := readPid(t, filepath.Join(logs, "leader")), readPid(t, filepath.Join(logs, "child"))
	if pgid, err := syscall.Getpgid(child); err != nil || pgid != leader {
		t.Fatalf("the runner's background process is in group %d (%v); want the runner's own group %d", pgid, err, leader)
	}
	cancel()
	var err error
	select {
	case err = <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("the launch did not end after its stop")
	}
	if !errors.Is(err, context.Canceled) || !errors.Is(err, ErrWrittenInPart) {
		t.Fatalf("the stop ends the launch as the run's error, after a write: %v", err)
	}
	for deadline := time.Now().Add(10 * time.Second); (alive(leader) || alive(child)) && time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
	}
	if alive(leader) || alive(child) {
		t.Fatalf("the runner's group lives on after the stop: leader %v, background %v", alive(leader), alive(child))
	}
	if !alive(decoy.Process.Pid) {
		t.Fatal("a process outside the runner's group was killed")
	}
	if len(res.Copies) != 1 || res.Copies[0].State != HandOffInterrupted || res.Copies[0].Status != "" {
		t.Fatalf("the stopped copy is interrupted and carries no status: %+v", res.Copies)
	}
	if _, ok := trialOf(t, r, m, rec, key); ok {
		t.Fatal("a stopped copy has no trial")
	}
	if h := handOffNow(t, r, m, rec, key); h.State != HandOffInterrupted {
		t.Fatalf("the copy's hand-off is interrupted: %+v", h)
	}
	if _, err := testLauncher(r, home, m, d).launch(context.Background(), rec.RunID, &key, nil); err == nil || !strings.Contains(err.Error(), "already launched") {
		t.Fatalf("an interrupted copy is not launched again: %v", err)
	}
}
