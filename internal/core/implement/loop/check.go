package loop

// check.go is the pre-start checks (spec piece 4; criteria 1 and 2): the
// readiness gate, the open-question count, the claim sections, the hold, the
// spec's steps, and the peers. Every check reads; none writes. A run starts
// only when every row passes, and a refusal carries every row, so the caller
// sees the whole picture rather than the first failure.

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/peers"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Check names, in the fixed order Check reports them.
const (
	CheckKey           = "key"
	CheckReady         = "ready"
	CheckOpenQuestions = intent.StartCheckOpenQuestions
	CheckClaimSections = intent.StartCheckClaimSections
	CheckHold          = intent.StartCheckHold
	CheckBlocked       = intent.StartCheckBlocked
	CheckSteps         = intent.StartCheckSteps
	CheckPeers         = "peers"
	// CheckEligible is an issue key's row: the drained repository's own rule
	// takes the issue (itd-82 scope 2; decision 10 on the parent).
	CheckEligible = "eligible"
)

// CheckRow is one pre-start check's verdict.
type CheckRow struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
	Remedy string `json:"remedy,omitempty"`
	// contention marks a failure that is a peer's hold, not the record's fault.
	contention bool
}

// CheckResult is every pre-start check for one key.
type CheckResult struct {
	Key    string     `json:"key"`
	Intent string     `json:"intent,omitempty"`
	Spec   string     `json:"spec,omitempty"`
	OK     bool       `json:"ok"`
	Checks []CheckRow `json:"checks"`

	// steps are the unlanded spec steps a run opens lanes for, in order.
	steps []PendingStep
}

// record is the record a run built from the result delivers: the intent, or
// the issue an issue key names.
func (r CheckResult) record() string {
	if r.Intent != "" {
		return r.Intent
	}
	return r.Key
}

// refusal renders a failed result as the refusal Start returns: the first
// failing row names the check, and every row rides along.
func (r CheckResult) refusal() *Refusal {
	for _, c := range r.Checks {
		if c.OK {
			continue
		}
		ref := refuse("check", c.Name, "", c.Detail, c.Remedy)
		ref.Contention = c.contention
		ref.Checks = r.Checks
		return ref
	}
	return nil
}

// maxIntentBytes caps an intent read the loop makes.
const maxIntentBytes = 256 * 1024

// Check runs the pre-start checks for key against the checkout at repoRoot. It
// returns an error only for a fault in reading this checkout (the store will
// not load, git will not answer); a record that may not start is a result with
// OK false.
//
// The rows, in order:
//
//   - key: an intent id, or an issue id (decision 10's key). An issue key
//     takes two rows after it and no others: eligible (the drained
//     repository's own rule takes the issue, as `abcd drain` reads it) and
//     peers (no peer holds it out of open/, and no session claims it).
//   - ready: the implement-readiness gate (intent.Ready) — planned, criteria,
//     the spec linked and written. Its advisory rows stay advisory here.
//   - open_questions: no open question under `## Open Questions`
//     (intent.StartChecksIn: every list item, less the settled markers).
//   - claim_sections: no unanswered claim section — the mechanism prompt
//     answered or the section absent, the scope conditions recorded.
//   - hold: no `held:` on the record (iss-2609200830076665).
//   - blocked: nothing the record names in `blocked_by` is unshipped
//     (itd-2609211116005482: an intent with an unshipped blocker is not one
//     a run may take); a superseded blocker is followed along
//     `superseded_by` to its replacement, transitively (ruling BZ2 of
//     2026-09-29).
//   - steps: the spec's `## Steps` reads, and leaves a step to build.
//   - peers: no peer holds the record — no sibling worktree or local branch
//     holds it in another bucket, and no session holds a live claim on it.
func Check(repoRoot, key string) (CheckResult, error) { return check(repoRoot, key, "", nil) }

