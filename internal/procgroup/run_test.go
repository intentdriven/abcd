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

// unseenEnd is the context a command was built with, as RunToEOF's wait
// sees it on a run where the command's pipes close first: its Done never
// closes, and its Err is the command's context's.
type unseenEnd struct{ context.Context }

func (unseenEnd) Done() <-chan struct{} { return nil }

// TestRunToEOFStopsACommandThatHangsWhosePipesCloseFirst: a command its
// context stopped is answered as the context's error even when the pipes the
// kill closed are drained before the wait sees the context end, which under
// load lost the race the other way and returned the bare signal: killed
// (iss-2610310054494441).
func TestRunToEOFStopsACommandThatHangsWhosePipesCloseFirst(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	cmd := shCommand(t, ctx, `exec sleep 10`)
	err := RunToEOF(unseenEnd{ctx}, cmd, &bytes.Buffer{}, &bytes.Buffer{})
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

// TestRunToEOFGivesOneWriterForBothStreamsOnePipe: a writer given as both
// stdout and stderr is written by one reader, in the order the command
// wrote, as exec's one descriptor for it would be.
func TestRunToEOFGivesOneWriterForBothStreamsOnePipe(t *testing.T) {
	cmd := shCommand(t, context.Background(), `echo a; echo b >&2; echo c`)
	var both bytes.Buffer
	if err := RunToEOF(context.Background(), cmd, &both, &both); err != nil {
		t.Fatal(err)
	}
	if both.String() != "a\nb\nc\n" {
		t.Errorf("read %q; want %q", both.String(), "a\nb\nc\n")
	}
}

// failingWriter refuses every write.
type failingWriter struct{}

var errRefused = errors.New("refused")

func (failingWriter) Write([]byte) (int, error) { return 0, errRefused }

// TestRunToEOFDrainsAndReportsAWriterThatFails: a writer that fails leaves
// the command neither blocked on a full pipe nor reported as a success; the
// rest of its output is drained and the writer's error returned.
func TestRunToEOFDrainsAndReportsAWriterThatFails(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := shCommand(t, ctx, `i=0; while [ $i -lt 4000 ]; do echo 0123456789012345678901234567890123456789; i=$((i+1)); done`)
	start := time.Now()
	err := RunToEOF(ctx, cmd, failingWriter{}, nil)
	if took := time.Since(start); took > 30*time.Second {
		t.Errorf("returned after %s; want the command to finish once drained", took)
	}
	if !errors.Is(err, errRefused) {
		t.Errorf("RunToEOF = %v; want the writer's error", err)
	}
}
