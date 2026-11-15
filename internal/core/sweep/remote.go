package sweep

// remote.go is the half of a sweep that judges branches on origin
// (spc-2610230730269295, step 3, Merged pull-request branches on origin): one
// row per branch this checkout last fetched from origin. A branch is deleted
// under apply when the store's merged judgement shows it merged into origin's
// default branch (the forge records its pull request merged at the tip, or
// its content is on the default branch) and nothing still needs it:
//
//   - no open pull request heads it, or targets it as its base, as the forge
//     lists them (a branch an open pull request merges into is still in use,
//     which keeps more than "heads it" alone);
//   - no reservation names it: a live reservation whose record id its name
//     holds; a reservation that cannot be read, or reservations never
//     fetched, hold every branch. A reservation carries no branch, and a
//     merged branch's own changes are already on the default branch, so a
//     footprint is compared with a branch only through a local worktree on
//     it, below;
//   - no running session works on it: the liveness probe, reservations made
//     from it or covering its changes among its proofs, holds no local
//     worktree on that branch (checked out under its name, or tracking it);
//   - and the shell guard's delete proof passes it (branchHazard): it is not
//     checked out in a worktree, held by a stopped rebase or bisect, or one
//     of abcd's coordination branches.
//
// Never deleted: the default branch, a release branch, a reservation branch
// (abcd-reserved/*, which only the reservation sweep's workflow deletes), a
// status branch, origin's HEAD, and a branch the sweep cannot show merged.
// Only refs under refs/heads/ are ever named in a delete, so a tag never is.
//
// The judgement reaches no remote: it reads origin's branches as the
// caller's fetch left them, and the forge's listings through Options.Forge,
// pinned to origin's repository (forge.go); a forge that cannot be matched
// to origin lists nothing, so every branch is kept. Under apply, origin's
// fetch and push addresses are read again once, before the first delete,
// with the person's own configuration (the one the push runs under): unless
// origin is one repository at every address (oneRepository), still the one
// the forge was matched to, the last check below cannot see the repository
// the push would change, and every delete is held. Each delete is then preceded by a last check of
// everything that could hold the branch, made again: origin's own default
// branch and its tip
// (asked of origin before the first delete, and required to be the very
// branch and commit the judgement read), the forge's pull requests from that
// branch, the worktrees, the local branches' upstreams and the guard's checks
// with a fresh liveness probe, origin's tip of it now, and, last, origin's
// default branch and its tip asked again, held to the same rule. Only then is
// it deleted, by one push of an empty ref with a
// lease on the exact tip that was judged (the reservation sweep's pattern),
// so a branch someone pushed to meanwhile survives. After a delete origin
// took, origin is asked for its default branch and tip once more: when they
// moved, the row notes the tip the branch was judged at and the command that
// restores it, in case its work is no longer there. The push runs through
// gitutil.Net, so the repository's own pre-push hook sees it, as it sees the
// reservation sweep's; a hook that refuses it keeps the branch and is the
// run's fault, as any other failed push is. Reservations are the ones the
// caller read after its fetch, as the worktree apply reads them.
//
// Every git command this file runs goes through gitutil. The reads are bound
// to the run's context and a deadline; the delete's push is bound to
// gitutil.NetTimeout, the hooked runner taking no context; and the shared
// merged judgement (store.Merged) and the liveness probe's changed-paths
// read run their own git reads with no deadline. Every forge call is bounded
// by forgeTimeout and the context.
//
// sweepRemote is an entry point the surfaces (step 5) call, and so stays
// unexported until then, as runDry does.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	store "github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/shellquote"
)

const (
	// forgeTimeout bounds one forge call: a forge that does not answer in time
	// is no answer, and no answer keeps every branch it was asked about.
	forgeTimeout = 30 * time.Second
	// openListLimit is how many open pull requests the run's listing asks
	// for. A listing that reaches it may have left some out, so it cannot
	// show any branch free of one, and every branch is kept.
	openListLimit = 1000
	// mergedListLimit is how many merged pull requests the run's listing asks
	// for, newest first. One left out costs only the forge's word on that
	// branch, which is then judged by its content alone.
	mergedListLimit = 500
	// remoteNetTimeout bounds each networked git command of the last check
	// and the delete.
	remoteNetTimeout = 2 * time.Minute
)

// remotePrefix is where this checkout keeps what it last fetched of origin's
// branches.
const remotePrefix = "refs/remotes/" + claim.Remote + "/"

// beforeRemoteDelete, when a test sets it, runs between the last check of a
// branch on origin and its delete: the window another writer's push lands in.
var beforeRemoteDelete func(ref string)

