package landing

// withdraw.go is the step before land's push. Where the push will move the head
// of an armed or queued pull request of the branch, the arming comes off first and
// is read back, and the enqueue step arms the new head where the merge gate allows,
// pinned to it. So does this account's arming at a local head the gate refuses;
// another account's there refuses the landing. Land withdraws when it is sure the
// head moves or the gate refuses it, and the arming is its to take off; where it
// cannot, it refuses and pushes nothing.

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// ArmingsQuery lists the open pull requests from one head branch, with what says
// whether each is armed (autoMergeRequest) or queued (isInMergeQueue), who did it,
// and the viewer's login: the forge clears autoMergeRequest as a pull request
// enters the queue, so both are read. A branch's name can be shared with forks, so
// the listing says whether more lie past the page, and a listing that does is no
// answer. The committed pre-push hook asks the same question.
const ArmingsQuery = `query($o:String!,$r:String!,$b:String!){viewer{login} repository(owner:$o,name:$r){pullRequests(states:OPEN,headRefName:$b,first:100){pageInfo{hasNextPage} nodes{number id headRefOid isCrossRepository isInMergeQueue mergeQueueEntry{enqueuer{login}} autoMergeRequest{mergeMethod enabledBy{login}}}}}}`

// armingActor is an account as ArmingsQuery reads it.
type armingActor struct {
	Login string `json:"login"`
}

// branchArming is one open pull request as ArmingsQuery lists it.
type branchArming struct {
	Number            int    `json:"number"`
	ID                string `json:"id"`
	HeadRefOid        string `json:"headRefOid"`
	IsCrossRepository bool   `json:"isCrossRepository"`
	IsInMergeQueue    bool   `json:"isInMergeQueue"`
	MergeQueueEntry   *struct {
		Enqueuer *armingActor `json:"enqueuer"`
	} `json:"mergeQueueEntry"`
	AutoMergeRequest *struct {
		EnabledBy *armingActor `json:"enabledBy"`
	} `json:"autoMergeRequest"`
}

func (a branchArming) auto() bool { return a.AutoMergeRequest != nil }

// own reports whether the forge says viewer made every arming on this pull
// request. One it attributes to no one counts as another account's.
func (a branchArming) own(viewer string) bool {
	by := func(who *armingActor) bool {
		return viewer != "" && who != nil && strings.EqualFold(who.Login, viewer)
	}
	if a.auto() && !by(a.AutoMergeRequest.EnabledBy) {
		return false
	}
	if a.IsInMergeQueue && (a.MergeQueueEntry == nil || !by(a.MergeQueueEntry.Enqueuer)) {
		return false
	}
	return true
}

// armings lists branch's pull requests from this repository that are armed or
// queued, and the forge client's login: a fork's from a branch of the same name
// is not this branch's, and this push does not move its head. Any answer that
// cannot be trusted is a refusal.
func armings(o Options, dir, owner, name, branch string) ([]branchArming, string, error) {
	cannot := func(why string) error {
		return refuse("the forge could not say whether a pull request for "+branch+" is armed, so land does not push over one: "+why,
			"settle what the forge reports (gh auth status), then run abcd land again; nothing was pushed")
	}
	raw, err := o.Forge(dir, "api", "graphql", "-f", "query="+ArmingsQuery, "-f", "o="+owner, "-f", "r="+name, "-f", "b="+branch)
	if errors.Is(err, ErrNoForgeClient) {
		return nil, "", noForgeClient()
	}
	if err != nil {
		return nil, "", cannot(forgeWhy(err))
	}
	out, viewer, err := parseArmings(raw)
	if err != nil {
		return nil, "", cannot(err.Error())
	}
	return out, viewer, nil
}

// parseArmings reads the forge's answer to ArmingsQuery: the pull requests from
// this repository that are armed or queued, and the forge client's login. An answer
// that cannot be trusted whole is an error.
func parseArmings(raw string) ([]branchArming, string, error) {
	var a struct {
		Data struct {
			Viewer     *armingActor `json:"viewer"`
			Repository *struct {
				PullRequests struct {
					PageInfo struct {
						HasNextPage bool `json:"hasNextPage"`
					} `json:"pageInfo"`
					Nodes []branchArming `json:"nodes"`
				} `json:"pullRequests"`
			} `json:"repository"`
		} `json:"data"`
		Errors []json.RawMessage `json:"errors"`
	}
	switch {
	case json.Unmarshal([]byte(raw), &a) != nil:
		return nil, "", errors.New("its answer does not parse")
	case len(a.Errors) > 0:
		return nil, "", errors.New(forgeErrorText(a.Errors))
	case a.Data.Repository == nil:
		return nil, "", errors.New("it answered no such repository")
	case a.Data.Repository.PullRequests.PageInfo.HasNextPage:
		return nil, "", errors.New("it lists more than 100 open pull requests from a branch of this name")
	}
	var out []branchArming
	for _, n := range a.Data.Repository.PullRequests.Nodes {
		if !n.IsCrossRepository && (n.auto() || n.IsInMergeQueue) {
			out = append(out, n)
		}
	}
	viewer := ""
	if a.Data.Viewer != nil {
		viewer = a.Data.Viewer.Login
	}
	return out, viewer, nil
}

