package gitutil

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// asStored makes an isolated command read history as stored: no
// refs/replace/* ref stands one object in for another
// (GIT_NO_REPLACE_OBJECTS), and no .git/info/grafts file gives a commit
// parents it does not store (noGraftFile), which git honours even with
// replace refs off.
func asStored(cmd *exec.Cmd) *exec.Cmd {
	cmd.Env = append(cmd.Env, "GIT_NO_REPLACE_OBJECTS=1", "GIT_GRAFT_FILE="+noGraftFile)
	return cmd
}

// RunCappedAsStored is RunCapped reading history as stored (asStored): for a
// read whose answer is a commit's parents or what a branch holds, which a
// replace ref or a graft could otherwise change.
func RunCappedAsStored(root string, maxBytes int, args ...string) (string, error) {
	out, overflowed, err := runBoundedCmd(asStored(isolatedGit(root, args...)), maxBytes)
	if err != nil {
		return "", err
	}
	if overflowed {
		return "", fmt.Errorf("git %s: output exceeded the %d-byte cap; the answer would be truncated, and a truncated history is a wrong one",
			strings.Join(args, " "), maxBytes)
	}
	return strings.TrimRight(string(out), " \t\r\n"), nil
}

// RunCappedExitOneAsStored is RunCappedExitOne reading history as stored
// (asStored).
func RunCappedExitOneAsStored(root string, maxBytes int, args ...string) ([]byte, bool, error) {
	return runCappedExitOneCmd(asStored(isolatedGit(root, args...)), maxBytes, args)
}

// RunCappedBytesAsStoredContext is RunCappedAsStored bound to ctx, with stdin
// as the command's standard input ("" for none), returning stdout verbatim:
// for a deadline-bound read of history as stored, a batch of objects named on
// standard input among them. When ctx ends first, git is killed
// (isolatedGitContext) and the error wraps ctx's own; output past maxBytes is
// an error saying the cap was exceeded, never a truncated answer. A WaitDelay
// miss over a git that answered is asked once more, as RunLimitedContext asks
// it, and a second miss is ErrGitTimedOut.
func RunCappedBytesAsStoredContext(ctx context.Context, root, stdin string, maxBytes int, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	run := func() ([]byte, bool, error) {
		cmd := asStored(isolatedGitContext(ctx, root, args...))
		if stdin != "" {
			cmd.Stdin = strings.NewReader(stdin)
		}
		return runBoundedCmd(cmd, maxBytes)
	}
	out, overflowed, err := run()
	if errors.Is(err, exec.ErrWaitDelay) && ctx.Err() == nil {
		out, overflowed, err = run()
		if errors.Is(err, exec.ErrWaitDelay) && ctx.Err() == nil {
			return nil, fmt.Errorf("git %s: %w: its output was still open when the wait for it ran out, on a retry too (%w)",
				strings.Join(args, " "), ErrGitTimedOut, err)
		}
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), ctxErr)
	}
	if err != nil {
		return nil, err
	}
	if overflowed {
		return nil, fmt.Errorf("git %s: output exceeded the %d-byte cap; the answer would be truncated, and a truncated history is a wrong one",
			strings.Join(args, " "), maxBytes)
	}
	return out, nil
}

// EachAsStoredContext runs one git command as stored (asStored) under ctx,
// for a command whose output may run to any length (a listing of every ref
// under a prefix a fetch of a whole namespace can fill): each line of stdout
// is handed to each as it is read, without its newline, and none is kept, so
// no bound on the whole output applies. A line longer than maxLine bytes is
// handed cut to its first maxLine. git is killed when ctx ends, and ctx's
// error is then answered; a failure carries git's bounded stderr.
func EachAsStoredContext(ctx context.Context, root string, maxLine int, each func(line string), args ...string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	cmd := asStored(isolatedGitContext(ctx, root, args...))
	w := &lineSink{max: maxLine, each: each}
	e := &capWriter{remaining: 4096}
	cmd.Stdout, cmd.Stderr = w, e
	err := cmd.Run()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("git %s: %w", strings.Join(args, " "), ctxErr)
	}
	if err != nil {
		return fmt.Errorf("%w (stderr: %q)", err, strings.TrimSpace(string(e.buf)))
	}
	w.flush()
	return nil
}
