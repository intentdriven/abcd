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
// as stored, detached, in a directory of a random name it makes under abcd's
// home (preflightCheckoutsDir), never the system's shared temporary
// directory, through the runner the lane's own worktree add uses
// (worktreeAddGit): the repository's filters and post-checkout hook run as on
// the person's own checkout, and in a drive under the stage's wall. Before
// the preflight and again after it, the checkout is compared with the head's
// stored tree by content, through an index the drive builds itself in the
// environment the checkout was made in (gitutil.RunWithIndexAsCheckedOut), so
// the filters that wrote the checkout read it back. Before the preflight
// every untracked file outside the checkout's local tier (preflightOutputs),
// ignored ones included, is a difference. After it, an untracked file is a
// difference unless the head's own .gitignore files ignore it, read with no
// other ignore source, or it is under the local tier; and an untracked
// .gitignore or .gitattributes anywhere is a difference, ignored or not, since
// either changes how the checkout reads. Any difference holds the landing and
// marks nothing. A passing preflight's receipt is taken into the main
// checkout's local tier, where the pre-push hook and the landing find it; that
// the drive saw it pass is held in the drive's own memory (terminal.minted),
// never read back from a file. The checkout is removed once the preflight has
// ended, whatever its outcome.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/localtier"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// preflightCheckoutsDir is the folder under abcd's home, beside the worktree
// store, in which a drive makes each preflight's directory: the account's
// own, at abcdhome.DirMode.
const preflightCheckoutsDir = "preflight-checkouts"

// preflightCheckoutPrefix leads the random name of the directory a drive
// makes for one preflight's checkout and its scratch index.
const preflightCheckoutPrefix = "abcd-preflight-"

// preflightCheckoutHome is the home preflightCheckoutsDir is made under, the
// machine's (abcdhome.MachineHome); the package's tests point it at a fixture
// home.
var preflightCheckoutHome = abcdhome.MachineHome

// maxPreflightReceipt bounds the receipt a preflight leaves, which is a few
// lines.
const maxPreflightReceipt = 1 << 16

// preflightOutputs are the directories, relative to the checkout, under which
// the preflight is known to write: the checkout's local tier, where it keeps
// its receipt, its test reports and its scratch files. Before the preflight an
// untracked file anywhere else differs from the head; after it, one the
// head's own .gitignore files do not ignore does.
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
// checkout, inside tmp, the directory the drive made under
// preflightCheckoutsDir, which also holds the scratch index the comparison
// builds.
type preflightCheckout struct {
	tmp, dir string
}

// makePreflightCheckout adds a detached linked worktree of head, as stored,
// in a directory of a random name it makes (os.MkdirTemp, mode 0700) under
// preflightCheckoutsDir in abcd's home, which it makes at abcdhome.DirMode,
// refusing a level that is a link (fsutil.EnsureHomeScope); the local tier is
// made inside the checkout. A checkout it could not finish is removed before
// it returns the error.
func makePreflightCheckout(repoRoot, head string) (*preflightCheckout, error) {
	if !gitutil.IsFullSHA(head) {
		return nil, errors.New("the lane records no head to check out")
	}
	home, err := preflightCheckoutHome()
	if err != nil {
		return nil, fmt.Errorf("abcd's home could not be found: %w", err)
	}
	scope, err := fsutil.EnsureHomeScope(home, abcdhome.Rel(preflightCheckoutsDir), abcdhome.DirMode)
	if err != nil {
		return nil, fmt.Errorf("%s could not be made: %w", abcdhome.Display(preflightCheckoutsDir), err)
	}
	_ = scope.Close()
	tmp, err := os.MkdirTemp(abcdhome.Path(home, preflightCheckoutsDir), preflightCheckoutPrefix)
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
// does not; after says whether the preflight has run. A read that fails is
// said as a difference, with the drive's reason when its check of what it
// holds stopped the read.
//
// It builds its own index from the head's tree and stages the checkout's
// tracked files into it, which reads each through the repository's filters,
// in the environment the checkout was made in
// (gitutil.RunWithIndexAsCheckedOut), and lists what differs from the head.
// Then it lists the files the index does not hold. Before the preflight any
// such file outside preflightOutputs is a difference, ignored or not. After
// it, a file is a difference when it is outside preflightOutputs and the
// head's own .gitignore files do not ignore it, read with no other ignore
// source (no info/exclude, no core.excludesFile); and any untracked
// .gitignore or .gitattributes is, ignored or not, since either changes what
// the listing ignores or how the tracked files are read.
func (c *preflightCheckout) differs(head string, after bool) string {
	index := filepath.Join(c.tmp, "index")
	if err := os.Remove(index); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "the drive's own index could not be made afresh"
	}
	if _, err := gitutil.RunWithIndexAsCheckedOut(c.dir, index, maxGitOutput, "read-tree", head); err != nil {
		return readWhy("the head's tree could not be read into the drive's own index", err)
	}
	if _, err := gitutil.RunWithIndexAsCheckedOut(c.dir, index, maxGitOutput, "add", "--update", "--", "."); err != nil {
		return readWhy("the checkout's files could not be read into the drive's own index", err)
	}
	changed, err := gitutil.RunWithIndexAsCheckedOut(c.dir, index, maxGitOutput, "diff-index", "--cached", "--name-only", "-z", "--no-renames", head, "--")
	if err != nil {
		return readWhy("the checkout could not be compared with the head", err)
	}
	if first, _, _ := strings.Cut(string(changed), "\x00"); first != "" {
		return fmt.Sprintf("%q is not the head's", termsafe.Sanitize(first))
	}
	outside := func(name string) bool { return !preflightOutput(name) }
	if !after {
		return c.strayUntracked(index, outside)
	}
	if why := c.strayUntracked(index, outside, "--exclude-per-directory=.gitignore"); why != "" {
		return why
	}
	return c.strayUntracked(index, ignoreOrAttributesFile)
}

