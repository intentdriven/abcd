package loop

// preflightcheckout.go is where a drive's own preflight runs: in a checkout the
// drive makes of the lane's stored head, never in the lane's worktree. The
// lane's worktree is where its stages ran, so what is in it, its index and its
// ignored files among them, is not the drive's to trust; the drive reads it
// for nothing but the lane's branch, and what the lane changed is read from
// stored commits (the landing's gate check).
//
// Before the preflight the drive reads again what it holds (heldBeforeGit:
// the settings that can name a program or change what a checkout writes, the
// hooks directories and info/attributes), so the checkout runs under the
// settings the drive started with. It then adds a linked worktree of the head
// as stored, detached, in a directory it makes, through the runner the lane's
// own worktree add uses (worktreeAddGit): the repository's filters and
// post-checkout hook run as on the person's own checkout, and in a drive under
// the stage's wall. Before the preflight and again after it, the checkout is
// compared with the head's stored tree by content, through an index the drive
// builds itself, and every untracked file, ignored ones included, is listed
// against the paths the preflight is known to write (preflightOutputs). Any
// difference holds the landing and marks nothing. A passing preflight's
// receipt is taken into the main checkout's local tier, where the pre-push
// hook and the landing find it; that the drive saw it pass is held in the
// drive's own memory (terminal.minted), never read back from a file. The
// checkout is removed once the preflight has ended, whatever its outcome.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/localtier"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// preflightCheckoutPrefix names the directory a drive makes for one
// preflight's checkout and its scratch index.
const preflightCheckoutPrefix = "abcd-preflight-"

// maxPreflightReceipt bounds the receipt a preflight leaves, which is a few
// lines.
const maxPreflightReceipt = 1 << 16

// preflightOutputs are the directories, relative to the checkout, under which
// the preflight is known to write: the checkout's local tier, where it keeps
// its receipt, its test reports and its scratch files. An untracked file
// anywhere else differs from the head.
var preflightOutputs = []string{landing.TierRelDir + "/"}

