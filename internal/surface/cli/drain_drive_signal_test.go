//go:build unix

package cli

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
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
