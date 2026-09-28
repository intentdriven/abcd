package loop

// check.go is the pre-start checks (spec piece 4; criteria 1 and 2): the
// readiness gate, the open-question count, the claim sections, the hold, the
// spec's steps, and the peers. Every check reads; none writes. A run starts
// only when every row passes, and a refusal carries every row, so the caller
// sees the whole picture rather than the first failure.

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/peers"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// Check names, in the fixed order Check reports them.
const (
	CheckKey           = "key"
	CheckReady         = "ready"
	CheckOpenQuestions = "open_questions"
	CheckClaimSections = "claim_sections"
	CheckHold          = "hold"
	CheckSteps         = "steps"
	CheckPeers         = "peers"
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

// maxIntentBytes caps the intent read the section checks make.
const maxIntentBytes = 256 * 1024

// Check runs the pre-start checks for key against the checkout at repoRoot. It
// returns an error only for a fault in reading this checkout (the store will
// not load, git will not answer); a record that may not start is a result with
// OK false.
//
// The rows, in order:
//
//   - key: an intent id. An issue id is decision 10's key, which a later piece
//     of the spec admits with drain's eligibility rule.
//   - ready: the implement-readiness gate (intent.Ready) — planned, criteria,
//     the spec linked and written. Its advisory rows stay advisory here.
//   - open_questions: no open question under `## Open Questions`
//     (intent.OpenQuestions: every list item, less the settled markers).
//   - claim_sections: no unanswered claim section — the mechanism prompt
//     answered or the section absent, the scope conditions recorded.
//   - hold: no `held:` on the record (iss-2609200830076665).
//   - steps: the spec's `## Steps` reads, and leaves a step to build.
//   - peers: no peer holds the record — no sibling worktree or local branch
//     holds it in another bucket, and no session holds a live claim on it.
func Check(repoRoot, key string) (CheckResult, error) {
	res := CheckResult{Key: key}
	if row, ok := keyCheck(key); !ok {
		res.Checks = append(res.Checks, row)
		return res, nil
	} else {
		res.Checks = append(res.Checks, row)
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

	content, err := fsutil.ReadGuarded(filepath.Join(repoRoot, filepath.FromSlash(ready.Path)), maxIntentBytes)
	if err != nil {
		return res, fmt.Errorf("reading %s: %w", ready.Path, err)
	}
	res.Checks = append(res.Checks, openQuestionsRow(ready.IntentID, string(content)))
	res.Checks = append(res.Checks, claimSectionsRow(ready, string(content)))

	corpus, err := intent.Load(repoRoot)
	if err != nil {
		return res, err
	}
	it, _ := corpus.Lookup(ready.IntentID)
	res.Checks = append(res.Checks, holdRow(it))

	stepsRow, steps, err := stepsCheck(repoRoot, ready)
	if err != nil {
		return res, err
	}
	res.steps = steps
	res.Checks = append(res.Checks, stepsRow)

	peersRow, err := peersCheck(repoRoot, ready)
	if err != nil {
		return res, err
	}
	res.Checks = append(res.Checks, peersRow)

	res.OK = true
	for _, c := range res.Checks {
		if !c.OK {
			res.OK = false
		}
	}
	return res, nil
}

// keyCheck admits an intent id and refuses everything else by name.
func keyCheck(key string) (CheckRow, bool) {
	row := CheckRow{Name: CheckKey}
	switch {
	case recordid.ValidIntentID(key):
		row.OK = true
		row.Detail = key + " is an intent"
		return row, true
	case strings.HasPrefix(key, "iss-"):
		row.Detail = key + " is an issue: the issue key (the intent's decision 10) is not built in this abcd yet"
		row.Remedy = "build an intent with `abcd build <itd-N>`, or fix the issue by hand"
	default:
		row.Detail = fmt.Sprintf("%q is not a record id this verb builds", key)
		row.Remedy = "name a planned intent: `abcd build <itd-N>`"
	}
	return row, false
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

// openQuestionsRow refuses a record that still asks a question.
func openQuestionsRow(id, content string) CheckRow {
	row := CheckRow{Name: CheckOpenQuestions}
	qs := intent.OpenQuestions(content)
	if len(qs) == 0 {
		row.OK = true
		row.Detail = "no open question"
		return row
	}
	row.Detail = fmt.Sprintf("%s asks %d open question(s): %s", id, len(qs), strings.Join(qs, " | "))
	row.Remedy = "answer each in the record's `## Decisions` and mark it settled (an item marked `resolved:` or `**Deferred**`, or a section opening `_All resolved …_`), through the planning interview (/abcd:intent)"
	return row
}

// claimSectionsRow refuses an unanswered claim section. The readiness gate
// reports both claim rows as advisory; a run is the point they bind, because an
// autonomous lane has nobody to ask what the record meant.
func claimSectionsRow(r intent.ReadyResult, content string) CheckRow {
	row := CheckRow{Name: CheckClaimSections}
	var why, remedy []string
	if intent.ParseClaims(content).MechanismPrompt {
		why = append(why, "the '## Mechanism' prompt is unanswered")
		remedy = append(remedy, "write the falsifiable claim under '## Mechanism', or record `"+intent.NullityToken+"` alone on its line to decline it")
	}
	for _, c := range r.Checks {
		if (c.Name == intent.CheckMechanismClaim || c.Name == intent.CheckScopeConditions) && !c.OK {
			why = append(why, c.Detail)
			remedy = append(remedy, c.Remedy)
		}
	}
	if len(why) == 0 {
		row.OK = true
		row.Detail = "the claim sections are answered"
		return row
	}
	row.Detail = strings.Join(why, "; ")
	row.Remedy = strings.Join(remedy, "; ")
	return row
}

// holdRow refuses a held record, and a malformed hold with it: the key's
// presence is somebody's attempt at a hold, whatever its shape.
func holdRow(it intent.Intent) CheckRow {
	row := CheckRow{Name: CheckHold}
	switch {
	case it.HeldMalformed:
		row.Detail = it.ID + " carries a `held:` key in a shape no verb writes"
		row.Remedy = "repair the line by hand (record-lint names it), then `abcd intent unhold " + it.ID + "`"
	case it.Held != "":
		row.Detail = it.ID + " is held: " + it.Held
		row.Remedy = "settle the hold, then `abcd intent unhold " + it.ID + "`"
	default:
		row.OK = true
		row.Detail = it.ID + " is not held"
	}
	return row
}

// stepsCheck reads the open spec's steps through the spec store's reader: the
// unlanded steps are the lanes, one at a time; a spec listing none is one
// implicit step. A section the reader refuses is refused here, because the
// lanes are made from it (unrecognized-input-never-writes).
func stepsCheck(repoRoot string, r intent.ReadyResult) (CheckRow, []PendingStep, error) {
	row := CheckRow{Name: CheckSteps}
	store, err := spec.Load(repoRoot)
	if err != nil {
		return row, nil, err
	}
	sp, ok := store.Lookup(r.SpecID)
	if !ok {
		row.Detail = "no spec to read steps from"
		row.Remedy = "link a written spec first: `abcd intent ready " + r.IntentID + "` names how"
		return row, nil, nil
	}
	listed, err := spec.ReadSteps(repoRoot, sp)
	if err != nil {
		row.Detail = err.Error()
		row.Remedy = "rewrite " + spec.StepsHeading + " in " + sp.Path + " as the numbered list `abcd intent ready " + r.IntentID + "` describes, or empty it to build the spec as one step"
		return row, nil, nil
	}
	if len(listed) == 0 {
		row.OK = true
		row.Detail = sp.ID + " lists no steps: it is built as one step"
		return row, []PendingStep{{Number: 1, Title: spec.ImplicitStepTitle}}, nil
	}
	var steps []PendingStep
	for _, s := range spec.Unlanded(listed) {
		steps = append(steps, PendingStep{Number: s.Number, Title: s.Title})
	}
	if len(steps) == 0 {
		row.Detail = fmt.Sprintf("every one of %s's %d step(s) is landed: nothing is left to build", sp.ID, len(listed))
		row.Remedy = "close the spec: `abcd spec close " + sp.ID + "`"
		return row, nil, nil
	}
	row.OK = true
	row.Detail = fmt.Sprintf("%s: %d of %d step(s) to build", sp.ID, len(steps), len(listed))
	return row, steps, nil
}

// peersCheck refuses a record a peer holds, from the two places a holding is
// visible from this checkout: the peer listing (a sibling worktree or a local
// branch holding the intent in another bucket than this checkout's — a lane
// that shipped or re-drafted it), and the run's claim store (a session holding
// a live claim on it). A peer holding the record in the same bucket holds a
// copy, not the record: every branch cut from the default branch does.
func peersCheck(repoRoot string, r intent.ReadyResult) (CheckRow, error) {
	row := CheckRow{Name: CheckPeers}
	rep, err := peers.Scan(repoRoot)
	if err != nil {
		return row, err
	}
	var holders []string
	for _, l := range rep.Locate(r.IntentID) {
		if l.Folder == r.Bucket {
			continue
		}
		holders = append(holders, peerName(l.Source, l.Branch, l.Path)+" holds it in "+l.Folder+"/")
	}
	// A peer the listing names and cannot read may hold the record; the check
	// fails closed on it, as it does on an unreadable claim below.
	for _, p := range rep.Unjudged() {
		holders = append(holders, peerName(p.Source, p.Branch, p.Path)+" could not be read, so what it holds is unknown ("+fsutil.DisplayPathsIn(p.NotRead, p.Path)+")")
	}
	if sha := gitutil.RootCommit(repoRoot); gitutil.IsFullSHA(sha) {
		run, err := implement.Peek(sha)
		if err != nil {
			return row, err
		}
		claims, err := run.Claims()
		if err != nil {
			return row, err
		}
		for _, c := range claims {
			if (c.Live || c.Unreadable) && recordid.SameID(c.Record, r.IntentID) {
				if c.Unreadable {
					holders = append(holders, "an unreadable claim file holds it")
					continue
				}
				holders = append(holders, fmt.Sprintf("session %s claims it for lane %s", c.Session, c.Lane))
			}
		}
	}
	if len(holders) == 0 {
		row.OK = true
		row.Detail = "no peer holds " + r.IntentID
		return row, nil
	}
	row.contention = true
	row.Detail = r.IntentID + " is held by a peer: " + strings.Join(holders, "; ")
	row.Remedy = "take other work, or coordinate with the peer; `abcd peers` and `abcd implement` show what each holds, and name why a peer is not read"
	return row, nil
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
