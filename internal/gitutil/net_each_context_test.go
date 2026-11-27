package gitutil_test

import (
	"context"
	"errors"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// The line-wise runners ask the check before git runs, as every hooked or
// networked runner does (TestEveryHookedRunnerAsksTheCheckBeforeGitRuns).
func init() {
	hookedRunners["NetEachContext"] = func(dir string, args ...string) error {
		return gitutil.NetEachContext(context.Background(), dir, 64, func(string) {}, args...)
	}
	hookedRunners["RunNetEachContext"] = func(dir string, args ...string) error {
		return gitutil.RunNetEachContext(context.Background(), dir, 64, func(string) {}, args...)
	}
}

// TestTheContextRunnersReadAListingPastTheBound: a listing longer than the
// bound NetContext keeps whole is handed over by NetEachContext and
// RunNetEachContext line by line, every line, and a context already ended
// runs neither.
func TestTheContextRunnersReadAListingPastTheBound(t *testing.T) {
	r, bare := remoteWorld(t, "main")
	tip, err := gitutil.Run(bare, "rev-parse", "refs/heads/main")
	if err != nil {
		t.Fatal(err)
	}
	headEnding(t, bare, tip, gitutil.NetOutputCap)
	_, err = gitutil.NetContext(context.Background(), r.Root(), "ls-remote", "origin", "HEAD")
	var capped *gitutil.OutputCapError
	if !errors.As(err, &capped) || capped.Cap != gitutil.NetOutputCap {
		t.Fatalf("NetContext = %v; want the listing refused as past its bound", err)
	}
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	for name, run := range map[string]func(context.Context, func(string)) error{
		"NetEachContext": func(ctx context.Context, each func(string)) error {
			return gitutil.NetEachContext(ctx, r.Root(), gitutil.RefLineMax, each, "ls-remote", "origin", "HEAD")
		},
		"RunNetEachContext": func(ctx context.Context, each func(string)) error {
			return gitutil.RunNetEachContext(ctx, r.Root(), gitutil.RefLineMax, each, "ls-remote", "origin", "HEAD")
		},
	} {
		bytes, head := 0, gitutil.RefListing{Ref: "HEAD"}
		if err := run(context.Background(), func(line string) { bytes += len(line) + 1; head.Line(line) }); err != nil {
			t.Fatalf("%s = %v", name, err)
		}
		if bytes <= gitutil.NetOutputCap || head.Tips != 1 || head.Tip != tip {
			t.Fatalf("%s handed over %d bytes, HEAD read %+v; want more than %d, and HEAD at %s", name, bytes, head, gitutil.NetOutputCap, tip)
		}
		if err := run(ended, func(string) {}); !errors.Is(err, context.Canceled) {
			t.Fatalf("%s under an ended context = %v; want the context's error", name, err)
		}
	}
}
