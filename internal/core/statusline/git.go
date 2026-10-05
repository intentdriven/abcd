package statusline

// The status line's git: every question the row asks git, asked under a
// deadline.
//
// The harness waits on the status verb on every refresh, so a git that hangs
// — a checkout on a stalled network mount, a wedged object store — would
// freeze the person's status line for as long as git takes
// (iss-2610050556383525). gitutil.Run has no deadline and no context seam, so
// the row's two git questions (where the checkout's root is, and which branch
// it is on) are asked here instead, under the caller's context, and the git
// is KILLED when the context ends rather than abandoned: an abandoned git
// outlives the verb, and one is left behind on every refresh for as long as
// the hang lasts.
//
// The isolation is gitutil's own, spelled through its exported pieces: the
// environment is gitutil.IsolatedEnv (repo-selection and config-injection
// variables scrubbed, global and system config neutralised, no background
// maintenance, C locale), and the command line turns hooks and the
// filesystem monitor off and asks for verbatim paths, as gitutil's
// isolatedGit does. A question this package does not ask stays on gitutil.

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// gitOutputCap bounds what one answer may buffer. The row asks for one path
// or one ref name, so the cap is generous and a hostile repository still
// cannot make the status line allocate.
const gitOutputCap = 4096

// gitWaitDelay is how long the kill waits for git's output pipe to close
// once the context has ended, so a process git started that kept the pipe
// open cannot hold the verb past its deadline.
const gitWaitDelay = 50 * time.Millisecond

// cappedBuffer keeps at most limit bytes and never fails a write, so a chatty
// git is neither blocked nor allowed to grow the buffer.
type cappedBuffer struct {
	buf   []byte
	limit int
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	if room := c.limit - len(c.buf); room > 0 {
		if len(p) < room {
			room = len(p)
		}
		c.buf = append(c.buf, p[:room]...)
	}
	return len(p), nil
}

// gitQuery runs one read-only git question under root and returns its
// trimmed stdout. It returns ctx's error, wrapped, when the context ended
// before git answered, so a caller can tell "git said no" from "git did not
// answer in time".
func gitQuery(ctx context.Context, root string, args ...string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	full := append([]string{
		"-c", "core.hooksPath=/dev/null",
		"-c", "core.fsmonitor=false",
		"-c", "core.quotePath=false",
		"-C", root,
	}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Env = gitutil.IsolatedEnv()
	cmd.WaitDelay = gitWaitDelay
	out := &cappedBuffer{limit: gitOutputCap}
	cmd.Stdout = out
	err := cmd.Run()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return "", fmt.Errorf("git %s: %w", args[0], ctxErr)
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out.buf)), nil
}

// ErrNoCheckout is CheckoutRoot's answer for a directory git places in no
// checkout, or would not answer for.
var ErrNoCheckout = errors.New("statusline: no checkout git will answer for")

// CheckoutRoot is the status line's checkout resolution: git's toplevel for
// cwd, held to gitutil.ToplevelShaped exactly as gitutil.Toplevel holds it,
// under ctx. It answers the one question the verb asks of
// gitutil.CheckoutRoot — is there a root, and which — and differs only in
// being bounded: a context that ends first is returned as its own error
// (context.DeadlineExceeded, wrapped), never as ErrNoCheckout, so the verb
// does not mistake a slow git for "not a repository" and run the person's
// previous command over a checkout abcd manages.
func CheckoutRoot(ctx context.Context, cwd string) (string, error) {
	top, err := gitQuery(ctx, cwd, "rev-parse", "--show-toplevel")
	if ctxErr := ctx.Err(); ctxErr != nil {
		return "", ctxErr
	}
	if err != nil || !gitutil.ToplevelShaped(cwd, top) {
		return "", ErrNoCheckout
	}
	return top, nil
}
