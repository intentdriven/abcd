package statusline

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/mode"
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

// TestCheckoutRootReadsAGitWhoseOutputClosesLate: a git that answered while
// a process it left holds its output open is read once the output closes, not
// taken for git not answering (a loaded machine's late copy of the answer
// once was, iss-2610272041031418), and never as ErrNoCheckout, which would run
// the person's previous command over a checkout abcd manages.
func TestCheckoutRootReadsAGitWhoseOutputClosesLate(t *testing.T) {
	root := composeRepo(t)
	want, err := gitutil.Toplevel(root)
	if err != nil {
		t.Fatal(err)
	}
	gittest.SlowPipeGit(t, "--show-toplevel", -1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if got, err := CheckoutRoot(ctx, root); got != want || err != nil {
		t.Errorf("CheckoutRoot = %q, %v; want %q, nil", got, err, want)
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

// TestComposeContextNeverBadgesAnUnreadModeAsManaged: a stored mode is read
// only once git says it tracks none of it, so with git's time spent the
// composition is an error, never a badge saying managed over a stop parked on
// someone; with time to answer it reads the mode.
func TestComposeContextNeverBadgesAnUnreadModeAsManaged(t *testing.T) {
	root := composeRepo(t)
	git(t, root, "commit", "--allow-empty", "-m", "root")
	if err := mode.SetAt(root, mode.Facilitator); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := ComposeContext(ctx, root, Payload{}, Defaults()); err == nil {
		t.Fatalf("a mode git was not asked about in time composed as %q: %q", res.State, res.Row.Plain())
	}
	res, err := ComposeContext(context.Background(), root, Payload{}, Defaults())
	if err != nil || res.State != mode.Facilitator {
		t.Fatalf("with time to answer: state %q, %v; want %q", res.State, err, mode.Facilitator)
	}
}
