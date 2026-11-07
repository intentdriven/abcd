//go:build unix

package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/runner"
)

const driveSignalChildEnv = "ABCD_DRIVE_SIGNAL_CHILD"

var driveSignalsByName = map[string]syscall.Signal{"int": syscall.SIGINT, "hup": syscall.SIGHUP, "term": syscall.SIGTERM}

// TestTheFirstSignalLeavesTheContextLiveAndTheSecondEndsIt runs a real drive's
// front door in a child: the stand-in loop signals the process, sees the stop
// asked for with its context live, signals again, and sees the context end.
func TestTheFirstSignalLeavesTheContextLiveAndTheSecondEndsIt(t *testing.T) {
	if name := os.Getenv(driveSignalChildEnv); name != "" {
		w := newDriveWorld(t)
		w.passWalls(t)
		sig := driveSignalsByName[name]
		driveTerminal = func(ctx context.Context, _ string, o loop.TerminalOptions) (loop.TerminalResult, error) {
			_ = syscall.Kill(os.Getpid(), sig)
			select {
			case <-o.Stop:
			case <-time.After(5 * time.Second):
				os.Exit(3)
			}
			// The call in flight would be killed by the context ending.
			time.Sleep(200 * time.Millisecond)
			if ctx.Err() != nil {
				os.Exit(4)
			}
			_ = syscall.Kill(os.Getpid(), sig)
			select {
			case <-ctx.Done():
				os.Exit(0)
			case <-time.After(5 * time.Second):
				os.Exit(5)
			}
			return loop.TerminalResult{}, nil
		}
		var out, errb strings.Builder
		Run([]string{"drain", "--drive", "--yes"}, &out, &errb)
		os.Exit(6)
	}
	for name := range driveSignalsByName {
		t.Run(name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestTheFirstSignalLeavesTheContextLiveAndTheSecondEndsIt$", "-test.count=1")
			cmd.Env = append(os.Environ(), driveSignalChildEnv+"="+name)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("the child: %v\n%s", err, out)
			}
		})
	}
}

// TestASecondSignalExitsEvenIfTheLoopDoesNotReturn: the process does not wait
// on anything that does not watch the context.
func TestASecondSignalExitsEvenIfTheLoopDoesNotReturn(t *testing.T) {
	exited := make(chan int, 1)
	was := driveExit
	t.Cleanup(func() { driveExit = was })
	driveExit = func(code int) { exited <- code }
	s := watchDriveSignals(func() {}, func() {})
	t.Cleanup(s.close)
	s.ch <- syscall.SIGINT
	<-s.stop
	s.ch <- syscall.SIGINT
	<-s.ctx.Done()
	select {
	case code := <-exited:
		if code != driveKilledExit {
			t.Fatalf("exit %d", code)
		}
	case <-time.After(driveKillGrace + 5*time.Second):
		t.Fatal("the process did not exit")
	}
}

// TestTheDrivesEndNamesTheRunTheLaneAndTheResume: a stop's line, and a kill's
// naming when the reservation lapses.
func TestTheDrivesEndNamesTheRunTheLaneAndTheResume(t *testing.T) {
	at := time.Date(2026, 11, 6, 9, 0, 0, 0, time.UTC)
	var b strings.Builder
	renderDriveEnd(&b, loop.TerminalResult{Ended: loop.TerminalStopped, Run: "run-1", Lane: "lane-1", Issue: "iss-1", Resume: loop.TerminalResumeCommand})
	for _, want := range []string{"run-1", "lane-1", "iss-1", "abcd drain --drive"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("the stop does not name %q:\n%s", want, b.String())
		}
	}
	b.Reset()
	renderDriveEnd(&b, loop.TerminalResult{Ended: loop.TerminalKilled, Run: "run-1", Lane: "lane-1", Issue: "iss-1", Resume: loop.TerminalResumeCommand, Lapses: &at})
	if !strings.Contains(b.String(), "2026-11-06T09:00:00Z") {
		t.Errorf("the kill names when the reservation lapses:\n%s", b.String())
	}
}