// armedIn reports whether pull request number is among the armings listed.
func armedIn(list []branchArming, number int) bool {
	for _, a := range list {
		if a.Number == number {
			return true
		}
	}
	return false
}

// StillArmed is the read-back after a withdrawal, for a caller that lets gh name
// the checkout's repository: whether the forge still reports pull request number
// from branch armed or queued. gh exits 0 on some failures, so the forge's word is
// asked, not the call's.
func StillArmed(forge ForgeCall, branch string, number int) (bool, error) {
	raw, err := forge("api", "graphql", "-f", "query="+ArmingsQuery, "-F", "o={owner}", "-F", "r={repo}", "-f", "b="+branch)
	if err != nil {
		return false, err
	}
	list, _, err := parseArmings(raw)
	if err != nil {
		return false, err
	}
	return armedIn(list, number), nil
}

// WithdrawArming takes pull request n, node id id, off the forge's merge path:
// auto-merge off first, so the forge cannot queue it again between the two
// calls, then out of the queue. `gh pr merge --disable-auto` does not dequeue,
// so the queue entry takes the forge's own mutation. A dequeue that fails after
// auto-merge came off says so.
func WithdrawArming(forge ForgeCall, n, id string, auto, queued bool) error {
	if auto {
		if _, err := forge("pr", "merge", n, "--disable-auto"); err != nil {
			return err
		}
	}
	if !queued {
		return nil
	}
	err := dequeue(forge, id)
	if err != nil && auto {
		return fmt.Errorf("its auto-merge came off, and taking it out of the merge queue failed: %w", err)
	}
	return err
}

// dequeue takes the pull request with node id id out of the merge queue.
func dequeue(forge ForgeCall, id string) error {
	raw, err := forge("api", "graphql", "-f", "query="+claim.DequeueMutation, "-f", "id="+id)
	if err != nil {
		return err
	}
	var a struct {
		Errors []json.RawMessage `json:"errors"`
	}
	if json.Unmarshal([]byte(raw), &a) != nil {
		return errors.New("the forge's answer to the dequeue does not parse")
	}
	if len(a.Errors) > 0 {
		return errors.New(forgeErrorText(a.Errors))
	}
	return nil
}

// RemoteTip is the commit origin's branch is at, asked of origin itself, or ""
// when origin has no such branch.
func RemoteTip(git GitFunc, dir, branch string) (string, error) {
	out, err := git(dir, "ls-remote", Remote, "refs/heads/"+branch)
	if err != nil {
		return "", err
	}
	for _, l := range strings.Split(out, "\n") {
		if f := strings.Fields(l); len(f) == 2 && f[1] == "refs/heads/"+branch {
			return f[0], nil
		}
	}
	return "", nil
}

