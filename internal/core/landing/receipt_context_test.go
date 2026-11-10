package landing

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// receiptRepo is a checkout holding this checkout's own receipt for HEAD.
func receiptRepo(t *testing.T) (root, sha string) {
	t.Helper()
	repo := gittest.NewRepo(t)
	repo.Write(".gitignore", TierRelDir+"/\n")
	repo.Commit("seed")
	sha = repo.Git("rev-parse", "HEAD")
	repo.Write(ReceiptsRelDir+"/"+sha, "commit "+sha+"\n")
	return repo.Root(), sha
}

// stallingGit puts a git first on PATH that logs every call and, on a call
// naming the argument stall, sleeps far past any deadline a test sets; every
// other call goes to the real git. An empty stall stalls nothing.
func stallingGit(t *testing.T, stall string) (calls func() []string) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not on PATH")
	}
	bin := t.TempDir()
	log := filepath.Join(t.TempDir(), "calls")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> '" + log + "'\n"
	if stall != "" {
		script += "for a in \"$@\"; do\n  [ \"$a\" = '" + stall + "' ] && exec sleep 60\ndone\n"
	}
	script += "exec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return func() []string {
		raw, _ := os.ReadFile(log)
		_ = os.WriteFile(log, nil, 0o600)
		var out []string
		for _, l := range strings.Split(string(raw), "\n") {
			if l != "" {
				out = append(out, l)
			}
		}
		return out
	}
}

// TestTheReceiptReadStartsNoGitOnASpentDeadline: the landing-receipt read the
// stop check makes runs its git under the caller's ctx, so a ctx already ended
// lists no worktree and checks no receipt, and says so with its own error
// rather than reading as no receipt found.
func TestTheReceiptReadStartsNoGitOnASpentDeadline(t *testing.T) {
	root, sha := receiptRepo(t)
	calls := stallingGit(t, "")
	if p, err := ReceiptContext(context.Background(), root, sha); err != nil || p == "" {
		t.Fatalf("a live ctx: %q, %v; want the receipt found", p, err)
	}
	if got := calls(); len(got) == 0 {
		t.Fatal("the receipt read ran no git with a live ctx; the count proves nothing")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p, err := ReceiptContext(ctx, root, sha)
	if p != "" || !errors.Is(err, context.Canceled) {
		t.Errorf("a spent ctx: %q, %v; want no receipt and the ctx's own error", p, err)
	}
	if got := calls(); len(got) != 0 {
		t.Errorf("a spent ctx still ran git: %q", got)
	}
}

// TestTheReceiptReadEndsAtItsDeadlineWhileGitHangs: each git call the receipt
// read makes, the worktree listing and the receipt's tracked check, is killed
// when the caller's deadline ends, so a git that hangs holds the read no
// longer than the deadline, and the read refuses rather than finding nothing.
// The read must end within endBound of its start: generous for a loaded
// machine over a 500 ms deadline, and far short of the stalled git's 60
// seconds, so a read that waited on git rather than killing it fails.
func TestTheReceiptReadEndsAtItsDeadlineWhileGitHangs(t *testing.T) {
	for name, stall := range map[string]string{
		"the worktree listing": "worktree",
		"the tracked check":    "ls-files",
	} {
		t.Run(name, func(t *testing.T) {
			root, sha := receiptRepo(t)
			calls := stallingGit(t, stall)
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()
			type answer struct {
				p   string
				err error
			}
			const endBound = 10 * time.Second
			done := make(chan answer, 1)
			start := time.Now()
			go func() {
				p, err := ReceiptContext(ctx, root, sha)
				done <- answer{p, err}
			}()
			var a answer
			select {
			case a = <-done:
			case <-time.After(30 * time.Second):
				t.Fatalf("the receipt read hung past its deadline on %s", name)
			}
			if took := time.Since(start); took > endBound {
				t.Errorf("a hanging %s held the receipt read %v past a 500ms deadline; want it ended within %v", name, took, endBound)
			}
			if a.p != "" || a.err == nil {
				t.Errorf("a hanging %s: %q, %v; want the read refused", name, a.p, a.err)
			}
			stalled := 0
			for _, c := range calls() {
				for _, f := range strings.Fields(c) {
					if f == stall {
						stalled++
						break
					}
				}
			}
			if stalled == 0 {
				t.Errorf("no %s call was made; the test proves nothing", name)
			}
		})
	}
}