// TestAKillExitsWithItsOwnCode.
func TestAKillExitsWithItsOwnCode(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	driveTerminal = func(context.Context, string, loop.TerminalOptions) (loop.TerminalResult, error) {
		return loop.TerminalResult{Ended: loop.TerminalKilled}, nil
	}
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != driveKilledExit {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
}

const driveProofChildEnv = "ABCD_DRIVE_PROOF_CHILD"

// proofHarness is a claude on PATH whose live launch records its pid and its
// child's in dir, waits for dir/go, says so in dir/finished, and answers with
// every wall holding.
func proofHarness(t *testing.T, dir string) {
	t.Helper()
	bin := t.TempDir()
	stream := filepath.Join(bin, "stream")
	if err := os.WriteFile(stream, []byte(wallInit+wallRefused+wallResult), 0o600); err != nil {
		t.Fatal(err)
	}
	script := fmt.Sprintf(`#!/bin/sh
if [ "$1" = auth ]; then echo '{"loggedIn":true}'; exit 0; fi
/bin/sleep 60 >/dev/null 2>&1 &
echo $! > '%[1]s/child.tmp'; mv '%[1]s/child.tmp' '%[1]s/child'
echo $$ > '%[1]s/leader.tmp'; mv '%[1]s/leader.tmp' '%[1]s/leader'
while [ ! -e '%[1]s/go' ]; do /bin/sleep 0.05; done
: > '%[1]s/finished'
kill $! 2>/dev/null
`+strings.ReplaceAll(wallHomeLine, "%", "%%")+`cat '%[2]s'
`, dir, stream)
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// waitPidFile reads a pid the harness wrote.
func waitPidFile(t *testing.T, path string) int {
	t.Helper()
	for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if raw, err := os.ReadFile(path); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil {
				t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
				return pid
			}
		}
	}
	t.Fatalf("the harness wrote no %s", path)
	return 0
}

func pidGone(pid int, wait time.Duration) bool {
	for deadline := time.Now().Add(wait); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) {
			return true
		}
	}
	return false
}

// TestSignalsThroughARealDriveAndARealProcessGroup runs the whole front door in
// a child, on real signals, with the launcher's real process group around a
// stand-in harness in its proof: the first signal leaves the harness to finish
// and ends the drive with nothing moved; the second kills the harness and its
// child and exits 130.
func TestSignalsThroughARealDriveAndARealProcessGroup(t *testing.T) {
	if dir := os.Getenv(driveProofChildEnv); dir != "" {
		w := newDriveWorld(t)
		w.recordTerminalStopped(t)
		proofHarness(t, dir)
		var out, errb strings.Builder
		code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb)
		_ = os.WriteFile(filepath.Join(dir, "output"), []byte(out.String()+errb.String()), 0o600)
		os.Exit(code)
	}
	start := func(t *testing.T) (*exec.Cmd, string) {
		dir := t.TempDir()
		cmd := exec.Command(os.Args[0], "-test.run=^TestSignalsThroughARealDriveAndARealProcessGroup$", "-test.count=1")
		cmd.Env = append(os.Environ(), driveProofChildEnv+"="+dir)
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = cmd.Process.Kill() })
		return cmd, dir
	}
	output := func(dir string) string {
		raw, _ := os.ReadFile(filepath.Join(dir, "output"))
		return string(raw)
	}
	t.Run("first", func(t *testing.T) {
		cmd, dir := start(t)
		leader := waitPidFile(t, filepath.Join(dir, "leader"))
		_ = cmd.Process.Signal(syscall.SIGINT)
		time.Sleep(300 * time.Millisecond)
		if err := syscall.Kill(leader, 0); err != nil {
			t.Fatalf("the first signal does not reach the agent: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "go"), nil, 0o600); err != nil {
			t.Fatal(err)
		}
		var stopped *exec.ExitError
		if err := cmd.Wait(); !errors.As(err, &stopped) || stopped.ExitCode() != 3 {
			t.Fatalf("a stop exits 3: %v\n%s", err, output(dir))
		}
		if _, err := os.Stat(filepath.Join(dir, "finished")); err != nil {
			t.Fatalf("the agent finished its call: %v\n%s", err, output(dir))
		}
		if !strings.Contains(output(dir), "ended: stopped") {
			t.Fatalf("the drive says it stopped:\n%s", output(dir))
		}
	})
	t.Run("second", func(t *testing.T) {
		cmd, dir := start(t)
		child := waitPidFile(t, filepath.Join(dir, "child"))
		leader := waitPidFile(t, filepath.Join(dir, "leader"))
		_ = cmd.Process.Signal(syscall.SIGINT)
		time.Sleep(300 * time.Millisecond)
		_ = cmd.Process.Signal(syscall.SIGINT)
		err := cmd.Wait()
		var ee *exec.ExitError
		if !errors.As(err, &ee) || ee.ExitCode() != driveKilledExit {
			t.Fatalf("the second signal exits %d: %v\n%s", driveKilledExit, err, output(dir))
		}
		if !pidGone(leader, 5*time.Second) || !pidGone(child, 5*time.Second) {
			t.Fatal("the agent's process group outlived the signal")
		}
		if _, err := os.Stat(filepath.Join(dir, "finished")); err == nil {
			t.Fatal("the agent was killed before it finished")
		}
		if strings.Contains(output(dir), "does not hold") {
			t.Fatalf("a kill is no wall refusing:\n%s", output(dir))
		}
	})
}

