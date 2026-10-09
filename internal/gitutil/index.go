package gitutil

import (
	"fmt"
	"path/filepath"
	"strings"
)

// RunWithIndex runs one isolated git command in root against the index file
// index instead of the repository's own, and returns its stdout verbatim. It
// is how a caller stages or diffs a working tree without disturbing what the
// person staged: `read-tree HEAD` and `add -A` into a scratch index, then a
// diff of it, leave the repository's index exactly as it was. The environment
// is the isolated one (gitEnv), which strips any inherited GIT_INDEX_FILE, with
// the caller's appended after it, so it is the one git reads. index must be an
// absolute path, kept outside root's working tree (a scratch index inside the
// tree would be staged by its own `add -A`). Output past maxBytes is an error,
// never a truncated answer: a cut-short patch is a wrong one.
func RunWithIndex(root, index string, maxBytes int, args ...string) ([]byte, error) {
	if !filepath.IsAbs(index) {
		return nil, fmt.Errorf("the scratch index %q is not an absolute path", index)
	}
	cmd := isolatedGit(root, args...)
	cmd.Env = append(cmd.Env, "GIT_INDEX_FILE="+index)
	out, overflowed, err := runBoundedCmd(cmd, maxBytes)
	if err != nil {
		return nil, err
	}
	if overflowed {
		return nil, fmt.Errorf("git %s: output exceeded the %d-byte cap; the answer would be truncated",
			strings.Join(args, " "), maxBytes)
	}
	return out, nil
}
