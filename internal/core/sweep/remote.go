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
// caller's fetch left them, and the forge's listings through Options.Forge.
// Under apply, each delete is preceded by a last check of everything that
// could hold the branch, made again: origin's own default branch and its tip
// (asked of origin before the first delete, and required to be the very
// branch and commit the judgement read), the forge's pull requests from that
// branch, the worktrees, the local branches' upstreams and the guard's checks
// with a fresh liveness probe, origin's tip of it now, and, last, origin's
// default branch and its tip asked again, held to the same rule. Only then is
// it deleted, by one push of an empty ref with a
// lease on the exact tip that was judged (the reservation sweep's pattern),
// so a branch someone pushed to meanwhile survives. The push runs through
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
	"strconv"
	"strings"
	"time"

	store "github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
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
			head, headTip, headFail = rj.originHead()
		}
		if err := rj.delete(branches[i], head, headTip, headFail, row); err != nil && fault == nil {
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
		if err := rj.askForge(); err != nil {
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

// errForgeTimedOut is a forge call that did not answer within forgeTimeout.
var errForgeTimedOut = errors.New("the forge did not answer in time")

// forge asks the forge, bounded by forgeTimeout and the run's context. A call
// still running at the deadline is abandoned, and its answer is no answer.
//
// An abandoned call may still be running in its goroutine when forge
// returns. That is safe: the channel is buffered, so the goroutine's one send
// never blocks and it ends as soon as the call does, and nothing it shares
// with forge is written after forge returns; its answer is dropped. Once the
// run's context has ended no call starts at all, here or at the callers, which
// check the context first, so a stopped run never has two forge calls in
// flight. After a timeout alone the next call may overlap the abandoned one,
// so Options.Forge must not rely on its calls being serialised.
func (rj *remoteJudge) forge(args ...string) (string, error) {
	if rj.o.Forge == nil {
		return "", errors.New("no forge was asked")
	}
	if err := rj.ctx.Err(); err != nil {
		return "", err
	}
	type answer struct {
		out string
		err error
	}
	ch := make(chan answer, 1)
	go func() {
		out, err := rj.o.Forge(rj.root, args...)
		ch <- answer{out, err}
	}()
	t := time.NewTimer(forgeTimeout)
	defer t.Stop()
	select {
	case a := <-ch:
		return a.out, a.err
	case <-t.C:
		return "", errForgeTimedOut
	case <-rj.ctx.Done():
		return "", rj.ctx.Err()
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
// with the outcome. head is origin's default branch as origin named it before
// the run's first delete and headTip its tip there, or headFail why they
// could not be asked; the last check asks origin for both again just before
// the push. A hold the last check finds keeps the branch, naming it; so does
// origin refusing the lease because the branch moved. Any other refusal of
// the delete keeps it, naming why, and is returned as the run's fault.
func (rj *remoteJudge) delete(b remoteBranch, head, headTip string, headFail error, row *Row) error {
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
	if reason, ev := rj.headHold(b, head, headTip, headFail); reason != "" {
		return held(reason, ev...)
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
	if err := rj.readLocal(); err != nil {
		return held(err.Error())
	}
	rj.live.probe = newProbe(rj.ctx, rj.o.Probe)
	if ev := rj.holds(b, map[string][]string{b.name: open}, bases, ""); len(ev) > 0 {
		return held(details(ev), ev...)
	}
	ctx, cancel := context.WithTimeout(rj.ctx, remoteNetTimeout)
	defer cancel()
	tip, found, err := lsRemoteTipContext(ctx, rj.root, claim.Remote, b.name)
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
		return nil
	}
	why := gitutil.ScrubURLCredentials(pushRefusal(out, err))
	if strings.Contains(why, "stale info") {
		return held("it moved on origin after the last check, and the lease refused the delete")
	}
	row.Reason = clean("the delete on origin failed, so the branch is kept: " + why)
	return errors.New(clean("sweep: the delete of " + b.name + " on origin failed: " + why))
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
