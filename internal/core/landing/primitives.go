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
	"regexp"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/tools"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/localtier"
	"github.com/intentdriven/abcd/internal/procgroup"
	"github.com/intentdriven/abcd/internal/termsafe"
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
//
// It stays on the deadline-free finder rather than calling ReceiptContext
// with a Background context, which the hook git audit would count as a new
// deadline-free call (dcn-2611101237202451).
func Receipt(repoRoot, sha string) (string, error) { return findReceipt(repoRoot, ReceiptsRelDir, sha) }

// ReceiptContext is Receipt with its git calls (the worktree listing and each
// receipt's tracked check) bound to ctx, for a reader on a deadline, the away
// mode's stop check among them: a ctx already ended starts no git and
// refuses with its own error, and one that ends while git runs kills it.
func ReceiptContext(ctx context.Context, repoRoot, sha string) (string, error) {
	return findReceiptContext(ctx, repoRoot, ReceiptsRelDir, sha)
}

// findReceipt finds the receipt file named sha under rel in any worktree git
// lists for the repository at repoRoot: a regular file, never a symlink, and
// that worktree's own: one git tracks came with a commit, and is refused
// (localtier.Untracked) rather than counted. Preflight and review receipts
// are found the same way, except that a review receipt or a recorded approval
// in a drive lane's worktree is passed over (laneBlind), and every one is
// when repoRoot is itself a drive lane's worktree.
func findReceipt(repoRoot, rel, sha string) (string, error) {
	return firstReceipt(findReceipts(repoRoot, rel, sha))
}

// findReceiptContext is findReceipt bound to ctx (findReceiptsContext).
func findReceiptContext(ctx context.Context, repoRoot, rel, sha string) (string, error) {
	return firstReceipt(findReceiptsContext(ctx, repoRoot, rel, sha))
}

// firstReceipt is the first worktree's receipt of found that may count: its
// path, or why it does not count, or "" when none was found. One in a drive
// lane's worktree (foundReceipt.lane) is passed over.
func firstReceipt(found []foundReceipt, err error) (string, error) {
	if err != nil {
		return "", err
	}
	for _, f := range found {
		if f.lane {
			continue
		}
		if f.err != nil {
			return "", f.err
		}
		return f.path, nil
	}
	return "", nil
}

// foundReceipt is one worktree's receipt: its path, or why it does not count.
type foundReceipt struct {
	path string
	err  error
	// lane marks a review receipt or a recorded approval found in a drive
	// lane's worktree (insideDriveLane), which never counts; it is kept so a
	// reader can say one was passed over. Its tracked check is not asked.
	lane bool
}

// laneBlind reports whether a receipt under rel is passed over in a drive
// lane's worktree: a review receipt or a recorded approval is, since the
// unattended drive runs model stages in that worktree and they can write any
// file there. A preflight receipt is not: the drive's preflight mints it in
// the lane's worktree, and the loop checks it is its own (landPush).
func laneBlind(rel string) bool {
	return rel == ReviewReceiptsRelDir || rel == ApprovalsRelDir
}

// driveLaneName is the name the implement loop gives a lane's worktree in the
// store, <run-id>-<lane-id> (loop.laneName: run-<16 digits>-lane-<n>).
var driveLaneName = regexp.MustCompile(`^run-[0-9]{16}-lane-[1-9][0-9]{0,3}$`)

// DriveLaneWorktree reports whether path names a drive lane's worktree by
// its last element alone: a lane's name as the implement loop mints it. It
// neither resolves path nor looks at its ancestors; the merge gate asks
// insideDriveLane, which does both.
func DriveLaneWorktree(path string) bool {
	return driveLaneName.MatchString(filepath.Base(filepath.Clean(path)))
}

