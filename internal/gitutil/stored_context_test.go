package gitutil_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// The deadline-bound stored read the status reader's walk and batch reads
// run through (itd-2609151838312703 C12): a git that hangs is killed at the
// deadline, a deadline already passed runs no git at all, and a git whose
// output stays open after it answered is read once the output closes, or
// ends at the deadline.

// stubGit puts a `git` first on PATH that answers `git version` as a git
// that honours GIT_NO_LAZY_FETCH, so the object-read guard asks it nothing
// more, and runs act for every other command, logging each run to the
// returned file.
func stubGit(t *testing.T, act string) (runs string) {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	bin := t.TempDir()
	runs = filepath.Join(t.TempDir(), "runs")
	body := "#!/bin/sh\nfor a in \"$@\"; do [ \"$a\" = version ] && { echo 'git version 2.52.0'; exit 0; }; done\n" +
		"echo run >> '" + runs + "'\n" + act + "\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return runs
}

// TestTheStoredReadKillsAGitThatHangs: a git that sleeps past the deadline
// is killed when it ends, and the read answers the context's own error.
func TestTheStoredReadKillsAGitThatHangs(t *testing.T) {
	stubGit(t, "exec sleep 10")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	out, err := gitutil.RunCappedBytesAsStoredContext(ctx, t.TempDir(), "", 1024, "for-each-ref")
	if took := time.Since(start); took > 2*time.Second {
		t.Fatalf("returned after %s; the deadline was 300ms", took)
	}
	if out != nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("read = %q, %v; want nothing and context.DeadlineExceeded", out, err)
	}
}

// TestTheStoredReadRunsNoGitPastItsDeadline: a context already ended is
// answered before any git starts.
func TestTheStoredReadRunsNoGitPastItsDeadline(t *testing.T) {
	runs := stubGit(t, "exec sleep 10")
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	<-ctx.Done()
	if _, err := gitutil.RunCappedBytesAsStoredContext(ctx, t.TempDir(), "", 1024, "for-each-ref"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v; want context.DeadlineExceeded", err)
	}
	if _, err := os.Stat(runs); !os.IsNotExist(err) {
		t.Fatalf("a git ran past an ended deadline: %v", err)
	}
}

// TestTheStoredReadReadsAnAnswerWhoseOutputClosesLate: a git that answers
// and leaves a process holding its output open is read once the output
// closes, its standard input fed whole, and asked once: a successful git is
// never reported as missing a wait (iss-2610272041031418).
func TestTheStoredReadReadsAnAnswerWhoseOutputClosesLate(t *testing.T) {
	runs := stubGit(t, "read line\necho \"got $line\"\nsleep 1 &")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := gitutil.RunCappedBytesAsStoredContext(ctx, t.TempDir(), "ask\n", 1024, "cat-file", "--batch")
	if err != nil || string(out) != "got ask\n" {
		t.Fatalf("read = %q, %v; want %q, nil", out, err, "got ask\n")
	}
	data, _ := os.ReadFile(runs)
	if n := strings.Count(string(data), "run\n"); n != 1 {
		t.Fatalf("git ran %d times; want once", n)
	}
}

// TestTheStoredReadEndsAtItsDeadlineWhileTheOutputStaysOpen: a process git
// left holding its output cannot hold the read past its deadline, and what
// was read before it is never answered as git's.
func TestTheStoredReadEndsAtItsDeadlineWhileTheOutputStaysOpen(t *testing.T) {
	stubGit(t, "sleep 5 &\necho answer")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	out, err := gitutil.RunCappedBytesAsStoredContext(ctx, t.TempDir(), "", 1024, "for-each-ref")
	if took := time.Since(start); took > 3*time.Second {
		t.Fatalf("returned after %s; the deadline was 300ms", took)
	}
	if out != nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("read = %q, %v; want nothing and context.DeadlineExceeded", out, err)
	}
}