// withdrawBeforePush is the step before the push. An arming at the head the push
// moves comes off, as does this account's at a local head the merge gate refuses;
// another account's at such a head stops the landing. It reports an arming left at
// the local head, which the enqueue step names when the gate would not make it.
func withdrawBeforePush(res *Result, o Options, dir, tracking, nameWithOwner string) (atHead bool, err error) {
	owner, name, ok := splitRepo(nameWithOwner)
	if !ok {
		return false, refuse(fmt.Sprintf("the forge client names the repository %q, not owner/name, so land cannot ask whether a pull request for %s is armed", nameWithOwner, res.Branch),
			"check the forge client (gh repo view), then run abcd land again; nothing was pushed")
	}
	armed, viewer, err := armings(o, dir, owner, name, res.Branch)
	if err != nil {
		return false, err
	}
	ask := func(args ...string) (string, error) { return o.Forge(dir, args...) }
	type withdrawal struct {
		a   branchArming
		why string
	}
	var off []withdrawal
	tip, tipRead := "", false
	for _, a := range armed {
		n := fmt.Sprint(a.Number)
		if a.HeadRefOid != res.Head {
			if !res.Push.On {
				res.add("withdraw", StatusLeft, "pull request #"+n+" is armed at "+short(a.HeadRefOid)+", and "+res.Push.offText()+", so land moves no head: its arming is left as it is", "")
				continue
			}
			if !tipRead {
				if tip, err = RemoteTip(o.Git, dir, res.Branch); err != nil {
					return atHead, refuse("land could not ask "+Remote+" where "+res.Branch+" is ("+forgeWhy(err)+"), so it cannot say its push would move pull request #"+n+"'s head; its arming is left as it is",
						"settle what git reports, then run abcd land again; nothing was pushed")
				}
				tipRead = true
			}
			if tip != res.Head {
				if err := movedByPush(res, dir, n, a.HeadRefOid, tip); err != nil {
					return atHead, err
				}
				if !a.own(viewer) {
					vouched, err := headVouched(dir, a.HeadRefOid)
					if err != nil {
						return atHead, refuse("the receipts for "+short(a.HeadRefOid)+" cannot be read ("+forgeWhy(err)+"), so land leaves pull request #"+n+"'s arming by another account as it is",
							"settle it, then run abcd land again; nothing was pushed")
					}
					if !vouched {
						return atHead, refuse("pull request #"+n+" is armed at "+short(a.HeadRefOid)+" by another account (or one the forge does not attribute), and no receipt here names that head, so land leaves the arming as it is and does not push over it",
							"ask whoever armed it, or withdraw it yourself (gh pr merge "+n+" --disable-auto, and take it out of the merge queue on its page), then run abcd land again; nothing was pushed")
					}
				}
				off = append(off, withdrawal{a, "armed at " + short(a.HeadRefOid) + ", and this push moves its head to " + short(res.Head)})
				continue
			}
		}
		// The arming stands at the local head (the forge's reading of its head
		// can lag origin's), so the push moves nothing and the gate decides.
		refused, unread := gateAtLocalHead(res, o, dir, tracking, nameWithOwner, n, ask)
		switch {
		case unread != "":
			atHead = true
			res.add("withdraw", StatusLeft, "the merge gate could not be read for "+short(res.Head)+" ("+unread+"), so pull request #"+n+"'s arming is left as it is", "")
		case refused == "":
			atHead = true
			if a.HeadRefOid != res.Head {
				res.add("withdraw", StatusLeft, Remote+" already holds "+res.Branch+" at "+short(res.Head)+", so the push moves no head: pull request #"+n+"'s arming, which the forge reads at "+short(a.HeadRefOid)+", is left as it is", "")
			}
		case a.own(viewer):
			off = append(off, withdrawal{a, "armed at " + short(res.Head) + ": " + refused})
		default:
			return atHead, refuse("pull request #"+n+" is armed at "+short(res.Head)+" by another account (or one the forge does not attribute), and land leaves that arming as it is: "+refused,
				"ask whoever armed it, or withdraw it yourself (gh pr merge "+n+" --disable-auto, and take it out of the merge queue on its page), then run abcd land again; nothing was pushed")
		}
	}
	if len(off) == 0 {
		return atHead, nil
	}
	var did []string
	// Every return from here says what came off.
	record := func() {
		if len(did) > 0 {
			res.add("withdraw", StatusDone, "land withdrew the arming of "+strings.Join(did, "; ")+"; land arms only a head the merge gate arms, pinned to it", "")
		}
	}
	for _, w := range off {
		n := fmt.Sprint(w.a.Number)
		if err := WithdrawArming(ask, n, w.a.ID, w.a.auto(), w.a.IsInMergeQueue); err != nil {
			record()
			if errors.Is(err, ErrNoForgeClient) {
				return atHead, noForgeClient()
			}
			return atHead, refuse("pull request #"+n+" is "+w.why+", and land could not withdraw its arming: "+forgeWhy(err),
				"settle what the forge reports, or withdraw it yourself (gh pr merge "+n+" --disable-auto, and take it out of the merge queue on its page), then run abcd land again; nothing was pushed")
		}
		did = append(did, "pull request #"+n+" ("+w.why+")")
	}
	record()
	// The forge's word for it, not the call's: gh exits 0 on some failures.
	again, _, err := armings(o, dir, owner, name, res.Branch)
	if err != nil {
		return atHead, err
	}
	for _, w := range off {
		if armedIn(again, w.a.Number) {
			n := fmt.Sprint(w.a.Number)
			return atHead, refuse("land withdrew the arming of pull request #"+n+", but the forge still reports it armed or queued",
				"check it (gh pr view "+n+"), then run abcd land again; nothing was pushed")
		}
	}
	return atHead, nil
}

