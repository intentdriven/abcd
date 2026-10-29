package gitutil_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestTheStoredLineReadHandsOnEveryLine: a stored read line by line hands on
// every line of git's output, however much of it there is, each cut to the
// line bound, and keeps none; the last line is handed on without its
// newline too.
func TestTheStoredLineReadHandsOnEveryLine(t *testing.T) {
	stubGit(t, "i=0; while [ $i -lt 30000 ]; do echo \"line $i of a long listing\"; i=$((i+1)); done; printf tail")
	var n int
	var last string
	err := gitutil.EachAsStoredContext(context.Background(), t.TempDir(), 8, func(line string) {
		n++
		last = line
		if len(line) > 8 {
			t.Fatalf("a line of %d bytes was handed on; want at most 8", len(line))
		}
	}, "for-each-ref")
	if err != nil || n != 30001 || last != "tail" {
		t.Fatalf("read %d lines, the last %q, %v; want 30001, the last \"tail\"", n, last, err)
	}
}

// TestTheStoredLineReadEndsWithItsContext: a git that hangs is killed when
// the context ends, and the read answers the context's own error; one whose
// context has already ended runs no git.
func TestTheStoredLineReadEndsWithItsContext(t *testing.T) {
	runs := stubGit(t, "echo first; exec sleep 10")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := gitutil.EachAsStoredContext(ctx, t.TempDir(), 64, func(string) {}, "for-each-ref")
	if took := time.Since(start); took > 3*time.Second {
		t.Fatalf("a hanging git held the read %s", took)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v; want the deadline", err)
	}
	// The first read's git may or may not have started before its deadline
	// on a loaded machine, so what is held is that the second, whose context
	// has already ended, starts none.
	countRuns := func() int {
		data, _ := os.ReadFile(runs)
		return strings.Count(string(data), "run\n")
	}
	before := countRuns()
	done, cancelDone := context.WithCancel(context.Background())
	cancelDone()
	if err := gitutil.EachAsStoredContext(done, t.TempDir(), 64, func(string) {}, "for-each-ref"); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v; want the ended context", err)
	}
	if got := countRuns(); got != before || got > 1 {
		t.Fatalf("git ran %d times (%d before the ended read); want no run for a read whose context had ended, and at most one for the first", got, before)
	}
}
