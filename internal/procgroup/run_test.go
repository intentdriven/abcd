package procgroup

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// shCommand is sh -c script under ctx with the short WaitDelay a
// context-bound git runs under, so a wait exec times from the command's exit
// would miss a pipe held a second past it.
func shCommand(t *testing.T, ctx context.Context, script string) *exec.Cmd {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", script)
	cmd.WaitDelay = 50 * time.Millisecond
	return cmd
}

// TestRunToEOFReadsACommandWhoseOutputClosesLate: a command that answers and
// exits 0, leaving a process holding its output for a second, is read whole
// and reported as the success it was, its standard input fed whole and its
// standard error read too (iss-2610272041031418).
func TestRunToEOFReadsACommandWhoseOutputClosesLate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := shCommand(t, ctx, `read line; echo "got $line"; echo note >&2; sleep 1 &`)
	cmd.Stdin = strings.NewReader("ask\n")
	var out, errOut bytes.Buffer
	if err := RunToEOF(ctx, cmd, &out, &errOut); err != nil {
		t.Fatalf("RunToEOF = %v; want nil", err)
	}
	if out.String() != "got ask\n" || errOut.String() != "note\n" {
		t.Errorf("read %q and %q; want %q and %q", out.String(), errOut.String(), "got ask\n", "note\n")
	}
}

// TestRunToEOFEndsAtTheContextWhileAPipeIsHeld: a process the command left
// holding its output cannot hold the caller past ctx, and the output read
// before then is answered as ctx's error, never as the command's success.
func TestRunToEOFEndsAtTheContextWhileAPipeIsHeld(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	cmd := shCommand(t, ctx, `echo partial; sleep 5 &`)
	start := time.Now()
	err := RunToEOF(ctx, cmd, &bytes.Buffer{}, nil)
	if took := time.Since(start); took > 3*time.Second {
		t.Errorf("returned after %s; the context ended at 300ms", took)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("RunToEOF = %v; want context.DeadlineExceeded", err)
	}
}

// TestRunToEOFStopsACommandThatHangs: a command still running when ctx ends
// is stopped, and its Cancel's answer is returned.
func TestRunToEOFStopsACommandThatHangs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	cmd := shCommand(t, ctx, `exec sleep 10`)
	start := time.Now()
	err := RunToEOF(ctx, cmd, &bytes.Buffer{}, &bytes.Buffer{})
	if took := time.Since(start); took > 3*time.Second {
		t.Errorf("returned after %s; the context ended at 300ms", took)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("RunToEOF = %v; want context.DeadlineExceeded", err)
	}
}

// TestRunToEOFReportsAFailureAsItself: a command that exits non-zero is its
// exit error, with what it wrote read whole.
func TestRunToEOFReportsAFailureAsItself(t *testing.T) {
	cmd := shCommand(t, context.Background(), `echo why >&2; exit 3`)
	var errOut bytes.Buffer
	err := RunToEOF(context.Background(), cmd, nil, &errOut)
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 3 || errOut.String() != "why\n" {
		t.Errorf("RunToEOF = %v with stderr %q; want exit 3 and %q", err, errOut.String(), "why\n")
	}
}