// check is Check on behalf of session: a live claim session itself holds on the
// record is its own, not a peer's. An empty session owns no claim. snap is the
// peers and claims read once for many keys (the pick's candidate set); nil
// reads them for this key alone.
func check(repoRoot, key, session string, snap *peerSnapshot) (CheckResult, error) {
	res := CheckResult{Key: key}
	if row, ok := keyCheck(key); !ok {
		res.Checks = append(res.Checks, row)
		return res, nil
	} else {
		res.Checks = append(res.Checks, row)
	}
	if validIssueKey(key) {
		return issueCheck(repoRoot, res, session, snap)
	}

	ready, err := intent.Ready(repoRoot, key)
	if err != nil {
		// A malformed or unknown id: the gate cannot judge it, and nothing below
		// has a record to read.
		res.Checks = append(res.Checks, CheckRow{Name: CheckReady, Detail: err.Error(),
			Remedy: "name a planned intent: `abcd intent` lists them"})
		return res, nil
	}
	res.Intent, res.Spec = ready.IntentID, ready.SpecID
	res.Checks = append(res.Checks, readyRow(ready))

	corpus, err := intent.Load(repoRoot)
	if err != nil {
		return res, err
	}
	store, err := spec.Load(repoRoot)
	if err != nil {
		return res, err
	}
	// The record-only checks are intent.StartChecksIn's, the one statement of
	// them the status board's "next up" reads too (iss-2609291803334904).
	record, err := intent.StartChecksIn(repoRoot, corpus, store, ready)
	if err != nil {
		return res, err
	}
	for _, r := range record.Rows {
		res.Checks = append(res.Checks, CheckRow{Name: r.Name, OK: r.OK, Detail: r.Detail, Remedy: r.Remedy})
	}
	for _, st := range record.Steps {
		res.steps = append(res.steps, PendingStep{Number: st.Number, Title: st.Title})
	}

	if snap == nil {
		if snap, err = readPeers(repoRoot); err != nil {
			return res, err
		}
	}
	peersRow := peersCheck(ready.IntentID, ready.Bucket, session, snap)
	res.Checks = append(res.Checks, peersRow)

	res.OK = true
	for _, c := range res.Checks {
		if !c.OK {
			res.OK = false
		}
	}
	return res, nil
}

// validIssueKey reports whether key is an issue id by shape (`iss-` and
// digits): the only issue key a run is built from, so no path is ever made of
// anything else.
func validIssueKey(key string) bool { return issueIDRe.MatchString(key) }

// keyCheck admits an intent id or an issue id and refuses everything else by
// name, quoting the refused key escaped.
func keyCheck(key string) (CheckRow, bool) {
	row := CheckRow{Name: CheckKey}
	switch {
	case recordid.ValidIntentID(key):
		row.OK = true
		row.Detail = key + " is an intent"
		return row, true
	case validIssueKey(key):
		row.OK = true
		row.Detail = key + " is an issue"
		return row, true
	default:
		row.Detail = fmt.Sprintf("%q is not a record id this verb builds (itd-N or iss-N)", key)
		row.Remedy = "name a planned intent (`abcd build <itd-N>`) or an open issue the drain rule takes (`abcd build <iss-N>`)"
	}
	return row, false
}

// issueCheck is the pre-start checks for an issue key (decision 10 on the
// parent): the drained repository's own rule, read as the drain reads it, takes
// the issue, and no peer holds it. The run then has one lane, for the issue.
func issueCheck(repoRoot string, res CheckResult, session string, snap *peerSnapshot) (CheckResult, error) {
	row := CheckRow{Name: CheckEligible}
	plan, err := capture.PlanDrain(capture.DrainPlanRequest{RepoRoot: repoRoot})
	if err != nil {
		if !drainRuleRefusal(err) {
			return res, err
		}
		row.Detail = fsutil.RedactHome(err.Error())
		row.Remedy = "record the repository's drain rule (`abcd ahoy install` offers it); an issue is built only under it"
		res.Checks = append(res.Checks, row)
		return res, nil
	}
	i := slices.IndexFunc(plan.Dispositions, func(v capture.DrainVerdict) bool { return recordid.SameID(v.ID, res.Key) })
	switch {
	case i < 0:
		row.Detail = res.Key + " is not an open issue in this checkout's ledger"
		row.Remedy = "name an open issue: `abcd drain --dry-run` lists every one with its disposition"
	case plan.Dispositions[i].Outcome != capture.DrainEligible:
		v := plan.Dispositions[i]
		row.Detail = fmt.Sprintf("%s is %s under %s (%s): %s", v.ID, v.Outcome, plan.Record, v.Rule, v.Reason)
		row.Remedy = "the issue is a person's; `abcd drain --dry-run` shows where each open issue goes"
	default:
		v := plan.Dispositions[i]
		row.OK = true
		row.Detail = fmt.Sprintf("%s is eligible under %s: %s", v.ID, plan.Record, v.Reason)
		res.steps = []PendingStep{{Number: 1, Title: issueStepTitle(v)}}
	}
	res.Checks = append(res.Checks, row)
	if !row.OK {
		return res, nil
	}
	if snap == nil {
		if snap, err = readPeers(repoRoot); err != nil {
			return res, err
		}
	}
	res.Checks = append(res.Checks, peersCheck(res.Key, string(capture.StateOpen), session, snap))
	res.OK = true
	for _, c := range res.Checks {
		if !c.OK {
			res.OK = false
		}
	}
	return res, nil
}

// maxIssueTitle caps the issue's title a lane carries as its step title.
const maxIssueTitle = 120