// gateAtLocalHead asks the merge gate about the local head. refused is its reason
// when it definitely does not arm that head, unread why it could not say; both ""
// is an arming it would make.
func gateAtLocalHead(res *Result, o Options, dir, tracking, nameWithOwner, n string, ask ForgeCall) (refused, unread string) {
	tip, err := gitutil.Run(dir, "rev-parse", "--verify", "--quiet", tracking+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(tip) {
		return "", Remote + "/" + res.Default + " cannot be read"
	}
	d, err := DecideMerge(MergeQuery{RepoRoot: dir, Roots: o.Roots, Tip: tip, TipName: Remote + "/" + res.Default, Default: res.Default,
		Repo: nameWithOwner, PR: n, Head: res.Head, Forge: ask})
	switch {
	case err != nil:
		return "", fsutil.RedactHome(err.Error())
	case d.Unread:
		return "", d.Why
	case d.Arm:
		return "", ""
	}
	return d.Why, ""
}

// movedByPush refuses unless the push will fast-forward: the forge's head and
// origin's branch (tip, "" when origin has none) are both ancestors of the local
// head. A commit this checkout does not hold is no ancestor it can vouch for.
func movedByPush(res *Result, dir, n, head, tip string) error {
	for _, c := range []struct{ what, sha string }{{"pull request #" + n + "'s head on the forge", head}, {Remote + "'s " + res.Branch, tip}} {
		if c.sha == "" {
			continue
		}
		if on, err := gitutil.IsAncestor(dir, c.sha, res.Head); err == nil && on {
			continue
		}
		return refuse(c.what+", "+short(c.sha)+", is not an ancestor of the local head "+short(res.Head)+", so the push would not move it: land leaves pull request #"+n+"'s arming as it is",
			"bring the branch up to it (git pull "+Remote+" "+res.Branch+"), then run abcd land again; nothing was pushed")
	}
	return nil
}

// headVouched reports whether a review receipt or a recorded approval in any
// worktree of the repository at dir names sha.
func headVouched(dir, sha string) (bool, error) {
	if p, err := reviewReceipt(dir, sha); err != nil || p != "" {
		return p != "", err
	}
	p, err := approvalRecord(dir, sha)
	return p != "", err
}

// splitRepo splits the forge's "owner/name", or reports it is not one.
func splitRepo(nameWithOwner string) (owner, name string, ok bool) {
	owner, name, ok = strings.Cut(nameWithOwner, "/")
	if !ok || owner == "" || name == "" || strings.Contains(name, "/") || !ownerName(owner) || !repoName(name) {
		return "", "", false
	}
	return owner, name, true
}

// rearmGate is the merge gate asked about a head already recorded, for a path
// that arms without landing a branch (the release hold's re-arm and admission):
// the mode at the default branch as last fetched, the author's role, the
// approvals and the receipts, in any worktree of root. The rule is left empty
// since it decides only the method, which the arming already carries. An answer
// the gate could not read is an error, never a refusal.
func rearmGate(ask ForgeCall, root, repo, pr, head string) (bool, string, error) {
	def := trackedDefault(root)
	if def == "" {
		return false, "", errors.New("origin's default branch is not tracked here, so the merge gate cannot be read")
	}
	tipName := Remote + "/" + def
	tip, err := gitutil.Run(root, "rev-parse", "--verify", "--quiet", "refs/remotes/"+tipName+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(tip) {
		return false, "", errors.New(tipName + " cannot be read, so the merge gate cannot be read")
	}
	roots, _ := layered.RootsFor(root)
	d, err := DecideMerge(MergeQuery{RepoRoot: root, Roots: roots, Tip: tip, TipName: tipName, Default: def, Repo: repo, PR: pr, Head: head, Forge: ask})
	if err != nil {
		return false, "", err
	}
	if d.Unread {
		return false, "", errors.New(d.Why)
	}
	return d.Arm, d.Why, nil
}

// The release hold's arming is held to the merge gate here: claim cannot import
// this package, and every binary that reaches a re-arm links both. One that
// links claim alone arms nothing.
func init() {
	claim.SetArmGate(func(root, repo, pr, head string) (bool, string, error) {
		return rearmGate(func(args ...string) (string, error) { return GH(root, args...) }, root, repo, pr, head)
	})
}
