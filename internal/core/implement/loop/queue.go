package loop

// queue.go is the landing's watch over the merge queue (spc-2609212015054359
// step 5, iss-2610090642376032). The landing arms a lane's pull request and
// waits for its head to reach the default branch, and the forge could leave
// it there unseen: twice an armed pull request sat CLEAN and never entered
// the queue (#837, #844; the forge's enqueue call failed, and only toggling
// auto-merge queued it), and four times a failed merge-group check dropped
// one from the queue, which also cleared its arming (#839, #854, #867, #869,
// and #1011's failed_checks). So each wait of an armed lane whose default
// branch a merge queue gates reads the pull request's place in the queue:
//
//   - in the queue: the ordinary wait, naming its place;
//   - at a head other than the one pushed, armed or queued, with origin's branch
//     moved too: the arming is withdrawn and refused for a person. A reading
//     origin does not bear out is a wait, the arming left as it is;
//   - armed, CLEAN and outside the queue: the landing's fence and the merge
//     gate are asked, as at the arming, and the pull request is queued, pinned
//     to the pushed head; when the forge refuses that, auto-merge is toggled
//     once its arming is past requeueGrace, at most maxToggles times at one head,
//     and otherwise the wait says when.
//     A head the gate refuses has its arming withdrawn and is left open, for
//     the gate to be asked again on the next step; a gate it could not read is a
//     wait, the arming left as it is;
//   - neither armed nor queued nor merged: a terminal state. It is refused
//     for a person, naming the forge's reason for the removal and the checks
//     that failed on the pull request's last merge group, never re-queued
//     into the same failure. A release being cut withdraws armed merges the
//     same way and re-arms them after, so while its hold stands the lane
//     waits on it instead.
//
// The forge's answers are untrusted text: every name and reason is
// sanitised and bounded, the merge group's commit must be a full object name
// before it is put in a request path, and the pull request's node id goes to
// the forge as a raw string variable, never read as a flag or a file.

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// QueueStateQuery is the forge's GraphQL query for an armed pull request's
// place in the merge queue: `gh pr view --json` has no merge-queue field, so
// it is read through the forge's API, as `abcd land` reads it. It reads the
// node id the enqueue call names, the head (the queue and a re-arm are pinned
// to it), whether it is queued, the arming and its age, the merge state, and
// the last removal from the queue with the forge's reason. Every field is
// pinned in internal/ghcontract.
const QueueStateQuery = `query($o:String!,$r:String!,$n:Int!){repository(owner:$o,name:$r){pullRequest(number:$n){id state headRefOid isInMergeQueue mergeQueueEntry{state position} autoMergeRequest{mergeMethod enabledAt} mergeStateStatus timelineItems(last:1,itemTypes:[REMOVED_FROM_MERGE_QUEUE_EVENT]){nodes{... on RemovedFromMergeQueueEvent{createdAt reason}}}}}}`

// queueStateArgs are the forge client's arguments for QueueStateQuery on
// pull request n of the repository gh resolves in the checkout: the owner and
// the name are gh's own placeholders, as the merge gate's REST reads name
// them, and n goes as a number.
func queueStateArgs(n string) []string {
	return []string{"api", "graphql", "-f", "query=" + QueueStateQuery, "-F", "o={owner}", "-F", "r={repo}", "-F", "n=" + n}
}

// mergeGroupRunsPath lists the repository's latest workflow runs the merge
// queue started, newest first, and checkRunsPath the check runs on one
// commit: the merge group's, where its failed check is named. Both are plain
// GETs, as the merge gate's reads are.
const mergeGroupRunsPath = "repos/{owner}/{repo}/actions/runs?event=merge_group&per_page=100"

func checkRunsPath(sha string) string {
	return "repos/{owner}/{repo}/commits/" + sha + "/check-runs?per_page=100"
}

// requeueGrace is how long the landing's arming may sit CLEAN outside the
// merge queue, the forge having refused to queue it, before the watch toggles
// it: the forge also queues an armed pull request by itself after a wait (a
// 15-minute pickup on #1018), which a toggle would only churn. It is
// `abcd land`'s own grace (landing's toggleGrace), counted from the arming.
const requeueGrace = 30 * time.Minute

