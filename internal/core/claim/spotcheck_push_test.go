package claim

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestPushHonoursCallerContext: the spot-check's push runs under its
// caller's context, its hooks included. A pre-push hook that hangs is
// stopped when the caller's deadline ends, within the runner's grace, and
// the push is an error that moved nothing on origin; with no hook in the way
// the same push goes through.
func TestPushHonoursCallerContext(t *testing.T) {
	r := spotcheckOrigin(t)
	bare := strings.TrimSpace(r.Git("remote", "get-url", Remote))
	tip := strings.TrimSpace(r.Git("rev-parse", "HEAD"))
	r.Write("b.txt", "b\n")
	r.Commit("the next")
	next := strings.TrimSpace(r.Git("rev-parse", "HEAD"))
	hooks := t.TempDir()
	if err := os.WriteFile(filepath.Join(hooks, "pre-push"), []byte("#!/bin/sh\nexec sleep 600\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("config", "core.hooksPath", hooks)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// grace is the runner's own once its context ends; margin is what the
	// test allows past the two for processes to start and be reaped.
	const grace, margin = 2 * time.Second, 4 * time.Second
	start := time.Now()
	err := PushBranch(ctx, r.Root(), next, spotcheckBranchRef)
	if took := time.Since(start); took > time.Second+grace+margin {
		t.Fatalf("the push took %s under a caller's deadline of a second", took)
	}
	if err == nil {
		t.Fatal("a push whose hook outlasted the caller's deadline succeeded")
	}
	remote := func() string {
		out, _ := os.ReadFile(filepath.Join(bare, filepath.FromSlash(spotcheckBranchRef)))
		if len(out) == 0 {
			return strings.TrimSpace(r.Git("ls-remote", Remote, spotcheckBranchRef))
		}
		return string(out)
	}
	if got := remote(); !strings.Contains(got, tip) {
		t.Fatalf("origin's branch moved under a push cut off by its caller: %s", got)
	}
	r.Git("config", "--unset", "core.hooksPath")
	if err := PushBranch(context.Background(), r.Root(), next, spotcheckBranchRef); err != nil {
		t.Fatalf("the push with no hook in the way: %v", err)
	}
	if got := remote(); !strings.Contains(got, next) {
		t.Fatalf("origin's branch is at %s after the push; want %s", got, next)
	}
}