// preflightOutput says whether an untracked path in the checkout is under one
// of preflightOutputs.
func preflightOutput(name string) bool {
	for _, p := range preflightOutputs {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// preflightCheckout is one checkout a drive made for a preflight: dir, the
// checkout, inside tmp, the directory the drive made, which also holds the
// scratch index the comparison builds.
type preflightCheckout struct {
	tmp, dir string
}

// makePreflightCheckout adds a detached linked worktree of head, as stored,
// in a directory it makes, with the local tier made inside it. A checkout it
// could not finish is removed before it returns the error.
func makePreflightCheckout(repoRoot, head string) (*preflightCheckout, error) {
	if !gitutil.IsFullSHA(head) {
		return nil, errors.New("the lane records no head to check out")
	}
	tmp, err := os.MkdirTemp("", preflightCheckoutPrefix)
	if err != nil {
		return nil, err
	}
	c := &preflightCheckout{tmp: tmp, dir: filepath.Join(tmp, "checkout")}
	if _, err := worktreeAddGit(repoRoot, "worktree", "add", "--detach", "--", c.dir, head); err != nil {
		_ = c.remove(repoRoot)
		return nil, err
	}
	root, err := os.OpenRoot(c.dir)
	if err == nil {
		err = root.MkdirAll(landing.TierRelDir, 0o700)
		_ = root.Close()
	}
	if err != nil {
		_ = c.remove(repoRoot)
		return nil, fmt.Errorf("the checkout's local tier could not be made: %w", err)
	}
	return c, nil
}

// remove takes the checkout out of the repository's worktrees and deletes the
// directory the drive made. git is asked to forget a worktree whose removal
// did not finish.
func (c *preflightCheckout) remove(repoRoot string) error {
	var errs []error
	if _, err := os.Lstat(c.dir); err == nil {
		if _, err := gitutil.Run(repoRoot, "worktree", "remove", "--force", "--", c.dir); err != nil {
			errs = append(errs, err)
		}
	}
	if err := os.RemoveAll(c.tmp); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		if _, err := gitutil.Run(repoRoot, "worktree", "prune"); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// differs says how the checkout differs from head's stored tree, "" when it
// holds that tree's content and nothing beside it outside preflightOutputs. It
// builds its own index from the head's tree, with replace refs off, stages the
// checkout's tracked files into it, which reads each through the repository's
// filters, and lists what differs from the head; then it lists every file the
// index does not hold, ignored ones included. A read that fails is said as a
// difference.
func (c *preflightCheckout) differs(head string) string {
	index := filepath.Join(c.tmp, "index")
	if err := os.Remove(index); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "the drive's own index could not be made afresh"
	}
	if _, err := gitutil.RunWithIndex(c.dir, index, maxGitOutput, "--no-replace-objects", "read-tree", head); err != nil {
		return "the head's tree could not be read into the drive's own index"
	}
	if _, err := gitutil.RunWithIndex(c.dir, index, maxGitOutput, "--no-replace-objects", "add", "--update", "--", "."); err != nil {
		return "the checkout's files could not be read into the drive's own index"
	}
	changed, err := gitutil.RunWithIndex(c.dir, index, maxGitOutput, "--no-replace-objects", "diff-index", "--cached", "--name-only", "-z", "--no-renames", head, "--")
	if err != nil {
		return "the checkout could not be compared with the head"
	}
	if first, _, _ := strings.Cut(string(changed), "\x00"); first != "" {
		return fmt.Sprintf("%q is not the head's", termsafe.Sanitize(first))
	}
	var stray string
	w := &nulNames{each: func(name string) {
		if stray == "" && !preflightOutput(name) {
			stray = name
		}
	}}
	if err := gitutil.StreamWithIndex(c.dir, index, w, "ls-files", "-z", "--others"); err != nil {
		return "the checkout's untracked files could not be listed"
	}
	w.flush()
	if stray != "" {
		return fmt.Sprintf("it holds %q, which the head does not", termsafe.Sanitize(stray))
	}
	return ""
}

// takeReceipt copies the receipt the preflight left in the checkout's local
// tier for head into the main checkout's, where the pre-push hook and the
// landing look for it: a regular file the checkout's git does not track,
// naming head on its first line.
func (c *preflightCheckout) takeReceipt(repoRoot, head string) error {
	rel := landing.ReceiptsRelDir + "/" + head
	from, err := os.OpenRoot(c.dir)
	if err != nil {
		return err
	}
	data, err := localtier.ReadInRoot(from, rel, maxPreflightReceipt)
	_ = from.Close()
	if err != nil {
		return err
	}
	if !bytes.HasPrefix(data, []byte("commit "+head+"\n")) {
		return errors.New("the receipt does not name the head on its first line")
	}
	to, err := os.OpenRoot(repoRoot)
	if err != nil {
		return err
	}
	defer to.Close()
	if err := to.MkdirAll(landing.ReceiptsRelDir, 0o700); err != nil {
		return err
	}
	return fsutil.WriteFileAtomicInRoot(to, rel, data, 0o600)
}

// nulNames hands each NUL-terminated name written to it to each, as it is
// read, keeping none but the one being read.
type nulNames struct {
	each func(string)
	part []byte
}

func (n *nulNames) Write(p []byte) (int, error) {
	for _, b := range p {
		if b == 0 {
			n.each(string(n.part))
			n.part = n.part[:0]
			continue
		}
		n.part = append(n.part, b)
	}
	return len(p), nil
}

// flush hands on a last name git did not end.
func (n *nulNames) flush() {
	if len(n.part) > 0 {
		n.each(string(n.part))
		n.part = n.part[:0]
	}
}

// ownPreflight runs a lane's preflight in the drive's own checkout of its head
// (the file's header). It returns why the landing is held, "" for no hold, and
// the preflight's own error when it ran and failed. A passing preflight whose
// checkout still holds the head's content, and whose receipt was taken, is
// the one outcome with neither.
func (t *terminal) ownPreflight(ctx context.Context, runID string, lane Lane) (held string, perr error) {
	if err := heldBeforeGit(t.repoRoot, t.o.Options.remote); err != nil {
		return heldWhy(err) + ", so the lane's preflight is not run", nil
	}
	c, err := makePreflightCheckout(t.repoRoot, lane.HeadSHA)
	if err != nil {
		return "the drive's own checkout of the head " + shortSHA(lane.HeadSHA) + " could not be made, so its preflight is not run: " + heldWhy(err), nil
	}
	defer func() {
		if err := c.remove(t.repoRoot); err != nil {
			t.say("%s %s: the drive's own checkout for the preflight could not be removed: %s", runID, lane.ID, err.Error())
		}
	}()
	if why := c.differs(lane.HeadSHA); why != "" {
		return "the drive's own checkout does not hold the head " + shortSHA(lane.HeadSHA) + " (" + why + "), so its preflight is not run", nil
	}
	t.say("%s %s: running the preflight in the drive's own checkout of %s", runID, lane.ID, shortSHA(lane.HeadSHA))
	if perr := t.preflight(ctx, c.dir); perr != nil {
		return "", perr
	}
	if why := c.differs(lane.HeadSHA); why != "" {
		return "the drive's own checkout no longer held the head " + shortSHA(lane.HeadSHA) + " once its preflight ran (" + why + "), so no receipt it left is taken", nil
	}
	if err := c.takeReceipt(t.repoRoot, lane.HeadSHA); err != nil {
		return "its preflight exited 0 in the drive's own checkout, and left no receipt for the head " + shortSHA(lane.HeadSHA) + " that could be taken: " + fsutil.RedactHome(err.Error()), nil
	}
	return "", nil
}

// heldWhy is an error's reason as a hold says it: a refusal's reason alone,
// any other error's text.
func heldWhy(err error) string {
	if r, ok := heldStopOf(err); ok {
		return r.Reason
	}
	if r, ok := AsRefusal(err); ok {
		return r.Reason
	}
	return fsutil.RedactHome(err.Error())
}
