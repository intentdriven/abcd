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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// TierRelDir is the local-ephemeral tier of a checkout, repo-relative and
// slash-separated: gitignored, per worktree.
const TierRelDir = ".abcd/.work.local"

// ReceiptsRelDir is where the preflight mints its receipts, one file named by
// the full commit id, in any worktree of the repository
// (scripts/preflight-receipt.sh). A receipt is a convenience check, not a
// security boundary: CI re-runs every gate on the pushed commit and is the
// authority, and the receipt says only that a preflight passed on a tree that
// matched the commit.
const ReceiptsRelDir = TierRelDir + "/preflight-receipts"

// Bounds on what the landing runs.
const (
	// ForgeTimeout is the longest one forge client call may run.
	ForgeTimeout = 2 * time.Minute
	// MaxOutput bounds what a forge client call may write to stdout.
	MaxOutput = 1 << 20
	// maxStderr bounds how much of a failed command's stderr its error carries.
	maxStderr = 2048
)

// Receipt finds the preflight receipt for sha in any worktree git lists for
// the repository at repoRoot, as the pre-push hook's check does, and returns
// its path, or "" when none names sha. A receipt must be a regular file, never
// a symlink.
func Receipt(repoRoot, sha string) (string, error) {
	wts, err := gitutil.ListWorktrees(repoRoot, gitutil.MaxWorktreeListing)
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

// GitFunc runs one networked git command (a fetch or a push) from dir:
// gitutil.Net, or a test's seam.
type GitFunc func(dir string, args ...string) (string, error)

// FetchError is a fetch of the default branch git refused.
type FetchError struct {
	Def string
	Err error
}

func (e *FetchError) Error() string {
	return "git could not fetch " + e.Def + " from " + Remote + ": " + e.Err.Error()
}
func (e *FetchError) Unwrap() error { return e.Err }

// FetchDefault fetches the default branch def from origin into its
// remote-tracking ref through git, and returns the ref. A refused fetch is a
// *FetchError.
func FetchDefault(git GitFunc, dir, def string) (string, error) {
	tracking := "refs/remotes/" + Remote + "/" + def
	if _, err := git(dir, "fetch", "--quiet", "--no-tags", Remote, "+refs/heads/"+def+":"+tracking); err != nil {
		return tracking, &FetchError{Def: def, Err: err}
	}
	return tracking, nil
}

// Landed fetches the default branch def and reports whether sha is on it as
// origin holds it (an ancestor of, or equal to, its tip): the one ancestry
// check by which a landing proves a head merged, the implement loop's and
// `abcd land`'s. A squash or rebase merge leaves no such ancestry; MergedAt is
// the forge's half of the proof for it. A refused fetch is a *FetchError; any
// other error is git unable to answer, which is not a no.
func Landed(git GitFunc, dir, def, sha string) (tracking string, on bool, err error) {
	if tracking, err = FetchDefault(git, dir, def); err != nil {
		return tracking, false, err
	}
	on, err = gitutil.IsAncestor(dir, sha, tracking)
	return tracking, on, err
}

// maxMergedListed bounds the forge's listing of a branch's merged pull
// requests: a fork's from a branch of the same name are listed too.
const maxMergedListed = 20

// mergedPR is a merged pull request as the forge lists it.
type mergedPR struct {
	Number            int    `json:"number"`
	HeadRefOid        string `json:"headRefOid"`
	IsCrossRepository bool   `json:"isCrossRepository"`
	MergedAt          string `json:"mergedAt"`
}

// MergedAt asks the forge, through ask (the forge client's arguments), for a
// pull request from this repository's own branch that MERGED with sha as its
// head, and returns its number, or 0 when none did: the proof a squash or a
// rebase merge leaves where the ancestor check (Landed) finds none, since the
// merged commit on the default branch is not sha. A pull request from another
// repository, or one merged at another head, proves nothing about sha. An
// error is the forge unable to answer, which is not a no.
func MergedAt(ask func(args ...string) (string, error), branch, sha string) (int, error) {
	out, err := ask("pr", "list", "--head="+branch, "--state=merged",
		"--json=number,headRefOid,isCrossRepository,mergedAt", fmt.Sprintf("--limit=%d", maxMergedListed))
	if err != nil {
		return 0, err
	}
	var prs []mergedPR
	if json.Unmarshal([]byte(out), &prs) != nil {
		return 0, errors.New("the forge's listing of " + branch + "'s merged pull requests does not parse")
	}
	for _, pr := range prs {
		if !pr.IsCrossRepository && pr.Number > 0 && pr.HeadRefOid == sha {
			return pr.Number, nil
		}
	}
	return 0, nil
}