// afterRemoteDelete, when a test sets it, runs between a delete origin took
// and the question to origin that follows it: the window in which origin's
// default branch can move after the delete.
var afterRemoteDelete func(ref string)

// readRemoteTip reads origin's tip of a branch for the last check before a
// delete: lsRemoteTipContext, which a test replaces to end the run's context
// while the read is under way.
var readRemoteTip = lsRemoteTipContext

// remoteBranch is one branch on origin as this checkout last fetched it.
type remoteBranch struct {
	// name is the branch's name on origin, ref its full ref there, and
	// tracking this checkout's ref for it.
	name, ref, tracking string
	tip                 string
}

// remoteJudge is what every branch on origin of one run is judged against.
type remoteJudge struct {
	ctx  context.Context
	root string
	o    Options
	// def is origin's default branch as last fetched, defName its name, and
	// defTip the commit it named when the branches were judged.
	def, defName, defTip string
	wts                  []gitutil.Worktree
	held                 gitutil.Held
	// upstreams maps each local branch's ref to its upstream's ref.
	upstreams map[string]string
	// live judges a worktree's liveness, against one reading of the process
	// table.
	live judge
	// open maps a branch to the numbers of the open pull requests from it,
	// openBases to those that target it as their base, and openUnread says
	// why no listing can show a branch free of either.
	open, openBases map[string][]string
	openUnread      string
	// merged is the forge's listing of merged pull requests, nil when it gave
	// none.
	merged []byte
}

// sweepRemote judges every branch on origin this checkout last fetched and,
// with apply, deletes each one judged deletable, after a last check, under a
// lease on the tip that was judged; each row carries its Outcome then. A
// failure to read what the branches are judged against refuses the run. A
// delete origin refuses for any reason but the branch having moved keeps that
// branch, naming why, and is the run's fault, carried as its Refusal and
// returned as the error, so ExitCode reads ExitRefused; the other rows are
// still acted on.
func sweepRemote(ctx context.Context, repoRoot string, o Options, apply bool) (Result, error) {
	rj, branches, err := newRemoteJudge(ctx, repoRoot, o)
	if err != nil {
		return refused(err)
	}
	res := Result{
		Rows:                    make([]Row, 0, len(branches)),
		ProcessCheckUnavailable: clean(rj.live.probe.ProcessCheckUnavailable()),
		ReservationsUnread:      clean(rj.live.probe.ReservationsUnread()),
		RunsUnread:              clean(rj.live.probe.runsUnread),
	}
	if rj.openUnread != "" {
		res.ForgeNote = clean("no open pull request can be shown absent (" + rj.openUnread + "), so no branch on origin is deleted")
	}
	for _, b := range branches {
		// A run whose context ended stops judging: a row judged after it
		// could rest on reads git was killed for.
		if err := ctx.Err(); err != nil {
			return refused(fmt.Errorf("the run was stopped while judging origin's branches: %w", err))
		}
		res.Rows = append(res.Rows, rj.row(b))
	}
	if !apply {
		return res, nil
	}
	res.Applied = true
	var (
		fault    error
		asked    bool
		addrHold string
		head     string
		headTip  string
		headFail error
	)
	for i := range res.Rows {
		row := &res.Rows[i]
		row.Outcome = OutcomeKept
		if row.Action != ActionDelete {
			continue
		}
		if err := ctx.Err(); err != nil {
			if fault == nil {
				fault = fmt.Errorf("the run was stopped before every delete was made: %w", err)
			}
			continue
		}
		if !asked {
			// The tracked default branch is a local reading and may be stale,
			// so an act on origin asks origin which branch is its default.
			asked = true
			addrHold = rj.addressHold()
			head, headTip, headFail = rj.originHead()
		}
		if err := rj.delete(branches[i], addrHold, head, headTip, headFail, row); err != nil && fault == nil {
			fault = err
		}
	}
	if fault != nil {
		_, err := refused(fault)
		res.Refusal = err.Error()
		return res, err
	}
	return res, nil
}