// strayUntracked lists the checkout's untracked files against index, with
// exclude as ls-files's exclude options (none lists every one, ignored ones
// included), and says the first that stray answers for, "" for none.
func (c *preflightCheckout) strayUntracked(index string, stray func(name string) bool, exclude ...string) string {
	var first string
	w := &nulNames{each: func(name string) {
		if first == "" && stray(name) {
			first = name
		}
	}}
	if err := gitutil.StreamWithIndexAsCheckedOut(c.dir, index, w, append([]string{"ls-files", "-z", "--others"}, exclude...)...); err != nil {
		return readWhy("the checkout's untracked files could not be listed", err)
	}
	w.flush()
	if first != "" {
		return fmt.Sprintf("it holds %q, which the head does not", termsafe.Sanitize(first))
	}
	return ""
}

// ignoreOrAttributesFile says whether a path's base name is .gitignore or
// .gitattributes, in any case, since a filesystem that folds case gives git
// the file under either spelling.
func ignoreOrAttributesFile(name string) bool {
	base := path.Base(name)
	return strings.EqualFold(base, ".gitignore") || strings.EqualFold(base, ".gitattributes")
}

// readWhy is a failed read of the comparison as a difference says it: what
// could not be read, and the drive's reason when its check of what it holds
// stopped git (gitutil.SetBeforeHookedGit).
func readWhy(what string, err error) string {
	if gitNotRun(err) {
		return what + ": " + heldWhy(err)
	}
	return what
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
// (the file's header). It returns why the landing is held, "" for no hold;
// whether the preflight ran and exited 0 (passed), with or without a hold
// after it; and the preflight's own error when it ran and failed. A passing
// preflight whose checkout still holds the head's content, and whose receipt
// was taken, is the one outcome with neither a hold nor an error.
func (t *terminal) ownPreflight(ctx context.Context, runID string, lane Lane) (held string, passed bool, perr error) {
	if err := heldBeforeGit(t.repoRoot, t.o.Options.remote); err != nil {
		return heldWhy(err) + ", so the lane's preflight is not run", false, nil
	}
	c, err := makePreflightCheckout(t.repoRoot, lane.HeadSHA)
	if err != nil {
		return "the drive's own checkout of the head " + shortSHA(lane.HeadSHA) + " could not be made, so its preflight is not run: " + heldWhy(err), false, nil
	}
	defer func() {
		if err := c.remove(t.repoRoot); err != nil {
			t.say("%s %s: the drive's own checkout for the preflight could not be removed: %s", runID, lane.ID, fsutil.RedactHome(err.Error()))
		}
	}()
	if why := c.differs(lane.HeadSHA, false); why != "" {
		return "the drive's own checkout does not hold the head " + shortSHA(lane.HeadSHA) + " (" + why + "), so its preflight is not run", false, nil
	}
	t.say("%s %s: running the preflight in the drive's own checkout of %s", runID, lane.ID, shortSHA(lane.HeadSHA))
	if perr := t.preflight(ctx, c.dir); perr != nil {
		return "", false, perr
	}
	if why := c.differs(lane.HeadSHA, true); why != "" {
		return "the drive's own checkout no longer held the head " + shortSHA(lane.HeadSHA) + " once its preflight ran (" + why + "), so no receipt it left is taken", true, nil
	}
	if err := c.takeReceipt(t.repoRoot, lane.HeadSHA); err != nil {
		return "its preflight exited 0 in the drive's own checkout, and left no receipt for the head " + shortSHA(lane.HeadSHA) + " that could be taken: " + fsutil.RedactHome(err.Error()), true, nil
	}
	return "", true, nil
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
