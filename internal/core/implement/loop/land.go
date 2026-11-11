package loop

// land.go is the landing (spec piece 9; criterion 6). After a lane's validators
// pass, the land stage takes the lane to the default branch one step per
// invocation, each recorded in the lane's `landing` as it completes, so a
// killed process resumes at the step that did not:
//
//  1. prepare: the lane's worktree holds no change beside the lane's own spec
//     record, which the records commit then carries, and its branch is at the head
//     the validators judged; the landing decides whether it closes the spec (the
//     lane that took the fidelity audit) and what it records.
//  2. records: in the lane's worktree, the lane's step marked landed on the
//     open spec (its `- landed:` line, naming the judged head), `spec close`
//     (intent.Reconcile) when the lane closes the spec, with the verdict of the audit the lane took
//     ingested into the parked receipt rather than asked for again, and
//     `capture resolve` for every capture the lane's receipts declared fixed,
//     each with the lane's commit that fixed it. The loop commits them on the
//     lane's branch with `Delivers:` (when the close ships the intent) and
//     `Resolves:` trailers, so RS001 and RS005 find the records in the change,
//     and an `Assisted-by:` naming the model the lane's receipts reported (the
//     records carry its prose), with the repository's hooks running.
//  3. push: refused unless the repository's preflight receipt names the lane's
//     head (the pre-push hook's gate, checked before any connection opens) and
//     the run's reservation is still this person's from this copy and no
//     release another session is cutting holds it (the push fence, the land
//     fence's check), then "merging <the lane's record>" announced on the
//     person's status branch (claim.AnnounceBeforePush, the code `abcd
//     announce` runs), the push refused when it cannot be, then a plain
//     `git push` of the lane's
//     branch from the checkout the run lives in, its hooks running: the loop
//     never skips a hook and never forces.
//  4. pull request: through the forge client the repository already uses
//     (`gh`), with a body built from the records and passed through the
//     outbound scrub; after creating it the loop re-reads the body the forge
//     holds and strips a session URL or a tool footer the harness appended.
//  5. arm: the merge gate `abcd land` asks (landing.DecideMerge, which
//     replaced ruling AM1): the facilitator mode at the default branch's
//     fetched tip, the author's permission, the approvals of the pushed head
//     and, in auto, a review receipt naming it, which the loop records itself
//     from a pair that read that head (recordOwnReview): in auto a records commit
//     goes to a round of its own (recordsRoundWanted) before the push. Where it allows, the merge is
//     armed by the ruleset mirror's queue method, or as auto-merge with no
//     queue; otherwise the pull request is left open, saying why, and each
//     later step asks again. A pull request the forge reports CONFLICTING
//     is refused instead of armed (conflict.go). Nothing is pushed to the lane
//     after this step.
//  6. merged: the lane's pushed head must be an ancestor of the default
//     branch as the remote holds it; until it is the step waits, and only then
//     is the lane's worktree removed and its branch deleted, and the lane done.
//     An open pull request the forge reports CONFLICTING never merges on its
//     own, so it is refused for a person rather than waited on. An armed one
//     a merge queue gates is watched in the queue while it waits (queue.go,
//     iss-2610090642376032): queued when it sits CLEAN outside it, and
//     refused for a person, naming the failed merge-group check, when it
//     left the queue and its arming with it.
//
// Every git and forge argument is derived from the state and the record: the
// branch from the run and lane ids, the pull request from the forge's own
// listing, the body and title from the records, through the scrub.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/assistedby"
	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/embargo"
	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/core/landings"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/relink"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// Landing is the land stage's progress on one lane.
type Landing struct {
	// Closes is whether this lane's landing closes the spec (and ships the
	// intent): the lane that took the fidelity audit.
	Closes bool `json:"closes"`
	// MarkStep is the spec step whose `- landed:` line the records write on
	// the open spec (iss-2610110918418459): the lane's step, on an intent lane
	// of a spec that lists its steps; 0 when there is none to mark.
	MarkStep int `json:"mark_step,omitempty"`
	// Records is the loop's records commit on the lane's branch, and
	// RecordsDone is set once it is made, or once there was nothing to record.
	Records     string `json:"records,omitempty"`
	RecordsDone bool   `json:"records_done"`
	// SpecSettled says the first step took an uncommitted change to the lane's own
	// spec record, so the records commit has that to carry whatever else it holds.
	SpecSettled bool `json:"spec_settled,omitempty"`
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
	// Toggles counts how many times the watch toggled the arming at the pushed
	// head, which maxToggles bounds.
	Toggles int `json:"toggles,omitempty"`
	// Merged is the default branch's tip, as the remote held it, that carried
	// the pushed head when the landing cleaned the lane up.
	Merged string `json:"merged,omitempty"`
	// GateUnreadSince is when the merge gate first could not read what it asks at
	// the arming; set by the call that finds it, and cleared once it reads.
	GateUnreadSince *time.Time `json:"gate_unread_since,omitempty"`
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
)

// landSubject is the records commit's subject.
func landSubject(st State, lane Lane) string {
	return "chore(record): land " + lane.ID + " of " + st.RunID
}

// recordsMessage is the records commit's message: its subject, one line saying
// what the landing did, and its trailers, one to a line.
func recordsMessage(st State, lane Lane, done, trailers []string) string {
	what := fmt.Sprintf("%s (%s, step %d)", st.Intent, st.Spec, lane.SpecStep)
	if iss := st.Issue(); iss != "" {
		what = iss
	}
	return landSubject(st, lane) + "\n\n" +
		fmt.Sprintf("The implement loop's landing for %s: %s.\n\n", what, strings.Join(done, "; ")) +
		strings.Join(trailers, "\n") + "\n"
}

// composedRecordsMessage is the whole message of a records commit under abcd's
// own label, which the run's state alone decides: the step's mark, or the
// admission alone when the landing moves no record.
func composedRecordsMessage(st State, lane Lane) string {
	var done []string
	if ld := lane.Landing; ld != nil {
		if ld.MarkStep > 0 {
			done = append(done, fmt.Sprintf("marked step %d landed at %s", ld.MarkStep, shortSHA(lane.HeadSHA)))
		}
		if recordsOnlyAdmission(*ld, lane) {
			done = append(done, recordsOnlyTheAdmission)
		}
	}
	var trailers []string
	if st.Admitted != nil {
		trailers = admittedTrailers(*st.Admitted, lane.Resolves)
	}
	return recordsMessage(st, lane, done, append(trailers, composedAssistedBy()))
}

// recordsOnlyTheAdmission is the records commit's prose when the landing moves no
// record: the commit is made to carry the admission alone.
const recordsOnlyTheAdmission = "records only the admission"

// recordsOnlyAdmission reports whether the landing moves no record: it marks no
// step, closes nothing, resolves no capture and carries no change to the spec.
func recordsOnlyAdmission(ld Landing, lane Lane) bool {
	return !ld.Closes && ld.MarkStep == 0 && len(lane.Resolves) == 0 && !ld.SpecSettled
}

// admittedTrailers are the records commit's and the pull request's lines for the
// admission: its `Admitted-by:` trailer, and, when it names an issue the
// resolutions do not, a `Refs:` line declaring it, since RS004 refuses a message
// naming an issue no `Resolves:` or `Refs:` line declares.
func admittedTrailers(a Admitted, resolves []Resolution) []string {
	out := []string{a.Trailer()}
	if strings.HasPrefix(a.Record, "iss-") && !slices.ContainsFunc(resolves, func(r Resolution) bool { return r.Issue == a.Record }) {
		out = append(out, "Refs: "+a.Record)
	}
	return out
}