// insideDriveLane reports whether path is a drive lane's worktree or lies
// anywhere under one: an element of the path as given, made absolute, or of
// the path with its symlinks resolved (fsutil.RealExistingPath) is a lane's
// name (DriveLaneWorktree). The drive's lane worktrees and everything under
// them are the drive's, so the merge gate reads no review receipt or recorded
// approval there, whatever path names them: a worktree listed under another
// name that resolves into a lane, a directory inside a lane, or a repository
// nested in one. Read from a checkout that is not in a lane, the paths git
// lists come from the repository's own worktree records, outside every lane.
// Read from a checkout in a lane they do not: git finds the repository
// through the lane's own .git file, which sits inside the lane, so the gate
// counts no review receipt or recorded approval at all from there
// (fromDriveLane). A worktree a person names that way is one the drive would
// adopt as a lane's, and is read as one.
func insideDriveLane(path string) bool {
	if path == "" {
		return false
	}
	given, err := filepath.Abs(path)
	if err != nil {
		given = filepath.Clean(path)
	}
	for _, p := range []string{given, fsutil.RealExistingPath(path)} {
		for {
			if DriveLaneWorktree(p) {
				return true
			}
			up := filepath.Dir(p)
			if up == p {
				break
			}
			p = up
		}
	}
	return false
}

// laneShown is path as a reason names it: resolved, so the lane it lies in
// shows, with the home directory masked and sanitised.
func laneShown(path string) string {
	return termsafe.Sanitize(fsutil.RedactHome(fsutil.RealExistingPath(path)))
}

// fromDriveLane says, for a reason, why the gate counts no review receipt or
// recorded approval for the checkout at dir, asked from the checkout at from
// ("" when dir is the one asked from): from, or else dir, is a drive lane's
// worktree or lies inside one. When the landing was asked from a checkout
// that is not in a lane and only the branch's own checkout is, it says the
// branch is checked out in a drive lane's worktree. "" when neither is in one.
func fromDriveLane(dir, from string) string {
	switch {
	case insideDriveLane(from):
		return inLaneWhy(from)
	case !insideDriveLane(dir):
		return ""
	case from != "":
		return "the branch is checked out in a drive lane's worktree (" + laneShown(dir) + "), and the merge gate counts no review receipt or recorded approval there: the drive lands that branch itself, and to land it by hand the lane must be handed back and its worktree removed first"
	}
	return inLaneWhy(dir)
}

// inLaneWhy is the reason for a gate asked from the checkout at dir, which is
// a drive lane's worktree or lies inside one.
func inLaneWhy(dir string) string {
	return laneShown(dir) + " is a drive lane's worktree or lies inside one, and the merge gate counts no review receipt or recorded approval when it runs from there, since the worktrees it would read are listed from inside the lane; run it from the main checkout or your own worktree"
}

// ErrDriveLaneRecord refuses a review receipt or a recorded approval written
// in a drive lane's worktree, which the merge gate would never count.
var ErrDriveLaneRecord = errors.New("the merge gate never counts a review receipt or a recorded approval in a drive lane's worktree")

// refuseInDriveLane refuses a record written at repoRoot when it is a drive
// lane's worktree or lies inside one (insideDriveLane), naming where to
// record it instead.
func refuseInDriveLane(repoRoot string) error {
	if !insideDriveLane(repoRoot) {
		return nil
	}
	return fmt.Errorf("%s is a drive lane's worktree or lies inside one, and %w: run it from the main checkout or your own worktree",
		laneShown(repoRoot), ErrDriveLaneRecord)
}

// findReceipts finds every worktree's receipt named sha under rel, in the order
// git lists the worktrees, by findReceipt's rule. A review receipt or a
// recorded approval in a worktree that is a drive lane's or lies inside one
// (insideDriveLane, on the path git lists) is passed over, and every one is
// when repoRoot is a drive lane's worktree or lies inside one, since the
// listing is then the lane's own. Its git calls run with no deadline; a reader
// on one uses findReceiptsContext.
func findReceipts(repoRoot, rel, sha string) ([]foundReceipt, error) {
	wts, err := gitutil.ListWorktrees(repoRoot, gitutil.MaxWorktreeListing)
	if err != nil {
		return nil, fmt.Errorf("listing the repository's worktrees: %w", err)
	}
	inLane := laneBlind(rel) && insideDriveLane(repoRoot)
	var found []foundReceipt
	for _, wt := range wts {
		p := receiptIn(wt, rel, sha)
		switch {
		case p == "":
		case inLane || laneBlind(rel) && insideDriveLane(wt.Path):
			found = append(found, foundReceipt{path: p, lane: true})
		default:
			found = append(found, foundReceipt{path: p, err: localtier.Untracked(wt.Path, rel+"/"+sha)})
		}
	}
	return found, nil
}

