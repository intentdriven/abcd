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