// maxToggles bounds how many times the watch toggles one lane's arming at one
// pushed head: an enqueue the forge goes on refusing is not asked for without end.
const maxToggles = 3

// The bounds on forge text a refusal or a note carries.
const (
	maxQueueReason   = 160
	maxFailedChecks  = 8
	maxCheckNameText = 80
)

// queueView is the forge's answer to QueueStateQuery.
type queueView struct {
	ID              string `json:"id"`
	State           string `json:"state"`
	HeadRefOid      string `json:"headRefOid"`
	IsInMergeQueue  bool   `json:"isInMergeQueue"`
	MergeQueueEntry *struct {
		State    string `json:"state"`
		Position int    `json:"position"`
	} `json:"mergeQueueEntry"`
	AutoMergeRequest *struct {
		MergeMethod string `json:"mergeMethod"`
		EnabledAt   string `json:"enabledAt"`
	} `json:"autoMergeRequest"`
	MergeStateStatus string `json:"mergeStateStatus"`
	TimelineItems    struct {
		Nodes []struct {
			CreatedAt string `json:"createdAt"`
			Reason    string `json:"reason"`
		} `json:"nodes"`
	} `json:"timelineItems"`
}

// queued reports whether the forge holds the pull request in the merge queue.
func (v queueView) queued() bool { return v.IsInMergeQueue || v.MergeQueueEntry != nil }

