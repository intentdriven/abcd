package statusline

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestCheckoutRootAnswersAsGitutilDoes: inside a checkout the bounded
// resolution names git's toplevel, from a subdirectory too; outside one it is
// ErrNoCheckout.
func TestCheckoutRootAnswersAsGitutilDoes(t *testing.T) {
	root := composeRepo(t)
	for _, dir := range []string{root, root + "/.abcd"} {
		got, err := CheckoutRoot(context.Background(), dir)
		if err != nil || got != root {
			t.Errorf("CheckoutRoot(%s) = %q, %v; want %q", dir, got, err, root)
		}
	}
	if _, err := CheckoutRoot(context.Background(), t.TempDir()); !errors.Is(err, ErrNoCheckout) {
		t.Errorf("outside a checkout: err = %v, want ErrNoCheckout", err)
	}
}

// TestCheckoutRootReportsAnEndedContextAsItself: a context that has ended is
// returned as its own error and never as ErrNoCheckout, so the verb cannot
// read "git did not answer in time" as "not a repository" and run the
// person's previous command over a checkout abcd manages.
func TestCheckoutRootReportsAnEndedContextAsItself(t *testing.T) {
	root := composeRepo(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := CheckoutRoot(ctx, root)
	if !errors.Is(err, context.Canceled) || errors.Is(err, ErrNoCheckout) {
		t.Errorf("err = %v, want context.Canceled and not ErrNoCheckout", err)
	}
}

// TestCheckoutRootReportsARepeatedWaitDelayMissAsATimeout: a git whose
// output misses the runner's WaitDelay on its retry too (a loaded machine;
// here, a process holding git's pipe) is git not answering in time, returned
// as gitutil.ErrGitTimedOut and never as ErrNoCheckout, so the verb blanks
// the row rather than running the person's previous command over a checkout
// abcd manages (iss-2610100846469473).
func TestCheckoutRootReportsARepeatedWaitDelayMissAsATimeout(t *testing.T) {
	root := composeRepo(t)
	gittest.SlowPipeGit(t, "--show-toplevel", -1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	_, err := CheckoutRoot(ctx, root)
	if !errors.Is(err, gitutil.ErrGitTimedOut) || errors.Is(err, ErrNoCheckout) {
		t.Errorf("err = %v, want gitutil.ErrGitTimedOut and not ErrNoCheckout", err)
	}
}

// TestComposeContextDropsOnlyTheBranchWhenGitRunsOut: with git's time already
// spent, the row still carries the badge and the repository; only the branch,
// the one element git supplies, drops.
func TestComposeContextDropsOnlyTheBranchWhenGitRunsOut(t *testing.T) {
	root := composeRepo(t)
	git(t, root, "commit", "--allow-empty", "-m", "root")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := ComposeContext(ctx, root, Payload{}, Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.Row.Element(KeyBranch); ok {
		t.Errorf("the branch rendered with git's time spent: %q", res.Row.Plain())
	}
	if _, ok := res.Row.Element(KeyPresence); !ok {
		t.Errorf("the badge dropped: %q", res.Row.Plain())
	}
	if _, ok := res.Row.Element(KeyRepo); !ok {
		t.Errorf("the repository dropped: %q", res.Row.Plain())
	}

	// The control: the same checkout with time to answer names its branch.
	res, err = ComposeContext(context.Background(), root, Payload{}, Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if got := plainOf(t, res.Row, KeyBranch); got != "main" {
		t.Errorf("branch = %q, want main", got)
	}
}