// newRemoteJudge reads what the branches on origin are judged against, and
// the branches themselves.
func newRemoteJudge(ctx context.Context, root string, o Options) (*remoteJudge, []remoteBranch, error) {
	rj := &remoteJudge{ctx: ctx, root: root, o: o}
	if err := rj.readLocal(); err != nil {
		return nil, nil, err
	}
	ctx2, cancel := context.WithTimeout(ctx, gitReadTimeout)
	rj.def = gitutil.TrackedDefaultBranchContext(ctx2, root, claim.Remote)
	err := ctx2.Err()
	cancel()
	if err != nil {
		return nil, nil, fmt.Errorf("reading origin's default branch: %w", err)
	}
	rj.defName = shortName(rj.def)
	if rj.def != "" {
		if rj.defTip, err = refTip(ctx, root, rj.def); err != nil {
			return nil, nil, fmt.Errorf("reading origin's default branch: %w", err)
		}
	}
	refs, err := listRefs(ctx, root, remotePrefix)
	if err != nil {
		return nil, nil, fmt.Errorf("listing origin's branches: %w", err)
	}
	var branches []remoteBranch
	for _, r := range refs {
		if r.symref != "" {
			// origin/HEAD names the default branch; it is no branch of its own.
			continue
		}
		name := strings.TrimPrefix(r.ref, remotePrefix)
		branches = append(branches, remoteBranch{name: name, ref: "refs/heads/" + name, tracking: r.ref, tip: r.tip})
	}
	rj.live = judge{def: rj.def, probe: newProbe(ctx, o.Probe), changes: o.changes}
	if rj.live.changes == nil {
		rj.live.changes = changedPaths
	}
	if len(branches) > 0 {
		rj.o = rj.o.pinnedForge(ctx, root)
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("the run was stopped while the forge's repository was matched to origin: %w", err)
		}
		if rj.o.forgeUnmatched != "" {
			// No pull request is listed from a repository that may not be
			// origin's, so none can be shown absent and every branch is kept.
			rj.openUnread = rj.o.forgeUnmatched
		} else if err := rj.askForge(); err != nil {
			return nil, nil, err
		}
	}
	return rj, branches, nil
}

// readLocal reads the worktrees, what a stopped rebase or bisect holds and
// each local branch's upstream, each time anew: the last check before a
// delete reads them again, so a worktree that began tracking the branch under
// another name since the judgement is judged for liveness too.
func (rj *remoteJudge) readLocal() error {
	ctx, cancel := context.WithTimeout(rj.ctx, gitReadTimeout)
	defer cancel()
	wts, err := gitutil.ListWorktreesContext(ctx, rj.root, gitutil.MaxWorktreeListing)
	if err != nil {
		return fmt.Errorf("listing the repository's worktrees: %w", err)
	}
	common, err := gitutil.CommonDirContext(ctx, rj.root)
	if err != nil {
		return fmt.Errorf("reading the checkout's common directory: %w", err)
	}
	locals, err := listRefs(rj.ctx, rj.root, "refs/heads/")
	if err != nil {
		return fmt.Errorf("listing the local branches: %w", err)
	}
	upstreams := map[string]string{}
	for _, l := range locals {
		if l.upstream != "" {
			upstreams[l.ref] = l.upstream
		}
	}
	rj.wts, rj.held, rj.upstreams = wts, gitutil.HeldBranches(common), upstreams
	return nil
}

// The stages of the last check before a delete on origin that
// Options.onLastCheck is told of: the worktrees and local branches read
// again, and the liveness probe made again.
const (
	lastCheckWorktrees = "worktrees"
	lastCheckProbe     = "probe"
)

// passed tells Options.onLastCheck, when set, that the last check passed
// stage.
func (o Options) passed(stage string) {
	if o.onLastCheck != nil {
		o.onLastCheck(stage)
	}
}

// errForgeTimedOut is a forge call that did not answer within forgeTimeout.
var errForgeTimedOut = errors.New("the forge did not answer in time")

// forge asks the forge for the run's judgement of origin's branches, through
// forgeCall.
func (rj *remoteJudge) forge(args ...string) (string, error) {
	return forgeCall(rj.ctx, rj.o.Forge, rj.root, args...)
}