// issueStepTitle is the lane's title for an issue: the record's one-line
// summary, sanitised and capped, or its id when it has none.
func issueStepTitle(v capture.DrainVerdict) string {
	t := strings.Join(strings.Fields(termsafe.Sanitize(v.Title)), " ")
	if r := []rune(t); len(r) > maxIssueTitle {
		t = string(r[:maxIssueTitle]) + "…"
	}
	if t == "" {
		return v.ID
	}
	return t
}

// drainRuleRefusal reports whether err is the drain rule's load refusing.
func drainRuleRefusal(err error) bool {
	for _, s := range []error{drainrule.ErrUnrecorded, drainrule.ErrMalformed, drainrule.ErrAmbiguous, drainrule.ErrUnreadable} {
		if errors.Is(err, s) {
			return true
		}
	}
	return false
}

// readyRow folds the readiness gate into one row: the first failing gating
// check names why, with its own remedy.
func readyRow(r intent.ReadyResult) CheckRow {
	row := CheckRow{Name: CheckReady}
	if r.Ready {
		row.OK = true
		row.Detail = r.IntentID + " is READY (planned, criteria written, " + r.SpecID + " linked and written)"
		return row
	}
	for _, c := range r.Checks {
		if c.OK || c.Advisory {
			continue
		}
		row.Detail = c.Name + ": " + c.Detail
		row.Remedy = c.Remedy
		break
	}
	if row.Remedy == "" {
		row.Remedy = "run `abcd intent ready " + r.IntentID + "` and settle what it reports"
	}
	return row
}

// peerSnapshot is what the peers check reads: the peer listing and the run's
// claims, read once and judged for any number of records.
type peerSnapshot struct {
	rep    peers.Report
	claims []implement.ClaimState
}

// readPeers reads the peer listing and, when the checkout has a root commit,
// the shared run's claims.
func readPeers(repoRoot string) (*peerSnapshot, error) {
	rep, err := peers.Scan(repoRoot)
	if err != nil {
		return nil, err
	}
	snap := &peerSnapshot{rep: rep}
	if sha := gitutil.RootCommit(repoRoot); gitutil.IsFullSHA(sha) {
		run, err := implement.Peek(sha)
		if err != nil {
			return nil, err
		}
		if snap.claims, err = run.Claims(); err != nil {
			return nil, err
		}
	}
	return snap, nil
}

// peersCheck refuses a record a peer holds, from the two places a holding is
// visible from this checkout: the peer listing (a sibling worktree or a local
// branch holding the intent in another bucket than this checkout's — a lane
// that shipped or re-drafted it), and the run's claim store (a session holding
// a live claim on it). A peer holding the record in the same bucket holds a
// copy, not the record: every branch cut from the default branch does. A live
// claim held by session — the one the build is started for — is its own.
func peersCheck(id, bucket, session string, snap *peerSnapshot) CheckRow {
	row := CheckRow{Name: CheckPeers}
	var holders []string
	for _, l := range snap.rep.Locate(id) {
		if l.Folder == bucket {
			continue
		}
		holders = append(holders, peerName(l.Source, l.Branch, l.Path)+" holds it in "+l.Folder+"/")
	}
	// A peer the listing names and cannot read may hold the record; the check
	// fails closed on it, as it does on an unreadable claim below.
	for _, p := range snap.rep.Unjudged() {
		holders = append(holders, peerName(p.Source, p.Branch, p.Path)+" could not be read, so what it holds is unknown ("+fsutil.DisplayPathsIn(p.NotRead, p.Path)+")")
	}
	for _, c := range snap.claims {
		if (c.Live || c.Unreadable) && recordid.SameID(c.Record, id) {
			if session != "" && !c.Unreadable && c.Session == session {
				continue
			}
			if c.Unreadable {
				holders = append(holders, "an unreadable claim file holds it")
				continue
			}
			holders = append(holders, fmt.Sprintf("session %s claims it for lane %s", c.Session, c.Lane))
		}
	}
	if len(holders) == 0 {
		row.OK = true
		row.Detail = "no peer holds " + id
		return row
	}
	row.contention = true
	row.Detail = id + " is held by a peer: " + strings.Join(holders, "; ")
	row.Remedy = "take other work, or coordinate with the peer; `abcd peers` and `abcd implement` show what each holds, and name why a peer is not read"
	return row
}

// peerName names a peer for a refusal, a worktree by fsutil.DisplayPath so one
// outside HOME is its directory name, not an absolute local path
// (iss-2609281329007423).
func peerName(src peers.Source, branch, path string) string {
	if src != peers.SourceWorktree {
		return "branch " + branch
	}
	who := "the worktree at " + fsutil.DisplayPath(path)
	if branch != "" {
		who += " (branch " + branch + ")"
	}
	return who
}
