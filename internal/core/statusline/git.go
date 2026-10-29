package statusline

// The status line's git: every question the row asks git, asked under a
// deadline.
//
// The harness waits on the status verb on every refresh, so a git that hangs
// — a checkout on a stalled network mount, a wedged object store — would
// freeze the person's status line for as long as git takes
// (iss-2610050556383525). Every question is asked through
// gitutil.RunLimitedContext, the one deadline-bound git primitive: git runs
// under gitutil's isolation (hooks and the filesystem monitor off, global and
// system config neutralised) and is KILLED when the context ends rather than
// abandoned, because an abandoned git outlives the verb and one is left behind
// on every refresh for as long as the hang lasts.

import (
	"context"
	"errors"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// gitOutputCap bounds what one answer may buffer. The row asks for one path
// or one ref name, so the cap is generous and a hostile repository still
// cannot make the status line allocate.
const gitOutputCap = 4096

// gitQuery runs one read-only git question under root and returns its
// trimmed stdout. It returns ctx's error, wrapped, when the context ended
// before git answered, so a caller can tell "git said no" from "git did not
// answer in time".
func gitQuery(ctx context.Context, root string, args ...string) (string, error) {
	out, err := gitutil.RunLimitedContext(ctx, root, gitOutputCap, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// ErrNoCheckout is CheckoutRoot's answer for a directory git places in no
// checkout, or would not answer for.
var ErrNoCheckout = errors.New("statusline: no checkout git will answer for")

// CheckoutRoot is the status line's checkout resolution: git's toplevel for
// cwd through gitutil.ToplevelContext, which holds the answer to the shape
// gitutil.Toplevel holds it to, under ctx. It answers the one question the
// verb asks of gitutil.CheckoutRoot — is there a root, and which — and differs
// only in being bounded: a context that ends first is returned as its own
// error (context.DeadlineExceeded, wrapped), never as ErrNoCheckout, so the
// verb does not mistake a slow git for "not a repository" and run the person's
// previous command over a checkout abcd manages.
func CheckoutRoot(ctx context.Context, cwd string) (string, error) {
	top, err := gitutil.ToplevelContext(ctx, cwd)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return "", ctxErr
	}
	if err != nil {
		return "", ErrNoCheckout
	}
	return top, nil
}