// landStage is the land stage's body: it performs the landing's next step.
func landStage(c Context, lane *Lane) (Outcome, error) {
	if c.State.Trial {
		// A trial lands nothing (trial.go), whatever its lane holds.
		return Outcome{Note: trialLandNote}, nil
	}
	if !gitutil.IsFullSHA(lane.BaseSHA) || !gitutil.IsFullSHA(lane.HeadSHA) || lane.Worktree == "" || lane.Branch == "" {
		return Outcome{}, refuse(string(StageLand), "", lane.ID, "the lane records no branch, worktree, base and head to land",
			"the earlier stages record them; restore the run's state file")
	}
	if err := loopWorktree(c, *lane, string(StageLand)); err != nil {
		return Outcome{}, err
	}
	// The landing pushes the lane's branch and, once it has landed, deletes it;
	// a branch outside the loop's prefix, which only a hand-written state file
	// names, is refused here as the discards refuse it, so a state naming
	// `main` neither pushes nor deletes the default branch
	// (iss-2610090821552801).
	if !strings.HasPrefix(lane.Branch, BranchPrefix) {
		return Outcome{}, refuse(string(StageLand), "", lane.ID, "the lane's branch is not one the loop made, so it is not the loop's to push or delete",
			"restore the run's state file")
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
		due, err := recordsRoundWanted(c, *lane)
		if err != nil {
			return Outcome{}, err
		}
		if due {
			if recordsParked(c, *lane) == "" {
				// The round can take no fix round, so it needs no wider reservation.
				return Outcome{Goto: StageValidate, LandLeft: true, Note: "the landing's records commit " + shortSHA(lane.HeadSHA) + " goes to a review round of its own, before anything is pushed"}, nil
			}
		}
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
	tip, err := gitutil.RunCappedAsStored(c.RepoRoot, maxGitOutput, "rev-parse", "--verify", "--quiet", "refs/heads/"+lane.Branch+"^{commit}", "--")
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
	ownSpec := ""
	if c.State.Issue() == "" {
		ownSpec = c.State.Spec
	}
	dirty, settled, err := worktreeChanges(lane.Worktree, ownSpec)
	if err != nil {
		return Outcome{}, fmt.Errorf("reading the lane's worktree: %w", err)
	}
	if dirty {
		return Outcome{}, refuse(string(StageLand), "", lane.ID, "the lane's worktree holds uncommitted changes, and the landing commits only what it writes",
			"remove them from the lane's worktree (the landing lands only what was validated, and a commit moves the branch off that head), then run `abcd implement step` again")
	}
	if settled {
		if err := refuseChangedCloseRecords(c, *lane, false); err != nil {
			return Outcome{}, err
		}
	}
	closes, err := auditsHere(c, *lane)
	if err != nil {
		return Outcome{}, err
	}
	if closes && passingAudit(*lane) == "" {
		return Outcome{}, refuse(string(StageLand), "", lane.ID, "this lane closes the spec, and its passing round recorded no fidelity audit to ingest at the close",
			"restore the run's state file; the validate stage records the audit on the lane that closes the spec")
	}
	mark, err := stepToMark(c, *lane)
	if err != nil {
		return Outcome{}, err
	}
	ld := &Landing{Closes: closes, MarkStep: mark, SpecSettled: settled}
	var does []string
	if settled {
		does = append(does, "carries the change to "+c.State.Spec+" left in the lane's worktree")
	}
	if mark > 0 {
		does = append(does, fmt.Sprintf("marks step %d landed", mark))
	}
	if closes {
		does = append(does, "closes "+c.State.Spec+" and ships "+c.State.Intent)
	}
	for _, r := range lane.Resolves {
		does = append(does, "resolves "+r.Issue)
	}
	if len(does) == 0 {
		// The records commit is made on every lane, so the rule the lane was
		// admitted under rides it into history (itd-2610301850454860).
		does = append(does, recordsOnlyTheAdmission+" (the lane marks no step, neither closes the spec nor fixed a capture)")
	}
	lane.Landing = ld
	return Outcome{Stay: true, Note: "landing prepared at " + shortSHA(lane.HeadSHA) + ": it " + strings.Join(does, ", ")}, nil
}

// stepToMark is the spec step the landing marks landed: the lane's own, on an
// intent lane whose spec lists its steps on the lane's branch and does not
// already mark it; 0 for an issue lane, an unstepped spec (its one implicit
// step names the spec) or a step marked already. A branch whose spec is not
// open, or lists the lane's step under another title, is refused, as the
// brief refuses it (laneSteps).
func stepToMark(c Context, lane Lane) (int, error) {
	st := c.State
	if st.Issue() != "" || lane.SpecStep < 1 {
		return 0, nil
	}
	store, err := spec.Load(lane.Worktree)
	if err != nil {
		return 0, fmt.Errorf("reading the spec store in the lane's worktree: %w", err)
	}
	sp, ok := store.Lookup(st.Spec)
	if !ok || sp.Status != spec.StatusOpen {
		return 0, refuse(string(StageLand), "", lane.ID, "the lane's branch holds no open "+st.Spec+" to mark its step landed on",
			"restore "+st.Spec+" in the open specs in the lane's worktree and leave it uncommitted, then run `abcd implement step` again")
	}
	steps, err := spec.ReadSteps(lane.Worktree, sp)
	if err != nil {
		return 0, refuse(string(StageLand), "", lane.ID, fmt.Sprintf("%s's steps cannot be read on the lane's branch: %s", st.Spec, fsutil.RedactHome(err.Error())),
			"rewrite "+spec.StepsHeading+" as `abcd intent ready "+st.Intent+"` describes in the lane's worktree and leave it uncommitted, then run `abcd implement step` again")
	}
	if len(steps) == 0 {
		return 0, nil
	}
	n := lane.SpecStep
	if n > len(steps) || steps[n-1].Title != lane.StepTitle {
		return 0, refuse(string(StageLand), "", lane.ID, fmt.Sprintf("%s on the lane's branch does not list step %d as %q, the step this lane was opened for", st.Spec, n, lane.StepTitle),
			"restore "+st.Spec+"'s steps as the run started from them in the lane's worktree and leave it uncommitted, then run `abcd implement step` again")
	}
	if steps[n-1].Landed != "" {
		return 0, nil
	}
	return n, nil
}

// landRecords makes the landing's records in the lane's worktree and commits
// them on the lane's branch, or finds the commit a killed call made.
func landRecords(c Context, lane *Lane) (Outcome, error) {
	st := c.State
	ld := lane.Landing
	// The trailer is written from the run's state and never guessed, so a run that
	// recorded no admission is refused before anything is read or written.
	if st.Admitted == nil {
		return Outcome{}, refuse(string(StageLand), "", lane.ID, "the run's state names no `admitted`, the rule its lanes were admitted under, so its records commit cannot carry its "+landings.TrailerKey+": trailer",
			"resume the run with this abcd, which records the admission its start would have: `abcd build "+st.Key+"` for a run `abcd build` started, or `abcd drain` for a drain's lane (under the rule the drain began under); then run `abcd implement step` again")
	}
	if st.Issue() == "" {
		if why, err := ownSpecTwice(c, *lane); err != nil {
			return Outcome{}, err
		} else if why != "" {
			return closeHandBack(c, *lane, why), nil
		}
	}
	tip, err := branchTip(c, *lane)
	if err != nil {
		return Outcome{}, err
	}
	if tip != lane.HeadSHA {
		// A commit found is read as stored throughout: what stands in for it, its parents
		// or its objects says nothing of what it holds.
		parents, _ := gitutil.RunCappedAsStored(c.RepoRoot, maxGitOutput, "log", "-1", "--format=%P", tip, "--")
		subject, _ := gitutil.RunCappedAsStored(c.RepoRoot, maxGitOutput, "log", "-1", "--format=%s", tip, "--")
		if parents != lane.HeadSHA || subject != landSubject(st, *lane) {
			return Outcome{}, refuse(string(StageLand), "", lane.ID,
				fmt.Sprintf("%s moved to %s, which is not the landing's records commit on the judged head %s", lane.Branch, shortSHA(tip), shortSHA(lane.HeadSHA)),
				"restore the branch to the judged head, then run `abcd implement step` again")
		}
		found, err := gitutil.RunCappedAsStored(c.RepoRoot, maxBaseListing, "diff-tree", "-r", "--raw", "--no-renames", "--no-abbrev", "--ignore-submodules=none", "-z", lane.HeadSHA, tip, "--")
		if err != nil {
			return Outcome{}, fmt.Errorf("reading what %s changes: %w", shortSHA(tip), err)
		}
		if why, err := judgeLandingDiff(c.RepoRoot, st, *lane, found); err != nil {
			return Outcome{}, err
		} else if why != "" {
			return Outcome{}, refuse(string(StageLand), "", lane.ID, shortSHA(tip)+", the commit found on the judged head "+shortSHA(lane.HeadSHA)+", holds "+why+", and "+strayRule,
				"restore the branch to the judged head, then run `abcd implement step` again")
		}
		if err := refuseFoundCloseRecords(c, *lane, tip, ld.Closes); err != nil {
			return Outcome{}, err
		}
		if lacks, err := foundCommitLacks(c, *lane, tip, found); err != nil {
			return Outcome{}, err
		} else if lacks != "" {
			return Outcome{}, refuse(string(StageLand), "", lane.ID, shortSHA(tip)+", the commit found on the judged head "+shortSHA(lane.HeadSHA)+", lacks "+lacks+", so it is not the landing's records",
				"restore the branch to the judged head, then run `abcd implement step` again; the landing makes its records itself")
		}
		// A call killed after its commit and before its state write: found.
		ld.Records, ld.RecordsDone, lane.HeadSHA = tip, true, tip
		return Outcome{Stay: true, Note: "found the landing's records commit " + shortSHA(tip) + " on " + lane.Branch}, nil
	}

	if why, err := strayChanges(st, *lane); err != nil {
		return Outcome{}, fmt.Errorf("reading the lane's worktree: %w", err)
	} else if why != "" {
		return Outcome{}, strayRefusal(*lane, why)
	}

	// What a close ships and what holds it back stay as the judged head has them,
	// whatever markdown the worktree holds; a lane with no close moves no intent or spec.
	if err := refuseChangedCloseRecords(c, *lane, ld.Closes); err != nil {
		return Outcome{}, err
	}
	if st.Issue() == "" {
		if why, err := ownSpecUnresolved(*lane, st.Spec); err != nil {
			return Outcome{}, err
		} else if why != "" {
			return closeHandBack(c, *lane, why), nil
		}
	}
	if ld.Closes {
		if why, err := ownIntentUnresolved(*lane, st.Intent); err != nil {
			return Outcome{}, err
		} else if why != "" {
			return closeHandBack(c, *lane, why), nil
		}
	}
	// A landing prepared over a change to the spec holds it no longer once it is put
	// back: nothing is written yet, so the first step runs again on the tree as it stands.
	if ld.SpecSettled {
		_, settled, err := worktreeChanges(lane.Worktree, st.Spec)
		if err != nil {
			return Outcome{}, fmt.Errorf("reading the lane's worktree: %w", err)
		}
		if !settled {
			lane.Landing = nil
			return Outcome{Stay: true, Note: "the change to " + st.Spec + " the landing's first step took is gone from the lane's worktree, so that step runs again"}, nil
		}
	}

	// The disclosure is settled before any record is written where the landing
	// writes a model's text, so a lane with no model to disclose is refused with its
	// worktree untouched. A records commit that only marks the step carries no
	// model's prose, only the mark abcd computed from the run's state, so it declares
	// abcd's label instead (composed_attribution.go); what is staged settles that
	// below, after the mark is written, and a refusal there leaves the mark in the
	// worktree for the next call.
	if note := rereadModels(c, lane); note != "" {
		return Outcome{Stay: true, Note: note}, nil
	}
	assisted := []string{composedAssistedBy()}
	composed := true
	if ld.Closes || len(lane.Resolves) > 0 || ld.SpecSettled {
		composed = false
		var gap string
		if assisted, gap = assistedByTrailers(lane.Receipts); gap != "" {
			return Outcome{}, modelGap(*lane, "the landing's records commit carries text the lane's implementer composed", gap)
		}
	}

	wt := lane.Worktree
	var trailers, done []string
	if ld.SpecSettled {
		done = append(done, "carried the change to "+st.Spec+" left in the lane's worktree")
	}
	wrote := false
	// The step is marked before the close moves the spec, since only an open spec takes
	// a mark, so a closed spec carries every step's mark. A close refused on what the
	// judged head holds takes the mark back with it: the hand-back leaves the worktree
	// as the call found it.
	takeBackMark := func() {}
	if ld.MarkStep > 0 {
		if ld.Closes {
			takeBackMark = ownSpecRestorer(wt, st.Spec)
		}
		mark := shortSHA(lane.HeadSHA)
		if _, err := spec.MarkLanded(wt, st.Spec, ld.MarkStep, mark); err != nil {
			return Outcome{}, refuse(string(StageLand), "", lane.ID, fmt.Sprintf("step %d of %s could not be marked landed in the lane's worktree: %s", ld.MarkStep, st.Spec, fsutil.RedactHome(err.Error())),
				"settle what the spec store names in the lane's worktree and leave it uncommitted, then run `abcd implement step` again")
		}
		done = append(done, fmt.Sprintf("marked step %d landed at %s", ld.MarkStep, mark))
	}
	if ld.Closes {
		// The close's audit emit parks its receipt under the worktree's local
		// tier; the lane's worktree is the loop's own, so the tier is made there.
		if err := fsutil.EnsureRealDirAll(wt, TierRelDir, dirPerm); err != nil {
			return Outcome{}, fmt.Errorf("making the local tier in the lane's worktree: %w", err)
		}
		// The close ships, so it meets what `spec close` meets: an issue the
		// intent names and leaves unresolved. This lane's own resolutions land in
		// this commit, just below, so they are not refused here. The remedy keeps
		// the settled link uncommitted: a commit moves the branch off the judged
		// head, which refuses above, and this commit carries a markdown change.
		store, err := spec.Load(wt)
		if err != nil {
			return Outcome{}, refuse(string(StageLand), "", lane.ID, "the spec store could not be read to judge what `spec close "+st.Spec+"` ships: "+fsutil.RedactHome(err.Error()),
				"settle what the spec store names in the lane's worktree and leave it uncommitted, then run `abcd implement step` again")
		}
		var ships []string
		if sp, ok := store.Lookup(st.Spec); ok {
			ships = store.Ships(sp)
		}
		links, err := capture.UnresolvedLinks(wt, ships)
		if err != nil {
			return Outcome{}, refuse(string(StageLand), "", lane.ID, "the issue ledger could not be read to judge what `spec close "+st.Spec+"` ships: "+fsutil.RedactHome(err.Error()),
				"settle what the capture store names in the lane's worktree and leave it uncommitted, then run `abcd implement step` again")
		}
		for _, l := range links {
			if !laneResolves(*lane, l.Issue) {
				return Outcome{}, refuse(string(StageLand), "", lane.ID, "`spec close "+st.Spec+"` would ship: "+l.String(),
					"settle the link in the lane's worktree and leave it uncommitted, then run `abcd implement step` again; the landing commits it with its records")
			}
		}
		res, err := intent.Reconcile(wt, st.Spec, "", intent.RemainderRequest{})
		var fact *intent.CloseFactError
		if errors.As(err, &fact) {
			// The worktree says what the judged head says of these, so only a new head
			// could change the answer.
			takeBackMark()
			return closeHandBack(c, *lane, "`spec close "+st.Spec+"` refused what the judged head holds: "+termsafe.Sanitize(fsutil.RedactHome(err.Error()))), nil
		}
		if err != nil {
			return Outcome{}, refuse(string(StageLand), "", lane.ID, "`spec close "+st.Spec+"` refused in the lane's worktree: "+fsutil.RedactHome(err.Error()),
				"settle what the close names in the lane's worktree and leave it uncommitted (an intent ships only with an impact:), then run `abcd implement step` again")
		}
		// The records stand moved, so the next call's close finds them and repoints again.
		if res.RelinkError != "" {
			return Outcome{}, relinkRefusal(*lane, st.Spec+" and "+st.Intent, res.RelinkError)
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
			// A call killed or refused after the move may have left links naming the
			// record where it stood.
			if err := capture.RepointResolved(wt, r.Issue); err != nil {
				return Outcome{}, relinkRefusal(*lane, r.Issue, err.Error())
			}
			done = append(done, "found "+r.Issue+" resolved")
		} else {
			moved, err := capture.Resolve(capture.ResolveRequest{RepoRoot: wt, ID: r.Issue, Resolution: r.Note, Impact: r.Impact,
				ByCommit: r.Commit, Grounds: r.Grounds})
			if err != nil {
				return Outcome{}, refuse(string(StageLand), "", lane.ID, "`capture resolve "+r.Issue+"` refused in the lane's worktree: "+fsutil.RedactHome(err.Error()),
					"the resolution is the lane receipt's; settle what the capture store names in the lane's worktree and leave it uncommitted, then run `abcd implement step` again")
			}
			if moved.RelinkError != "" {
				return Outcome{}, relinkRefusal(*lane, r.Issue, moved.RelinkError)
			}
			wrote = true
			done = append(done, "resolved "+r.Issue+" with "+shortSHA(r.Commit))
		}
		trailers = append(trailers, "Resolves: "+r.Issue)
	}
	trailers = append(trailers, admittedTrailers(*st.Admitted, lane.Resolves)...)
	if beforeLandingStage != nil {
		beforeLandingStage(wt)
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
	// What the commit would hold is judged as staged, so a write during the close meets
	// the rule the top of the call read.
	raw, err := pickGit(wt, stagedRawArgs(lane.HeadSHA)...)
	if err != nil {
		return Outcome{}, fmt.Errorf("reading the landing's staged records: %w", err)
	}
	if why, err := judgeLandingDiff(wt, st, *lane, raw); err != nil {
		return Outcome{}, err
	} else if why != "" {
		return Outcome{}, strayRefusal(*lane, why)
	}
	staged, err := pickGit(wt, "--no-replace-objects", "diff", "--cached", "--no-renames", "--ignore-submodules=none", "--name-only", "-z")
	if err != nil {
		return Outcome{}, fmt.Errorf("reading the landing's staged records: %w", err)
	}
	commit := []string{"commit", "-q"}
	if staged == "" {
		// A resolution written on this call that stages nothing is one the branch
		// already holds byte for byte, and a landing with nothing to move has nothing
		// to stage: either still makes the records commit, changing no file, so the
		// admission rides it.
		switch {
		case recordsOnlyAdmission(*ld, *lane):
			done = append(done, recordsOnlyTheAdmission)
		case wrote:
			done = append(done, "the branch already held the records, so the commit changes no file")
		default:
			return Outcome{}, refuse(string(StageLand), "", lane.ID, "the landing's records changed nothing on the lane's branch",
				"the close and the resolutions should move records; put back anything the branch already moved, or the spec left for the landing to carry, in the lane's worktree and leave it uncommitted, then run `abcd implement step` again")
		}
		commit = append(commit, "--allow-empty")
	}
	// abcd's label covers the mark it computed and nothing else, so it is chosen from
	// what is staged: anything beside the mark names the lane's model.
	if composed && staged != "" {
		only, err := stagedOnlyTheMark(wt, st.Spec, staged, lane.HeadSHA, shortSHA(lane.HeadSHA))
		if err != nil {
			return Outcome{}, fmt.Errorf("reading the landing's staged records: %w", err)
		}
		// A mark is abcd's only when this landing writes it.
		if !only || ld.MarkStep == 0 {
			var gap string
			if assisted, gap = assistedByTrailers(lane.Receipts); gap != "" {
				return Outcome{}, modelGap(*lane, "the landing's records commit carries text abcd did not compose", gap)
			}
			done = append(done, "carried what else was left in the lane's worktree")
		}
	}
	// Unlike the pick commit (pickcommit.go), whose text abcd computes and
	// which declares `Assisted-by: abcd:<version>`, this commit's diff carries prose a
	// model composed: the receipt's resolution note and grounds, and the
	// audit's verdict ingested into the intent. So it names that model, and
	// it is made with the repository's hooks running (never through pickGit's
	// hooks-off configuration), so the commit-msg outbound gate judges it.
	msg := recordsMessage(st, *lane, done, append(trailers, assisted...))
	// gitutil.LocalGit runs a local git command under the developer's own
	// configuration with the repository's hooks running, and none of the
	// networked runner's settings.
	if _, err := gitutil.LocalGit(wt, append(commit, "-m", msg)...); err != nil {
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

// relinkRefusal stops a call whose moves stand with links not repointed: a commit made
// now would carry links that resolve to nothing, and the next call repoints them.
func relinkRefusal(lane Lane, records, why string) error {
	return refuse(string(StageLand), "", lane.ID, "the links to "+records+" could not be repointed in the lane's worktree: "+fsutil.RedactHome(why),
		"settle what it names in the lane's worktree and leave it uncommitted, then run `abcd implement step` again")
}

// stagedOnlyTheMark reports whether what is staged is the lane's own spec record with
// its mode unchanged, and its blob the judged head's plus the one `- landed:` line
// naming mark. staged is the NUL-separated list of staged paths.
func stagedOnlyTheMark(worktree, specID, staged, head, mark string) (bool, error) {
	path, rest, _ := strings.Cut(staged, "\x00")
	if strings.Trim(rest, "\x00") != "" || !isSpecRecordOf(path, specID) {
		return false, nil
	}
	was, err := pickGit(worktree, "--no-replace-objects", "ls-tree", head, "--", path)
	if err != nil {
		return false, err
	}
	now, err := pickGit(worktree, "ls-files", "--stage", "--", ":(literal)"+path)
	if err != nil {
		return false, err
	}
	before, after := strings.Fields(was), strings.Fields(now) // <mode> blob <object> <path>, <mode> <object> <stage> <path>
	if len(before) < 3 || len(after) < 3 || before[0] != after[0] {
		return false, nil
	}
	return addsOnlyTheMark(worktree, before[2], after[1], mark)
}

// addsOnlyTheMark reports whether the second blob is the first plus the one `- landed:`
// line naming mark.
func addsOnlyTheMark(repo, was, now, mark string) (bool, error) {
	blobs, err := gitutil.CatFileBatch(repo, []string{was, now}, 2*(maxSpecRecordBytes+256))
	if err != nil {
		return false, err
	}
	lines := strings.SplitAfter(blobs[1].Content, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "- landed: "+mark && strings.Join(lines[:i], "")+strings.Join(lines[i+1:], "") == blobs[0].Content {
			return true, nil
		}
	}
	return false, nil
}

// worktreeChanges reads the lane's worktree for the first step. settled is an
// uncommitted change to the lane's own spec record, in any bucket: stepToMark reads
// the record there, and a record the judged head leaves unfit is settled only that
// way, since a commit moves the branch off that head. The records commit carries it
// (Landing.SpecSettled), for the round to read. dirty is any other change. ownSpec
// is "" on a lane with no spec, which has no such record.
func worktreeChanges(worktree, ownSpec string) (dirty, settled bool, err error) {
	out, err := pickGit(worktree, "status", "--porcelain", "-z", "--untracked-files=all", "--no-renames")
	if err != nil {
		return false, false, err
	}
	for _, entry := range strings.Split(out, "\x00") {
		if len(entry) < 4 {
			continue
		}
		if ownSpec != "" && isSpecRecordOf(entry[3:], ownSpec) {
			settled = true
			continue
		}
		dirty = true
	}
	return dirty, settled, nil
}

// isSpecRecordOf reports whether the repository-relative path is a record of the
// spec, directly inside one of the store's buckets.
func isSpecRecordOf(path, specID string) bool {
	rest, ok := strings.CutPrefix(path, spec.SpecsRelDir+"/")
	if !ok {
		return false
	}
	bucket, name, ok := strings.Cut(rest, "/")
	if !ok || (bucket != spec.StatusOpen && bucket != spec.StatusClosed) || strings.Contains(name, "/") {
		return false
	}
	id, ok := spec.RecordFileID(name)
	return ok && recordid.SameID(id, specID)
}

// maxSpecRecordBytes bounds the read of a spec record, as the spec store bounds its own.
const maxSpecRecordBytes = 256 * 1024

// closeFields are the frontmatter fields of each store a close decides by: what a spec
// realises, and what an intent's id, back-link, bundle and hold say. Anything else in a
// record is the worktree's to settle, an intent's related_issues among them, and its
// impact where the judged head declares none.
var (
	specCloseFields   = []string{"id", "intent", "intents", "bundle"}
	intentCloseFields = []string{"id", "spec_id", "bundle", "held"}
)

// closeRecord is one record of a store's bucket, under a name the store reads.
type closeRecord struct {
	rel, bucket, name, content string
}

// field is the record's frontmatter value for key.
func (r closeRecord) field(key string) string {
	return frontmatter.Fields(strings.Split(r.content, "\n"))[key].Value
}

// closeFacts is what the record says in fields, each present or absent.
func (r closeRecord) closeFacts(fields []string) string {
	read := frontmatter.Fields(strings.Split(r.content, "\n"))
	var out []string
	for _, key := range fields {
		f, ok := read[key]
		switch {
		case !ok:
			out = append(out, "")
		case key == "intents":
			out = append(out, "="+strings.Join(frontmatter.StringList(f.Value), ","))
		default:
			out = append(out, "="+f.Value)
		}
	}
	return strings.Join(out, "\x00")
}

// realises reports whether a spec record names the intent.
func (r closeRecord) realises(id string) bool {
	if recordid.SameID(r.field("intent"), id) {
		return true
	}
	return slices.ContainsFunc(frontmatter.StringList(r.field("intents")), func(m string) bool { return recordid.SameID(m, id) })
}

// closeMembers are the intent records a close of this spec moves or finds moved, in the
// close's own order: the one intent: names, or for a bundle each member it does not pass
// over (intent.BundleMemberPassedOver).
func (r closeRecord) closeMembers(intents []closeRecord) []closeRecord {
	bundle, listed := r.bundleName(), frontmatter.StringList(r.field("intents"))
	names := []string{r.field("intent")}
	inBundle := bundle != "" && len(listed) > 0
	if inBundle {
		names = append(names, listed...)
	}
	var out []closeRecord
	for _, name := range names {
		for _, i := range intents {
			if !recordid.SameID(i.field("id"), name) || slices.ContainsFunc(out, func(o closeRecord) bool { return o.rel == i.rel }) {
				continue
			}
			if !inBundle || !intent.BundleMemberPassedOver(i.bucket, i.bundleName(), bundle) {
				out = append(out, i)
			}
		}
	}
	return out
}

// bundleName is the record's bundle: "" when it names none.
func (r closeRecord) bundleName() string {
	if b := r.field("bundle"); !frontmatter.IsNull(b) {
		return b
	}
	return ""
}

// closeStore is one store as the close reads it: where it lives, its buckets, the cap
// on a record and the file names it reads.
type closeStore struct {
	rel     string
	buckets []string
	limit   int64
	reads   func(name string) bool
}

var (
	specCloseStore = closeStore{spec.SpecsRelDir, []string{spec.StatusOpen, spec.StatusClosed}, maxSpecRecordBytes, func(name string) bool {
		_, ok := spec.RecordFileID(name)
		return ok
	}}
	intentCloseStore = closeStore{intent.IntentsRelDir, intent.Buckets, intent.MaxFileBytes, intent.IsRecordFile}
	issueStore       = closeStore{capture.LedgerRelPath, issueschema.StatusDirs, 0, func(name string) bool {
		return strings.HasSuffix(name, ".md") && issueFileRe.MatchString(name)
	}}
	issueFileRe = recordid.FilenameNumRe("iss")
)

// treeRecords reads a store's records out of a commit.
func treeRecords(at baseTree, store closeStore) ([]closeRecord, error) {
	entries, err := at.list(store.rel+"/", true)
	if err != nil {
		return nil, fmt.Errorf("reading %s at %s: %w", store.rel, shortSHA(at.sha), err)
	}
	var records []closeRecord
	var objects []string
	budget := 0
	for _, e := range entries {
		bucket, name, ok := strings.Cut(strings.TrimPrefix(e.path, store.rel+"/"), "/")
		if !ok || strings.Contains(name, "/") || !store.reads(name) || !slices.Contains(store.buckets, bucket) {
			continue
		}
		if err := at.readable(e, store.limit); err != nil {
			return nil, fmt.Errorf("reading %s at %s: %w", e.path, shortSHA(at.sha), err)
		}
		records = append(records, closeRecord{rel: e.path, bucket: bucket, name: name})
		objects = append(objects, e.object)
		budget += len(e.object) + 32 + int(e.size)
	}
	got, err := gitutil.CatFileBatch(at.root, objects, budget)
	if err != nil {
		return nil, fmt.Errorf("reading %s at %s: %w", store.rel, shortSHA(at.sha), err)
	}
	for i := range records {
		if got[i].Missing || got[i].Type != "blob" {
			return nil, fmt.Errorf("reading %s at %s: it holds no blob there", records[i].rel, shortSHA(at.sha))
		}
		records[i].content = got[i].Content
	}
	return records, nil
}

// worktreeRecords reads a store's records from the lane's worktree, as the close will.
// odd names one the close could not read.
func worktreeRecords(worktree string, store closeStore) (records []closeRecord, odd string, err error) {
	for _, bucket := range store.buckets {
		dir := filepath.Join(worktree, filepath.FromSlash(store.rel), bucket)
		info, err := os.Lstat(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, "", fmt.Errorf("reading %s/%s in the lane's worktree: %w", store.rel, bucket, err)
		}
		if !info.IsDir() {
			return nil, store.rel + "/" + bucket, nil
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, "", fmt.Errorf("reading %s/%s in the lane's worktree: %w", store.rel, bucket, err)
		}
		for _, e := range entries {
			if e.IsDir() || !store.reads(e.Name()) {
				continue
			}
			rel := store.rel + "/" + bucket + "/" + e.Name()
			info, err := e.Info()
			if err != nil {
				return nil, "", fmt.Errorf("reading %s in the lane's worktree: %w", rel, err)
			}
			if !info.Mode().IsRegular() || info.Size() > store.limit {
				return nil, rel, nil
			}
			body, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				return nil, "", fmt.Errorf("reading %s in the lane's worktree: %w", rel, err)
			}
			records = append(records, closeRecord{rel: rel, bucket: bucket, name: e.Name(), content: string(body)})
		}
	}
	return records, "", nil
}

// closeRefusal says what differs from the judged head, and where.
type closeRefusal func(rel, how string) error

// refuseChangedCloseRecords holds the lane's worktree to the judged head.
func refuseChangedCloseRecords(c Context, lane Lane, closes bool) error {
	refused := func(rel, how string) error {
		return refuse(string(StageLand), "", lane.ID, strconv.Quote(termsafe.Sanitize(rel))+" "+how+" in the lane's worktree, and "+headDecides(c.State),
			"put it back as the judged head has it, removing it if the head holds no such record, in the lane's worktree, then run `abcd implement step` again")
	}
	return judgeCloseRecords(c, lane, closes, refused, func(store closeStore) ([]closeRecord, error) {
		records, odd, err := worktreeRecords(lane.Worktree, store)
		if err == nil && odd != "" {
			return nil, refused(odd, "is not a regular file within its cap")
		}
		return records, err
	})
}

// refuseFoundCloseRecords holds a commit found on the judged head to it.
func refuseFoundCloseRecords(c Context, lane Lane, tip string, closes bool) error {
	refused := func(rel, how string) error {
		return refuse(string(StageLand), "", lane.ID, strconv.Quote(termsafe.Sanitize(rel))+" "+how+" in "+shortSHA(tip)+", the commit found on the judged head "+shortSHA(lane.HeadSHA)+", and "+headDecides(c.State),
			"restore the branch to the judged head, then run `abcd implement step` again")
	}
	return judgeCloseRecords(c, lane, closes, refused, func(store closeStore) ([]closeRecord, error) {
		return treeRecords(baseTree{root: c.RepoRoot, sha: tip}, store)
	})
}

// headDecides says what the comparison holds to the judged head.
func headDecides(st State) string {
	if st.Issue() != "" {
		return "the judged head decides what the spec and intent stores hold, and an issue's landing moves neither"
	}
	return "the judged head decides what the close ships and what holds it"
}

// judgeCloseRecords reads the spec and intent stores as the judged head and read each
// hold them and refuses any difference a close decides by, so nothing uncommitted
// chooses what it ships or lifts what holds it back. The lane's own spec record may
// stand in either bucket. A closing lane's own intent may stand in planned or shipped,
// where a call a hook refused left it, but in shipped only as the close would have
// moved it: the head holds no hold and no other open spec naming it, and the record
// carries a valid impact. A head holding no record of the spec links only the run's own
// intent. An issue's run has no spec of its own, so every record stands as the head has it.
func judgeCloseRecords(c Context, lane Lane, closes bool, refused closeRefusal, read func(closeStore) ([]closeRecord, error)) error {
	st := c.State
	at := baseTree{root: c.RepoRoot, sha: lane.HeadSHA}
	headSpecs, err := treeRecords(at, specCloseStore)
	if err != nil {
		return err
	}
	workSpecs, err := read(specCloseStore)
	if err != nil {
		return err
	}
	var own []closeRecord
	for _, r := range workSpecs {
		if isSpecRecordOf(r.rel, st.Spec) {
			own = append(own, r)
		}
	}
	want := closeRecord{content: "---\nid: " + st.Spec + "\nintent: " + st.Intent + "\n---\n"}
	var otherSpecs []closeRecord
	for _, r := range headSpecs {
		if isSpecRecordOf(r.rel, st.Spec) {
			want = r
			continue
		}
		otherSpecs = append(otherSpecs, r)
	}
	if st.Spec != "" {
		ownChanged := refused(st.Spec, "changes what it links (its record, or its id:, intent:, intents: or bundle:)")
		if len(own) != 1 || want.closeFacts(specCloseFields) != own[0].closeFacts(specCloseFields) {
			return ownChanged
		}
	}
	others := slices.DeleteFunc(slices.Clone(workSpecs), func(r closeRecord) bool { return isSpecRecordOf(r.rel, st.Spec) })
	if err := refuseRecordDifferences(otherSpecs, others, specCloseFields, nil, nil, refused); err != nil {
		return err
	}

	headIntents, err := treeRecords(at, intentCloseStore)
	if err != nil {
		return err
	}
	workIntents, err := read(intentCloseStore)
	if err != nil {
		return err
	}
	moves := func(r, w closeRecord) bool {
		id := r.field("id")
		if !closes || len(own) != 1 || !own[0].realises(id) || !isCloseBucket(r.bucket) || !isCloseBucket(w.bucket) {
			return false
		}
		if w.bucket != intent.BucketShipped {
			return true
		}
		if _, held := frontmatter.Fields(strings.Split(r.content, "\n"))["held"]; held {
			return false
		}
		if slices.ContainsFunc(otherSpecs, func(o closeRecord) bool { return o.bucket == spec.StatusOpen && o.realises(id) }) {
			return false
		}
		return intent.ShipImpactValid(w.content)
	}
	impact := func(r, w closeRecord) string {
		if intent.ShipImpactValid(r.content) && w.field("impact") != r.field("impact") {
			return "changes the impact: the judged head declares"
		}
		return ""
	}
	return refuseRecordDifferences(headIntents, workIntents, intentCloseFields, moves, impact, refused)
}

// isCloseBucket reports whether a close may find a record of its own in the bucket.
func isCloseBucket(bucket string) bool {
	return bucket == intent.BucketPlanned || bucket == intent.BucketShipped
}

// refuseRecordDifferences holds the records read to the judged head's, by file name:
// none new, none gone, none in another bucket unless moves allows it, each saying the
// same in fields, and none differing as also says.
func refuseRecordDifferences(head, work []closeRecord, fields []string, moves func(r, w closeRecord) bool, also func(r, w closeRecord) string, refused closeRefusal) error {
	at := map[string]closeRecord{}
	for _, r := range work {
		if _, twice := at[r.name]; twice {
			return refused(r.rel, "stands in two buckets")
		}
		at[r.name] = r
	}
	held := map[string]bool{}
	for _, r := range head {
		held[r.name] = true
		w, ok := at[r.name]
		switch {
		case !ok:
			return refused(r.rel, "is gone")
		case w.bucket != r.bucket && (moves == nil || !moves(r, w)):
			return refused(r.rel, "moved to "+w.bucket+"/")
		case w.closeFacts(fields) != r.closeFacts(fields):
			return refused(w.rel, "changes its "+strings.Join(fields, ":, ")+":")
		}
		if also != nil {
			if how := also(r, w); how != "" {
				return refused(w.rel, how)
			}
		}
	}
	for _, r := range work {
		if !held[r.name] {
			return refused(r.rel, "is new")
		}
	}
	return nil
}

// ownSpecUnresolved reads the spec store as the close does, once the worktree agrees
// with the judged head, and says why the head leaves the close no way to read the lane's
// own record: "" when it reads it. What the worktree changed in the store, it settles.
func ownSpecUnresolved(lane Lane, specID string) (string, error) {
	store, err := spec.Load(lane.Worktree)
	if err != nil {
		changed, gerr := pickGit(lane.Worktree, "status", "--porcelain", "--", spec.SpecsRelDir)
		if gerr != nil {
			return "", fmt.Errorf("reading the lane's worktree: %w", gerr)
		}
		if changed != "" {
			return "", refuse(string(StageLand), "", lane.ID, "the spec store in the lane's worktree could not be read: "+fsutil.RedactHome(err.Error()),
				"settle what the spec store names in the lane's worktree and leave it uncommitted, then run `abcd implement step` again")
		}
		return "the spec store cannot be read at the judged head: " + termsafe.Sanitize(fsutil.RedactHome(err.Error())), nil
	}
	if sp, ok := store.Lookup(specID); ok && !isSpecRecordOf(filepath.ToSlash(sp.Path), specID) {
		return strconv.Quote(termsafe.Sanitize(filepath.ToSlash(sp.Path))) + " carries the id of " + specID + " at the judged head, and the close would read it in place of the lane's own record", nil
	}
	return "", nil
}

// ownSpecTwice says why the judged head leaves the lane no one record under its spec's
// number, "" when it has one: which of them stands is no choice for the worktree.
func ownSpecTwice(c Context, lane Lane) (string, error) {
	records, err := treeRecords(baseTree{root: c.RepoRoot, sha: lane.HeadSHA}, specCloseStore)
	if err != nil {
		return "", err
	}
	var holding []string
	for _, r := range records {
		if isSpecRecordOf(r.rel, c.State.Spec) {
			holding = append(holding, strconv.Quote(termsafe.Sanitize(r.rel)))
		}
	}
	if len(holding) > 1 {
		return strings.Join(holding, " and ") + " stand under the number of " + c.State.Spec + " at the judged head, and the close reads only one of them", nil
	}
	return "", nil
}

// ownSpecRestorer returns what writes the lane's own spec record back as it stands now.
// What cannot be read is left: the hand-back's way out names any change it leaves.
func ownSpecRestorer(worktree, specID string) func() {
	nothing := func() {}
	store, err := spec.Load(worktree)
	if err != nil {
		return nothing
	}
	sp, ok := store.Lookup(specID)
	if !ok {
		return nothing
	}
	abs := filepath.Join(worktree, sp.Path)
	before, err := os.ReadFile(abs)
	if err != nil {
		return nothing
	}
	return func() { _ = fsutil.WriteFileAtomicPreserveMode(abs, before) }
}

// ownIntentUnresolved reads the intent store as the close does, once the worktree agrees
// with the judged head, and says why the head leaves the close no one record to read for
// the run's intent: "" when it has one.
func ownIntentUnresolved(lane Lane, id string) (string, error) {
	corpus, err := intent.Load(lane.Worktree)
	if err != nil {
		changed, gerr := pickGit(lane.Worktree, "status", "--porcelain", "--", intent.IntentsRelDir)
		if gerr != nil {
			return "", fmt.Errorf("reading the lane's worktree: %w", gerr)
		}
		if changed != "" {
			return "", refuse(string(StageLand), "", lane.ID, "the intent store in the lane's worktree could not be read: "+fsutil.RedactHome(err.Error()),
				"settle what the intent store names in the lane's worktree and leave it uncommitted, then run `abcd implement step` again")
		}
		return "the intent store cannot be read at the judged head: " + termsafe.Sanitize(fsutil.RedactHome(err.Error())), nil
	}
	var holding []string
	for _, it := range corpus.Intents {
		if recordid.SameID(it.ID, id) {
			holding = append(holding, strconv.Quote(termsafe.Sanitize(filepath.ToSlash(it.Path))))
		}
	}
	if len(holding) > 1 {
		return strings.Join(holding, " and ") + " carry the id of " + id + " at the judged head, and the close reads only one of them", nil
	}
	return "", nil
}

// closeHandBack hands the lane back for what the judged head holds against its close:
// no edit in the worktree may settle that, and a lane takes a new head only from a
// sibling's landing.
func closeHandBack(c Context, lane Lane, why string) Outcome {
	st := c.State
	var others []string
	for _, l := range st.Lanes {
		if l.ID != lane.ID && l.Stage != StageDone && l.Stage != StageHandedBack && l.Stage != StageDiscarded {
			others = append(others, l.ID)
		}
	}
	if !ValidRunID(st.RunID) || !ValidLaneID(lane.ID) {
		return Outcome{HandBack: &HandBack{Kind: HandBackStuck, Reason: why,
			Home: "a person: the judged head holds this, and the run's state names a lane the loop did not open, so restore the run's state file"}}
	}
	then := "remove the run's directory " + runRel(st.RunID)
	if len(others) > 0 {
		then = "once `abcd implement status` shows " + strings.Join(others, ", ") + " finished or decided, " + then
	}
	// What a person pastes comes from the ids the loop validates, as the lane's branch did.
	name := st.RunID + "-" + lane.ID
	branch := BranchPrefix + name
	return Outcome{HandBack: &HandBack{Kind: HandBackStuck, Reason: why,
		Home: "a person: the judged head holds this, and this lane takes no new head from the default branch. To keep the lane's commits, save its branch " + branch + " first with `git tag saved-" + name + " " + branch +
			"`: the discard deletes that branch and every commit on it not pushed, and removing the run's directory deletes the record naming them. Then settle it on the default branch, put back anything `git status` shows uncommitted in the lane's worktree, discard the lane with `abcd implement step --run " +
			st.RunID + " --discard " + lane.ID + "`, and " + then + " and build " + st.Intent + " again"}}
}

// recordStores are the stores whose records the landing moves, and where its remedies
// ask for an uncommitted edit.
var recordStores = []closeStore{specCloseStore, intentCloseStore, issueStore}

// storeRecord places a repository-relative path as a record a store reads, directly in
// one of its buckets.
func storeRecord(path string) (store closeStore, bucket, name string, ok bool) {
	for _, st := range recordStores {
		rest, in := strings.CutPrefix(path, st.rel+"/")
		if !in {
			continue
		}
		b, n, in := strings.Cut(rest, "/")
		if in && !strings.Contains(n, "/") && slices.Contains(st.buckets, b) && st.reads(n) {
			return st, b, n, true
		}
	}
	return closeStore{}, "", "", false
}

// strayRule is what the landing commits, in the record stores' modes and outside them.
const strayRule = "the landing commits a record a store reads only with the mode its writes give it, and outside the record stores only the links its own moves repointed"

// strayRefusal names a change the landing does not commit, and how to take it out.
func strayRefusal(lane Lane, why string) error {
	return refuse(string(StageLand), "", lane.ID, "the lane's worktree holds the uncommitted "+why+", and "+strayRule,
		"remove it, or put it back as the judged head has it, in the lane's worktree (the landing repoints its own links itself, lands only what was validated, and a commit moves the branch off that head), then run `abcd implement step` again")
}

// stagedRawArgs is git's list of what the index changes against head, one form for
// every judgement to read, with head's objects as stored.
func stagedRawArgs(head string) []string {
	return []string{"--no-replace-objects", "diff", "--cached", "--raw", "--no-renames", "--no-abbrev", "--no-ext-diff", "--ignore-submodules=none", "-z", head, "--"}
}

// strayChanges judges the lane's worktree as `git add` would stage it, on a copy of the
// lane's index, so the lane's own index and its flags stay as they are and the person's
// filters and ignore files read the tree. "" when nothing in it stops the landing.
func strayChanges(st State, lane Lane) (string, error) {
	wt := lane.Worktree
	own, err := pickGit(wt, "rev-parse", "--git-path", "index")
	if err != nil {
		return "", err
	}
	own = strings.TrimRight(own, "\n")
	if !filepath.IsAbs(own) {
		own = filepath.Join(wt, own)
	}
	data, err := os.ReadFile(own)
	if err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp("", "abcd-land-index-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	index := filepath.Join(tmp, "index")
	if err := os.WriteFile(index, data, 0o600); err != nil {
		return "", err
	}
	if _, err := pickGitIndex(wt, index, "add", "-A", "--", "."); err != nil {
		return "", err
	}
	if _, err := pickGitIndex(wt, index, "reset", "-q", "--", TierRelDir); err != nil {
		return "", err
	}
	raw, err := pickGitIndex(wt, index, stagedRawArgs(lane.HeadSHA)...)
	if err != nil {
		return "", err
	}
	return judgeLandingDiff(wt, st, lane, raw)
}

// parseRawDiff reads git's raw NUL-separated list of changes.
func parseRawDiff(raw string) ([]rawEntry, error) {
	if raw == "" {
		return nil, nil
	}
	fields := strings.Split(strings.TrimSuffix(raw, "\x00"), "\x00")
	if len(fields)%2 != 0 {
		return nil, fmt.Errorf("git's list of what the landing changes does not parse")
	}
	var entries []rawEntry
	for i := 0; i < len(fields); i += 2 {
		f := strings.Fields(fields[i])
		if len(f) != 5 {
			return nil, fmt.Errorf("git's list of what the landing changes does not parse")
		}
		entries = append(entries, rawEntry{oldMode: strings.TrimPrefix(f[0], ":"), newMode: f[1], oldObject: f[2], newObject: f[3], status: f[4], path: fields[i+1]})
	}
	return entries, nil
}

// landingDiff reads what the landing would commit: the moves of the landing's own records
// the list holds, each entry outside what a store reads, and a record whose mode no write
// of the landing's gives it, "" when none is.
func landingDiff(st State, lane Lane, raw string) ([]relink.Move, []rawEntry, string, error) {
	entries, err := parseRawDiff(raw)
	if err != nil {
		return nil, nil, "", err
	}
	from, to, left := map[string]string{}, map[string]string{}, map[string]string{}
	var outside []rawEntry
	for _, e := range entries {
		store, _, name, ok := storeRecord(e.path)
		if !ok {
			outside = append(outside, e)
			continue
		}
		key := store.rel + "/" + name
		if e.status == "D" {
			left[key] = e.oldMode
		}
		if !ownMovedRecord(st, lane, store, name) {
			continue
		}
		switch e.status {
		case "D":
			from[key] = e.path
		case "A":
			to[key] = e.path
		}
	}
	for _, e := range entries {
		store, _, name, ok := storeRecord(e.path)
		if !ok || e.status == "D" {
			continue
		}
		had := e.oldMode
		if e.status == "A" {
			had = left[store.rel+"/"+name]
		}
		if !recordModeHolds(e.newMode, had) {
			return nil, nil, e.path, nil
		}
	}
	var moves []relink.Move
	for key, was := range from {
		if reached, ok := to[key]; ok {
			moves = append(moves, relink.Move{From: was, To: reached})
		}
	}
	return moves, outside, "", nil
}

// recordModeHolds reports whether a record's mode is one a store's write gives it: an
// ordinary file, or the executable bit it already had, which a move and a write that
// preserves the mode both keep.
func recordModeHolds(mode, had string) bool {
	return mode == "100644" || (mode == "100755" && had == "100755")
}

// standingMoves are the moves the judged head already holds of the records this landing
// repoints for: a close and a resolution repoint for a record standing moved, whichever
// commit moved it. A move whose source the head still holds is none, as a repoint reads it.
func standingMoves(repo string, st State, lane Lane) ([]relink.Move, error) {
	at := baseTree{root: repo, sha: lane.HeadSHA}
	var moves []relink.Move
	add := func(from, to string) error {
		held, err := at.list(from, false)
		if err != nil {
			return err
		}
		if len(held) == 0 {
			moves = append(moves, relink.Move{From: from, To: to})
		}
		return nil
	}
	if lane.Landing != nil && lane.Landing.Closes {
		specs, err := treeRecords(at, specCloseStore)
		if err != nil {
			return nil, err
		}
		for _, r := range specs {
			if !isSpecRecordOf(r.rel, st.Spec) {
				continue
			}
			if r.bucket == spec.StatusClosed {
				if err := add(spec.SpecsRelDir+"/"+spec.StatusOpen+"/"+r.name, r.rel); err != nil {
					return nil, err
				}
			}
			intents, err := treeRecords(at, intentCloseStore)
			if err != nil {
				return nil, err
			}
			for _, m := range r.closeMembers(intents) {
				if m.bucket != intent.BucketShipped {
					continue
				}
				if err := add(intent.IntentsRelDir+"/"+intent.BucketPlanned+"/"+m.name, m.rel); err != nil {
					return nil, err
				}
			}
		}
	}
	if len(lane.Resolves) > 0 {
		resolved, err := at.list(capture.LedgerRelPath+"/"+string(capture.StateResolved)+"/", false)
		if err != nil {
			return nil, err
		}
		for _, e := range resolved {
			name := e.path[strings.LastIndex(e.path, "/")+1:]
			if !slices.ContainsFunc(lane.Resolves, func(r Resolution) bool { return strings.HasPrefix(name, r.Issue+"-") }) {
				continue
			}
			if err := add(capture.LedgerRelPath+"/"+string(capture.StateOpen)+"/"+name, e.path); err != nil {
				return nil, err
			}
		}
	}
	return moves, nil
}

// rawEntry is one line of git's raw diff.
type rawEntry struct {
	oldMode, newMode, oldObject, newObject, status, path string
}

// judgeLandingDiff judges one list of what the landing would commit against the judged
// head: a record a store reads is the reader's in what it says and keeps a mode its writes
// give it, and any other path must keep its mode and hold exactly what the repoint of the
// landing's own moves writes over the head's file, those in this same list and those the
// head already holds. It says which path fails, "" when none does. Everything it needs is
// in the list and the object store, so a call after a refusal or a kill reads the same
// answer.
func judgeLandingDiff(repo string, st State, lane Lane, raw string) (string, error) {
	moves, outside, odd, err := landingDiff(st, lane, raw)
	if err != nil {
		return "", err
	}
	if odd != "" {
		return strconv.Quote(termsafe.Sanitize(odd)), nil
	}
	if len(outside) > 0 {
		standing, err := standingMoves(repo, st, lane)
		if err != nil {
			return "", err
		}
		moves = append(moves, standing...)
	}
	for _, e := range outside {
		says := strconv.Quote(termsafe.Sanitize(e.path))
		if e.status != "M" || e.oldMode != e.newMode || (e.newMode != "100644" && e.newMode != "100755") {
			return says, nil
		}
		blobs, err := gitutil.CatFileBatch(repo, []string{e.oldObject, e.newObject}, 2*(maxRepointedBytes+256))
		if err != nil || len(blobs) != 2 || blobs[0].Missing || blobs[1].Missing {
			return says, nil
		}
		if blobs[1].Content != relink.Rewritten(e.path, blobs[0].Content, moves) {
			return says, nil
		}
	}
	return "", nil
}

// foundCommitLacks says what of the landing's own a commit found on its judged head does
// not hold, "" when it holds it all: the lane's moves, its step's mark, the audit on the
// intent's record and the trailers the landing writes. Such a commit is the landing's
// records, and any other leaves its close and resolutions unrun. raw is git's list of what
// it changes.
func foundCommitLacks(c Context, lane Lane, tip, raw string) (string, error) {
	st, ld := c.State, lane.Landing
	// Only a landing that moves no spec record may change no file: one with nothing to
	// move, or one whose resolutions the branch already held, each checked below.
	if raw == "" && (ld.Closes || ld.MarkStep > 0 || ld.SpecSettled) {
		return "any change", nil
	}
	at := baseTree{root: c.RepoRoot, sha: tip}
	specs, err := treeRecords(at, specCloseStore)
	if err != nil {
		return "", err
	}
	var own *closeRecord
	for i := range specs {
		if isSpecRecordOf(specs[i].rel, st.Spec) {
			own = &specs[i]
		}
	}
	if (ld.Closes || ld.MarkStep > 0) && own == nil {
		return "the record of " + st.Spec, nil
	}
	if ld.Closes && own.bucket != spec.StatusClosed {
		return "the close of " + st.Spec, nil
	}
	if ld.MarkStep > 0 {
		steps, err := spec.ParseSteps(own.content)
		if err != nil || ld.MarkStep > len(steps) || steps[ld.MarkStep-1].Landed != shortSHA(lane.HeadSHA) {
			return fmt.Sprintf("the mark of step %d", ld.MarkStep), nil
		}
	}
	delivers := false
	if ld.Closes {
		intents, err := treeRecords(at, intentCloseStore)
		if err != nil {
			return "", err
		}
		members := own.closeMembers(intents)
		for _, r := range members {
			id := r.field("id")
			kept := slices.ContainsFunc(specs, func(o closeRecord) bool { return o.bucket == spec.StatusOpen && o.realises(id) })
			if !kept && r.bucket != intent.BucketShipped {
				return "the move of " + termsafe.Sanitize(id) + " to shipped/", nil
			}
		}
		delivers = len(members) > 0 && members[0].bucket == intent.BucketShipped
		verdict, err := intent.ReadVerdict(abs(c.RepoRoot, passingAudit(lane)))
		if err != nil {
			return "the audit on the record of " + st.Intent + ", whose verdict cannot be read", nil
		}
		holds := false
		for _, r := range intents {
			if !recordid.SameID(r.field("id"), st.Intent) {
				continue
			}
			on, err := intent.VerdictOnRecord(lane.Worktree, st.Intent, r.content, verdict)
			if err != nil {
				return "", fmt.Errorf("reading the audit on %s: %w", st.Intent, err)
			}
			holds = holds || on
		}
		if !holds {
			return "the audit on the record of " + st.Intent, nil
		}
	}
	if len(lane.Resolves) > 0 {
		resolved, err := at.list(capture.LedgerRelPath+"/"+string(capture.StateResolved)+"/", false)
		if err != nil {
			return "", err
		}
		for _, r := range lane.Resolves {
			if !slices.ContainsFunc(resolved, func(e baseEntry) bool {
				name := e.path[strings.LastIndex(e.path, "/")+1:]
				return strings.HasPrefix(name, r.Issue+"-") || name == r.Issue+".md"
			}) {
				return "the issue " + r.Issue + " resolved", nil
			}
		}
	}
	return foundTrailersLack(c, lane, tip, raw, delivers)
}

// foundTrailersLack says which trailers a commit found on the judged head lacks, "" when
// the ones it carries are the ones the landing writes: its delivery, each resolution and
// its disclosure, which is abcd's only on a commit of the mark alone.
func foundTrailersLack(c Context, lane Lane, tip, raw string, delivers bool) (string, error) {
	st, ld := c.State, lane.Landing
	var want []string
	if delivers {
		want = append(want, "Delivers: "+st.Intent)
	}
	for _, r := range lane.Resolves {
		want = append(want, "Resolves: "+r.Issue)
	}
	if st.Admitted != nil {
		want = append(want, admittedTrailers(*st.Admitted, lane.Resolves)...)
	}
	composed := !ld.Closes && len(lane.Resolves) == 0 && !ld.SpecSettled
	if composed && raw != "" {
		only, err := rawOnlyTheMark(c.RepoRoot, st.Spec, raw, shortSHA(lane.HeadSHA))
		if err != nil {
			return "", err
		}
		// As landRecords judges it: a mark is abcd's only when the landing writes it.
		composed = only && ld.MarkStep > 0
	}
	if composed {
		want = append(want, composedAssistedBy())
	} else {
		assisted, gap := assistedByTrailers(lane.Receipts)
		if gap != "" {
			return "a model to disclose: " + gap, nil
		}
		want = append(want, assisted...)
	}
	slices.Sort(want)
	want = slices.Compact(want)
	msg, err := gitutil.RunCappedAsStored(c.RepoRoot, maxGitOutput, "log", "-1", "--format=%B", tip, "--")
	if err != nil {
		return "", fmt.Errorf("reading the message of %s: %w", shortSHA(tip), err)
	}
	if !slices.Equal(landingTrailers(msg), want) {
		return "the landing's trailers (" + termsafe.Sanitize(strings.Join(want, ", ")) + ")", nil
	}
	// landingTrailers reads a line wherever it stands; the admission counts only
	// in the trailer block, read through the grammar that wrote it.
	if a := st.Admitted; a != nil {
		if got, ok := landings.ParseAdmitted(msg); !ok || got.Kind != a.Kind || got.Record != a.Record {
			return "its admission (" + termsafe.Sanitize(a.Trailer()) + ") in its trailer block", nil
		}
	}
	// A commit under abcd's label carries text abcd composed and nothing else, so
	// its whole message is the one the run's state decides, not its trailers alone.
	if composed {
		norm := func(s string) string { return strings.TrimRight(strings.ReplaceAll(s, "\r\n", "\n"), "\n") }
		if norm(msg) != norm(composedRecordsMessage(st, lane)) {
			return "the message abcd composes for it under its own label", nil
		}
	}
	return "", nil
}

// landingTrailerRe is a line git or a gate could read as one of the trailers the landing
// writes.
var landingTrailerRe = regexp.MustCompile(`(?i)^(admitted-by|assisted-by|delivers|refs|resolves)\s*:`)

// landingTrailers lists a commit message's lines of those trailers, wherever in it they
// stand and with a folded value unfolded, sorted and without repeats.
func landingTrailers(msg string) []string {
	var out []string
	folding := false
	for line := range strings.SplitSeq(msg, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case landingTrailerRe.MatchString(trimmed):
			out = append(out, trimmed)
			folding = true
		case folding && trimmed != "" && (line[0] == ' ' || line[0] == '\t'):
			out[len(out)-1] += " " + trimmed
		default:
			folding = false
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// rawOnlyTheMark is stagedOnlyTheMark over git's raw list of what a commit changes.
func rawOnlyTheMark(repo, specID, raw, mark string) (bool, error) {
	entries, err := parseRawDiff(raw)
	if err != nil {
		return false, err
	}
	if len(entries) != 1 || entries[0].status != "M" || entries[0].oldMode != entries[0].newMode || !isSpecRecordOf(entries[0].path, specID) {
		return false, nil
	}
	return addsOnlyTheMark(repo, entries[0].oldObject, entries[0].newObject, mark)
}

// maxRepointedBytes bounds the read of a file the landing's moves repointed; a repoint
// reads no larger a file.
const maxRepointedBytes = 4 << 20

// ownMovedRecord reports whether the landing moves this record of the store: its own
// spec, an intent a close ships (the comparison holds which), or an issue it resolves.
func ownMovedRecord(st State, lane Lane, store closeStore, name string) bool {
	switch store.rel {
	case spec.SpecsRelDir:
		id, ok := spec.RecordFileID(name)
		return ok && recordid.SameID(id, st.Spec)
	case intent.IntentsRelDir:
		return lane.Landing != nil && lane.Landing.Closes
	default:
		return slices.ContainsFunc(lane.Resolves, func(r Resolution) bool {
			return strings.HasPrefix(name, r.Issue+"-") || name == r.Issue+".md"
		})
	}
}

// recordsRoundWanted reports whether the lane's head is a records commit no round has
// read and the mode is auto, where the receipt that arms a head rests on a pair that
// read it. It reads the default branch as last fetched and opens no connection: the
// push's own fence says what a lost origin means, and a mode changed since is read at
// the arming, where a head no pair read is left open.
func recordsRoundWanted(c Context, lane Lane) (bool, error) {
	if !lane.recordsRoundDue() {
		return false, nil
	}
	def, err := defaultBranch(c, lane)
	if err != nil {
		return false, err
	}
	tip, err := gitutil.Run(c.RepoRoot, "rev-parse", "--verify", "--quiet", Remote+"/"+def+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(tip) {
		return false, nil
	}
	roots, _ := layered.RootsFor(c.RepoRoot)
	return landing.AutoAt(roots, c.RepoRoot, tip, Remote+"/"+def), nil
}

// recordsParked says why the lane's records commit goes to a person rather than to a
// round, "" when it does not: it moves an issue's record the full round was not told of,
// as that round's own state keeps what it was shown, or it holds more than records
// (landing.RecordsCommitShape). It holds only on the lane's head.
func recordsParked(c Context, lane Lane) string {
	if lane.Landing == nil || lane.Landing.Records == "" || lane.Landing.Records != lane.HeadSHA || len(lane.Validation) == 0 {
		return ""
	}
	last := lane.Validation[len(lane.Validation)-1]
	if last.HeadSHA == lane.HeadSHA {
		return ""
	}
	why, err := landing.RecordsCommitShape(c.RepoRoot, last.HeadSHA, lane.HeadSHA, last.ResolvesShown)
	if err != nil {
		return termsafe.Sanitize(fsutil.RedactHome(err.Error()))
	}
	return termsafe.Sanitize(why)
}

// recordsRoundDue reports whether the lane's head is the landing's records commit and
// its latest round read another head.
func (l Lane) recordsRoundDue() bool {
	n := len(l.Validation)
	return n > 0 && l.Landing != nil && l.Landing.Records != "" && l.Landing.Records == l.HeadSHA && l.Validation[n-1].HeadSHA != l.HeadSHA
}

// readsRecords reports whether r is the round that reads the landing's records commit.
func (l Lane) readsRecords(r ValidationRound) bool {
	return l.Landing != nil && l.Landing.Records != "" && r.HeadSHA == l.Landing.Records
}

// recordsBase is the head the round before the records commit judged.
func (l Lane) recordsBase(r ValidationRound) string {
	for i := len(l.Validation) - 1; i >= 0; i-- {
		if h := l.Validation[i].HeadSHA; h != r.HeadSHA {
			return h
		}
	}
	return diffBase(l)
}

// resolutionsSection lists what a full round's brief asks it to judge: each issue the
// lane's receipts declare fixed, by which commit and in what words, which the landing
// then records as resolved. "" for a lane that declares none, and for the records round.
func resolutionsSection(lane Lane, r ValidationRound) string {
	if len(lane.Resolves) == 0 || lane.readsRecords(r) {
		return ""
	}
	var b strings.Builder
	for _, res := range lane.Resolves {
		fmt.Fprintf(&b, "- %s, by %s: %s\n", res.Issue, shortSHA(res.Commit), termsafe.Sanitize(res.Note))
	}
	return "## What the landing will record as resolved\n\nThe lane's receipts declare each of these fixed, and the landing may resolve these and no others. Judge each: a commit that does not fix its issue is a finding, whoever's record it is.\n\n" + fenced("text", b.String()) + "\n\n"
}

// recordsRoundScope is what a round over the records commit is told to read.
func recordsRoundScope(base, head string) string {
	return fmt.Sprintf("## This round reads the landing's records commit alone\n\n"+
		"The lane's own change passed its full round at `%s`, which judged each resolution its receipt declared. Read only `%s..%s` (`git "+readerDiff+" %s..%s`), the commit the landing added on it; this should take a couple of minutes, so do not read the lane again or judge the fix again. "+
		"Judge the records' text against what the lane says it did: each resolution's note, impact and grounds, and the folder each record lands in. "+
		"This commit normally holds records and link rewrites, and anything else in it is a finding. "+
		"It can also carry a change to the lane's own spec, left in its worktree for the landing: a change to that spec beyond its `- landed:` line and its folder must be named and justified in your verdict, or it is a finding. "+
		"A verdict that does not pass parks the lane for a person; no fix round follows.\n\n", base, base, head, base, head)
}

// bareClaudeRe is a bare Claude model id, which takes the Claude vendor prefix.
// A value already in the vendor form the attribution gate takes is judged by
// assistedby.IsModelValue, the gate's TRAILER_RE in its one Go home.
var bareClaudeRe = regexp.MustCompile(`^claude-[A-Za-z0-9._-]+(\[[A-Za-z0-9._-]+\])?$`)

// assistedByTrailers are the records commit's Assisted-by: trailers, one per
// distinct model the lane's receipts reported, in the order first reported.
// Every receipt's runner may have composed text the commit carries, so a
// receipt that reports no model, or one in no form the trailer takes, is a gap
// named in the returned description, and no trailers are returned; a lane with
// no receipt at all is a gap too. So is a receipt naming abcd as its model: the
// abcd label says abcd composed the text from record facts (ruling PC1), which
// is never true of a runner's prose. The model is the runner's report, which the
// binary cannot verify: a refused value is described, never quoted.
func assistedByTrailers(rs []ReceiptRecord) ([]string, string) {
	if len(rs) == 0 {
		return nil, "the lane has no implementer's receipt to report a model"
	}
	var out []string
	for i, r := range rs {
		v, problem := modelTrailer(r.Model)
		if problem != "" {
			return nil, fmt.Sprintf("receipt %d (%s) %s", i+1, r.Receipt, problem)
		}
		if t := "Assisted-by: " + v; !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out, ""
}

// modelTrailer is the Assisted-by: value a receipt's model names, or, when it
// names none, what is wrong with it: no model, abcd's own label, or a value in
// no form the trailer takes (described, never quoted).
func modelTrailer(model string) (string, string) {
	switch {
	case model == "":
		return "", "reports no model"
	case assistedby.NamesAbcd(model):
		return "", "reports abcd as its model, a label reserved for text abcd composes from record facts"
	case bareClaudeRe.MatchString(model):
		return "Claude:" + model, ""
	case assistedby.IsModelValue(model):
		return model, ""
	}
	return "", "reports a model in no form the trailer takes (" + termsafe.DescribeRefused(model) + ")"
}

// nameable reports whether model is one the trailer can name.
func nameable(model string) bool {
	_, problem := modelTrailer(model)
	return problem == ""
}

// rereadModels replaces the model of each receipt the lane recorded with none
// the trailer can name (iss-2611012309408370): with the model the runner that
// ran it reported, when that one is nameable, else with the model the receipt
// file the record points at reports, read again, when that one is nameable and
// differs. So a receipt corrected after it verified reaches the landing without
// anyone editing the run's state. Only the model is taken from the file;
// everything else stays as it verified. It returns the run log's note naming
// each receipt it changed, "" when it changed none; a receipt it could not
// mend is still refused by assistedByTrailers.
func rereadModels(c Context, lane *Lane) string {
	var filled []string
	var root *os.Root
	defer func() {
		if root != nil {
			root.Close()
		}
	}()
	for i := range lane.Receipts {
		r := &lane.Receipts[i]
		if nameable(r.Model) {
			continue
		}
		if r.Route != nil && r.Route.Model != "" && plainWritten(r.Route.Model) && nameable(r.Route.Model) {
			r.Model = r.Route.Model
			filled = append(filled, r.Receipt+" from the route that ran it")
			continue
		}
		if root == nil {
			var err error
			if root, err = os.OpenRoot(c.RepoRoot); err != nil {
				return ""
			}
		}
		rc, err := readReceipt(c.RepoRoot, root, r.Receipt, lane.ID)
		if err != nil || rc.Model == r.Model || !plainWritten(rc.Model) || !nameable(rc.Model) {
			continue
		}
		r.Model = rc.Model
		filled = append(filled, r.Receipt+" from its file, read again")
	}
	if len(filled) == 0 {
		return ""
	}
	return "took the model of " + strings.Join(filled, "; ") + ", since the receipt verified with none the trailer can name"
}

// modelGap is the refusal of a landing step whose text must name the models
// the lane's receipts reported, when gap (assistedByTrailers') says one cannot
// be named. what says why the step's text discloses a model.
func modelGap(lane Lane, what, gap string) error {
	return refuse(string(StageLand), "", lane.ID,
		what+", and "+gap+", so its Assisted-by: trailer cannot name the model",
		"have the implementer's receipt report the model its harness runs (\"model\": \"<vendor>:<model-id>\", or a bare claude-* id), "+
			"correct the receipt file so it reports the model (the landing reads a receipt's model again from its file), then run `abcd implement step` again; the loop never claims no assistance for a model's text")
}

// laneResolves reports whether the lane's receipts declared issue fixed, so the
// landing resolves it in the commit that closes the spec.
func laneResolves(lane Lane, issue string) bool {
	for _, r := range lane.Resolves {
		if r.Issue == issue {
			return true
		}
	}
	return false
}

// resolvedIn reports whether issue is already in the worktree's resolved/.
func resolvedIn(wt, issue string) bool {
	m, _ := filepath.Glob(filepath.Join(wt, ".abcd", "work", "issues", "resolved", issue+"-*.md"))
	return len(m) > 0
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
	rcp, err := landing.Receipt(c.RepoRoot, lane.HeadSHA)
	if rf := tierRefusal(string(StageLand), lane.ID, err); rf != nil {
		return Outcome{}, rf
	}
	if err != nil {
		return Outcome{}, err
	}
	if rcp != "" && c.receiptOwn != nil && !c.receiptOwn(lane.HeadSHA) {
		rcp = ""
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
	// The reservation is checked before the push as well as at the arming
	// (spc-2610161044461448 step 7): one that moved to another while the
	// lane worked pushes nothing, so it never becomes a duplicate pull
	// request; and a release another session is cutting holds the push, as
	// it holds `abcd land`'s (C7).
	holds, notice, err := pushFence(c, *lane)
	if err != nil {
		return Outcome{}, err
	}
	// Before the push, "merging <the lane's record>" is posted on the
	// person's status branch, through the code `abcd announce` runs, and the
	// push waits for it to reach origin (iss-2610161645224876,
	// dcn-2610170655202547); one that cannot be published refuses the push,
	// nothing pushed (dcn-2610170704189712), and the next step tries again.
	// A record not yet public is passed over, and the note says so.
	notices, err := announceBeforePush(c.RepoRoot, claim.PushItems([]string{lane.Key}), announceOptions(c.Now))
	if err != nil {
		return Outcome{}, refuse(string(StageLand), "", lane.ID,
			gitutil.ScrubURLCredentials(fsutil.RedactHome(err.Error()))+", so "+lane.Branch+" was not pushed: the landing announces before it pushes",
			"settle what it names, then run `abcd implement step` again; the branch was not pushed, and an announcement already sent stands")
	}
	// The head the receipt names is pushed by its commit id, not the branch's
	// ref, so a branch that moves after the check cannot change what is pushed.
	if _, err := gitutil.Net(c.RepoRoot, "push", "--porcelain", Remote, lane.HeadSHA+":refs/heads/"+lane.Branch); err != nil {
		if r := netFailure(StageLand, lane.ID, "git push "+lane.Branch, err); r != nil {
			return Outcome{}, r
		}
		return Outcome{}, refuse(string(StageLand), "", lane.ID, "git could not push "+lane.Branch+" to "+Remote+": "+gitutil.ScrubURLCredentials(fsutil.RedactHome(err.Error())),
			"settle what git or the pre-push hook reports, then run `abcd implement step` again")
	}
	note := "pushed " + lane.Branch + " at " + shortSHA(lane.HeadSHA) + " to " + Remote + " on its preflight receipt"
	if ld.CheckWaitSince != nil {
		note += fmt.Sprintf(", after waiting %d minute(s) for its full check", int(c.Now.Sub(*ld.CheckWaitSince)/time.Minute))
	}
	if !holds.IsZero() {
		note += "; the reservation on " + c.State.Key + " holds until " + stamp(holds)
	}
	if notice != "" {
		note += "; " + notice
	}
	for _, n := range notices {
		note += "; " + n.Line()
	}
	ld.PreflightReceipt, ld.Pushed, ld.CheckWaitSince = fsutil.RedactHome(rcp), lane.HeadSHA, nil
	return Outcome{Stay: true, Note: note}, nil
}

// announceBeforePush is claim.AnnounceBeforePush, the one path abcd's pushes
// announce by; a seam the package's tests replace, since most of them land
// with no status branch to post to.
var announceBeforePush = claim.AnnounceBeforePush

// announceOptions are the options the announcement before a lane's push is
// made with: the step's clock, and the embargo check `abcd announce` holds an
// announcement to.
func announceOptions(now time.Time) claim.Options {
	o := claimOptions(now)
	o.Embargo = embargo.AnnouncementCheck
	return o
}

// forge runs the forge client with args from the checkout the run lives in
// (landing.GH), refused when the client is not installed.
func forge(c Context, lane Lane, args ...string) (string, error) {
	out, err := landing.GH(c.RepoRoot, args...)
	if err != nil {
		return "", forgeRefusal(lane, args, err)
	}
	return out, nil
}

// forgeRefusal is a forge client failure as the loop refuses it.
func forgeRefusal(lane Lane, args []string, err error) error {
	if errors.Is(err, landing.ErrNoForgeClient) {
		return refuse(string(StageLand), "", lane.ID, "the forge client `gh` is not installed, and the landing opens the pull request through it",
			"install gh and sign in with your own identity (`gh auth login`), then run `abcd implement step` again")
	}
	if err != nil {
		what := "gh " + strings.Join(args[:min(2, len(args))], " ")
		if r := netFailure(StageLand, lane.ID, what, err); r != nil {
			return r
		}
		return refuse(string(StageLand), "", lane.ID, gitutil.ScrubURLCredentials(fsutil.RedactHome(err.Error())),
			"settle what the forge client reports, then run `abcd implement step` again")
	}
	return nil
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
// records. The body ends with its trailers: the Delivers: and Resolves: lines
// the records commit carries, then the abcd label (abcd composed the text from
// record facts, as it does the pick commit's), then assisted, the Assisted-by:
// lines assistedByTrailers derives from the lane's receipts, since the change
// the body describes carries those models' work and a squash merge may adopt
// the body as its message (ruling R4, the technical facilitator, 2026-10-02).
func prTitle(st State, lane Lane) string {
	if iss := st.Issue(); iss != "" {
		return fmt.Sprintf("fix(%s): %s", iss, lane.StepTitle)
	}
	return fmt.Sprintf("build(%s): %s, step %d of %s", st.Intent, lane.StepTitle, lane.SpecStep, st.Spec)
}

func prBody(st State, lane Lane, assisted []string) string {
	return prBodyParked(st, lane, assisted, false)
}

// prBodyParked is prBody, saying the lane is parked for a person when no round read its
// records commit. What it was parked over stays in the run's own record.
func prBodyParked(st State, lane Lane, assisted []string, parked bool) string {
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
		if slices.ContainsFunc(r.Validators, func(v ValidatorRun) bool { return !v.Pass }) {
			p("- Validation round %d at %s did not pass (%s), so the lane is parked at this pull request for a person.\n", r.Round, shortSHA(r.HeadSHA), verdictsLine(r))
		} else {
			p("- Validation round %d at %s passed: %s.\n", r.Round, shortSHA(r.HeadSHA), verdictsLine(r))
		}
		if parked {
			p("- No round read its records commit %s, so the lane is parked at this pull request for a person.\n", shortSHA(lane.HeadSHA))
		}
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
	if st.Admitted != nil {
		trailers = append(trailers, admittedTrailers(*st.Admitted, lane.Resolves)...)
	}
	trailers = append(append(trailers, composedAssistedBy()), assisted...)
	p("\n%s\n", strings.Join(trailers, "\n"))
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
		// The disclosure is settled before the forge is called. The records
		// commit settled it already on a lane that records something; a lane
		// that records nothing made no records commit, so it is settled here.
		if note := rereadModels(c, lane); note != "" {
			return Outcome{Stay: true, Note: note}, nil
		}
		assisted, gap := assistedByTrailers(lane.Receipts)
		if gap != "" {
			return Outcome{}, modelGap(*lane, "the pull request's body describes a change carrying text the lane's implementer composed", gap)
		}
		due, err := recordsRoundWanted(c, *lane)
		if err != nil {
			return Outcome{}, err
		}
		body, _, err := scanner.ScrubOutbound(c.RepoRoot, prBodyParked(c.State, *lane, assisted, due && recordsParked(c, *lane) != ""), "the pull request's body")
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

// readMergeRule reads the ruleset mirror at the lane's base through the
// landing's one reader of the mirror (landing.ReadMergeRule), its refusals
// worded for the loop: the merge queue's method ("" when none gates the
// default branch), whether an active ruleset on the default branch requires a
// person's approval (ruling AM1), and whether a mirror exists at all, which
// decides the land fence's lease. A missing mirror requires nothing. The base
// is the default branch the lane was cut from, so the lane's own commits
// cannot change the rule it lands by.
func readMergeRule(c Context, lane Lane, def string) (landing.MergeRule, error) {
	r, err := landing.ReadMergeRule(c.RepoRoot, lane.BaseSHA, def, "the lane's base")
	var le *landing.ListError
	var me *landing.MirrorError
	switch {
	case errors.As(err, &le):
		return landing.MergeRule{}, fmt.Errorf("listing the ruleset mirror at the lane's base: %v", le.Err)
	case errors.As(err, &me):
		return landing.MergeRule{}, refuse(string(StageLand), "", lane.ID, me.Reason, me.Fix+", then run `abcd implement step` again")
	case err != nil:
		return landing.MergeRule{}, err
	}
	return r, nil
}

// gateUnreadBound is how long a merge gate that cannot read what it asks is asked
// again before its lane is left open.
const gateUnreadBound = 30 * time.Minute

// landArm asks the merge gate (landing.DecideMerge, the one decision `abcd
// land` asks too, spc-2610161609350616 step 4) whether to arm the lane's pull
// request, and arms it by the method the gate names, or leaves it open with
// the gate's reason: a contributor's pull request until a maintainer's
// approval of its head counts, a maintainer's in human mode until the
// technical facilitator's does, and in auto until a review receipt names it,
// which the loop records from a pair that read that head (recordOwnReview). A gate
// that could not read what it asks leaves the lane at this step, to ask again.
// A lane left open is parked: landMerged asks the gate again on each step, so
// an approval given later arms it. A pull request the forge reports
// conflicting is refused, unarmed.
func landArm(c Context, lane *Lane) (Outcome, error) {
	ld := lane.Landing
	def, err := defaultBranch(c, *lane)
	if err != nil {
		return Outcome{}, err
	}
	rule, err := readMergeRule(c, *lane, def)
	if err != nil {
		return Outcome{}, err
	}
	// The land fence (spc-2610031753359849 spec 2): before the rule decides,
	// the run's reservation must still be this person's from this copy, and
	// it is renewed for fenceLease, a day.
	holds, notice, err := landFence(c, *lane, fenceLease)
	if err != nil {
		return Outcome{}, err
	}
	held := ""
	if !holds.IsZero() {
		held = "; the reservation on " + c.State.Key + " holds until " + stamp(holds)
	}
	if notice != "" {
		held += "; " + notice
	}
	n := strconv.Itoa(lane.PR)
	// A pull request the forge already reports conflicting never enters the
	// queue, and nothing is pushed to a lane after arming: it is refused here,
	// unarmed, naming what conflicts (iss-2610110908443341). A verdict the
	// forge has not computed yet arms as before; the merged check reads it.
	m, err := prMergeability(c, *lane)
	if err != nil {
		return Outcome{}, err
	}
	if m.conflicting() {
		tracking, err := fetchDefault(c, *lane, def)
		if err != nil {
			return Outcome{}, err
		}
		// An issue lane enters its sync again (sync.go, step 13).
		if c.State.Issue() != "" {
			return resyncLane(c, lane, def, tracking, m)
		}
		return Outcome{}, conflictRefusal(c, *lane, def, tracking, m)
	}
	d, err := askMergeGate(c, *lane, def, rule)
	if err != nil {
		return Outcome{}, err
	}
	if d.Unread && (ld.GateUnreadSince == nil || c.Now.Sub(*ld.GateUnreadSince) < gateUnreadBound) {
		// What the gate could not read says nothing of the lane: it stays at this
		// step, and a later turn asks again, where a left-open lane would be
		// stepped only when a person acts. An answer that cannot change is passed
		// over once the bound has run out, as a lane left open.
		r := contend(string(StageLand), "", lane.ID,
			"the merge gate could not read what it asks of pull request #"+n+": "+d.Why, "run `abcd implement step` again once the forge answers")
		r.gateUnread = true
		return Outcome{}, r
	}
	ld.GateUnreadSince = nil
	if !d.Arm {
		ld.Merge = "left open: " + d.Why
		return Outcome{Stay: true, Note: "pull request #" + n + " " + ld.Merge + "; the loop asks again on each step, and it lands when it is armed or a person merges it" + held}, nil
	}
	return armLane(c, lane, def, rule, d, held)
}

// askMergeGate asks the merge gate about the lane's pull request: the mode as
// the default branch's tip just fetched and the machine layer declare it,
// never the lane's checkout (dcn-2610230058333177), with the ruleset mirror's
// rule at the lane's base. A mode that cannot be read is refused, masked,
// and nothing is armed.
func askMergeGate(c Context, lane Lane, def string, rule landing.MergeRule) (landing.MergeDecision, error) {
	tip, err := defaultTip(c, lane, def)
	if err != nil {
		return landing.MergeDecision{}, err
	}
	roots, _ := layered.RootsFor(c.RepoRoot)
	unrecorded := recordOwnReview(c, lane, roots, tip, Remote+"/"+def)
	d, err := landing.DecideMerge(landing.MergeQuery{RepoRoot: c.RepoRoot, Roots: roots, Tip: tip, TipName: Remote + "/" + def, Default: def,
		Rule: rule, Repo: "{owner}/{repo}", PR: strconv.Itoa(lane.PR), Head: lane.Landing.Pushed, Forge: gateForge(c, lane)})
	if err != nil {
		r := refuse(string(StageLand), "", lane.ID, termsafe.Sanitize(fsutil.RedactHome(err.Error()))+" at "+Remote+"/"+def+", so nothing was armed",
			"correct the file it names, then run `abcd implement step` again")
		r.modeUnread = true
		return landing.MergeDecision{}, r
	}
	if !d.Arm && unrecorded != "" {
		d.Why += "; the loop did not record its own review: " + unrecorded
	}
	return d, nil
}

// defaultTip is the default branch's tip, just fetched.
func defaultTip(c Context, lane Lane, def string) (string, error) {
	tracking, err := fetchDefault(c, lane, def)
	if err != nil {
		return "", err
	}
	tip, err := gitutil.Run(c.RepoRoot, "rev-parse", "--verify", "--quiet", tracking+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(tip) {
		return "", fmt.Errorf("resolving %s: %v", tracking, err)
	}
	return tip, nil
}

// recordOwnReview writes the review receipt of a pair that read the pushed head,
// where the mode at the default branch's tip is auto, so a lane whose reviewers passed
// needs nobody to say so: in process, by RecordReview's rule, from the returns of the
// lane's latest round. It says why when it does not; the merge gate then reads the
// receipts as it finds them, a person's among them.
func recordOwnReview(c Context, lane Lane, roots layered.Roots, tip, tipName string) string {
	if !landing.AutoAt(roots, c.RepoRoot, tip, tipName) {
		return ""
	}
	if driveStopped(c.State.RunID) {
		return "a drive stopped on this run because its state changed in an agent's call"
	}
	n := len(lane.Validation)
	if n == 0 {
		return "the lane records no review round"
	}
	round := lane.Validation[n-1]
	head := lane.Landing.Pushed
	if round.HeadSHA != head {
		if why := recordsParked(c, lane); why != "" {
			return "the records commit goes to a person before any round: " + why
		}
		return fmt.Sprintf("its latest round read %s, not the pushed head %s", shortSHA(round.HeadSHA), shortSHA(head))
	}
	var returns []landing.ReviewReturn
	for _, v := range round.Validators {
		if v.Role == RoleAuditor {
			continue
		}
		text, err := readReturn(c.RepoRoot, v.Return, lane.ID)
		if err != nil {
			return termsafe.Sanitize(fsutil.RedactHome(err.Error()))
		}
		if !v.Pass {
			return fmt.Sprintf("round %d's %s said %s at %s (%s)%s", round.Round, v.Role, v.Verdict, shortSHA(head), v.Return, firstFinding(string(text)))
		}
		returns = append(returns, landing.ReviewReturn{Name: v.Return, Text: string(text)})
	}
	if _, err := landing.RecordReview(c.RepoRoot, head, returns); err != nil {
		return termsafe.Sanitize(fsutil.RedactHome(err.Error()))
	}
	return ""
}

// failedRecordsRound says which verdict the round that read the lane's records commit
// gave, "" when that round passed or there was none.
func failedRecordsRound(c Context, lane Lane) string {
	n := len(lane.Validation)
	if n == 0 || !lane.readsRecords(lane.Validation[n-1]) {
		return ""
	}
	r := lane.Validation[n-1]
	for _, v := range r.Validators {
		if !v.Pass {
			text, err := readReturn(c.RepoRoot, v.Return, lane.ID)
			finding := ""
			if err == nil {
				finding = firstFinding(string(text))
			}
			return fmt.Sprintf("round %d's %s said %s of the landing's records (%s)%s", r.Round, v.Role, v.Verdict, v.Return, finding)
		}
	}
	return ""
}

// firstFinding is the first item of a return's findings, as a clause for a note, ""
// when it lists none the loop can read.
func firstFinding(text string) string {
	at := findingsHeading.FindStringIndex(text)
	if at == nil {
		return ""
	}
	for _, line := range strings.Split(text[at[1]:], "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			return ""
		}
		if item, ok := strings.CutPrefix(strings.TrimSpace(line), "- "); ok {
			if r := []rune(item); len(r) > maxFindingRunes {
				item = string(r[:maxFindingRunes]) + "..."
			}
			return ": " + termsafe.Sanitize(fsutil.RedactHome(item))
		}
	}
	return ""
}

// findingsHeading is a return's Findings heading, at any depth.
var findingsHeading = regexp.MustCompile(`(?m)^[ \t]{0,3}#{1,6}[ \t]+Findings[ \t]*\r?$`)

// maxFindingRunes bounds what a note quotes of a finding.
const maxFindingRunes = 200

// gateForge is the forge client as the merge gate reads through it: run from
// the checkout the run lives in (gh resolves {owner}/{repo} there, as it does
// for the loop's pull request), its failures returned as they are, so the gate
// can tell a 404 from a read it could not make.
func gateForge(c Context, lane Lane) landing.ForgeCall {
	return func(args ...string) (string, error) { return landing.GH(c.RepoRoot, args...) }
}

// armLane arms the lane's pull request by the gate's method, pinned to the
// pushed head: the forge refuses it when the pull request's head is another
// commit. With no queue the arming is the forge's auto-merge, which a
// repository can leave switched off: that refusal is said, naming it.
func armLane(c Context, lane *Lane, def string, rule landing.MergeRule, d landing.MergeDecision, held string) (Outcome, error) {
	ld := lane.Landing
	n := strconv.Itoa(lane.PR)
	args := []string{"pr", "merge", n, "--auto", d.Method, "--match-head-commit=" + ld.Pushed}
	if _, err := landing.GH(c.RepoRoot, args...); err != nil {
		if rule.Method == "" && !errors.Is(err, landing.ErrNoForgeClient) {
			return Outcome{}, refuse(string(StageLand), "", lane.ID,
				"no merge queue gates "+def+", and the forge refused to arm pull request #"+n+"'s auto-merge: "+gitutil.ScrubURLCredentials(termsafe.Sanitize(fsutil.RedactHome(err.Error()))),
				"allow auto-merge in the repository's settings, or merge it yourself, then run `abcd implement step` again")
		}
		return Outcome{}, forgeRefusal(*lane, args, err)
	}
	ld.Armed = true
	ld.Merge = d.Why
	return Outcome{Stay: true, Note: "pull request #" + n + ": " + ld.Merge + "; nothing is pushed to the lane after this" + held}, nil
}

// landMerged waits for the pushed head to be an ancestor of the default branch
// as the remote holds it, then removes the lane's worktree and branch. An open
// pull request the forge reports conflicting is refused, not waited on.
func landMerged(c Context, lane *Lane) (Outcome, error) {
	ld := lane.Landing
	def, err := defaultBranch(c, *lane)
	if err != nil {
		return Outcome{}, err
	}
	tracking, on, err := landing.Landed(gitutil.Net, c.RepoRoot, def, ld.Pushed)
	if fe := (*landing.FetchError)(nil); errors.As(err, &fe) {
		return Outcome{}, fetchRefusal(*lane, def, fe.Err)
	} else if err != nil {
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
		// An open pull request the forge reports conflicting never merges on
		// its own: it is refused for a person, not waited on
		// (iss-2610110908443341). Any other verdict, UNKNOWN included (the
		// forge computes it lazily), is the merge's ordinary wait.
		m, err := prMergeability(c, *lane)
		if err != nil {
			return Outcome{}, err
		}
		if m.conflicting() {
			// An issue lane enters its sync again (sync.go, step 13).
			if c.State.Issue() != "" {
				return resyncLane(c, lane, def, tracking, m)
			}
			return Outcome{}, conflictRefusal(c, *lane, def, tracking, m)
		}
		// A pull request left open is asked about again: an approval of its
		// head, or a review receipt naming it, given since arms it now, and
		// the reason it waits is brought up to date otherwise.
		if !ld.Armed {
			rule, err := readMergeRule(c, *lane, def)
			if err != nil {
				return Outcome{}, err
			}
			d, err := askMergeGate(c, *lane, def, rule)
			if err != nil {
				return Outcome{}, err
			}
			if d.Arm {
				return armLane(c, lane, def, rule, d, "")
			}
			ld.Merge = "left open: " + d.Why
		}
		// An armed pull request a merge queue gates is watched in the queue
		// (queue.go): queued when it sits CLEAN outside it, and reported,
		// naming the failed check, when it left it disarmed.
		queued := ""
		if ld.Armed {
			rule, err := readMergeRule(c, *lane, def)
			if err != nil {
				return Outcome{}, err
			}
			if rule.Method != "" {
				out, wait, err := watchQueue(c, lane, def, rule)
				if err != nil {
					return Outcome{}, err
				}
				if out != nil {
					return *out, nil
				}
				queued = wait
			} else {
				wait, err := watchAutoMerge(c, *lane)
				if err != nil {
					return Outcome{}, err
				}
				queued = wait
			}
		}
		return Outcome{}, contend(string(StageLand), "", lane.ID,
			"pull request #"+n+" is not merged yet: "+shortSHA(ld.Pushed)+" is not on "+Remote+"/"+def+" ("+m.note()+")"+queued,
			"run `abcd implement step` again once it has merged; "+ld.Merge)
	}
	merged, err := retireLanded(context.Background(), c, *lane, tracking, string(StageLand), "run `abcd implement step` again")
	if err != nil {
		return Outcome{}, err
	}
	ld.Merged = merged
	note := fmt.Sprintf("pull request #%s landed: %s is on %s/%s at %s; removed the lane's worktree and branch %s",
		n, shortSHA(ld.Pushed), Remote, def, shortSHA(merged), lane.Branch)
	// The run's reservation is tidied once its last lane has landed
	// (reservation.go); the lane branch removal above touches only build/.
	if tidy := tidyReservation(c, *lane); tidy != "" {
		note += "; " + tidy
	}
	return Outcome{Note: note}, nil
}

// retireLanded removes a landed lane's worktree and branch once tracking, the
// default branch's remote-tracking ref, holds what the lane pushed; it returns
// tracking's commit. The branch is deleted only at a tip tracking holds: a
// commit added after the push is work that did not land, and is refused, as a
// branch another checkout or a stopped rebase or bisect holds is, and a
// worktree git will not remove (it refuses one with changes). Nothing is
// removed before every refusal has been asked. The land stage's last step
// retires a lane this way (landMerged), with no deadline, and so does a build
// that resumes a run whose lane merged (closeMergedLanes), whose ctx bounds each
// ref read, the worktree listing, the derivation of the lane's worktree and the
// branch's deletion. The worktree's removal is never bound to ctx: git killed
// part-way through deleting a worktree leaves it half removed, which no rerun
// settles. stage labels each refusal and again says what to run once it is
// settled.
func retireLanded(ctx context.Context, c Context, lane Lane, tracking, stage, again string) (string, error) {
	merged, err := gitutil.RunCappedContext(ctx, c.RepoRoot, 4096, "rev-parse", "--verify", "--quiet", tracking+"^{commit}", "--")
	if merged = strings.TrimSpace(merged); err != nil || !gitutil.IsFullSHA(merged) {
		return "", fmt.Errorf("resolving %s: %v", tracking, err)
	}
	ref := "refs/heads/" + lane.Branch
	// A branch git does not name a commit for is gone, as the land step has
	// always read it: there is nothing of it to delete.
	tip, tipErr := gitutil.RunCappedContext(ctx, c.RepoRoot, 4096, "rev-parse", "--verify", "--quiet", ref+"^{commit}", "--")
	if tip = strings.TrimSpace(tip); tipErr != nil || !gitutil.IsFullSHA(tip) {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", fmt.Errorf("resolving the lane branch %s: %w", lane.Branch, ctxErr)
		}
		tip = ""
	}
	if tip != "" {
		_, notOn, err := gitutil.RunCappedExitOneContext(ctx, c.RepoRoot, 4096, "merge-base", "--is-ancestor", tip, tracking)
		if err != nil || notOn {
			pushed := ""
			if lane.Landing != nil {
				pushed = lane.Landing.Pushed
			}
			return "", refuse(stage, "", lane.ID,
				lane.Branch+" carries "+shortSHA(tip)+", past what landed, and the loop never deletes work that did not land",
				"land or move that work, reset the branch to "+shortSHA(pushed)+", then "+again)
		}
		if err := branchCheckedOutElsewhereContext(ctx, c, lane, stage, again); err != nil {
			return "", err
		}
	}
	if err := removeLaneWorktreeAt(ctx, c, lane, stage, again); err != nil {
		return "", err
	}
	if tip != "" {
		if _, err := gitutil.RunCappedContext(ctx, c.RepoRoot, 4096, "update-ref", "-d", ref, tip); err != nil {
			return "", refuse(stage, "", lane.ID, "git could not delete "+lane.Branch+": "+fsutil.RedactHome(err.Error()),
				"settle what git reports, then "+again)
		}
	}
	return merged, nil
}

// fetchDefault fetches the default branch def from the remote into its
// remote-tracking ref (landing.FetchDefault), and returns the ref.
func fetchDefault(c Context, lane Lane, def string) (string, error) {
	tracking, err := landing.FetchDefault(gitutil.Net, c.RepoRoot, def)
	if fe := (*landing.FetchError)(nil); errors.As(err, &fe) {
		return "", fetchRefusal(lane, def, fe.Err)
	}
	return tracking, err
}

// fetchRefusal is a fetch of the default branch def that git refused: a lost
// network waited out on the shared probe, anything else a refusal.
func fetchRefusal(lane Lane, def string, err error) error {
	if r := netFailure(StageLand, lane.ID, "git fetch "+def, err); r != nil {
		return r
	}
	return refuse(string(StageLand), "", lane.ID, "git could not fetch "+def+" from "+Remote+": "+gitutil.ScrubURLCredentials(fsutil.RedactHome(err.Error())),
		"settle what git reports, then run `abcd implement step` again")
}

// branchCheckedOutElsewhere refuses, before anything is removed, when git
// lists a worktree other than the lane's own on the lane's branch, naming it.
// The branch is deleted with `git update-ref -d`, which does not ask, as `git
// branch -d` does, whether a checkout stands on it: deleting it there would
// leave that checkout's HEAD naming nothing (the main checkout after a person
// removed the clean lane worktree and checked the branch out, or a second
// worktree a person added on it), or under a rebase or bisect stopped in any
// worktree, the lane's own included, which git lists detached
// (gitutil.HeldBranches; a state file it cannot read refuses too, since the
// branch it names is unknown). The landing, the held discard and the
// hand-back discard each delete the branch that way, so each asks first; stage
// names the refusing stage and again what to run once the checkout is moved.
func branchCheckedOutElsewhere(c Context, lane Lane, stage, again string) error {
	return branchCheckedOutElsewhereContext(context.Background(), c, lane, stage, again)
}

// branchCheckedOutElsewhereContext is branchCheckedOutElsewhere with its git
// bound to ctx.
func branchCheckedOutElsewhereContext(ctx context.Context, c Context, lane Lane, stage, again string) error {
	wts, err := gitutil.ListWorktreesContext(ctx, c.RepoRoot, gitutil.MaxWorktreeListing)
	if err != nil {
		return fmt.Errorf("listing the repository's worktrees: %w", err)
	}
	own := fsutil.RealExistingPath(lane.Worktree)
	for _, wt := range wts {
		if !sameBranchRef(wt.Branch, lane.Branch) || fsutil.RealExistingPath(wt.Path) == own {
			continue
		}
		at := termsafe.Sanitize(fsutil.RedactHome(wt.Path))
		return refuse(stage, "", lane.ID,
			lane.Branch+" is checked out in "+at+", a checkout the loop did not make for this lane, and deleting the branch would leave its HEAD naming nothing",
			"check out another branch in "+at+" (or remove that worktree), then "+again)
	}
	// git lists a worktree whose rebase or bisect stopped as detached, yet
	// counts the branch it started from, and each branch a rebase's
	// --update-refs will rewrite, as in use there: deleted under it, the
	// person's `rebase --continue` cannot write the branch back.
	common, err := gitutil.CommonDirContext(ctx, c.RepoRoot)
	if err != nil {
		return fmt.Errorf("naming the repository's common dir: %w", err)
	}
	held := gitutil.HeldBranches(common)
	if held.Unreadable != "" {
		at := termsafe.Sanitize(fsutil.RedactHome(held.Unreadable))
		return refuse(stage, "", lane.ID,
			"a rebase or bisect state file at "+at+" could not be read, so whether "+lane.Branch+" is checked out under a stopped rebase or bisect is unknown",
			"finish or abort that rebase or bisect (or remove the file if git did not write it), then "+again)
	}
	// Each names the checkout (gitutil.Held.Checkout), never git's admin dir
	// for it, whose id need not be the checkout's folder name.
	for b, dir := range held.Branches {
		if sameBranchRef("refs/heads/"+b, lane.Branch) {
			at := fsutil.RedactHome(held.Checkout(dir, wts))
			return refuse(stage, "", lane.ID,
				lane.Branch+" is checked out in "+at+", where a rebase or bisect stopped on it, and deleting the branch would leave that rebase or bisect unable to write it back",
				"finish or abort the rebase or bisect in that checkout, then "+again)
		}
	}
	for b, dir := range held.Rewrites {
		if sameBranchRef("refs/heads/"+b, lane.Branch) {
			at := fsutil.RedactHome(held.Checkout(dir, wts))
			return refuse(stage, "", lane.ID,
				"a rebase stopped in "+at+" will rewrite "+lane.Branch+" (its --update-refs list names the branch), and deleting the branch would leave that rebase unable to write it back",
				"finish or abort the rebase in that checkout, then "+again)
		}
	}
	return nil
}

// removeLaneWorktree removes the lane's worktree when git lists it at the lane's
// path on the lane's branch; git refuses a worktree with changes, and the loop
// never forces it. git judges the changes as the person's own git does
// (pickGit), through the filters and ignore files their global config names
// (iss-2610111729375108).
func removeLaneWorktree(c Context, lane Lane, again string) error {
	return removeLaneWorktreeAt(context.Background(), c, lane, string(StageLand), again)
}

// removeLaneWorktreeAt is removeLaneWorktree with each refusal labelled stage
// and the derivation of the lane's worktree and the worktree listing bound to
// ctx. The removal itself is not: it runs as the land stage has always run it,
// to its end, since git killed part-way through deleting a worktree leaves a
// half-removed one that reads as one with changes.
func removeLaneWorktreeAt(ctx context.Context, c Context, lane Lane, stage, again string) error {
	if err := loopWorktreeContext(ctx, c, lane, stage); err != nil {
		return err
	}
	wts, err := gitutil.ListWorktreesContext(ctx, c.RepoRoot, gitutil.MaxWorktreeListing)
	if err != nil {
		return fmt.Errorf("listing the repository's worktrees: %w", err)
	}
	want := fsutil.RealExistingPath(lane.Worktree)
	for _, wt := range wts {
		if fsutil.RealExistingPath(wt.Path) != want {
			continue
		}
		if wt.Branch != "refs/heads/"+lane.Branch {
			return refuse(stage, "", lane.ID, "git lists a worktree at the lane's path on another branch, and the loop removes only what it made",
				"remove it yourself, then "+again)
		}
		if _, err := pickGit(c.RepoRoot, "worktree", "remove", "--", wt.Path); err != nil {
			return refuse(stage, "", lane.ID, "git could not remove the lane's worktree: "+fsutil.RedactHome(err.Error()),
				"settle what git reports (it refuses a worktree with changes), then "+again)
		}
		return nil
	}
	if _, err := os.Lstat(lane.Worktree); err == nil {
		return refuse(stage, "", lane.ID, fsutil.RedactHome(lane.Worktree)+" stands where git lists no worktree, and the loop removes only what it made",
			"move it aside, then "+again)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("checking the lane's worktree: %w", err)
	}
	return nil
}

// caseFoldingRefs reports whether the platform's filesystem folds case, held
// as a var so a test can provoke either answer on any host.
var caseFoldingRefs = fsutil.CaseFoldingFS

// sameBranchRef reports whether listed, a full ref git lists for a worktree,
// names the branch branch. On a filesystem that folds case and Unicode
// normalisation, git resolves a loose ref however it is spelled, so a state
// naming build/Lane-1, an NFD spelling of an NFC name, or build/strasse for
// build/stra\u00dfe reads and deletes the branch. Such a filesystem (APFS) folds
// with full Unicode case folding over normalised text, which a simple
// lower-case misses, so the names are compared through refFold there.
func sameBranchRef(listed, branch string) bool {
	want := "refs/heads/" + branch
	if !caseFoldingRefs() {
		return listed == want
	}
	return refFold(listed) == refFold(want)
}

// refFold is a ref name's key on a filesystem that folds case: full Unicode
// case folding, then NFC. A Caser is not safe to share, so each call makes
// its own.
func refFold(s string) string {
	return norm.NFC.String(cases.Fold().String(norm.NFC.String(s)))
}

// beforeLandingStage is a test seam, nil outside tests: called just before the landing
// stages what it wrote, so a test can change the worktree during the close.
var beforeLandingStage func(worktree string)
