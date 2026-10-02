package loop

// land.go is the landing (spec piece 9; criterion 6). After a lane's validators
// pass, the land stage takes the lane to the default branch one step per
// invocation, each recorded in the lane's `landing` as it completes, so a
// killed process resumes at the step that did not:
//
//  1. prepare: the lane's worktree is clean and its branch is at the head the
//     validators judged; the landing decides whether it closes the spec (the
//     lane that took the fidelity audit) and what it records.
//  2. records: in the lane's worktree, `spec close` (intent.Reconcile) when the
//     lane closes the spec, with the verdict of the audit the lane took
//     ingested into the parked receipt rather than asked for again, and
//     `capture resolve` for every capture the lane's receipts declared fixed,
//     each with the lane's commit that fixed it. The loop commits them on the
//     lane's branch with `Delivers:` (when the close ships the intent) and
//     `Resolves:` trailers, so RS001 and RS005 find the records in the change,
//     and an `Assisted-by:` naming the model the lane's receipts reported (the
//     records carry its prose), with the repository's hooks running.
//  3. push: refused unless the repository's preflight receipt names the lane's
//     head (the pre-push hook's gate, checked before any connection opens), then
//     a plain `git push` of the lane's branch from the checkout the run lives
//     in, its hooks running: the loop never skips a hook and never forces.
//  4. pull request: through the forge client the repository already uses
//     (`gh`), with a body built from the records and passed through the
//     outbound scrub; after creating it the loop re-reads the body the forge
//     holds and strips a session URL or a tool footer the harness appended.
//  5. arm: the merge rule from the ruleset mirror at the lane's base
//     (.abcd/work/rulesets/): auto-merge armed with the queue's method where a
//     merge queue gates the default branch and a ruleset requires a person's
//     approval, the pull request left open for a person where either is absent
//     (decision 3; ruling AM1). Nothing is pushed to the lane after this step.
//  6. merged: the lane's pushed head must be an ancestor of the default
//     branch as the remote holds it; until it is the step waits, and only then
//     is the lane's worktree removed and its branch deleted, and the lane done.
//
// Every git and forge argument is derived from the state and the record: the
// branch from the run and lane ids, the pull request from the forge's own
// listing, the body and title from the records, through the scrub.

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
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Landing is the land stage's progress on one lane.
type Landing struct {
	// Closes is whether this lane's landing closes the spec (and ships the
	// intent): the lane that took the fidelity audit.
	Closes bool `json:"closes"`
	// Records is the loop's records commit on the lane's branch, and
	// RecordsDone is set once it is made, or once there was nothing to record.
	Records     string `json:"records,omitempty"`
	RecordsDone bool   `json:"records_done"`
	// PreflightReceipt is the preflight receipt that named the pushed head,
	// home-redacted, and Pushed the head the loop pushed.
	PreflightReceipt string `json:"preflight_receipt,omitempty"`
	Pushed           string `json:"pushed,omitempty"`
	// Body is the pull request's body as the loop composed it, relative to the
	// checkout root; PRURL is the pull request, and BodyChecked is set once the
	// body the forge holds was re-read and found clean.
	Body        string `json:"body,omitempty"`
	PRURL       string `json:"pr_url,omitempty"`
	BodyChecked bool   `json:"body_checked"`
	// Merge is the merge rule the ruleset gave, and Armed whether auto-merge
	// was armed by it. Nothing is pushed to the lane once Merge is set.
	Merge string `json:"merge,omitempty"`
	Armed bool   `json:"armed"`
	// Merged is the default branch's tip, as the remote held it, that carried
	// the pushed head when the landing cleaned the lane up.
	Merged string `json:"merged,omitempty"`
	// CheckWaitSince is when the landing began waiting for its full check: the
	// preflight receipt naming the head it pushes (ruling DR6d-2). It is set
	// by the first call that finds no receipt, kept until the push, and
	// cleared by it.
	CheckWaitSince *time.Time `json:"check_wait_since,omitempty"`
}

// The landing's fixed names.
const (
	// Remote is the remote a lane lands through: the one the default branch
	// is read from.
	Remote = "origin"
	// LandDirName is the lane's directory the landing writes into.
	LandDirName = "land"
	// PRBodyFileName is the pull request's body as the loop composed it, and
	// PRStrippedFileName the body re-read from the forge and stripped.
	PRBodyFileName     = "pr-body.md"
	PRStrippedFileName = "pr-body.stripped.md"
	// PreflightReceiptsRelDir is where the preflight mints its receipts, one
	// file named by the full commit id, in any worktree of the repository.
	PreflightReceiptsRelDir = TierRelDir + "/preflight-receipts"
	// RulesetsRelDir is the committed mirror of the default branch's rulesets.
	RulesetsRelDir = ".abcd/work/rulesets"
)

// Timeouts and caps for what the landing runs.
const (
	netTimeout      = 10 * time.Minute
	ghTimeout       = 2 * time.Minute
	maxForgeOutput  = 1 << 20
	maxRulesetBytes = 256 << 10
	maxRulesets     = 32
)

// landSubject is the records commit's subject.
func landSubject(st State, lane Lane) string {
	return "chore(record): land " + lane.ID + " of " + st.RunID
}

// landStage is the land stage's body: it performs the landing's next step.
func landStage(c Context, lane *Lane) (Outcome, error) {
	if !gitutil.IsFullSHA(lane.BaseSHA) || !gitutil.IsFullSHA(lane.HeadSHA) || lane.Worktree == "" || lane.Branch == "" {
		return Outcome{}, refuse(string(StageLand), "", lane.ID, "the lane records no branch, worktree, base and head to land",
			"the earlier stages record them; restore the run's state file")
	}
	if lane.Landing == nil {
		// A sibling lane of the run that landed since this lane's base is
		// merged in first, and a fresh round judges the merge head.
		if out, synced, err := syncLane(c, lane); err != nil || synced {
			return out, err
		}
		return landPrepare(c, lane)
	}
	ld := *lane.Landing
	lane.Landing = &ld
	switch {
	case !ld.RecordsDone:
		return landRecords(c, lane)
	case ld.Pushed == "":
		return landPush(c, lane)
	case lane.PR == 0 || !ld.BodyChecked:
		return landPullRequest(c, lane)
	case ld.Merge == "":
		return landArm(c, lane)
	}
	return landMerged(c, lane)
}

