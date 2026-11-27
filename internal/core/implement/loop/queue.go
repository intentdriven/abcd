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
//   - neither armed nor queued nor merged: the merge queue's own removal
//     (a failed merge-group check clears the arming with it) is armed again
//     (rearmDropped, itd-2611141610201875), pinned to the pushed head, at most
//     maxRearms times at one head, past the fence and the merge gate, and only
//     a removal later than the loop's own last arming of that head and than
//     the removal it last armed again for. The forge's data cannot tell the
//     change's own failure from another's, so the reason never chooses. Any
//     other removal, a withdrawn arming and a removal past the bound are
//     refused for a person, naming the forge's reason, the checks that failed
//     on the pull request's last merge group, the command that re-runs them
//     and the one that arms it again. A release being cut withdraws armed
//     merges the same way and re-arms them after, so while its hold stands
//     the lane waits on it instead.
//
// The forge's answers are untrusted text: every name and reason is
// sanitised and bounded, the merge group's commit must be a full object name
// before it is put in a request path, and the pull request's node id goes to
// the forge as a raw string variable, never read as a flag or a file.

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
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
// the last removal from the queue with the forge's reason and its actor, whose
// type and login say whether the queue itself removed it. Every field is
// pinned in internal/ghcontract.
const QueueStateQuery = `query($o:String!,$r:String!,$n:Int!){repository(owner:$o,name:$r){pullRequest(number:$n){id state headRefOid isInMergeQueue mergeQueueEntry{state position} autoMergeRequest{mergeMethod enabledAt} mergeStateStatus timelineItems(last:1,itemTypes:[REMOVED_FROM_MERGE_QUEUE_EVENT]){nodes{... on RemovedFromMergeQueueEvent{createdAt reason actor{__typename login}}}}}}}`

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

// mergeGroupRunsFilter is the --jq filter the runs listing is read through:
// it keeps the listing's own shape, cut to the six fields of each run that
// groupRun decodes. A hundred whole workflow runs outgrow the forge client's
// output bound (landing.MaxOutput), so every dropped lane read "could not be
// read" and its failed check went unnamed (iss-2611201352116252); the six
// fields keep the listing far under it. gh prints an object result as JSON.
const mergeGroupRunsFilter = `{workflow_runs: [.workflow_runs[] | {id, name, head_branch, head_sha, conclusion, created_at}]}`

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

// maxRearms bounds how many times the watch arms again, at one pushed head, a
// pull request the merge queue removed: a check that goes on failing is
// handed back to a person rather than queued into it without end.
const maxRearms = 3

// The merge queue as a removal's actor names it: the forge's own application,
// with this login. An application is a Bot too, and one that takes a pull
// request out of the queue was told to by someone, so only this one shape is
// armed again; a forge that names its queue otherwise arms nothing again.
const (
	mergeQueueActorType  = "Bot"
	mergeQueueActorLogin = "github-merge-queue"
)

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
		Nodes []queueRemovalNode `json:"nodes"`
	} `json:"timelineItems"`
}

// queueRemovalNode is the forge's last removal of the pull request from the
// merge queue: when, the forge's reason, and who removed it (nil when the
// forge names no one).
type queueRemovalNode struct {
	CreatedAt string      `json:"createdAt"`
	Reason    string      `json:"reason"`
	Actor     *queueActor `json:"actor"`
}

// queueActor is a removal's actor: its concrete type and its login.
type queueActor struct {
	Typename string `json:"__typename"`
	Login    string `json:"login"`
}

// byTheQueue reports whether the forge's merge queue itself made the removal.
func (e queueRemovalNode) byTheQueue() bool {
	return e.Actor != nil && e.Actor.Typename == mergeQueueActorType && e.Actor.Login == mergeQueueActorLogin
}

