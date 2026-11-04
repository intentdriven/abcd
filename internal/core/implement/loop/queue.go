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
//   - armed, CLEAN and outside the queue: the landing's fence is asked, as
//     at the arming, and the pull request is queued, pinned to the pushed
//     head; when the forge refuses that, auto-merge is toggled once its
//     arming is past requeueGrace, and otherwise the wait says when;
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
	case v.queued():
		return nil, "; it is in " + def + "'s merge queue" + entryText(v), nil
	case v.HeadRefOid != "" && v.HeadRefOid != ld.Pushed:
		r := refuse(string(StageLand), "", lane.ID,
			fmt.Sprintf("pull request #%s's head on the forge is %s, not %s, the head the loop pushed and armed, and it is not in the merge queue, so the loop neither queues nor re-arms it",
				n, shortSHA(bounded(v.HeadRefOid, 40)), shortSHA(ld.Pushed)),
			"settle who moved the pull request's head (nothing is pushed to a lane after its arming), then merge it yourself or close it")
		r.outOfQueue = true
		return nil, "", r
	case v.AutoMergeRequest == nil:
		return nil, "", droppedRefusal(c, *lane, def, v)
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
	return toggleArming(c, lane, def, rule, why)
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
	rearm := []string{"pr", "merge", n, "--auto", method, "--match-head-commit=" + ld.Pushed}
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
	return &Outcome{Stay: true, Note: fmt.Sprintf("pull request #%s sat armed and CLEAN outside %s's merge queue past the %d-minute grace, and the forge refused to queue it (%s), so the loop toggled its auto-merge once, re-armed by %s pinned to %s",
		n, def, int(requeueGrace/time.Minute), why, strings.TrimPrefix(method, "--"), shortSHA(ld.Pushed))}, "", nil
}

// droppedRefusal is the terminal report of an armed lane's pull request that
// is open, outside the merge queue and no longer armed: the forge removed it
// from the queue (a failed merge-group check clears the arming with it) or
// someone withdrew the arming. It names the forge's reason and the checks
// that failed on the pull request's last merge group. While a release being
// cut holds every merge, the lane waits on the hold instead: the hold
// withdraws armed merges and re-arms them after.
func droppedRefusal(c Context, lane Lane, def string, v queueView) error {
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
	r := refuse(string(StageLand), "", lane.ID, reason,
		"fix or re-run the failed check, then arm it again pinned to the pushed head (`gh pr merge "+n+" --auto --match-head-commit="+lane.Landing.Pushed+
			"`, with the queue's merge method), and run `abcd implement step` again: the loop does not re-queue it into the same failure")
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
