// Package landing is the landing of a branch whose preflight passed: the gate
// (a preflight receipt naming the branch's head), a plain push of the branch
// to origin, and the merge the repository's own rules allow. `abcd land` is its
// front door for any branch; the implement loop's land stage
// (internal/core/implement/loop) shares its lane-free primitives, so the
// receipt lookup, the bounded runner, the forge client's invocation and the
// ruleset mirror's reading are spelled once.
//
// The package is transport-agnostic: it never writes to stdout. What a step
// did comes back as a Result for the front door to render.
package landing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// ReceiptsRelDir is where the preflight mints its receipts, one file named by
// the full commit id, in any worktree of the repository
// (scripts/preflight-receipt.sh). A receipt is a convenience check, not a
// security boundary: CI re-runs every gate on the pushed commit and is the
// authority, and the receipt says only that a preflight passed on a tree that
// matched the commit.
const ReceiptsRelDir = ".abcd/.work.local/preflight-receipts"

// Bounds on what the landing runs.
const (
	// ForgeTimeout is the longest one forge client call may run.
	ForgeTimeout = 2 * time.Minute
	// MaxOutput bounds what a forge client call may write to stdout.
	MaxOutput = 1 << 20
	// maxStderr bounds how much of a failed command's stderr its error carries.
	maxStderr = 2048
	// maxWorktreeListing bounds `git worktree list`.
	maxWorktreeListing = 16 << 20
)

// Receipt finds the preflight receipt for sha in any worktree git lists for
// the repository at repoRoot, as the pre-push hook's check does, and returns
// its path, or "" when none names sha. A receipt must be a regular file, never
// a symlink.
func Receipt(repoRoot, sha string) (string, error) {
	wts, err := gitutil.ListWorktrees(repoRoot, maxWorktreeListing)
	if err != nil {
		return "", fmt.Errorf("listing the repository's worktrees: %w", err)
	}
	for _, wt := range wts {
		if wt.Bare {
			continue
		}
		p := filepath.Join(wt.Path, filepath.FromSlash(ReceiptsRelDir), sha)
		if fi, err := os.Lstat(p); err == nil && fi.Mode().IsRegular() {
			return p, nil
		}
	}
	return "", nil
}

// RunCapped runs cmd, returning its stdout within MaxOutput and, on a failure,
// an error naming what and carrying its bounded stderr.
func RunCapped(cmd *exec.Cmd, what string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if len(msg) > maxStderr {
			msg = msg[:maxStderr]
		}
		return "", fmt.Errorf("%s: %v (%s)", what, err, msg)
	}
	if stdout.Len() > MaxOutput {
		return "", fmt.Errorf("%s wrote more than %d bytes", what, MaxOutput)
	}
	return stdout.String(), nil
}

// ErrNoForgeClient is GH's answer when the forge client is not installed.
var ErrNoForgeClient = errors.New("the forge client `gh` is not installed")

// GH runs the forge client the repository already uses (`gh`) with args from
// dir, bounded by ForgeTimeout and MaxOutput, with no prompt and no update
// notice. Its environment is scrubbed as the networked git runner's is
// (gitutil.ScrubbedEnv): an inherited GIT_DIR or an injected GIT_CONFIG_*
// cannot aim the client's own git reads at another repository. Every value
// that comes from a record or a branch rides in its flag's own argument
// (--flag=value) at the caller, so none can be read as a flag of its own.
func GH(dir string, args ...string) (string, error) {
	bin, err := exec.LookPath("gh")
	if err != nil {
		return "", ErrNoForgeClient
	}
	ctx, cancel := context.WithTimeout(context.Background(), ForgeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	cmd.Env = append(gitutil.ScrubbedEnv(), "GH_PROMPT_DISABLED=1", "GH_NO_UPDATE_NOTIFIER=1")
	return RunCapped(cmd, "gh "+strings.Join(args[:min(2, len(args))], " "))
}