// forgeCall asks the forge client gh from root, bounded by forgeTimeout and
// ctx. A call still running at the deadline is abandoned, and its answer is
// no answer.
//
// An abandoned call may still be running in its goroutine when forgeCall
// returns. That is safe: the channel is buffered, so the goroutine's one send
// never blocks and it ends as soon as the call does, and nothing it shares
// with forgeCall is written after forgeCall returns; its answer is dropped.
// Once ctx has ended no call starts at all, here or at the callers, which
// check the context first, so a stopped run never has two forge calls in
// flight. After a timeout alone the next call may overlap the abandoned one,
// so Options.Forge must not rely on its calls being serialised.
func forgeCall(ctx context.Context, gh store.GH, root string, args ...string) (string, error) {
	if gh == nil {
		return "", errors.New("no forge was asked")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	type answer struct {
		out string
		err error
	}
	ch := make(chan answer, 1)
	go func() {
		out, err := gh(root, args...)
		ch <- answer{out, err}
	}()
	t := time.NewTimer(forgeTimeout)
	defer t.Stop()
	select {
	case a := <-ch:
		return a.out, a.err
	case <-t.C:
		return "", errForgeTimedOut
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// askForge reads the run's two listings: every open pull request, which must
// be whole to show a branch free of one, and the newest merged ones, the
// forge's word on which branches landed. A run whose context ended asks
// nothing further and is refused, as judging is.
func (rj *remoteJudge) askForge() error {
	if err := rj.ctx.Err(); err != nil {
		return fmt.Errorf("the run was stopped before the forge was asked: %w", err)
	}
	out, err := rj.forge("pr", "list", "--state", "open", "--limit", strconv.Itoa(openListLimit), "--json", "number,headRefName,baseRefName,state")
	if err != nil {
		rj.openUnread = "the forge could not be asked: " + answerText(err)
	} else if heads, bases, n, perr := openByBranch(out); perr != nil {
		rj.openUnread = "the forge's listing of open pull requests does not parse"
	} else if n >= openListLimit {
		rj.openUnread = fmt.Sprintf("the forge listed %d open pull requests, its limit, so more may be open", n)
	} else {
		rj.open, rj.openBases = heads, bases
	}
	if err := rj.ctx.Err(); err != nil {
		return fmt.Errorf("the run was stopped while the forge was asked: %w", err)
	}
	if out, err := rj.forge("pr", "list", "--state", "merged", "--limit", strconv.Itoa(mergedListLimit),
		"--json", "headRefName,headRefOid,baseRefName,isCrossRepository,state"); err == nil {
		rj.merged = []byte(out)
	}
	if err := rj.ctx.Err(); err != nil {
		return fmt.Errorf("the run was stopped while the forge was asked: %w", err)
	}
	return nil
}

// openByBranch reads a listing of open pull requests into the numbers of the
// open ones from each head branch and into each base branch, and how many the
// listing held. One from a fork's branch of the same name counts too: this
// errs towards keeping.
func openByBranch(listing string) (heads, bases map[string][]string, n int, err error) {
	var prs []struct {
		Number      int    `json:"number"`
		HeadRefName string `json:"headRefName"`
		BaseRefName string `json:"baseRefName"`
		State       string `json:"state"`
	}
	if err := json.Unmarshal([]byte(listing), &prs); err != nil {
		return nil, nil, 0, err
	}
	heads, bases = map[string][]string{}, map[string][]string{}
	for _, pr := range prs {
		if pr.State == "OPEN" {
			num := strconv.Itoa(pr.Number)
			heads[pr.HeadRefName] = append(heads[pr.HeadRefName], num)
			if pr.BaseRefName != "" {
				bases[pr.BaseRefName] = append(bases[pr.BaseRefName], num)
			}
		}
	}
	return heads, bases, len(prs), nil
}

// neverDeleted is why a branch on origin is on the never-delete list, "" when
// it is not. A release branch is matched loosely, on its name opening with
// "release" in any case, erring towards keeping.
func neverDeleted(name, defName string) string {
	ref := "refs/heads/" + name
	switch {
	case name == "HEAD":
		return "origin's HEAD"
	case defName != "" && name == defName:
		return "the default branch"
	case strings.HasPrefix(ref, claim.RefPrefix):
		return "a reservation branch, which only the reservation sweep's workflow deletes"
	case strings.HasPrefix(ref, claim.StatusRefPrefix):
		return "a status branch, which holds a person's announcements"
	case strings.HasPrefix(strings.ToLower(name), "release"):
		return "a release branch"
	}
	return ""
}

// row judges one branch on origin: the never-delete list first, then the
// merged judgement, then everything that could still need it.
func (rj *remoteJudge) row(b remoteBranch) Row {
	row := Row{Kind: RowRemoteBranch, Name: clean(b.name), State: clean(displayRef(b.tracking) + " at " + short(b.tip)), Action: ActionKeep}
	keep := func(reason string, ev ...Evidence) Row {
		row.Reason = clean(reason)
		for _, e := range ev {
			e.Detail = clean(e.Detail)
			row.Evidence = append(row.Evidence, e)
		}
		return row
	}
	if why := neverDeleted(b.name, rj.defName); why != "" {
		return keep("never deleted: "+why, Evidence{Kind: KindProtected, Strength: Proven, Detail: why})
	}
	if rj.def == "" {
		return keep("origin's default branch has not been fetched here, so nothing can be shown merged into it",
			Evidence{Kind: KindMerged, Strength: Hint, Detail: "no default branch of origin"})
	}
	if !gitutil.IsFullSHA(b.tip) {
		return keep("cannot classify: git named no commit for it", Evidence{Kind: KindMerged, Strength: Hint, Detail: "no commit"})
	}
	merged, by := store.Merged(rj.root, rj.def, b.tracking, b.tip, rj.merged)
	if !merged {
		return keep("never deleted: the sweep cannot show it is merged into "+displayRef(rj.def),
			Evidence{Kind: KindMerged, Strength: Proven, Detail: "unmerged"})
	}
	ev := []Evidence{{Kind: KindMerged, Strength: Proven, Detail: "merged, judged by " + by}}
	holds := rj.holds(b, rj.open, rj.openBases, rj.openUnread)
	if len(holds) > 0 {
		return keep("merged, judged by "+by+", but still needed: "+details(holds), append(ev, holds...)...)
	}
	row.Action = ActionDelete
	return keep("merged, judged by "+by+"; no open pull request heads it, no reservation names it and no running session works on it", ev...)
}

// details is the evidence's details joined.
func details(ev []Evidence) string {
	out := make([]string, len(ev))
	for i, e := range ev {
		out[i] = e.Detail
	}
	return strings.Join(out, "; ")
}

// holds is everything that still needs the branch b: an open pull request
// from it (open) or targeting it (bases), or every branch when openUnread
// says no listing can show one absent; a reservation; the guard's checks; and
// the liveness of each local worktree on the branch.
func (rj *remoteJudge) holds(b remoteBranch, open, bases map[string][]string, openUnread string) []Evidence {
	var ev []Evidence
	if openUnread != "" {
		ev = append(ev, Evidence{Kind: KindPullRequest, Strength: Hint, Detail: "no open pull request can be shown absent: " + openUnread})
	} else {
		if nums := open[b.name]; len(nums) > 0 {
			ev = append(ev, Evidence{Kind: KindPullRequest, Strength: Proven, Detail: "pull request #" + strings.Join(nums, ", #") + " is open on it"})
		}
		if nums := bases[b.name]; len(nums) > 0 {
			ev = append(ev, Evidence{Kind: KindPullRequest, Strength: Proven, Detail: "open pull request #" + strings.Join(nums, ", #") + " targets it as its base"})
		}
	}
	ev = append(ev, rj.reservationsNaming(b)...)
	if pr, bad := branchHazard(rj.def, rj.wts, rj.held, b.name); bad {
		ev = append(ev, Evidence{Kind: KindDeleteProof, Strength: Proven, Detail: pr.Failed})
	}
	for _, wt := range rj.wts {
		if wt.Bare || wt.Branch == "" || (wt.Branch != b.ref && rj.upstreams[wt.Branch] != b.tracking) {
			continue
		}
		l := rj.live.liveness(wt.Path)
		if !l.Held {
			continue
		}
		where := "the worktree " + fsutil.RedactHome(wt.Path) + " on it: "
		for _, e := range l.Evidence {
			e.Detail = where + e.Detail
			ev = append(ev, e)
		}
	}
	return ev
}

// reservationsNaming is the reservations that name the branch b: a live one
// whose record id the branch's name holds. Reservations never fetched hold
// every branch, as they hold every worktree, and so does an unreadable one
// still within its hold, its footprint unknown.
func (rj *remoteJudge) reservationsNaming(b remoteBranch) []Evidence {
	p := rj.live.probe
	var ev []Evidence
	if p.unread != "" {
		ev = append(ev, Evidence{Kind: KindReservationCheck, Strength: Hint, Detail: "reservations unread: " + p.unread})
	}
	lower := strings.ToLower(b.name)
	for _, r := range p.reservations.Reservations {
		until := r.EffectiveExpiry.UTC().Format(time.RFC3339)
		switch {
		case r.State == claim.StateLive && r.Record != "" && strings.Contains(lower, strings.ToLower(r.Record)):
			ev = append(ev, Evidence{Kind: KindReservation, Strength: Proven, Detail: fmt.Sprintf(
				"reserved as %s by %s until %s, the record its name holds", r.Record, r.Holder, until)})
		case r.State == claim.StateUnreadable && r.EffectiveExpiry.After(p.now):
			holder := r.Holder
			if holder == "" {
				holder = "a holder its tip does not name readably"
			}
			ev = append(ev, Evidence{Kind: KindReservation, Strength: Hint, Detail: fmt.Sprintf(
				"reservation %s by %s is unreadable (%s) and holds until %s, its footprint unknown", r.Record, holder, r.Unreadable, until)})
		}
	}
	return ev
}

// originHead asks origin which branch is its default and that branch's tip,
// bounded by the run's context and remoteNetTimeout.
func (rj *remoteJudge) originHead() (branch, tip string, err error) {
	ctx, cancel := context.WithTimeout(rj.ctx, remoteNetTimeout)
	defer cancel()
	return gitutil.RemoteHeadTip(ctx, gitutil.NetContext, rj.root, claim.Remote)
}

// addressHold is why origin's addresses hold every delete, "" when they do
// not. Origin, as git resolves it under the person's own configuration (the
// one the delete's push runs under, rewrites included), must be one
// repository (oneRepository): exactly one fetch address, and every push
// address naming the repository it names, so the last check's reads of
// origin and the push reach the same repository. It must also still be the
// repository the forge was matched to, when one was (pinnedForge), so the
// pull requests the judgement read are that repository's. A read that fails
// holds every delete too. No address is shown: one may carry credentials,
// and the reason needs none.
func (rj *remoteJudge) addressHold() string {
	fetch, push, err := originAddresses(rj.ctx, rj.root)
	if err != nil {
		return answerText(err) + ", so nothing on origin is deleted"
	}
	key, why := oneRepository(fetch, push)
	switch {
	case why != "":
		return why
	case rj.o.forgeKey != "" && key != rj.o.forgeKey:
		return "origin names another repository than the one the forge was matched to when the run began, so the sweep cannot check what it would delete"
	}
	return ""
}

// headHold is why origin's default branch, as origin named it, holds the
// delete of the branch b, "" when it does not: head is that branch and
// headTip its tip there, or headFail why they could not be asked. Only the
// very branch and tip the judgement read let a delete through.
func (rj *remoteJudge) headHold(b remoteBranch, head, headTip string, headFail error) (string, []Evidence) {
	switch {
	case headFail != nil:
		return "origin's default branch could not be asked (" + answerText(headFail) + "), so nothing on origin is deleted", nil
	case b.name == head:
		return "it is origin's default branch", []Evidence{{Kind: KindRemote, Strength: Proven, Detail: "origin's default branch"}}
	case head != rj.defName:
		return "origin's default branch is " + head + ", but it was judged against " + displayRef(rj.def) +
			" as last fetched; fetch origin and run the sweep again", nil
	case headTip != rj.defTip:
		// Equality is the simple rule, and the safe one: a default branch
		// that moved on origin since the fetch may hold, or have dropped,
		// what the merged judgement read, so only the exact tip judged
		// against lets a delete through.
		return "origin's default branch is at " + short(headTip) + ", but it was judged against " + displayRef(rj.def) +
			" at " + short(rj.defTip) + " as last fetched; fetch origin and run the sweep again", nil
	}
	return "", nil
}

// delete makes the last check of the branch b on origin and, when nothing
// holds it, deletes it under a lease on the tip that was judged, marking row
// with the outcome. addrHold is why origin's addresses hold every delete
// (addressHold), "" when they do not. head is origin's default branch as
// origin named it before the run's first delete and headTip its tip there, or
// headFail why they could not be asked; the last check asks origin for both
// again just before the push. A hold the last check finds keeps the branch, naming it; so does
// origin refusing the lease because the branch moved. Any other refusal of
// the delete keeps it, naming why, and is returned as the run's fault.
func (rj *remoteJudge) delete(b remoteBranch, addrHold, head, headTip string, headFail error, row *Row) error {
	held := func(reason string, ev ...Evidence) error {
		row.Reason = clean("held at the last check before the delete: " + reason)
		for _, e := range ev {
			e.Detail = clean(e.Detail)
			row.Evidence = append(row.Evidence, e)
		}
		return nil
	}
	// stopped keeps the branch once the run's context has ended: no further
	// forge call or delete is made, and the stop is the run's fault.
	stopped := func() error {
		err := rj.ctx.Err()
		if err != nil {
			row.Reason = clean("the run was stopped before the delete, so the branch is kept")
			return fmt.Errorf("the run was stopped before every delete was made: %w", err)
		}
		return nil
	}
	// The run's questions to origin before its first delete may have been
	// cut short by the run's end, which reads as no answer: a stopped run is
	// reported as stopped, never as a hold. Every read below is followed by
	// the same check, before its answer is judged.
	if err := stopped(); err != nil {
		return err
	}
	if reason, ev := rj.headHold(b, head, headTip, headFail); reason != "" {
		return held(reason, ev...)
	}
	if addrHold != "" {
		return held(addrHold)
	}
	if !strings.HasPrefix(b.ref, "refs/heads/") || neverDeleted(b.name, head) != "" || !gitutil.IsFullSHA(b.tip) {
		return held("it is not a branch the sweep deletes")
	}
	if err := stopped(); err != nil {
		return err
	}
	// The branch is passed as one argument, --head=<name>, so a name is never
	// read as a flag of its own.
	prs, err := rj.forge("pr", "list", "--head="+b.name, "--state", "all", "--limit", strconv.Itoa(proofPRLimit),
		"--json", "number,headRefName,headRefOid,baseRefName,isCrossRepository,state")
	if serr := stopped(); serr != nil {
		return serr
	}
	if err != nil {
		return held("the forge could not be asked again whether a pull request is open on it (" + answerText(err) + ")")
	}
	open, err := openPullRequests(prs, b.name)
	if err != nil {
		return held("the forge's listing of its pull requests does not parse")
	}
	targeting, err := rj.forge("pr", "list", "--base="+b.name, "--state", "open", "--limit", strconv.Itoa(proofPRLimit),
		"--json", "number,headRefName,baseRefName,state")
	if serr := stopped(); serr != nil {
		return serr
	}
	if err != nil {
		return held("the forge could not be asked again whether an open pull request targets it (" + answerText(err) + ")")
	}
	_, bases, n, err := openByBranch(targeting)
	switch {
	case err != nil:
		return held("the forge's listing of the pull requests targeting it does not parse")
	case n >= proofPRLimit:
		return held(fmt.Sprintf("the forge listed %d open pull requests targeting it, its limit, so more may be open", n))
	}
	if merged, _ := store.Merged(rj.root, rj.def, b.tracking, b.tip, []byte(prs)); !merged {
		return held("it no longer reads as merged into " + displayRef(rj.def))
	}
	// The repository's own state is read last, closest to the push: the
	// worktrees, their upstreams and the guard's checks, and a fresh process
	// table.
	lerr := rj.readLocal()
	rj.o.passed(lastCheckWorktrees)
	if err := stopped(); err != nil {
		return err
	}
	if lerr != nil {
		return held(lerr.Error())
	}
	rj.live.probe = newProbe(rj.ctx, rj.o.Probe)
	ev := rj.holds(b, map[string][]string{b.name: open}, bases, "")
	rj.o.passed(lastCheckProbe)
	if err := stopped(); err != nil {
		return err
	}
	if len(ev) > 0 {
		return held(details(ev), ev...)
	}
	ctx, cancel := context.WithTimeout(rj.ctx, remoteNetTimeout)
	defer cancel()
	tip, found, err := readRemoteTip(ctx, rj.root, claim.Remote, b.name)
	if serr := stopped(); serr != nil {
		return serr
	}
	switch {
	case err != nil:
		return held("origin's tip of it could not be read (" + answerText(err) + ")")
	case !found:
		return held("origin no longer holds it", Evidence{Kind: KindRemote, Strength: Proven, Detail: "not on origin"})
	case tip != b.tip:
		return held("it moved on origin since it was judged (now "+short(tip)+")",
			Evidence{Kind: KindRemote, Strength: Proven, Detail: "origin's tip is " + short(tip)})
	}
	// origin's default branch is asked again, the last check before the push:
	// the answer the run asked for before its first delete may be older than
	// this delete, and the merged judgement holds only against the branch and
	// tip it read.
	head, headTip, headFail = rj.originHead()
	if err := stopped(); err != nil {
		return err
	}
	if reason, ev := rj.headHold(b, head, headTip, headFail); reason != "" {
		return held(reason, ev...)
	}
	if beforeRemoteDelete != nil {
		beforeRemoteDelete(b.ref)
	}
	if err := stopped(); err != nil {
		return err
	}
	// The delete goes through the push the repository's own hooks see
	// (gitutil.Net, the runner the reservation sweep's delete uses), so its
	// pre-push hook judges this push as it judges the person's own. No
	// exported hooked runner takes a context, so this one call is bounded by
	// gitutil.NetTimeout rather than the run's context, which is checked just
	// before it starts; bounding the wait here would leave git running past
	// it.
	out, err := gitutil.Net(rj.root, remoteDeleteArgs(b.ref, b.tip)...)
	if err == nil {
		row.Outcome = OutcomeDeleted
		row.Reason = clean("deleted on origin under a lease on its judged tip " + short(b.tip) + "; " + row.Reason)
		return rj.afterDelete(b, row)
	}
	why := gitutil.ScrubURLCredentials(pushRefusal(out, err))
	if strings.Contains(why, "stale info") {
		return held("it moved on origin after the last check, and the lease refused the delete")
	}
	row.Reason = clean("the delete on origin failed, so the branch is kept: " + why)
	return errors.New(clean("sweep: the delete of " + b.name + " on origin failed: " + why))
}

// afterDelete asks origin once more for its default branch and that branch's
// tip, after origin took the delete of the branch b, row's outcome being
// deleted. The last check asked just before the push, but origin's default
// branch can move between that answer and the push, and the merged judgement
// holds only against the branch and tip it read. When origin's answer no
// longer matches them, which an ordinary merge landing in that window also
// does, the outcome stays deleted and the row adds a note: the tip the branch
// was judged at and the command that puts it back, in case the branch's work
// is no longer on the default branch. That is no fault. An answer origin
// does not give adds a note saying so, with no restore line and no fault;
// one cut short by the run's end is the run's fault, as any stop is.
func (rj *remoteJudge) afterDelete(b remoteBranch, row *Row) error {
	if afterRemoteDelete != nil {
		afterRemoteDelete(b.ref)
	}
	head, headTip, err := rj.originHead()
	if cerr := rj.ctx.Err(); cerr != nil {
		row.Reason += clean("; origin could not be asked for its default branch after the delete, because the run was stopped")
		return fmt.Errorf("the run was stopped after a delete, before origin was asked for its default branch again: %w", cerr)
	}
	if err != nil {
		row.Reason += clean("; origin could not be asked for its default branch after the delete (" + answerText(err) + ")")
		return nil
	}
	if head == rj.defName && headTip == rj.defTip {
		return nil
	}
	moved := "origin's default branch is " + head + " at " + short(headTip) + " after the delete, but the branch was judged against " +
		displayRef(rj.def) + " at " + short(rj.defTip)
	note := "; note: " + moved + "; in case the branch's work is no longer on " + rj.defName + ", its judged tip was " + b.tip
	if restore, ok := remoteRestoreCommand(b.ref, b.tip); ok {
		note += ", and `" + restore + "` restores it"
	} else {
		note += "; its name holds a character no printed command carries safely, so quote the name by hand to restore it at that tip"
	}
	row.Reason += clean(note)
	row.Evidence = append(row.Evidence, Evidence{Kind: KindRemote, Strength: Hint, Detail: clean(moved)})
	return nil
}

// remoteRestoreCommand is the command that puts back the branch at ref on
// origin at its judged tip sha, after a delete, and whether one can be
// printed. The ref is a name origin holds, so the refspec is one shell word:
// left bare when every character is one a shell takes unquoted, and
// single-quoted otherwise (shellquote.Single). A ref holding a backquote or
// any character outside printable ASCII gets no command at all.
func remoteRestoreCommand(ref, sha string) (string, bool) {
	for _, r := range ref {
		if r == '`' || r < 0x20 || r > 0x7e {
			return "", false
		}
	}
	return "git push " + claim.Remote + " " + shellWord(sha+":"+ref), true
}

// shellSafeRe is the text a shell takes unquoted as one word.
var shellSafeRe = regexp.MustCompile(`^[A-Za-z0-9._/:@%+=-]+$`)

// shellWord is s as one shell word: bare when the shell takes it unquoted,
// single-quoted otherwise.
func shellWord(s string) string {
	if shellSafeRe.MatchString(s) {
		return s
	}
	return shellquote.Single(s)
}

// answerText is the first line of an error origin or the forge gave, for a
// row's reason: credentials a URL in it carries are scrubbed first, before
// the cut, so a cut across a URL cannot leave part of one behind, and the
// row's own pass (clean) then redacts the home and masks control bytes.
func answerText(err error) string {
	return firstLine(gitutil.ScrubURLCredentials(err.Error()))
}

// remoteDeleteArgs is the one push a delete makes: an empty ref pushed to ref
// on origin, leased on its tip sha, which origin's receive-pack checks under
// its ref lock.
func remoteDeleteArgs(ref, sha string) []string {
	return []string{"push", "--porcelain", "--force-with-lease=" + ref + ":" + sha, claim.Remote, ":" + ref}
}

// pushRefusal is a failed push's reason: the porcelain lines naming a refused
// ref, then git's error.
func pushRefusal(stdout string, err error) string {
	var lines []string
	for _, l := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(l, "!") {
			lines = append(lines, strings.Join(strings.Fields(l), " "))
		}
	}
	if len(lines) == 0 {
		return err.Error()
	}
	return strings.Join(lines, "; ") + "; " + err.Error()
}