// defaultBranch is the name of the default branch on the remote the lane lands
// through, refused when the repository has no such remote-tracking branch.
func defaultBranch(c Context, lane Lane) (string, error) {
	ref := gitutil.DefaultRef(c.RepoRoot)
	name, ok := strings.CutPrefix(ref, "refs/remotes/"+Remote+"/")
	if !ok || name == "" || name == "HEAD" {
		return "", refuse(string(StageLand), "", lane.ID, "the repository has no default branch on the remote "+Remote+" to land the lane on",
			"add the remote and fetch it (`git fetch "+Remote+"`), then run `abcd implement step` again")
	}
	return name, nil
}

// branchTip is the lane branch's tip.
func branchTip(c Context, lane Lane) (string, error) {
	tip, err := gitutil.Run(c.RepoRoot, "rev-parse", "--verify", "--quiet", "refs/heads/"+lane.Branch+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(tip) {
		return "", refuse(string(StageLand), "", lane.ID, "the lane's branch "+lane.Branch+" cannot be read",
			"restore the branch at the lane's head, then run `abcd implement step` again")
	}
	return tip, nil
}

// passingAudit is the return of the intent-auditor the lane's passing round
// recorded, relative to the checkout root, or "" when the round took none.
func passingAudit(lane Lane) string {
	if n := len(lane.Validation); n > 0 {
		for _, v := range lane.Validation[n-1].Validators {
			if v.Role == RoleAuditor && v.Pass {
				return v.Return
			}
		}
	}
	return ""
}

// landPrepare is the landing's first step.
func landPrepare(c Context, lane *Lane) (Outcome, error) {
	tip, err := branchTip(c, *lane)
	if err != nil {
		return Outcome{}, err
	}
	if tip != lane.HeadSHA {
		return Outcome{}, refuse(string(StageLand), "", lane.ID,
			fmt.Sprintf("%s is at %s, not at the head %s its validators judged", lane.Branch, shortSHA(tip), shortSHA(lane.HeadSHA)),
			"restore the branch to the judged head (the landing lands only what was validated), then run `abcd implement step` again")
	}
	if dirty, err := pickGit(lane.Worktree, "status", "--porcelain", "-z", "--untracked-files=all"); err != nil {
		return Outcome{}, fmt.Errorf("reading the lane's worktree: %w", err)
	} else if dirty != "" {
		return Outcome{}, refuse(string(StageLand), "", lane.ID, "the lane's worktree holds uncommitted changes, and the landing commits only what it writes",
			"commit or remove them in the lane's worktree (a change the validators did not judge goes back through a fix round), then run `abcd implement step` again")
	}
	closes, err := auditsHere(c, *lane)
	if err != nil {
		return Outcome{}, err
	}
	if closes && passingAudit(*lane) == "" {
		return Outcome{}, refuse(string(StageLand), "", lane.ID, "this lane closes the spec, and its passing round recorded no fidelity audit to ingest at the close",
			"restore the run's state file; the validate stage records the audit on the lane that closes the spec")
	}
	ld := &Landing{Closes: closes}
	var does []string
	if closes {
		does = append(does, "closes "+c.State.Spec+" and ships "+c.State.Intent)
	}
	for _, r := range lane.Resolves {
		does = append(does, "resolves "+r.Issue)
	}
	if len(does) == 0 {
		ld.RecordsDone = true
		does = append(does, "records nothing (the lane neither closes the spec nor fixed a capture)")
	}
	lane.Landing = ld
	return Outcome{Stay: true, Note: "landing prepared at " + shortSHA(lane.HeadSHA) + ": it " + strings.Join(does, ", ")}, nil
}

// landRecords makes the landing's records in the lane's worktree and commits
// them on the lane's branch, or finds the commit a killed call made.
func landRecords(c Context, lane *Lane) (Outcome, error) {
	st := c.State
	ld := lane.Landing
	tip, err := branchTip(c, *lane)
	if err != nil {
		return Outcome{}, err
	}
	if tip != lane.HeadSHA {
		parent, _ := gitutil.Run(c.RepoRoot, "rev-parse", "--verify", "--quiet", tip+"^1^{commit}", "--")
		subject, _ := gitutil.Run(c.RepoRoot, "log", "-1", "--format=%s", tip, "--")
		if parent != lane.HeadSHA || subject != landSubject(st, *lane) {
			return Outcome{}, refuse(string(StageLand), "", lane.ID,
				fmt.Sprintf("%s moved to %s, which is not the landing's records commit on the judged head %s", lane.Branch, shortSHA(tip), shortSHA(lane.HeadSHA)),
				"restore the branch to the judged head, then run `abcd implement step` again")
		}
		// A call killed after its commit and before its state write: found.
		ld.Records, ld.RecordsDone, lane.HeadSHA = tip, true, tip
		return Outcome{Stay: true, Note: "found the landing's records commit " + shortSHA(tip) + " on " + lane.Branch}, nil
	}

	// The disclosure is settled before any record is written, so a lane with
	// no model to disclose is refused with its worktree untouched.
	assisted, gap := assistedByTrailers(lane.Receipts)
	if gap != "" {
		return Outcome{}, refuse(string(StageLand), "", lane.ID,
			"the landing's records commit carries text the lane's implementer composed, and "+gap+", so its Assisted-by: trailer cannot name the model",
			"have the implementer's receipt report the model its harness runs (\"model\": \"<vendor>:<model-id>\", or a bare claude-* id), "+
				"send the lane back through a fix round whose receipt reports it, then run `abcd implement step` again; the loop never claims no assistance for a model's text")
	}

	wt := lane.Worktree
	var trailers, done []string
	if ld.Closes {
		// The close's audit emit parks its receipt under the worktree's local
		// tier; the lane's worktree is the loop's own, so the tier is made there.
		if err := fsutil.EnsureRealDirAll(wt, TierRelDir, dirPerm); err != nil {
			return Outcome{}, fmt.Errorf("making the local tier in the lane's worktree: %w", err)
		}
		res, err := intent.Reconcile(wt, st.Spec, "", intent.RemainderRequest{})
		if err != nil {
			return Outcome{}, refuse(string(StageLand), "", lane.ID, "`spec close "+st.Spec+"` refused in the lane's worktree: "+fsutil.RedactHome(err.Error()),
				"settle what the close names on the lane's branch (an intent ships only with an impact:), then run `abcd implement step` again")
		}
		if res.AuditEmitError != "" {
			return Outcome{}, refuse(string(StageLand), "", lane.ID, "the close parked no fidelity receipt for the audit to answer: "+fsutil.RedactHome(res.AuditEmitError),
				"settle what the emit names, then run `abcd implement step` again")
		}
		verdict := abs(c.RepoRoot, passingAudit(*lane))
		ing, err := intent.IngestVerdict(wt, verdict)
		if err != nil {
			return Outcome{}, refuse(string(StageLand), "", lane.ID, "the audit's verdict did not ingest at the close: "+fsutil.RedactHome(err.Error()),
				"the verdict is the one the closing lane's audit returned; restore it, then run `abcd implement step` again")
		}
		if ing.Status != "ingested" && ing.Status != "noop" {
			return Outcome{}, refuse(string(StageLand), "", lane.ID, "the audit's verdict was "+ing.Status+" at the close, not ingested: "+fsutil.RedactHome(ing.Reason),
				"the landing ships only an intent whose audit is on its record; settle the verdict, then run `abcd implement step` again")
		}
		done = append(done, fmt.Sprintf("closed %s (%s %s -> %s), audit %s %s", st.Spec, st.Intent, res.From, res.To, ing.ReceiptID, ing.Status))
		if res.To == intent.BucketShipped {
			trailers = append(trailers, "Delivers: "+st.Intent)
		}
	}
	for _, r := range lane.Resolves {
		if resolvedIn(wt, r.Issue) {
			done = append(done, "found "+r.Issue+" resolved")
		} else {
			if _, err := capture.Resolve(capture.ResolveRequest{RepoRoot: wt, ID: r.Issue, Resolution: r.Note, Impact: r.Impact,
				ByCommit: r.Commit, Grounds: r.Grounds}); err != nil {
				return Outcome{}, refuse(string(StageLand), "", lane.ID, "`capture resolve "+r.Issue+"` refused in the lane's worktree: "+fsutil.RedactHome(err.Error()),
					"the resolution is the lane receipt's; settle what the capture store names on the lane's branch, then run `abcd implement step` again")
			}
			done = append(done, "resolved "+r.Issue+" with "+shortSHA(r.Commit))
		}
		trailers = append(trailers, "Resolves: "+r.Issue)
	}
	if _, err := pickGit(wt, "add", "-A", "--", "."); err != nil {
		return Outcome{}, fmt.Errorf("staging the landing's records: %w", err)
	}
	// The local tier holds the close's review request and the preflight's
	// receipts; it is never part of the change, even in a repository that does
	// not ignore it. The reset takes back only what the add staged there.
	if _, err := pickGit(wt, "reset", "-q", "--", TierRelDir); err != nil {
		return Outcome{}, fmt.Errorf("keeping the local tier out of the landing's records: %w", err)
	}
	staged, err := pickGit(wt, "diff", "--cached", "--name-only", "-z")
	if err != nil {
		return Outcome{}, fmt.Errorf("reading the landing's staged records: %w", err)
	}
	if staged == "" {
		return Outcome{}, refuse(string(StageLand), "", lane.ID, "the landing's records changed nothing on the lane's branch",
			"the close and the resolutions should move records; check the lane's branch holds them open, then run `abcd implement step` again")
	}
	// Unlike the pick commit (pickcommit.go), whose text abcd computes and
	// which declares `Assisted-by: None`, this commit's diff carries prose a
	// model composed: the receipt's resolution note and grounds, and the
	// audit's verdict ingested into the intent. So it names that model, and
	// it is made with the repository's hooks running (never through pickGit's
	// hooks-off configuration), so the commit-msg outbound gate judges it.
	what := fmt.Sprintf("%s (%s, step %d)", st.Intent, st.Spec, lane.SpecStep)
	if iss := st.Issue(); iss != "" {
		what = iss
	}
	msg := landSubject(st, *lane) + "\n\n" +
		fmt.Sprintf("The implement loop's landing for %s: %s.\n\n", what, strings.Join(done, "; ")) +
		strings.Join(append(trailers, assisted...), "\n") + "\n"
	if _, err := hookedGit(wt, "commit", "-q", "-m", msg); err != nil {
		return Outcome{}, refuse(string(StageLand), "", lane.ID,
			"git could not commit the landing's records (a repository hook refused it, or no git identity is configured): "+fsutil.RedactHome(err.Error()),
			"settle what git or the hook reports (the records stay staged in the lane's worktree), then run `abcd implement step` again; the loop never skips a hook")
	}
	head, err := branchTip(c, *lane)
	if err != nil {
		return Outcome{}, err
	}
	ld.Records, ld.RecordsDone, lane.HeadSHA = head, true, head
	return Outcome{Stay: true, Note: "committed the landing's records as " + shortSHA(head) + ": " + strings.Join(done, "; ")}, nil
}

// assistedVendorRe is an Assisted-by: value in the vendor form the attribution
// gate takes (scripts/check-attribution.sh TRAILER_RE), and bareClaudeRe a bare
// Claude model id, which takes the Claude vendor prefix.
var (
	assistedVendorRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9._-]*:[A-Za-z0-9._-]+(\[[A-Za-z0-9._-]+\])?$`)
	bareClaudeRe     = regexp.MustCompile(`^claude-[A-Za-z0-9._-]+(\[[A-Za-z0-9._-]+\])?$`)
)

// assistedByTrailers are the records commit's Assisted-by: trailers, one per
// distinct model the lane's receipts reported, in the order first reported.
// Every receipt's runner may have composed text the commit carries, so a
// receipt that reports no model, or one in no form the trailer takes, is a gap
// named in the returned description, and no trailers are returned; a lane with
// no receipt at all is a gap too. The model is the runner's report, which the
// binary cannot verify: a refused value is described, never quoted.
func assistedByTrailers(rs []ReceiptRecord) ([]string, string) {
	if len(rs) == 0 {
		return nil, "the lane has no implementer's receipt to report a model"
	}
	var out []string
	for i, r := range rs {
		var v string
		switch {
		case r.Model == "":
			return nil, fmt.Sprintf("receipt %d (%s) reports no model", i+1, r.Receipt)
		case bareClaudeRe.MatchString(r.Model):
			v = "Claude:" + r.Model
		case assistedVendorRe.MatchString(r.Model):
			v = r.Model
		default:
			return nil, fmt.Sprintf("receipt %d (%s) reports a model in no form the trailer takes (%s)", i+1, r.Receipt, termsafe.DescribeRefused(r.Model))
		}
		if t := "Assisted-by: " + v; !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out, ""
}

// hookedGit runs one git command in the lane's worktree with the repository's
// hooks running, under the developer's own configuration less any injected
// GIT_DIR or GIT_CONFIG_* (as netGit), with no terminal prompt.
func hookedGit(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), netTimeout)
	defer cancel()
	full := append([]string{"-c", "core.quotePath=false", "-C", dir}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Env = append(gitutil.ScrubbedEnv(), "GIT_TERMINAL_PROMPT=0")
	return runCapped(cmd, "git "+args[0])
}

// resolvedIn reports whether issue is already in the worktree's resolved/.
func resolvedIn(wt, issue string) bool {
	m, _ := filepath.Glob(filepath.Join(wt, ".abcd", "work", "issues", "resolved", issue+"-*.md"))
	return len(m) > 0
}

// preflightReceipt finds the preflight receipt for sha in any worktree git
// lists for the repository, as the pre-push hook's check does. A receipt must
// be a regular file, never a symlink.
func preflightReceipt(repoRoot, sha string) (string, error) {
	wts, err := gitutil.ListWorktrees(repoRoot, maxWorktreeListing)
	if err != nil {
		return "", fmt.Errorf("listing the repository's worktrees: %w", err)
	}
	for _, wt := range wts {
		if wt.Bare {
			continue
		}
		p := filepath.Join(wt.Path, filepath.FromSlash(PreflightReceiptsRelDir), sha)
		if fi, err := os.Lstat(p); err == nil && fi.Mode().IsRegular() {
			return p, nil
		}
	}
	return "", nil
}

// landPush pushes the lane's branch once the preflight receipt names its head.
func landPush(c Context, lane *Lane) (Outcome, error) {
	ld := lane.Landing
	if ld.Armed || ld.Merge != "" {
		return Outcome{}, refuse(string(StageLand), "", lane.ID, "the lane's merge is already armed, and nothing is pushed to a lane after arming",
			"restore the run's state file")
	}
	if _, err := defaultBranch(c, *lane); err != nil {
		return Outcome{}, err
	}
	tip, err := branchTip(c, *lane)
	if err != nil {
		return Outcome{}, err
	}
	if tip != lane.HeadSHA {
		return Outcome{}, refuse(string(StageLand), "", lane.ID,
			fmt.Sprintf("%s is at %s, not at the head %s the landing lands", lane.Branch, shortSHA(tip), shortSHA(lane.HeadSHA)),
			"restore the branch to that head, then run `abcd implement step` again")
	}
	rcp, err := preflightReceipt(c.RepoRoot, lane.HeadSHA)
	if err != nil {
		return Outcome{}, err
	}
	if rcp == "" {
		// The lane waits for its full check as a landing waits on the forge's
		// merge (ruling DR6d-2): a contention that holds only this lane, from
		// the time the first call found it.
		since := c.Now
		if ld.CheckWaitSince != nil {
			since = *ld.CheckWaitSince
		}
		r := contend(string(StageLand), "", lane.ID,
			checkWaitText(since)+": no preflight receipt names the lane's head "+lane.HeadSHA+", so the pre-push gate would refuse its push",
			"run the repository's preflight (`make preflight`) on a clean tree in the lane's worktree "+fsutil.RedactHome(lane.Worktree)+
				", which mints the receipt, then run `abcd implement step` again; the loop never bypasses the receipt")
		r.checkWait = since
		return Outcome{}, r
	}
	ref := "refs/heads/" + lane.Branch
	if _, err := netGit(c.RepoRoot, "push", "--porcelain", Remote, ref+":"+ref); err != nil {
		return Outcome{}, refuse(string(StageLand), "", lane.ID, "git could not push "+lane.Branch+" to "+Remote+": "+fsutil.RedactHome(err.Error()),
			"settle what git or the pre-push hook reports, then run `abcd implement step` again")
	}
	note := "pushed " + lane.Branch + " at " + shortSHA(lane.HeadSHA) + " to " + Remote + " on its preflight receipt"
	if ld.CheckWaitSince != nil {
		note += fmt.Sprintf(", after waiting %d minute(s) for its full check", int(c.Now.Sub(*ld.CheckWaitSince)/time.Minute))
	}
	ld.PreflightReceipt, ld.Pushed, ld.CheckWaitSince = fsutil.RedactHome(rcp), lane.HeadSHA, nil
	return Outcome{Stay: true, Note: note}, nil
}

// netGit runs one git command that reaches the remote, from the checkout the
// run lives in. Its hooks run (the pre-push gate is the point), under the
// developer's own configuration less any injected GIT_DIR or GIT_CONFIG_*.
func netGit(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), netTimeout)
	defer cancel()
	full := append([]string{"-c", "core.quotePath=false", "-C", dir}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Env = append(gitutil.ScrubbedEnv(), "GIT_TERMINAL_PROMPT=0")
	return runCapped(cmd, "git "+args[0])
}

// runCapped runs cmd, returning its stdout within the forge cap and its
// stderr, bounded, in the error.
func runCapped(cmd *exec.Cmd, what string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if len(msg) > 2048 {
			msg = msg[:2048]
		}
		return "", fmt.Errorf("%s: %v (%s)", what, err, msg)
	}
	if stdout.Len() > maxForgeOutput {
		return "", fmt.Errorf("%s wrote more than %d bytes", what, maxForgeOutput)
	}
	return stdout.String(), nil
}

// forge runs the forge client with args from the checkout the run lives in,
// refused when the client is not installed.
func forge(c Context, lane Lane, args ...string) (string, error) {
	bin, err := exec.LookPath("gh")
	if err != nil {
		return "", refuse(string(StageLand), "", lane.ID, "the forge client `gh` is not installed, and the landing opens the pull request through it",
			"install gh and sign in with your own identity (`gh auth login`), then run `abcd implement step` again")
	}
	ctx, cancel := context.WithTimeout(context.Background(), ghTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = c.RepoRoot
	cmd.Env = append(os.Environ(), "GH_PROMPT_DISABLED=1", "GH_NO_UPDATE_NOTIFIER=1")
	out, err := runCapped(cmd, "gh "+strings.Join(args[:min(2, len(args))], " "))
	if err != nil {
		return "", refuse(string(StageLand), "", lane.ID, fsutil.RedactHome(err.Error()),
			"settle what the forge client reports, then run `abcd implement step` again")
	}
	return out, nil
}

// openPR is the lane branch's open pull request as the forge lists it.
type openPR struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

// findPR lists the lane branch's open pull requests: none, or the one.
func findPR(c Context, lane Lane) (*openPR, error) {
	out, err := forge(c, lane, "pr", "list", "--head", lane.Branch, "--state", "open", "--json", "number,url", "--limit", "2")
	if err != nil {
		return nil, err
	}
	var prs []openPR
	if err := json.Unmarshal([]byte(out), &prs); err != nil {
		return nil, refuse(string(StageLand), "", lane.ID, "the forge's listing of "+lane.Branch+"'s pull requests does not parse", "check the forge client (`gh --version`), then run `abcd implement step` again")
	}
	switch len(prs) {
	case 0:
		return nil, nil
	case 1:
		if prs[0].Number <= 0 {
			return nil, refuse(string(StageLand), "", lane.ID, "the forge lists a pull request for "+lane.Branch+" with no number", "check the forge client, then run `abcd implement step` again")
		}
		return &prs[0], nil
	}
	return nil, refuse(string(StageLand), "", lane.ID, "the forge lists more than one open pull request for "+lane.Branch,
		"close all but one, then run `abcd implement step` again")
}

// prTitle and prBody are the pull request's title and body, built from the run's
// records.
func prTitle(st State, lane Lane) string {
	if iss := st.Issue(); iss != "" {
		return fmt.Sprintf("fix(%s): %s", iss, lane.StepTitle)
	}
	return fmt.Sprintf("build(%s): %s, step %d of %s", st.Intent, lane.StepTitle, lane.SpecStep, st.Spec)
}

func prBody(st State, lane Lane) string {
	var b strings.Builder
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	if iss := st.Issue(); iss != "" {
		p("This pull request fixes %s (%q) by its remedy. The implement loop built it in run %s as %s, and wrote this text from the run's records.\n\n",
			iss, lane.StepTitle, st.RunID, lane.ID)
	} else {
		p("This pull request lands step %d of %s (%q) for %s. The implement loop built it in run %s as %s, and wrote this text from the run's records.\n\n",
			lane.SpecStep, st.Spec, lane.StepTitle, st.Intent, st.RunID, lane.ID)
	}
	p("- Branch `%s`, from %s to %s.\n", lane.Branch, shortSHA(lane.BaseSHA), shortSHA(lane.HeadSHA))
	if n := len(lane.Validation); n > 0 {
		r := lane.Validation[n-1]
		p("- Validation round %d at %s passed: %s.\n", r.Round, shortSHA(r.HeadSHA), verdictsLine(r))
	}
	var trailers []string
	if lane.Landing != nil && lane.Landing.Closes {
		p("- It closes %s and ships %s; the fidelity audit's verdict is on the intent's record.\n", st.Spec, st.Intent)
		trailers = append(trailers, "Delivers: "+st.Intent)
	}
	for _, r := range lane.Resolves {
		p("- It resolves %s, fixed by %s.\n", r.Issue, shortSHA(r.Commit))
		trailers = append(trailers, "Resolves: "+r.Issue)
	}
	if len(trailers) > 0 {
		p("\n%s\n", strings.Join(trailers, "\n"))
	}
	return b.String()
}

// landPullRequest opens the lane's pull request, or finds the one a killed
// call opened, and re-reads the body the forge holds.
func landPullRequest(c Context, lane *Lane) (Outcome, error) {
	ld := lane.Landing
	def, err := defaultBranch(c, *lane)
	if err != nil {
		return Outcome{}, err
	}
	pr, err := findPR(c, *lane)
	if err != nil {
		return Outcome{}, err
	}
	verb := "found"
	if pr == nil {
		body, _, err := scanner.ScrubOutbound(c.RepoRoot, prBody(c.State, *lane), "the pull request's body")
		if err != nil {
			return Outcome{}, refuse(string(StageLand), "", lane.ID, fsutil.RedactHome(err.Error()), "settle what the scrub names, then run `abcd implement step` again")
		}
		title, _, err := scanner.ScrubOutbound(c.RepoRoot, prTitle(c.State, *lane), "the pull request's title")
		if err != nil {
			return Outcome{}, refuse(string(StageLand), "", lane.ID, fsutil.RedactHome(err.Error()), "settle what the scrub names, then run `abcd implement step` again")
		}
		rel, err := laneFile(c.State.RunID, lane.ID, StageLand, LandDirName+"/"+PRBodyFileName)
		if err != nil {
			return Outcome{}, err
		}
		if err := writeRoundFile(c.RepoRoot, rel, []byte(body)); err != nil {
			return Outcome{}, err
		}
		ld.Body = rel
		// Values that come from the record ride in the flag's own argument
		// (--flag=value), so none can be read as a flag of its own.
		if _, err := forge(c, *lane, "pr", "create", "--base", def, "--head", lane.Branch,
			"--title="+strings.TrimSpace(title), "--body-file="+abs(c.RepoRoot, rel)); err != nil {
			return Outcome{}, err
		}
		if pr, err = findPR(c, *lane); err != nil {
			return Outcome{}, err
		}
		if pr == nil {
			return Outcome{}, refuse(string(StageLand), "", lane.ID, "the forge lists no pull request for "+lane.Branch+" after creating one",
				"check the forge, then run `abcd implement step` again; the next step finds a pull request it opened")
		}
		verb = "opened"
	}
	lane.PR, ld.PRURL = pr.Number, pr.URL
	stripped, err := recheckBody(c, lane)
	if err != nil {
		return Outcome{}, err
	}
	ld.BodyChecked = true
	note := fmt.Sprintf("%s pull request #%d for %s into %s; its body re-read clean", verb, pr.Number, lane.Branch, def)
	if stripped {
		note = fmt.Sprintf("%s pull request #%d for %s into %s; its body arrived carrying what the outbound policy bans, and was stripped and re-read clean", verb, pr.Number, lane.Branch, def)
	}
	return Outcome{Stay: true, Note: note}, nil
}

// recheckBody re-reads the body the forge holds and, when it carries a session
// URL or a tool footer the loop did not write, strips it and reads it again. It
// reports whether it stripped, and refuses a body still dirty.
func recheckBody(c Context, lane *Lane) (bool, error) {
	n := strconv.Itoa(lane.PR)
	held, err := forge(c, *lane, "pr", "view", n, "--json", "body", "--jq", ".body")
	if err != nil {
		return false, err
	}
	if _, err := scanner.CheckOutbound(c.RepoRoot, held, "pull request #"+n); err == nil {
		return false, nil
	}
	clean, _, err := scanner.ScrubOutbound(c.RepoRoot, held, "pull request #"+n)
	if err != nil {
		return false, refuse(string(StageLand), "", lane.ID, fsutil.RedactHome(err.Error()), "strip the pull request's body by hand, then run `abcd implement step` again")
	}
	rel, err := laneFile(c.State.RunID, lane.ID, StageLand, LandDirName+"/"+PRStrippedFileName)
	if err != nil {
		return false, err
	}
	if err := writeRoundFile(c.RepoRoot, rel, []byte(clean)); err != nil {
		return false, err
	}
	if _, err := forge(c, *lane, "pr", "edit", n, "--body-file="+abs(c.RepoRoot, rel)); err != nil {
		return false, err
	}
	again, err := forge(c, *lane, "pr", "view", n, "--json", "body", "--jq", ".body")
	if err != nil {
		return false, err
	}
	if _, err := scanner.CheckOutbound(c.RepoRoot, again, "pull request #"+n); err != nil {
		return false, refuse(string(StageLand), "", lane.ID, "pull request #"+n+"'s body still carries what the outbound policy bans after the strip",
			"strip it by hand on the forge, then run `abcd implement step` again")
	}
	return true, nil
}

// ruleset is the part of a ruleset mirror the merge rule is read from.
type ruleset struct {
	Enforcement string `json:"enforcement"`
	Conditions  struct {
		RefName struct {
			Include []string `json:"include"`
			Exclude []string `json:"exclude"`
		} `json:"ref_name"`
	} `json:"conditions"`
	Rules []struct {
		Type       string          `json:"type"`
		Parameters json.RawMessage `json:"parameters"`
	} `json:"rules"`
}

// targets reports whether the ruleset applies to the default branch def.
func (r ruleset) targets(def string) bool {
	match := func(pats []string) bool {
		for _, p := range pats {
			if p == "~DEFAULT_BRANCH" || p == "~ALL" || p == "refs/heads/"+def {
				return true
			}
		}
		return false
	}
	return r.Enforcement == "active" && match(r.Conditions.RefName.Include) && !match(r.Conditions.RefName.Exclude)
}

// mergeMethods maps a merge queue's method to the forge client's flag.
var mergeMethods = map[string]string{"MERGE": "--merge", "SQUASH": "--squash", "REBASE": "--rebase"}

// codeOwnersPaths are where the forge reads a CODEOWNERS file from, in the
// order it looks: the first file found is the only one it reads.
var codeOwnersPaths = []string{".github/CODEOWNERS", "CODEOWNERS", "docs/CODEOWNERS"}

// namesCodeOwners reports whether the CODEOWNERS file the forge reads at the
// lane's base — the first found in codeOwnersPaths, so a file there shadows
// the later ones even when it names nobody — names at least one owner: a line
// whose pattern is followed by an owner token (codeOwner).
func namesCodeOwners(c Context, lane Lane) bool {
	for _, p := range codeOwnersPaths {
		raw, err := gitutil.RunCappedBytes(c.RepoRoot, maxRulesetBytes, "cat-file", "blob", lane.BaseSHA+":"+p)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(raw), "\n") {
			if lineNamesOwner(line) {
				return true
			}
		}
		return false
	}
	return false
}

// lineNamesOwner reports whether one CODEOWNERS line names an owner: after its
// pattern, a field before any comment is an owner token. A blank line, a
// comment and a pattern with no owner name nobody.
func lineNamesOwner(line string) bool {
	fields := strings.Fields(line)
	if len(fields) < 2 || strings.HasPrefix(fields[0], "#") {
		return false
	}
	for _, f := range fields[1:] {
		if strings.HasPrefix(f, "#") {
			return false
		}
		if codeOwner(f) {
			return true
		}
	}
	return false
}

// codeOwner reports whether a token is one the forge takes as an owner:
// @username, @org/team-name, or an e-mail address.
func codeOwner(tok string) bool {
	if name, ok := strings.CutPrefix(tok, "@"); ok {
		org, team, isTeam := strings.Cut(name, "/")
		if isTeam {
			return ownerName(org) && team != "" && !strings.ContainsAny(team, "/@")
		}
		return ownerName(name)
	}
	local, domain, ok := strings.Cut(tok, "@")
	return ok && local != "" && strings.Contains(domain, ".") && !strings.HasPrefix(domain, ".") &&
		!strings.HasSuffix(domain, ".") && !strings.Contains(domain, "@")
}

// ownerName reports whether s is a forge account name: letters, digits and
// hyphens, not empty.
func ownerName(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return true
}

// requiresApproval reports whether a pull_request rule's parameters require a
// person's approval: an approving count of one or more, or a code-owner review
// where a CODEOWNERS file at the lane's base names an owner. Parameters that
// do not parse require nothing, so the merge is left for a person.
func requiresApproval(c Context, lane Lane, params json.RawMessage) bool {
	var p struct {
		Count     int  `json:"required_approving_review_count"`
		CodeOwner bool `json:"require_code_owner_review"`
	}
	if json.Unmarshal(params, &p) != nil {
		return false
	}
	return p.Count >= 1 || (p.CodeOwner && namesCodeOwners(c, lane))
}

// mergeRule reads the ruleset mirror at the lane's base: the merge queue's
// method when an active ruleset gates the default branch through one, or ""
// when none does, and whether an active ruleset on the default branch requires
// a person's approval (ruling AM1). A missing mirror requires nothing. The
// base is the default branch the lane was cut from, so the lane's own commits
// cannot change the rule it lands by.
func mergeRule(c Context, lane Lane, def string) (string, bool, error) {
	names, err := gitutil.RunCappedBytes(c.RepoRoot, maxGitOutput, "ls-tree", "-z", "--name-only", lane.BaseSHA, "--", RulesetsRelDir+"/")
	if err != nil {
		return "", false, fmt.Errorf("listing the ruleset mirror at the lane's base: %v", err)
	}
	method := ""
	approval := false
	count := 0
	for _, name := range strings.Split(string(names), "\x00") {
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		if count++; count > maxRulesets {
			return "", false, refuse(string(StageLand), "", lane.ID, fmt.Sprintf("the ruleset mirror holds more than %d files", maxRulesets), "trim the mirror, then run `abcd implement step` again")
		}
		raw, err := gitutil.RunCappedBytes(c.RepoRoot, maxRulesetBytes, "cat-file", "blob", lane.BaseSHA+":"+name)
		if err != nil {
			return "", false, refuse(string(StageLand), "", lane.ID, name+" cannot be read at the lane's base", "restore the ruleset mirror, then run `abcd implement step` again")
		}
		var rs ruleset
		if err := json.Unmarshal(raw, &rs); err != nil {
			return "", false, refuse(string(StageLand), "", lane.ID, name+" does not parse as a ruleset, so the merge rule cannot be read",
				"restore the ruleset mirror (its README says how to refresh it), then run `abcd implement step` again")
		}
		if !rs.targets(def) {
			continue
		}
		for _, rule := range rs.Rules {
			if rule.Type == "pull_request" && requiresApproval(c, lane, rule.Parameters) {
				approval = true
			}
			if rule.Type != "merge_queue" {
				continue
			}
			var p struct {
				MergeMethod string `json:"merge_method"`
			}
			_ = json.Unmarshal(rule.Parameters, &p)
			m := strings.ToUpper(p.MergeMethod)
			if m == "" {
				m = "MERGE"
			}
			if _, ok := mergeMethods[m]; !ok {
				return "", false, refuse(string(StageLand), "", lane.ID, name+" names a merge-queue method the forge client has no flag for",
					"correct the ruleset mirror, then run `abcd implement step` again")
			}
			if method != "" && method != m {
				return "", false, refuse(string(StageLand), "", lane.ID, "the ruleset mirror gates the default branch through merge queues with different methods",
					"correct the ruleset mirror, then run `abcd implement step` again")
			}
			method = m
		}
	}
	return method, approval, nil
}

// landArm arms the merge by the ruleset's rule, or leaves the pull request
// open where no merge queue gates the default branch, or where no ruleset
// requires a person's approval (ruling AM1): a merge the loop arms is one a
// person must still approve.
func landArm(c Context, lane *Lane) (Outcome, error) {
	ld := lane.Landing
	def, err := defaultBranch(c, *lane)
	if err != nil {
		return Outcome{}, err
	}
	method, approval, err := mergeRule(c, *lane, def)
	if err != nil {
		return Outcome{}, err
	}
	n := strconv.Itoa(lane.PR)
	if method == "" {
		ld.Merge = "left open: no ruleset gates " + def + " through a merge queue"
		return Outcome{Stay: true, Note: "pull request #" + n + " " + ld.Merge + "; it lands when a person merges it"}, nil
	}
	if !approval {
		ld.Merge = "left open for a person to merge: the ruleset requires no approval on " + def
		return Outcome{Stay: true, Note: "pull request #" + n + " " + ld.Merge + ", so the loop does not arm auto-merge; it lands when a person merges it"}, nil
	}
	if _, err := forge(c, *lane, "pr", "merge", n, "--auto", mergeMethods[method]); err != nil {
		return Outcome{}, err
	}
	ld.Armed = true
	ld.Merge = "auto-merge armed through " + def + "'s merge queue (" + strings.ToLower(method) + ")"
	return Outcome{Stay: true, Note: "pull request #" + n + ": " + ld.Merge + "; nothing is pushed to the lane after this"}, nil
}

// landMerged waits for the pushed head to be an ancestor of the default branch
// as the remote holds it, then removes the lane's worktree and branch.
func landMerged(c Context, lane *Lane) (Outcome, error) {
	ld := lane.Landing
	def, err := defaultBranch(c, *lane)
	if err != nil {
		return Outcome{}, err
	}
	tracking := "refs/remotes/" + Remote + "/" + def
	if _, err := netGit(c.RepoRoot, "fetch", "--quiet", "--no-tags", Remote, "+refs/heads/"+def+":"+tracking); err != nil {
		return Outcome{}, refuse(string(StageLand), "", lane.ID, "git could not fetch "+def+" from "+Remote+": "+fsutil.RedactHome(err.Error()),
			"settle what git reports, then run `abcd implement step` again")
	}
	on, err := gitutil.IsAncestor(c.RepoRoot, ld.Pushed, tracking)
	if err != nil {
		return Outcome{}, fmt.Errorf("placing %s on %s: %v", shortSHA(ld.Pushed), tracking, err)
	}
	n := strconv.Itoa(lane.PR)
	if !on {
		state, err := forge(c, *lane, "pr", "view", n, "--json", "state", "--jq", ".state")
		if err != nil {
			return Outcome{}, err
		}
		switch strings.TrimSpace(state) {
		case "MERGED":
			return Outcome{}, refuse(string(StageLand), "", lane.ID,
				"pull request #"+n+" merged, but the lane's head "+shortSHA(ld.Pushed)+" is not on "+Remote+"/"+def+" (a squash or rebase rewrote it), so the ancestor check cannot prove the lane landed",
				"confirm the change is on "+def+", then remove the lane's worktree and branch yourself; the loop cleans up only what the ancestor check proves")
		case "CLOSED":
			return Outcome{}, refuse(string(StageLand), "", lane.ID, "pull request #"+n+" was closed without merging",
				"reopen it, or hand the lane back; the loop cleans up nothing that did not land")
		}
		return Outcome{}, contend(string(StageLand), "", lane.ID,
			"pull request #"+n+" is not merged yet: "+shortSHA(ld.Pushed)+" is not on "+Remote+"/"+def,
			"run `abcd implement step` again once it has merged; "+ld.Merge)
	}
	merged, err := gitutil.Run(c.RepoRoot, "rev-parse", "--verify", "--quiet", tracking+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(merged) {
		return Outcome{}, fmt.Errorf("resolving %s: %v", tracking, err)
	}
	// The branch is deleted only at a tip the ancestor check proves landed: a
	// commit added after the push is work that did not land.
	tip, tipErr := gitutil.Run(c.RepoRoot, "rev-parse", "--verify", "--quiet", "refs/heads/"+lane.Branch+"^{commit}", "--")
	if tipErr == nil && gitutil.IsFullSHA(tip) {
		if landed, err := gitutil.IsAncestor(c.RepoRoot, tip, tracking); err != nil || !landed {
			return Outcome{}, refuse(string(StageLand), "", lane.ID,
				lane.Branch+" carries "+shortSHA(tip)+", past what landed, and the loop never deletes work that did not land",
				"land or move that work, reset the branch to "+shortSHA(ld.Pushed)+", then run `abcd implement step` again")
		}
	}
	if err := removeLaneWorktree(c, *lane); err != nil {
		return Outcome{}, err
	}
	if tipErr == nil && gitutil.IsFullSHA(tip) {
		if _, err := gitutil.Run(c.RepoRoot, "update-ref", "-d", "refs/heads/"+lane.Branch, tip); err != nil {
			return Outcome{}, refuse(string(StageLand), "", lane.ID, "git could not delete "+lane.Branch+": "+fsutil.RedactHome(err.Error()),
				"settle what git reports, then run `abcd implement step` again")
		}
	}
	ld.Merged = merged
	return Outcome{Note: fmt.Sprintf("pull request #%s landed: %s is on %s/%s at %s; removed the lane's worktree and branch %s",
		n, shortSHA(ld.Pushed), Remote, def, shortSHA(merged), lane.Branch)}, nil
}

// removeLaneWorktree removes the lane's worktree when git lists it at the lane's
// path on the lane's branch; git refuses a worktree with changes, and the loop
// never forces it.
func removeLaneWorktree(c Context, lane Lane) error {
	wts, err := gitutil.ListWorktrees(c.RepoRoot, maxWorktreeListing)
	if err != nil {
		return fmt.Errorf("listing the repository's worktrees: %w", err)
	}
	want := fsutil.RealExistingPath(lane.Worktree)
	for _, wt := range wts {
		if fsutil.RealExistingPath(wt.Path) != want {
			continue
		}
		if wt.Branch != "refs/heads/"+lane.Branch {
			return refuse(string(StageLand), "", lane.ID, "git lists a worktree at the lane's path on another branch, and the loop removes only what it made",
				"remove it yourself, then run `abcd implement step` again")
		}
		if _, err := gitutil.Run(c.RepoRoot, "worktree", "remove", "--", wt.Path); err != nil {
			return refuse(string(StageLand), "", lane.ID, "git could not remove the lane's worktree: "+fsutil.RedactHome(err.Error()),
				"settle what git reports (it refuses a worktree with changes), then run `abcd implement step` again")
		}
		return nil
	}
	if _, err := os.Lstat(lane.Worktree); err == nil {
		return refuse(string(StageLand), "", lane.ID, fsutil.RedactHome(lane.Worktree)+" stands where git lists no worktree, and the loop removes only what it made",
			"move it aside, then run `abcd implement step` again")
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("checking the lane's worktree: %w", err)
	}
	return nil
}
