package gitutil

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/procgroup"
)

// RunWithIndex runs one isolated git command in root against the index file
// index instead of the repository's own, and returns its stdout verbatim. It
// is how a caller stages or diffs a working tree without disturbing what the
// person staged: `read-tree HEAD` and `add -A` into a scratch index, then a
// diff of it, leave the repository's index exactly as it was. The environment
// is the isolated one (gitEnv), which strips any inherited GIT_INDEX_FILE, with
// the caller's appended after it, so it is the one git reads. index must be an
// absolute path, kept outside root's working tree (a scratch index inside the
// tree would be staged by its own `add -A`). The scratch index is written
// whole, never split (core.splitIndex off), so a repository set to split its
// index gains no shared index file in its git directory from it. Output past
// maxBytes is an error, never a truncated answer: a cut-short patch is a
// wrong one.
func RunWithIndex(root, index string, maxBytes int, args ...string) ([]byte, error) {
	cmd, err := indexCommand(root, index, args)
	if err != nil {
		return nil, err
	}
	out, overflowed, err := runBoundedCmd(context.Background(), cmd, maxBytes)
	if err != nil {
		return nil, err
	}
	if overflowed {
		return nil, fmt.Errorf("git %s: output exceeded the %d-byte cap; the answer would be truncated",
			strings.Join(args, " "), maxBytes)
	}
	return out, nil
}

// StreamWithIndex is RunWithIndex for an answer the caller reads as git
// writes it, such as a listing of every entry it hashes, so no cap is needed:
// git's stdout goes to w, and its stderr, bounded, into the error.
func StreamWithIndex(root, index string, w io.Writer, args ...string) error {
	cmd, err := indexCommand(root, index, args)
	if err != nil {
		return err
	}
	e := &capWriter{remaining: 4096}
	if err := procgroup.RunToEOF(context.Background(), cmd, w, e); err != nil {
		return fmt.Errorf("git %s: %w (stderr: %q)", treeSub(args), err, strings.TrimSpace(string(e.buf)))
	}
	return nil
}

// indexCommand is the isolated git command RunWithIndex and StreamWithIndex
// run on the scratch index, with core.splitIndex off.
func indexCommand(root, index string, args []string) (*exec.Cmd, error) {
	if !filepath.IsAbs(index) {
		return nil, fmt.Errorf("the scratch index %q is not an absolute path", index)
	}
	cmd := isolatedGit(root, append([]string{"-c", "core.splitIndex=false"}, args...)...)
	cmd.Env = append(cmd.Env, "GIT_INDEX_FILE="+index)
	return cmd, nil
}

// RunWithIndexAsCheckedOut is RunWithIndex in the environment a checkout made
// through RunCommitCommandBoundedAsStored was made in, for a comparison of
// that checkout with the commit it was made from: the person's global and
// system config, or in a drive the stage's (SetHookedEnv), so the content
// filters, attributes and line-ending settings the checkout was written under
// read it back; objects read as stored (AsStoredEnv); the program-starting
// knobs, the hooks among them, pinned off (ExecPins), as WorkingTreeCommand
// pins them; GIT_NO_LAZY_FETCH=1 and the lazy-fetch floor guard. Its content
// filters run, so the check SetBeforeHookedGit set is asked first, and git
// does not run on its error.
func RunWithIndexAsCheckedOut(root, index string, maxBytes int, args ...string) ([]byte, error) {
	cmd, err := checkedOutIndexCommand(root, index, args)
	if err != nil {
		return nil, err
	}
	out, overflowed, err := runBoundedCmd(context.Background(), cmd, maxBytes)
	if err != nil {
		return nil, err
	}
	if overflowed {
		return nil, fmt.Errorf("git %s: output exceeded the %d-byte cap; the answer would be truncated",
			strings.Join(args, " "), maxBytes)
	}
	return out, nil
}

// StreamWithIndexAsCheckedOut is StreamWithIndex in RunWithIndexAsCheckedOut's
// environment.
func StreamWithIndexAsCheckedOut(root, index string, w io.Writer, args ...string) error {
	cmd, err := checkedOutIndexCommand(root, index, args)
	if err != nil {
		return err
	}
	e := &capWriter{remaining: 4096}
	if err := procgroup.RunToEOF(context.Background(), cmd, w, e); err != nil {
		return fmt.Errorf("git %s: %w (stderr: %q)", treeSub(args), err, strings.TrimSpace(string(e.buf)))
	}
	return nil
}

// checkedOutIndexCommand is the command RunWithIndexAsCheckedOut and
// StreamWithIndexAsCheckedOut run on the scratch index, with core.splitIndex
// off. Its environment is RunCommitCommandBoundedAsStored's
// (WorkingTreeCommitCommand's, narrowed by SetHookedEnv, then AsStoredEnv),
// with GIT_INDEX_FILE last; its pins are WorkingTreeCommand's.
func checkedOutIndexCommand(root, index string, args []string) (*exec.Cmd, error) {
	if !filepath.IsAbs(index) {
		return nil, fmt.Errorf("the scratch index %q is not an absolute path", index)
	}
	cmd := filteredCommand(context.Background(), root, append([]string{"-c", "core.splitIndex=false"}, args...))
	cmd.Env = append(AsStoredEnv(narrowHooked(cmd.Env)), "GIT_INDEX_FILE="+index)
	return cmd, nil
}