// lastRemoval is the last removal on record, at whatever head; ok is false
// when the forge records none.
func (v queueView) lastRemoval() (queueRemovalNode, bool) {
	nodes := v.TimelineItems.Nodes
	if len(nodes) == 0 {
		return queueRemovalNode{}, false
	}
	return nodes[len(nodes)-1], true
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

// fenceRearm is the land fence's act before the watch arms again a pull
// request the merge queue removed.
var fenceRearm = fenceAct{not: "arm its pull request again", none: "the pull request is not armed again",
	stays: "the pull request stays outside the queue, disarmed"}

// fenceDropped is the merge hold's act before the watch reports a pull
// request gone from the queue: a release being cut withdraws armed merges
// and re-arms them after, so the lane waits on it rather than report a drop.
var fenceDropped = fenceAct{not: "report its pull request dropped from the merge queue", none: "nothing is reported",
	stays: "the pull request stays outside the queue for the release's hold to re-arm"}

// watchQueue is the merge-queue half of an armed lane's wait for its merge,
// where a queue gates def. It returns the step's outcome when it acted (the
// pull request queued, or its arming toggled), or wait, a clause the ordinary
// wait adds; a pull request gone from the queue and no longer armed is armed
// again where the queue removed it (rearmDropped), and refused otherwise,
// naming why.
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
		return rearmDropped(c, lane, def, rule, v)
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
	now := c.Now
	ld.ArmedAt = &now
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

// rearmDropped is the watch's act on an armed lane's pull request that is
// open, outside the merge queue and no longer armed: the forge's merge queue
// removed it (a failed merge-group check clears the arming with it), or
// someone withdrew the arming. It arms the pull request again, by the ruleset
// mirror's method pinned to the pushed head, when all of these hold, and
// otherwise refuses for a person (droppedReport):
//
//   - no release's hold stands, and the holds were read: a held lane waits
//     on the hold, and holds it could not read leave a release's withdrawal
//     not ruled out;
//   - the forge records a removal and the merge queue itself made it
//     (byTheQueue);
//   - the removal is later than the loop's own last arming of the pushed head
//     (Landing.ArmedAt, the run's clock), so a removal from before a resync
//     or before a person armed it again is never read as new;
//   - the removal is later than the one the landing last armed again for
//     (Landing.RearmedRemoval, the forge's clock), so a person's withdrawal
//     after a re-arm stands;
//   - fewer than maxRearms re-arms at the pushed head;
//   - the pushed head is a full object name and the mirror names a method;
//   - the fence and the merge gate pass, asked as the watch asks them before
//     it queues or toggles.
//
// The count, the removal's time and the arming's are kept only once the forge
// took the arming. The removal's reason is printed and never chooses
// (decision 1 of itd-2611141610201875: the forge's data cannot tell the
// change's own failure from another's).
func rearmDropped(c Context, lane *Lane, def string, rule landing.MergeRule, v queueView) (*Outcome, string, error) {
	held, err := droppedHold(c, *lane)
	if err != nil {
		return nil, "", err
	}
	if held != "" {
		return nil, "", droppedReport(c, *lane, def, rule, v, held)
	}
	ld := lane.Landing
	removedAt, why := rearmable(*ld, v)
	method := landing.MergeMethods[rule.Method]
	switch {
	case why != "":
	case method == "":
		why = "the ruleset mirror names no merge method to arm it by"
	case !gitutil.IsFullSHA(ld.Pushed):
		why = "the loop has no full pushed head to pin the arming to"
	}
	if why != "" {
		return nil, "", droppedReport(c, *lane, def, rule, v, notRearmed(*ld, why))
	}
	// Arming again is the arming's own act, so the fence it passed is asked
	// again (the reservation still this person's, no release held), and so is
	// the merge gate: nothing is armed at a head the gate would not arm now.
	if _, _, err := fence(c, *lane, fenceLease, fenceRearm); err != nil {
		return nil, "", err
	}
	d, err := askMergeGate(c, *lane, def, rule)
	if r, ok := AsRefusal(err); ok && r.modeUnread {
		return nil, "; the merge queue removed it, and facilitator.mode could not be read at " + Remote + "/" + def + ", so the loop does not arm it again yet", nil
	}
	if err != nil {
		return nil, "", err
	}
	if d.Unread {
		return nil, "; the merge queue removed it, and the merge gate could not be read (" + d.Why + "), so the loop does not arm it again yet", nil
	}
	n := strconv.Itoa(lane.PR)
	if !d.Arm {
		ld.Merge, ld.Armed = "left open: "+d.Why, false
		return &Outcome{Stay: true, Note: "the merge queue removed pull request #" + n + ", and the loop does not arm it again: " + d.Why + "; the loop asks again on each step"}, "", nil
	}
	args := rearmArgs(n, method, ld.Pushed)
	if _, err := landing.GH(c.RepoRoot, args...); err != nil {
		if errors.Is(err, landing.ErrNoForgeClient) {
			return nil, "", forgeRefusal(*lane, args, err)
		}
		if r := netFailure(StageLand, lane.ID, "gh pr merge", err); r != nil {
			return nil, "", r
		}
		return nil, "", droppedReport(c, *lane, def, rule, v, "; the loop armed it again and the forge refused the arming ("+forgeWhy(err)+")")
	}
	ld.Rearms++
	now := c.Now
	ld.RearmedRemoval, ld.ArmedAt = &removedAt, &now
	e, _ := v.lastRemoval()
	reason := ""
	if r := bounded(e.Reason, maxQueueReason); r != "" {
		reason = " (its reason: " + r + ")"
	}
	return &Outcome{Stay: true, Note: fmt.Sprintf("pull request #%s was removed from %s's merge queue by the queue%s and is armed again, %d of %d at this head, by %s pinned to %s",
		n, def, reason, ld.Rearms, maxRearms, strings.TrimPrefix(method, "--"), shortSHA(ld.Pushed))}, "", nil
}

// rearmable reads the forge's last removal against the landing: its time, and
// why it is not armed again, "" when the removal is the queue's own, later than
// the loop's own last arming of the pushed head and than the removal last armed
// again for, and under the bound. Both comparisons are strict, and a time that
// does not parse arms nothing.
func rearmable(ld Landing, v queueView) (time.Time, string) {
	e, ok := v.lastRemoval()
	if !ok {
		return time.Time{}, "the forge records no removal by the merge queue, so its arming was withdrawn"
	}
	if !e.byTheQueue() {
		return time.Time{}, "the merge queue itself did not remove it"
	}
	at, err := time.Parse(time.RFC3339, e.CreatedAt)
	switch {
	case err != nil:
		return time.Time{}, "the forge's time of the removal does not read"
	case ld.ArmedAt == nil:
		return time.Time{}, "the landing keeps no time of its own arming of this head"
	case !at.After(*ld.ArmedAt):
		return time.Time{}, "the removal on record is not later than the loop's own last arming of this head"
	case ld.RearmedRemoval != nil && !at.After(*ld.RearmedRemoval):
		return time.Time{}, "no removal on record is later than the one it was last armed again for, so its arming was withdrawn since"
	case ld.Rearms >= maxRearms:
		return time.Time{}, "past the bound"
	}
	return at, ""
}

// notRearmed is the report's clause saying the loop does not arm the pull
// request again and why, and how many times it already did at this head.
func notRearmed(ld Landing, why string) string {
	if ld.Rearms >= maxRearms {
		return fmt.Sprintf("; the loop armed it again %d times at this head, the most it does", ld.Rearms)
	}
	s := "; the loop does not arm it again: " + why
	if ld.Rearms > 0 {
		s += fmt.Sprintf(" (it armed it again %d of %d times at this head)", ld.Rearms, maxRearms)
	}
	return s
}

// droppedHold is the merge hold's answer for a dropped pull request: the
// hold's wait while a release being cut holds every merge (the hold withdraws
// armed merges and re-arms them after), or the clause saying the holds could
// not be read, "" when they were read and none stands.
func droppedHold(c Context, lane Lane) (string, error) {
	m, err := claim.FetchMergeHold(c.RepoRoot, c.Now, claimOptions(c.Now))
	if err != nil {
		return "; the release holds could not be read (" + forgeWhy(err) + "), so a release's withdrawal is not ruled out", nil
	}
	return "", heldMerge(lane, m, fenceDropped)
}

// droppedReport is the refusal for a person of a pull request the watch does
// not arm again: the forge's reason for the removal, the checks that failed on
// the pull request's last merge group, then extra (why the loop does not arm
// it again, or that the holds were not read); and as the remedy, the failed
// check, the command that re-runs its workflow run, the command that arms it
// again by rule's method (the one toggleArming runs), and the step that reads
// it again.
func droppedReport(c Context, lane Lane, def string, rule landing.MergeRule, v queueView, extra string) error {
	n := strconv.Itoa(lane.PR)
	removed := "the forge records no removal from the merge queue, so its arming was withdrawn"
	if e, ok := v.lastRemoval(); ok {
		removed = "the forge removed it from the queue"
		if t, err := time.Parse(time.RFC3339, e.CreatedAt); err == nil {
			removed += " at " + stamp(t)
		}
		if r := bounded(e.Reason, maxQueueReason); r != "" {
			removed += " (its reason: " + r + ")"
		}
	}
	g := failedGroupChecks(c, lane, def)
	reason := fmt.Sprintf("pull request #%s left %s's merge queue and is no longer armed, so it will not merge on its own: %s; %s%s",
		n, def, removed, g.clause, extra)
	rearm := lane.Landing.Pushed + " by the queue's merge method (the ruleset mirror names none the loop can quote)"
	if method := landing.MergeMethods[rule.Method]; method != "" {
		rearm = "(`gh " + strings.Join(rearmArgs(n, method, lane.Landing.Pushed), " ") + "`)"
	}
	// The failed checks are the forge's text: they stand in a clause of their
	// own before the commands, quoted, and as forgeLabel left them, so a name
	// can neither close nor open a back-quoted command.
	named := ""
	if g.failed != "" {
		named = "the failed check, as the forge names it, is \"" + forgeLabel(g.failed) + "\"; "
	}
	remedy := named + "the forge names no failed run to re-run, so settle the failed check, then arm it again pinned to the pushed head " + rearm
	if len(g.reruns) > 0 {
		cmds := make([]string, len(g.reruns))
		for i, id := range g.reruns {
			cmds[i] = "`gh run rerun " + id + " --failed`"
		}
		remedy = named + "re-run it with " + strings.Join(cmds, " and ") + ", then arm it again pinned to the pushed head " + rearm
	}
	r := refuse(string(StageLand), "", lane.ID, reason,
		remedy+", then run `abcd implement step` again, or leave it to `abcd drain --drive`, whose next move reads it again")
	r.outOfQueue = true
	return r
}

// groupRun is one workflow run the merge queue started, as the forge lists it.
type groupRun struct {
	ID         runID  `json:"id"`
	Name       string `json:"name"`
	HeadBranch string `json:"head_branch"`
	HeadSHA    string `json:"head_sha"`
	Conclusion string `json:"conclusion"`
	CreatedAt  string `json:"created_at"`
}

// runID is a workflow run's id as the forge lists it, kept only when it is 1
// to 20 digits and "" otherwise: it is put in a command a person runs, so
// nothing else read from the forge reaches one.
type runID string

// runIDShape is the one shape of a run id put in a command.
var runIDShape = regexp.MustCompile(`^[0-9]{1,20}$`)

// UnmarshalJSON keeps a bare number of 1 to 20 digits, and drops anything
// else (a string, a sign, a fraction, null) without failing the listing.
func (r *runID) UnmarshalJSON(b []byte) error {
	*r = ""
	if runIDShape.Match(b) {
		*r = runID(b)
	}
	return nil
}

// checkRun is one check run on a commit, as the forge lists it.
type checkRun struct {
	Name       string `json:"name"`
	Conclusion string `json:"conclusion"`
}

// failedConclusions are the conclusions that fail a merge group. A cancelled
// run is left out: the queue cancels the rest of a group it dropped.
var failedConclusions = map[string]bool{"failure": true, "timed_out": true, "action_required": true, "startup_failure": true}

// groupChecks is what the watch read of a pull request's last merge group:
// the clause the report says, which always says what it could not read; the
// failed checks it named, "" for none; and the ids of the group's failed
// workflow runs, each 1 to 20 digits, which a person re-runs.
type groupChecks struct {
	clause string
	failed string
	reruns []string
}

// failedGroupChecks names the checks that failed on the pull request's last
// merge group: its newest workflow run the queue started on a
// `gh-readonly-queue/<def>/pr-<n>-` branch gives the group's commit, and the
// check runs on that commit are read for the failed ones (any app's), the
// failed workflow runs named instead when they cannot be. The failed workflow
// runs of that commit are the ones to re-run.
func failedGroupChecks(c Context, lane Lane, def string) groupChecks {
	out, err := landing.GH(c.RepoRoot, "api", mergeGroupRunsPath, "--jq", mergeGroupRunsFilter)
	if err != nil {
		return groupChecks{clause: "the merge group's checks could not be read (" + forgeWhy(err) + ")"}
	}
	var listed struct {
		WorkflowRuns []groupRun `json:"workflow_runs"`
	}
	if json.Unmarshal([]byte(out), &listed) != nil {
		return groupChecks{clause: "the forge's listing of the merge queue's runs does not parse, so the failed check is not named"}
	}
	prefix := "gh-readonly-queue/" + def + "/pr-" + strconv.Itoa(lane.PR) + "-"
	var runs []groupRun
	for _, r := range listed.WorkflowRuns {
		if strings.HasPrefix(r.HeadBranch, prefix) && gitutil.IsFullSHA(r.HeadSHA) {
			runs = append(runs, r)
		}
	}
	if len(runs) == 0 {
		return groupChecks{clause: "the forge lists no merge-queue run of it among the repository's latest, so the failed check is not named"}
	}
	// The newest group: the forge's RFC 3339 times, all in UTC, sort as
	// strings.
	slices.SortStableFunc(runs, func(a, b groupRun) int { return strings.Compare(b.CreatedAt, a.CreatedAt) })
	sha := runs[0].HeadSHA
	at := "on its last merge group " + shortSHA(sha)
	var g groupChecks
	var failedRuns []string
	for _, r := range runs {
		if r.HeadSHA != sha || !failedConclusions[r.Conclusion] {
			continue
		}
		failedRuns = append(failedRuns, checkName(r.Name, r.Conclusion))
		if r.ID != "" && !slices.Contains(g.reruns, string(r.ID)) && len(g.reruns) < maxFailedChecks {
			g.reruns = append(g.reruns, string(r.ID))
		}
	}
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
				g.failed = namedChecks(failed)
				g.clause = "the failed check " + at + ": " + g.failed
				return g
			}
		}
	}
	if len(failedRuns) > 0 {
		g.failed = namedChecks(failedRuns)
		g.clause = "the failed workflow " + at + ": " + g.failed
		return g
	}
	g.clause = "the forge names no failed check " + at
	return g
}

// checkName is a check's name and conclusion as the forge gives them,
// sanitised and bounded.
func checkName(name, conclusion string) string {
	n := forgeLabel(bounded(name, maxCheckNameText))
	if n == "" {
		n = "an unnamed check"
	}
	return n + " (" + strings.ReplaceAll(forgeLabel(bounded(conclusion, 40)), "_", " ") + ")"
}

// forgeLabel is forge text named beside the loop's own commands: one
// terminal-safe line (termsafe.SanitizeLine masks a newline and the line
// separators), with every back-quote and double quote made a single quote, so
// it can close no quoted command or name and open none.
func forgeLabel(s string) string {
	return labelQuotes.Replace(termsafe.SanitizeLine(s))
}

// labelQuotes turns the quotes a remedy delimits by into a single quote.
var labelQuotes = strings.NewReplacer("`", "'", `"`, "'")

// namedChecks joins the failed checks, at most maxFailedChecks of them.
func namedChecks(names []string) string {
	shown := names[:min(len(names), maxFailedChecks)]
	s := strings.Join(shown, ", ")
	if more := len(names) - len(shown); more > 0 {
		s += fmt.Sprintf(", and %d more", more)
	}
	return s
}