// gqlAnswer is a GraphQL answer's errors, which may come beside its data:
// gh exits 0 on some of them, so they are read, never left to gh.
type gqlAnswer struct {
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (a gqlAnswer) text() string {
	var msgs []string
	for _, e := range a.Errors {
		msgs = append(msgs, e.Message)
	}
	t := bounded(strings.Join(msgs, "; "), maxQueueReason)
	if t == "" {
		return "an error with no message"
	}
	return t
}

// bounded is forge text made safe for a record: credentials scrubbed,
// home-redacted, sanitised, cut to max runes.
func bounded(s string, max int) string {
	s = strings.TrimSpace(termsafe.Sanitize(fsutil.RedactHome(gitutil.ScrubURLCredentials(s))))
	if r := []rune(s); len(r) > max {
		s = string(r[:max]) + "…"
	}
	return s
}

// forgeWhy is a forge client failure as a clause: its message, bounded.
func forgeWhy(err error) string {
	return bounded(err.Error(), maxQueueReason)
}

// readQueue asks the forge for the lane's pull request's place in the merge
// queue. unread says why the answer could not be had, the watch then waiting
// as before; a missing forge client and a lost network are refused as every
// forge call of the landing refuses them.
func readQueue(c Context, lane Lane) (v queueView, unread string, err error) {
	args := queueStateArgs(strconv.Itoa(lane.PR))
	out, err := landing.GH(c.RepoRoot, args...)
	if errors.Is(err, landing.ErrNoForgeClient) {
		return v, "", forgeRefusal(lane, args, err)
	}
	if err != nil {
		if r := netFailure(StageLand, lane.ID, "gh api graphql", err); r != nil {
			return v, "", r
		}
		return v, forgeWhy(err), nil
	}
	var a struct {
		gqlAnswer
		Data struct {
			Repository *struct {
				PullRequest *queueView `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}
	if json.Unmarshal([]byte(out), &a) != nil {
		return v, "the forge's answer does not parse", nil
	}
	if len(a.Errors) > 0 {
		return v, "the forge answered with errors: " + a.text(), nil
	}
	if a.Data.Repository == nil || a.Data.Repository.PullRequest == nil {
		return v, "the forge reports no such pull request", nil
	}
	return *a.Data.Repository.PullRequest, "", nil
}

// fenceQueue is the land fence's act before the watch queues or toggles an
// armed pull request.
var fenceQueue = fenceAct{not: "queue its pull request", none: "the pull request is not queued",
	stays: "the pull request stays armed outside the queue"}

// fenceDropped is the merge hold's act before the watch reports a pull
// request gone from the queue: a release being cut withdraws armed merges
// and re-arms them after, so the lane waits on it rather than report a drop.
var fenceDropped = fenceAct{not: "report its pull request dropped from the merge queue", none: "nothing is reported",
	stays: "the pull request stays outside the queue for the release's hold to re-arm"}

// watchQueue is the merge-queue half of an armed lane's wait for its merge,
// where a queue gates def. It returns the step's outcome when it acted (the
// pull request queued, or its arming toggled), or wait, a clause the ordinary
// wait adds; a pull request gone from the queue and no longer armed is
// refused, naming why.
func watchQueue(c Context, lane *Lane, def string, rule landing.MergeRule) (*Outcome, string, error) {
	ld := lane.Landing
	n := strconv.Itoa(lane.PR)
	v, unread, err := readQueue(c, *lane)
	switch {
	case err != nil:
		return nil, "", err
	case unread != "":
		return nil, "; its place in the merge queue could not be read (" + unread + "), so the loop waits as before", nil
	case v.State != "" && v.State != "OPEN":
		// Merged or closed since the state was read: the next step's read
		// of its state settles it.
		return nil, "", nil
	case v.HeadRefOid != "" && v.HeadRefOid != ld.Pushed:
		if wait := unconfirmedMove(c, *lane, v); wait != "" {
			return nil, wait, nil
		}
		return nil, "", movedHeadRefusal(c, *lane, v)
	case v.queued():
		return nil, "; it is in " + def + "'s merge queue" + entryText(v), nil
	case v.AutoMergeRequest == nil:
		return nil, "", droppedRefusal(c, *lane, def, rule, v)
	case v.MergeStateStatus != "CLEAN":
		return nil, "; it is armed and not yet in the merge queue (the forge reports " + bounded(v.MergeStateStatus, 40) + ")", nil
	}
	if !gitutil.IsFullSHA(ld.Pushed) || v.ID == "" {
		return nil, "; it is armed and CLEAN outside the merge queue, and the loop has no pushed head or node id to queue it by", nil
	}
	// Queuing is the arming's own act, so the fence the arming passed is
	// asked again: the reservation still this person's, no release held.
	if _, _, err := fence(c, *lane, fenceLease, fenceQueue); err != nil {
		return nil, "", err
	}
	// So is the merge gate: a queue entry and a toggle both make the arming
	// again, and neither is made at a head the gate would not arm now.
	d, err := askMergeGate(c, *lane, def, rule)
	if r, ok := AsRefusal(err); ok && r.modeUnread {
		return nil, "; it is armed and CLEAN outside the merge queue, and facilitator.mode could not be read at " + Remote + "/" + def + ", so its arming is left as it is and the loop neither queues nor toggles it", nil
	}
	if err != nil {
		return nil, "", err
	}
	if d.Unread {
		return nil, "; it is armed and CLEAN outside the merge queue, and the merge gate could not be read (" + d.Why + "), so its arming is left as it is and the loop neither queues nor toggles it", nil
	}
	if !d.Arm {
		if _, err := forge(c, *lane, "pr", "merge", n, "--disable-auto"); err != nil {
			return nil, "", err
		}
		// The forge took the withdrawal, so a read-back that fails leaves it
		// unconfirmed, not undone: the lane is recorded left open, and the next
		// step asks the gate again rather than read it as dropped from the queue.
		still, err := landing.StillArmed(gateForge(c, *lane), lane.Branch, lane.PR)
		if err == nil && still {
			return nil, "", stillArmedRefusal(*lane, StageLand)
		}
		note := "the loop withdrew pull request #" + n + "'s arming and neither queues nor toggles it: " + d.Why + "; the loop asks again on each step"
		if err != nil {
			note += "; it could not read the withdrawal back (" + forgeWhy(err) + ")"
		}
		ld.Merge, ld.Armed = "left open: "+d.Why, false
		return &Outcome{Stay: true, Note: note}, "", nil
	}
	why, err := enqueue(c, *lane, v.ID)
	if err != nil {
		return nil, "", err
	}
	if why == "" {
		return &Outcome{Stay: true, Note: fmt.Sprintf("pull request #%s sat armed and CLEAN outside %s's merge queue, so the loop queued it, pinned to %s",
			n, def, shortSHA(ld.Pushed))}, "", nil
	}
	enabled, perr := time.Parse(time.RFC3339, v.AutoMergeRequest.EnabledAt)
	if perr != nil || c.Now.Sub(enabled) < requeueGrace {
		after := "once its arming is " + strconv.Itoa(int(requeueGrace/time.Minute)) + " minutes old"
		if perr == nil {
			after = "after " + enabled.Add(requeueGrace).UTC().Format("15:04 UTC")
		}
		return nil, "; it is armed and CLEAN outside the merge queue, and the forge refused to queue it (" + why +
			"); the forge may still queue it by itself, and the loop toggles its auto-merge " + after, nil
	}
	if ld.Toggles >= maxToggles {
		return nil, fmt.Sprintf("; it is armed and CLEAN outside the merge queue, and the forge refused to queue it (%s); the loop toggled its auto-merge %d times at this head and no longer toggles it, so the forge queues it by itself or a person settles it", why, ld.Toggles), nil
	}
	return toggleArming(c, lane, def, rule, why)
}

// watchAutoMerge is the same watch where no queue gates the default branch: the
// forge's auto-merge is the arming, and a head moved under it is withdrawn and
// refused. It returns the clause the wait adds; a reading that fails adds none.
func watchAutoMerge(c Context, lane Lane) (string, error) {
	v, unread, err := readQueue(c, lane)
	switch {
	case err != nil:
		return "", err
	case unread != "", v.State != "" && v.State != "OPEN", v.HeadRefOid == "", v.HeadRefOid == lane.Landing.Pushed:
		return "", nil
	}
	if wait := unconfirmedMove(c, lane, v); wait != "" {
		return wait, nil
	}
	return "", movedHeadRefusal(c, lane, v)
}

// unconfirmedMove is the clause for an arming whose head the forge reads as moved
// and origin does not bear out: its branch still at the pushed head, gone, or
// out of reach. The forge's reading can lag, so nothing comes off for it. "" when
// there is no arming, or origin's branch has moved.
func unconfirmedMove(c Context, lane Lane, v queueView) string {
	if v.AutoMergeRequest == nil && !v.queued() {
		return ""
	}
	tip, err := landing.RemoteTip(gitutil.Net, c.RepoRoot, lane.Branch)
	if err == nil && tip != "" && tip != lane.Landing.Pushed {
		return ""
	}
	return fmt.Sprintf("; the forge reads its head as %s, and %s does not show that move, so its arming is left as it is", shortSHA(bounded(v.HeadRefOid, 40)), Remote)
}

// movedHeadRefusal is the refusal of a pull request whose head on the forge is
// not the one the loop pushed: nothing is pushed to a lane after its arming, so
// someone else moved it, and any arming still on it is withdrawn first. A
// withdrawal the network refuses waits on the shared probe, as every forge call
// of the landing does.
func movedHeadRefusal(c Context, lane Lane, v queueView) error {
	ld := lane.Landing
	n := strconv.Itoa(lane.PR)
	head := fmt.Sprintf("pull request #%s's head on the forge is %s, not %s, the head the loop pushed and armed", n, shortSHA(bounded(v.HeadRefOid, 40)), shortSHA(ld.Pushed))
	settle := "settle who moved the pull request's head (nothing is pushed to a lane after its arming), then merge it yourself or close it"
	reason := head + ", and it is not in the merge queue, so the loop neither queues nor re-arms it"
	if v.AutoMergeRequest != nil || v.queued() {
		if err := landing.WithdrawArming(gateForge(c, lane), n, v.ID, v.AutoMergeRequest != nil, v.queued()); err != nil {
			if r := netFailure(StageLand, lane.ID, "gh pr merge", err); r != nil {
				return r
			}
			r := refuse(string(StageLand), "", lane.ID, head+", and the loop could not withdraw its arming: "+forgeWhy(err),
				"withdraw it yourself (`gh pr merge "+n+" --disable-auto`, and take it out of the merge queue on its page), then "+settle)
			r.outOfQueue = true
			return r
		}
		if err := confirmWithdrawn(c, lane, StageLand); err != nil {
			return err
		}
		reason = head + ", so the loop withdrew its arming and neither queues nor re-arms it"
	}
	r := refuse(string(StageLand), "", lane.ID, reason, settle)
	r.outOfQueue = true
	return r
}

// confirmWithdrawn reads back a withdrawal, as land does: the lane records nothing
// withdrawn until the forge stops reporting the arming. A read the network refuses
// waits on the shared probe.
func confirmWithdrawn(c Context, lane Lane, stage Stage) error {
	n := strconv.Itoa(lane.PR)
	still, err := landing.StillArmed(gateForge(c, lane), lane.Branch, lane.PR)
	if err != nil {
		if r := netFailure(stage, lane.ID, "gh api graphql", err); r != nil {
			return r
		}
		return refuse(string(stage), "", lane.ID, "the loop withdrew pull request #"+n+"'s arming and could not read it back: "+forgeWhy(err),
			"check it (`gh pr view "+n+"`), then run `abcd implement step` again")
	}
	if still {
		return stillArmedRefusal(lane, stage)
	}
	return nil
}

// stillArmedRefusal is the refusal of a withdrawal the forge took and goes on
// reporting armed or queued.
func stillArmedRefusal(lane Lane, stage Stage) error {
	n := strconv.Itoa(lane.PR)
	return refuse(string(stage), "", lane.ID, "the loop withdrew pull request #"+n+"'s arming, but the forge still reports it armed or queued",
		"check it (`gh pr view "+n+"`), then run `abcd implement step` again")
}

// entryText is the queue entry's state and place, as a clause.
func entryText(v queueView) string {
	e := v.MergeQueueEntry
	if e == nil {
		return ""
	}
	var parts []string
	if s := bounded(e.State, 40); s != "" {
		parts = append(parts, strings.ToLower(s))
	}
	if e.Position > 0 {
		parts = append(parts, "position "+strconv.Itoa(e.Position))
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

// enqueue asks the forge to queue the pull request with node id, pinned to
// the lane's pushed head (claim.EnqueueMutation, the release hold's own
// call). why is the forge's refusal, "" when it queued it.
func enqueue(c Context, lane Lane, id string) (why string, err error) {
	args := []string{"api", "graphql", "-f", "query=" + claim.EnqueueMutation, "-f", "id=" + id, "-f", "h=" + lane.Landing.Pushed}
	out, err := landing.GH(c.RepoRoot, args...)
	if errors.Is(err, landing.ErrNoForgeClient) {
		return "", forgeRefusal(lane, args, err)
	}
	if err != nil {
		if r := netFailure(StageLand, lane.ID, "gh api graphql", err); r != nil {
			return "", r
		}
		return forgeWhy(err), nil
	}
	var a gqlAnswer
	if json.Unmarshal([]byte(out), &a) != nil {
		return "the forge's answer does not parse", nil
	}
	if len(a.Errors) > 0 {
		return a.text(), nil
	}
	return "", nil
}

// toggleArming withdraws the lane's arming and makes it again, by the ruleset
// mirror's method at the lane's base and pinned to the pushed head, as the
// arming made it: the toggle that queued #837 and #844 where the forge's
// enqueue call failed. An arming the forge refuses to make again leaves the
// pull request disarmed, which is refused, naming the command that re-arms it,
// and marked out of the queue, so a drain reads its parked lane as dropped
// with that refusal as its note, not as still armed.
func toggleArming(c Context, lane *Lane, def string, rule landing.MergeRule, why string) (*Outcome, string, error) {
	ld := lane.Landing
	n := strconv.Itoa(lane.PR)
	method := landing.MergeMethods[rule.Method]
	if method == "" {
		return nil, "; it is armed and CLEAN outside the merge queue, and the forge refused to queue it (" + why + "); the ruleset mirror names no merge method to re-arm it by, so the loop leaves it", nil
	}
	if _, err := forge(c, *lane, "pr", "merge", n, "--disable-auto"); err != nil {
		return nil, "", err
	}
	rearm := rearmArgs(n, method, ld.Pushed)
	if _, err := landing.GH(c.RepoRoot, rearm...); err != nil {
		if r := netFailure(StageLand, lane.ID, "gh pr merge", err); r != nil {
			r.Reason = "pull request #" + n + " is now DISARMED: " + r.Reason
			r.Remedy = "arm it again with `gh " + strings.Join(rearm, " ") + "` once the network is back; " + r.Remedy
			r.outOfQueue = true
			return nil, "", r
		}
		r := refuse(string(StageLand), "", lane.ID,
			"pull request #"+n+" is now DISARMED: the loop withdrew its auto-merge to queue it again and the forge refused the new arming ("+forgeWhy(err)+")",
			"arm it again yourself with `gh "+strings.Join(rearm, " ")+"`, then run `abcd implement step` again")
		r.outOfQueue = true
		return nil, "", r
	}
	ld.Toggles++
	return &Outcome{Stay: true, Note: fmt.Sprintf("pull request #%s sat armed and CLEAN outside %s's merge queue past the %d-minute grace, and the forge refused to queue it (%s), so the loop toggled its auto-merge, toggle %d of %d, re-armed by %s pinned to %s",
		n, def, int(requeueGrace/time.Minute), why, ld.Toggles, maxToggles, strings.TrimPrefix(method, "--"), shortSHA(ld.Pushed))}, "", nil
}

// rearmArgs is the forge client's command that arms pull request n again by
// the merge method's flag, pinned to head: the toggle's own re-arming, and
// the one a dropped pull request's report quotes for a person to run. gh
// refuses `pr merge --auto` with no method when it cannot prompt, so the
// method is never left out.
func rearmArgs(n, method, head string) []string {
	return []string{"pr", "merge", n, "--auto", method, "--match-head-commit=" + head}
}

// droppedRefusal is the terminal report of an armed lane's pull request that
// is open, outside the merge queue and no longer armed: the forge removed it
// from the queue (a failed merge-group check clears the arming with it) or
// someone withdrew the arming. It names the forge's reason and the checks
// that failed on the pull request's last merge group, and quotes the command
// that arms it again by rule's method, the one toggleArming runs. While a
// release being cut holds every merge, the lane waits on the hold instead:
// the hold withdraws armed merges and re-arms them after.
func droppedRefusal(c Context, lane Lane, def string, rule landing.MergeRule, v queueView) error {
	held := ""
	m, err := claim.FetchMergeHold(c.RepoRoot, c.Now, claimOptions(c.Now))
	if err != nil {
		held = "; the release holds could not be read (" + forgeWhy(err) + "), so a release's withdrawal is not ruled out"
	} else if err := heldMerge(lane, m, fenceDropped); err != nil {
		return err
	}
	n := strconv.Itoa(lane.PR)
	removed := "the forge records no removal from the merge queue, so its arming was withdrawn"
	if nodes := v.TimelineItems.Nodes; len(nodes) > 0 {
		e := nodes[len(nodes)-1]
		removed = "the forge removed it from the queue"
		if t, err := time.Parse(time.RFC3339, e.CreatedAt); err == nil {
			removed += " at " + stamp(t)
		}
		if r := bounded(e.Reason, maxQueueReason); r != "" {
			removed += " (its reason: " + r + ")"
		}
	}
	reason := fmt.Sprintf("pull request #%s left %s's merge queue and is no longer armed, so it will not merge on its own: %s; %s%s",
		n, def, removed, failedGroupChecks(c, lane, def), held)
	rearm := lane.Landing.Pushed + " by the queue's merge method (the ruleset mirror names none the loop can quote)"
	if method := landing.MergeMethods[rule.Method]; method != "" {
		rearm = "(`gh " + strings.Join(rearmArgs(n, method, lane.Landing.Pushed), " ") + "`)"
	}
	r := refuse(string(StageLand), "", lane.ID, reason,
		"fix or re-run the failed check, then arm it again pinned to the pushed head "+rearm+
			", and run `abcd implement step` again: the loop does not re-queue it into the same failure")
	r.outOfQueue = true
	return r
}

// groupRun is one workflow run the merge queue started, as the forge lists it.
type groupRun struct {
	Name       string `json:"name"`
	HeadBranch string `json:"head_branch"`
	HeadSHA    string `json:"head_sha"`
	Conclusion string `json:"conclusion"`
	CreatedAt  string `json:"created_at"`
}

// checkRun is one check run on a commit, as the forge lists it.
type checkRun struct {
	Name       string `json:"name"`
	Conclusion string `json:"conclusion"`
}

// failedConclusions are the conclusions that fail a merge group. A cancelled
// run is left out: the queue cancels the rest of a group it dropped.
var failedConclusions = map[string]bool{"failure": true, "timed_out": true, "action_required": true, "startup_failure": true}

// failedGroupChecks names the checks that failed on the pull request's last
// merge group: its newest workflow run the queue started on a
// `gh-readonly-queue/<def>/pr-<n>-` branch gives the group's commit, and the
// check runs on that commit are read for the failed ones (any app's), the
// failed workflow runs named instead when they cannot be. It always returns
// a clause, saying what it could not read.
func failedGroupChecks(c Context, lane Lane, def string) string {
	out, err := landing.GH(c.RepoRoot, "api", mergeGroupRunsPath)
	if err != nil {
		return "the merge group's checks could not be read (" + forgeWhy(err) + ")"
	}
	var listed struct {
		WorkflowRuns []groupRun `json:"workflow_runs"`
	}
	if json.Unmarshal([]byte(out), &listed) != nil {
		return "the forge's listing of the merge queue's runs does not parse, so the failed check is not named"
	}
	prefix := "gh-readonly-queue/" + def + "/pr-" + strconv.Itoa(lane.PR) + "-"
	var runs []groupRun
	for _, r := range listed.WorkflowRuns {
		if strings.HasPrefix(r.HeadBranch, prefix) && gitutil.IsFullSHA(r.HeadSHA) {
			runs = append(runs, r)
		}
	}
	if len(runs) == 0 {
		return "the forge lists no merge-queue run of it among the repository's latest, so the failed check is not named"
	}
	// The newest group: the forge's RFC 3339 times, all in UTC, sort as
	// strings.
	slices.SortStableFunc(runs, func(a, b groupRun) int { return strings.Compare(b.CreatedAt, a.CreatedAt) })
	sha := runs[0].HeadSHA
	at := "on its last merge group " + shortSHA(sha)
	if out, err := landing.GH(c.RepoRoot, "api", checkRunsPath(sha)); err == nil {
		var checks struct {
			CheckRuns []checkRun `json:"check_runs"`
		}
		if json.Unmarshal([]byte(out), &checks) == nil {
			var failed []string
			for _, r := range checks.CheckRuns {
				if failedConclusions[r.Conclusion] {
					failed = append(failed, checkName(r.Name, r.Conclusion))
				}
			}
			if len(failed) > 0 {
				return "the failed check " + at + ": " + namedChecks(failed)
			}
		}
	}
	var failed []string
	for _, r := range runs {
		if r.HeadSHA == sha && failedConclusions[r.Conclusion] {
			failed = append(failed, checkName(r.Name, r.Conclusion))
		}
	}
	if len(failed) > 0 {
		return "the failed workflow " + at + ": " + namedChecks(failed)
	}
	return "the forge names no failed check " + at
}

// checkName is a check's name and conclusion as the forge gives them,
// sanitised and bounded.
func checkName(name, conclusion string) string {
	n := bounded(name, maxCheckNameText)
	if n == "" {
		n = "an unnamed check"
	}
	return n + " (" + strings.ReplaceAll(bounded(conclusion, 40), "_", " ") + ")"
}

// namedChecks joins the failed checks, at most maxFailedChecks of them.
func namedChecks(names []string) string {
	shown := names[:min(len(names), maxFailedChecks)]
	s := strings.Join(shown, ", ")
	if more := len(names) - len(shown); more > 0 {
		s += fmt.Sprintf(", and %d more", more)
	}
	return s
}