// recordTerminalStopped stands in for the loop as a stop finds it: asked to stop
// before its first move.
func (w *driveWorld) recordTerminalStopped(t *testing.T) {
	t.Helper()
	term := driveTerminal
	t.Cleanup(func() { driveTerminal = term })
	driveTerminal = func(_ context.Context, _ string, o loop.TerminalOptions) (loop.TerminalResult, error) {
		select {
		case <-o.Stop:
			return loop.TerminalResult{Ended: loop.TerminalStopped}, nil
		default:
			return loop.TerminalResult{Ended: loop.DrainStoppedEmpty}, nil
		}
	}
}

const driveSigpipeChildEnv = "ABCD_DRIVE_SIGPIPE_CHILD"

// TestADrivesAgentsKeepTheDefaultSigpipe: a drive keeps writing after its
// terminal closes, but what it starts inherits an ignored signal across exec, so
// an agent's `yes | head` must still end on SIGPIPE, not spin on EPIPE.
func TestADrivesAgentsKeepTheDefaultSigpipe(t *testing.T) {
	if os.Getenv(driveSigpipeChildEnv) != "" {
		s := watchDriveSignals(func() {}, func() {})
		defer s.close()
		cmd := exec.Command("/bin/sh", "-c", `{ yes 2>/dev/null; echo "status $?" >&2; } | head -c 1 >/dev/null`)
		out, _ := cmd.CombinedOutput()
		fmt.Printf("%s\n", out)
		os.Exit(0)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestADrivesAgentsKeepTheDefaultSigpipe$", "-test.count=1")
	cmd.Env = append(os.Environ(), driveSigpipeChildEnv+"=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(string(out), "status 141") {
		t.Fatalf("yes did not end on SIGPIPE:\n%s", out)
	}
}

const driveClosedTerminalChildEnv = "ABCD_DRIVE_CLOSED_TERMINAL_CHILD"

// TestAClosedTerminalDoesNotEndAResumesWait: nothing is listening for the signal between
// two attempts, and a line to a terminal that went away is no reason to die there.
func TestAClosedTerminalDoesNotEndAResumesWait(t *testing.T) {
	if dir := os.Getenv(driveClosedTerminalChildEnv); dir != "" {
		goBackWorld(t)
		driveWalls.static = func(context.Context, runner.Profile) ([]string, error) {
			if err := os.WriteFile(filepath.Join(dir, "ready"), nil, 0o600); err != nil {
				os.Exit(4)
			}
			for range 2000 {
				if _, err := os.Stat(filepath.Join(dir, "closed")); err == nil {
					return nil, errors.New("a file was held for a moment")
				}
				time.Sleep(5 * time.Millisecond)
			}
			os.Exit(5)
			return nil, nil
		}
		driveStartWait = func(context.Context, time.Duration) error {
			fmt.Println("the wait was reached")
			os.Exit(0)
			return nil
		}
		Run(resumeArgs(), os.Stdout, os.Stderr)
		os.Exit(6)
	}
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestAClosedTerminalDoesNotEndAResumesWait$", "-test.count=1", "-test.timeout=2m")
	cmd.Env = append(os.Environ(), driveClosedTerminalChildEnv+"="+dir)
	var stdout strings.Builder
	cmd.Stdout = &stdout
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	for range 4000 {
		if _, err := os.Stat(filepath.Join(dir, "ready")); err == nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := stderr.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "closed"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil || !strings.Contains(stdout.String(), "the wait was reached") {
		t.Fatalf("%v:\n%s", err, stdout.String())
	}
}