// findReceiptsContext is findReceipts with its git calls bound to ctx: a ctx
// already ended refuses before any git starts, one that ends while git runs
// kills it, and each worktree's tracked check is asked only while ctx lives.
func findReceiptsContext(ctx context.Context, repoRoot, rel, sha string) ([]foundReceipt, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	wts, err := gitutil.ListWorktreesContext(ctx, repoRoot, gitutil.MaxWorktreeListing)
	if err != nil {
		return nil, fmt.Errorf("listing the repository's worktrees: %w", err)
	}
	return receiptsIn(ctx, repoRoot, wts, rel, sha)
}

// receiptsIn is findReceiptsContext over worktrees already listed for the
// repository at repoRoot, by findReceipts' rule: when repoRoot is a drive
// lane's worktree or lies inside one, the listing is the lane's own, so no
// review receipt or recorded approval it names counts.
func receiptsIn(ctx context.Context, repoRoot string, wts []gitutil.Worktree, rel, sha string) ([]foundReceipt, error) {
	inLane := laneBlind(rel) && insideDriveLane(repoRoot)
	var found []foundReceipt
	for _, wt := range wts {
		p := receiptIn(wt, rel, sha)
		if p == "" {
			continue
		}
		if inLane || laneBlind(rel) && insideDriveLane(wt.Path) {
			found = append(found, foundReceipt{path: p, lane: true})
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		found = append(found, foundReceipt{path: p, err: localtier.UntrackedContext(ctx, wt.Path, rel+"/"+sha)})
	}
	return found, nil
}

// receiptIn is the path of the receipt named sha under rel in wt when one is
// there as a regular file, never a symlink; "" otherwise, and for a bare
// repository's entry.
func receiptIn(wt gitutil.Worktree, rel, sha string) string {
	if wt.Bare {
		return ""
	}
	p := filepath.Join(wt.Path, filepath.FromSlash(rel), sha)
	if fi, err := os.Lstat(p); err == nil && fi.Mode().IsRegular() {
		return p
	}
	return ""
}

// runCapped runs cmd, returning its stdout within MaxOutput and, on a failure,
// an error naming what and carrying its bounded stderr, scrubbed of
// credential URLs (gitutil.ScrubURLCredentials) before it is cut.
// A failed exit is judged first, so its error carries the client's stderr
// whatever it wrote to stdout; only a call that succeeded is held to the bound.
func runCapped(ctx context.Context, cmd *exec.Cmd, what string) (string, error) {
	var stdout, stderr bytes.Buffer
	if err := procgroup.RunToEOF(ctx, cmd, &stdout, &stderr); err != nil {
		return "", exitError(what, err, &stderr)
	}
	if stdout.Len() > MaxOutput {
		return "", fmt.Errorf("%s wrote more than %d bytes", what, MaxOutput)
	}
	return stdout.String(), nil
}

// exitError is a failed command's error: what, the exit, and its stderr,
// scrubbed BEFORE the cut, as gitutil's runners do: a cut across a credential
// URL leaves a fragment the scrubber no longer recognises.
func exitError(what string, err error, stderr *bytes.Buffer) error {
	msg := gitutil.ScrubURLCredentials(strings.TrimSpace(stderr.String()))
	if len(msg) > maxStderr {
		msg = msg[:maxStderr]
	}
	return fmt.Errorf("%s: %v (%s)", what, err, msg)
}

// ErrOutputCap marks a forge client's answer refused for passing MaxOutput.
var ErrOutputCap = errors.New("the forge client's output bound")

// runCappedAnswer is runCapped that keeps what the command wrote to stdout
// within MaxOutput beside the error of a failed exit: the forge client exits
// non-zero on a GraphQL answer that carries errors, and writes that answer,
// data and errors, to stdout. An answer past the bound is never returned, and
// its error wraps ErrOutputCap, with the exit and the stderr of a failed call
// beside it. Only the landings reader runs through it (GHAnswerContext).
func runCappedAnswer(ctx context.Context, cmd *exec.Cmd, what string) (string, error) {
	var stdout, stderr bytes.Buffer
	err := procgroup.RunToEOF(ctx, cmd, &stdout, &stderr)
	if stdout.Len() > MaxOutput {
		if err != nil {
			return "", fmt.Errorf("%s wrote more than %d bytes (%w); %w", what, MaxOutput, ErrOutputCap, exitError(what, err, &stderr))
		}
		return "", fmt.Errorf("%s wrote more than %d bytes (%w)", what, MaxOutput, ErrOutputCap)
	}
	if err != nil {
		return stdout.String(), exitError(what, err, &stderr)
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
	return GHContext(context.Background(), dir, args...)
}

// GHContext is GH bound to ctx as well as to ForgeTimeout: the client is
// killed at whichever ends first, and its output is read to EOF before it is
// waited for (procgroup.RunToEOF), so a client that answered is read whole
// however late its output closes, and a process it started that still holds
// that output open holds the call no further than the bound: a caller's own
// deadline (the away mode's stop check, which lists the forge under ten
// seconds) holds.
func GHContext(parent context.Context, dir string, args ...string) (string, error) {
	return gh(parent, dir, runCapped, args)
}

// GHAnswerContext is GHContext that keeps the client's answer beside the
// error of a failed exit (runCappedAnswer): the forge client exits non-zero on
// a GraphQL answer carrying errors, and a reader that judges each error by the
// part of the answer it names reads that answer whole.
func GHAnswerContext(parent context.Context, dir string, args ...string) (string, error) {
	return gh(parent, dir, runCappedAnswer, args)
}

// gh runs the forge client as GHContext describes, through run.
func gh(parent context.Context, dir string, run func(context.Context, *exec.Cmd, string) (string, error), args []string) (string, error) {
	// abcd's own recorded copy runs whatever comes first on PATH
	// (itd-2610240918594138); otherwise PATH finds gh, as before.
	bin, err := tools.Resolve("gh")
	if err != nil {
		return "", ErrNoForgeClient
	}
	ctx, cancel := context.WithTimeout(parent, ForgeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	cmd.Env = append(gitutil.ScrubbedEnv(), "GH_PROMPT_DISABLED=1", "GH_NO_UPDATE_NOTIFIER=1")
	cmd.WaitDelay = time.Second
	return run(ctx, cmd, "gh "+strings.Join(args[:min(2, len(args))], " "))
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
// `abcd land`'s. A squash or rebase merge leaves no such ancestry; mergedAt is
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
	URL               string `json:"url"`
	HeadRefOid        string `json:"headRefOid"`
	BaseRefName       string `json:"baseRefName"`
	IsCrossRepository bool   `json:"isCrossRepository"`
	MergedAt          string `json:"mergedAt"`
}

// MergedListingFullError is a listing of a branch's merged pull requests that
// the bound filled with none at the head: the proof may lie past the bound, so
// the listing cannot say the head has not merged.
type MergedListingFullError struct {
	Branch string
	Listed int
}

func (e *MergedListingFullError) Error() string {
	return fmt.Sprintf("the forge lists %d merged pull requests from branches named %s (its bound) and none at this head, so the proof that it merged may lie past the bound", e.Listed, e.Branch)
}

// mergedAt asks the forge, through ask (the forge client's arguments), for a
// pull request from this repository's own branch that MERGED into the default
// branch def with sha as its head, and returns its number and URL, or 0 when
// none did: the proof a squash or a rebase merge leaves where the ancestor
// check (Landed) finds none, since the merged commit on the default branch is
// not sha. A pull request from another repository, one merged into another
// base (a stacked pull request, whose work is not on def), or one merged at
// another head proves nothing about sha. A listing the bound filled with none
// at sha is a *MergedListingFullError, and any other error is the forge
// unable to answer: neither is a no.
func mergedAt(ask func(args ...string) (string, error), branch, def, sha string) (int, string, error) {
	out, err := ask("pr", "list", "--head="+branch, "--state=merged",
		"--json=number,url,headRefOid,baseRefName,isCrossRepository,mergedAt", fmt.Sprintf("--limit=%d", maxMergedListed))
	if err != nil {
		return 0, "", err
	}
	var prs []mergedPR
	if json.Unmarshal([]byte(out), &prs) != nil {
		return 0, "", errors.New("the forge's listing of " + branch + "'s merged pull requests does not parse")
	}
	for _, pr := range prs {
		if !pr.IsCrossRepository && pr.Number > 0 && pr.HeadRefOid == sha && pr.BaseRefName == def {
			return pr.Number, pr.URL, nil
		}
	}
	if len(prs) >= maxMergedListed {
		return 0, "", &MergedListingFullError{Branch: branch, Listed: len(prs)}
	}
	return 0, "", nil
}
