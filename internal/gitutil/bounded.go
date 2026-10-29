package gitutil

// bounded.go runs the working-tree commands abcd makes where the person's own
// git would (WorkingTreeCommand, WorkingTreeCommitCommand) and reads their
// output back, and bounds the ones that run the repository's own programs. It
// lives here, beside the commands it runs, so the build loop's lanes and the
// worktree store's add share one bounding: the loop imports the store, so
// neither could hold it for the other.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// maxTreeOutput caps what RunTreeCommand reads back from git.
const maxTreeOutput = 1 << 20

// stopGrace is how long a bounded command that ran past its bound and the
// processes it started have, once told to stop (SIGTERM to git and each of
// its descendants, on which git removes its lock files), before they are
// killed and git's output pipes closed; and how long a process a hook or
// signing program left behind may hold those pipes once git has exited.
const stopGrace = 5 * time.Second

// CheckoutBound bounds a `git worktree add` abcd makes through
// RunCommitCommandBounded. Its checkout writes every file of the tree through
// the repository's content filters, and a Git LFS smudge downloads each object
// it names, so its time scales with the tree and the network rather than with
// a person's answer. The bound is there only so a post-checkout hook that
// waits never holds the command forever.
const CheckoutBound = 60 * time.Minute

// ErrCheckoutStalled names a `git worktree add` stopped at its bound.
var ErrCheckoutStalled = errors.New("the checkout (its content filters, such as a Git LFS download, and its post-checkout hook) did not finish")

// RunTreeCommand runs cmd, a WorkingTreeCommand or WorkingTreeCommitCommand
// built for args, and returns its stdout verbatim: a porcelain status line
// opens with a space when the change is unstaged, and a caller compares
// against the line as git wrote it. A failure names the subcommand and git's
// bounded stderr; output past the cap is refused rather than cut.
func RunTreeCommand(cmd *exec.Cmd, args []string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	sub := treeSub(args)
	if err := cmd.Run(); err != nil {
		msg := bytes.TrimSpace(stderr.Bytes())
		if len(msg) > 2048 {
			msg = msg[:2048]
		}
		return "", fmt.Errorf("git %s: %w (%s)", sub, err, msg)
	}
	if stdout.Len() > maxTreeOutput {
		return "", fmt.Errorf("git %s wrote more than %d bytes", sub, maxTreeOutput)
	}
	return stdout.String(), nil
}

// RunCommitCommandBounded runs WorkingTreeCommitCommand(dir, args...), so the
// repository's hooks, its signing and its signature checks run as on the
// person's own git, and bounds it: past limit git and the processes below it
// (its hooks, their children, a signing program it ran) are told to stop
// together (SIGTERM to git and each descendant, stopWithDescendants, so git
// removes its lock files), killed stopGrace later if they have not, and the
// call is refused with stalled, naming the bound. Git stays in abcd's process
// group, as the person's own git runs in the shell's, so a prompt on the
// terminal and an interrupt behave as they do there. A program a hook started
// that outlives a git which finished in time is not killed (the hook's own
// children are the repository's), but its hold on git's output is cut after
// stopGrace. A git that exited 0 made its change even then.
func RunCommitCommandBounded(dir string, limit time.Duration, stalled error, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	cmd := WorkingTreeCommitCommand(ctx, dir, args...)
	sweep := stopWithDescendants(cmd)
	cmd.WaitDelay = stopGrace
	out, err := RunTreeCommand(cmd, args)
	sweep()
	switch {
	case err == nil:
		return out, nil
	case cmd.ProcessState != nil && cmd.ProcessState.Success() && errors.Is(err, exec.ErrWaitDelay):
		// git finished; something it started held the pipes past the grace.
		return "", nil
	case ctx.Err() != nil:
		return "", fmt.Errorf("git %s: %w within %s, so git was stopped", treeSub(args), stalled, limit)
	}
	return "", err
}

// treeSub is the subcommand args runs, past any `-c` overrides the caller put
// before it.
func treeSub(args []string) string {
	sub := args
	for len(sub) > 2 && sub[0] == "-c" {
		sub = sub[2:]
	}
	if len(sub) == 0 {
		return ""
	}
	return sub[0]
}
