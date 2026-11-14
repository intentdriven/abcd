package sweep

// branches.go is the local-branch half of a sweep (spc-2610230730269295,
// step 3, Local branches): one row per local branch, judged by the store's
// merged judgement (store.Merged, the one the worktree rows and the shell
// guard's delete proof read), and under apply each merged branch nothing
// keeps deleted with `git branch -d`, never -D, so git re-checks the merge at
// the moment it deletes and a refusal of git's is kept as the reason.
//
// Kept, with the reason: the default branch; a branch a worktree has checked
// out, or a stopped rebase or bisect holds, and abcd's reservation and status
// branches (branchHazard, the guard's own checks); a branch with commits not
// on its upstream; and a branch the judgement cannot show merged. A merged
// branch `git branch -d` would refuse (it checks the branch's upstream, or
// HEAD when the upstream is unset or gone, not the default branch) is kept as
// well, so the dry run never promises a delete git will refuse.
//
// Every git command this file runs goes through gitutil. Its reads use the
// isolated runner, bound to the run's context and gitReadTimeout; apply's
// `git branch -d` uses the runner for a local write (LocalGit), bound to
// gitutil.NetTimeout; the shared merged judgement (store.Merged) runs its own
// git reads with no deadline. The judgement reads refs and history and writes
// nothing; only apply's `git branch -d` changes a ref.
//
// sweepBranches is an entry point the surfaces (step 5) call, and so stays
// unexported until then, as runDry does.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	store "github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/gitutil"
)

const (
	// gitReadTimeout bounds each git read a branch judgement or a last check
	// runs: a git that hangs is killed, and the branch kept, rather than
	// holding the whole run.
	gitReadTimeout = 2 * time.Minute
	// refListCap bounds a listing of every branch under a prefix.
	refListCap = 16 << 20
	// gitMessageCap bounds the output of one git command whose answer is a
	// line or two.
	gitMessageCap = 64 << 10
)

// beforeBranchDelete, when a test sets it, runs between a local branch's
// judgement and its `git branch -d`: the window in which the repository can
// change under the run.
var beforeBranchDelete func(name string)

// gitRead runs one isolated git command in dir, bound to ctx and
// gitReadTimeout, its whole output refused past max.
func gitRead(ctx context.Context, dir string, max int, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, gitReadTimeout)
	defer cancel()
	return gitutil.RunCappedContext(ctx, dir, max, args...)
}

// isAncestor reports whether commit a is b or an ancestor of it, read in dir
// under ctx and gitReadTimeout. An error is no answer, which keeps the branch.
func isAncestor(ctx context.Context, dir, a, b string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, gitReadTimeout)
	defer cancel()
	_, notAncestor, err := gitutil.RunCappedExitOneContext(ctx, dir, gitMessageCap, "merge-base", "--is-ancestor", a, b)
	if err != nil {
		return false, err
	}
	return !notAncestor, nil
}

// branchRef is one ref as for-each-ref lists it: its full name, the commit it
// names, its upstream's full ref and git's word on the two ("ahead 2",
// "behind 1", "gone"), and the ref it points to when it is a symbolic one.
type branchRef struct {
	ref, tip, upstream, track, symref string
}

// listRefs lists every ref under prefix in dir, bound to ctx.
func listRefs(ctx context.Context, dir, prefix string) ([]branchRef, error) {
	out, err := gitRead(ctx, dir, refListCap, "for-each-ref",
		"--format=%(refname)%00%(objectname)%00%(upstream)%00%(upstream:track,nobracket)%00%(symref)", prefix)
	if err != nil {
		return nil, err
	}
	var refs []branchRef
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		f := strings.Split(line, "\x00")
		if len(f) != 5 {
			return nil, errors.New("git listed a ref in a shape the sweep does not read")
		}
		refs = append(refs, branchRef{ref: f[0], tip: f[1], upstream: f[2], track: f[3], symref: f[4]})
	}
	return refs, nil
}

// refTip is the commit the exact ref names, "" when no ref of that name
// exists. for-each-ref's pattern also matches the refs below ref, so only the
// exact name counts.
func refTip(ctx context.Context, dir, ref string) (string, error) {
	out, err := gitRead(ctx, dir, refListCap, "for-each-ref", "--format=%(refname)%00%(objectname)", ref)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(out, "\n") {
		if name, sha, ok := strings.Cut(line, "\x00"); ok && name == ref {
			return sha, nil
		}
	}
	return "", nil
}

// branchJudge is what every local branch of one run is judged against.
type branchJudge struct {
	ctx  context.Context
	root string
	// def is the default branch's full ref, defName its short name.
	def, defName string
	wts          []gitutil.Worktree
	held         gitutil.Held
}

// newBranchJudge reads what the local branches are judged against: the
// default branch, the worktrees and what a stopped rebase or bisect holds.
func newBranchJudge(ctx context.Context, root string) (*branchJudge, error) {
	ctx2, cancel := context.WithTimeout(ctx, gitReadTimeout)
	defer cancel()
	wts, err := gitutil.ListWorktreesContext(ctx2, root, gitutil.MaxWorktreeListing)
	if err != nil {
		return nil, fmt.Errorf("listing the repository's worktrees: %w", err)
	}
	common, err := gitutil.CommonDirContext(ctx2, root)
	if err != nil {
		return nil, fmt.Errorf("reading the checkout's common directory: %w", err)
	}
	def := gitutil.DefaultRefContext(ctx2, root)
	if err := ctx2.Err(); err != nil {
		return nil, fmt.Errorf("reading the default branch: %w", err)
	}
	return &branchJudge{ctx: ctx, root: root, def: def, defName: shortName(def), wts: wts, held: gitutil.HeldBranches(common)}, nil
}

// shortName is a branch's name as a person writes it: a full local or
// remote-tracking ref with its prefix taken off.
func shortName(ref string) string {
	if b, ok := strings.CutPrefix(ref, "refs/heads/"); ok {
		return b
	}
	if r, ok := strings.CutPrefix(ref, "refs/remotes/"); ok {
		if _, b, ok := strings.Cut(r, "/"); ok {
			return b
		}
	}
	return ref
}

// sweepBranches judges every local branch of the repository at repoRoot and,
// with apply, deletes each one judged deletable with `git branch -d`; each
// row carries its Outcome then. A failure to read what the branches are
// judged against refuses the run. A delete git refuses keeps that branch,
// naming git's refusal, and is the run's fault, carried as its Refusal and
// returned as the error, so ExitCode reads ExitRefused; the other rows are
// still acted on.
func sweepBranches(ctx context.Context, repoRoot string, apply bool) (Result, error) {
	bj, err := newBranchJudge(ctx, repoRoot)
	if err != nil {
		return refused(err)
	}
	refs, err := listRefs(ctx, repoRoot, "refs/heads/")
	if err != nil {
		return refused(fmt.Errorf("listing the local branches: %w", err))
	}
	res := Result{Rows: make([]Row, 0, len(refs))}
	for _, b := range refs {
		// A run whose context ended stops judging: a row judged after it
		// could rest on reads git was killed for.
		if err := ctx.Err(); err != nil {
			return refused(fmt.Errorf("the run was stopped while judging the local branches: %w", err))
		}
		res.Rows = append(res.Rows, bj.row(b))
	}
	if !apply {
		return res, nil
	}
	res.Applied = true
	var fault error
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
		if err := bj.delete(refs[i], row); err != nil && fault == nil {
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

// row judges one local branch, in the spec's order: the default branch and
// the guard's checks first, then commits not on its upstream, then the merged
// judgement, then whether `git branch -d` would take it.
func (bj *branchJudge) row(b branchRef) Row {
	name := strings.TrimPrefix(b.ref, "refs/heads/")
	row := Row{Kind: RowBranch, Name: clean(name), State: clean(branchState(b)), Action: ActionKeep}
	keep := func(reason string, ev ...Evidence) Row {
		row.Reason = clean(reason)
		for _, e := range ev {
			e.Detail = clean(e.Detail)
			row.Evidence = append(row.Evidence, e)
		}
		return row
	}
	switch {
	case bj.def == "":
		return keep("there is no default branch to judge it merged against", Evidence{Kind: KindMerged, Strength: Hint, Detail: "no default branch"})
	case name == bj.defName:
		return keep("the default branch", Evidence{Kind: KindProtected, Strength: Proven, Detail: "the default branch"})
	}
	if pr, bad := branchHazard(bj.def, bj.wts, bj.held, name); bad {
		return keep(pr.Failed, Evidence{Kind: KindDeleteProof, Strength: Proven, Detail: pr.Failed})
	}
	if !gitutil.IsFullSHA(b.tip) {
		return keep("cannot classify: git named no commit for it", Evidence{Kind: KindMerged, Strength: Hint, Detail: "no commit"})
	}
	if ahead, ok := aheadOf(b.track); ok {
		why := fmt.Sprintf("%s commit(s) not on its upstream %s", ahead, displayRef(b.upstream))
		return keep(why, Evidence{Kind: KindUpstream, Strength: Proven, Detail: why})
	}
	merged, by := store.Merged(bj.root, bj.def, b.ref, b.tip, nil)
	if !merged {
		return keep("unmerged: the sweep cannot show it is merged into "+displayRef(bj.def),
			Evidence{Kind: KindMerged, Strength: Proven, Detail: "unmerged"})
	}
	mergedEv := Evidence{Kind: KindMerged, Strength: Proven, Detail: "merged, judged by " + by}
	// git branch -d checks the branch against its upstream when that still
	// resolves, and against HEAD otherwise; a branch merged by content alone
	// is not git's ancestor there, and -d refuses it.
	against := "HEAD"
	if b.upstream != "" && b.track != "gone" {
		against = b.upstream
	}
	ok, err := isAncestor(bj.ctx, bj.root, b.tip, against)
	switch {
	case err != nil:
		return keep("merged, judged by "+by+", but whether `git branch -d` would take it could not be read: "+firstLine(err.Error()), mergedEv)
	case !ok:
		return keep(fmt.Sprintf("merged, judged by %s, but not into %s, which `git branch -d` checks, so it would refuse; the sweep never forces a delete",
			by, displayRef(against)), mergedEv)
	}
	row.Action = ActionDelete
	return keep(fmt.Sprintf("merged, judged by %s, with no worktree on it; `git branch -d` re-checks it against %s as it deletes",
		by, displayRef(against)), mergedEv)
}

// aheadOf is how many commits git's track word says the branch holds that its
// upstream does not ("ahead 2" or "ahead 2, behind 1").
func aheadOf(track string) (string, bool) {
	for _, part := range strings.Split(track, ",") {
		if n, ok := strings.CutPrefix(strings.TrimSpace(part), "ahead "); ok {
			return n, true
		}
	}
	return "", false
}

// branchState is what the sweep read of a local branch, in words.
func branchState(b branchRef) string {
	parts := []string{"at " + short(b.tip)}
	switch {
	case b.upstream == "":
		parts = append(parts, "no upstream")
	case b.track == "gone":
		parts = append(parts, "its upstream "+displayRef(b.upstream)+" gone")
	case b.track != "":
		parts = append(parts, "upstream "+displayRef(b.upstream)+" ("+b.track+")")
	default:
		parts = append(parts, "upstream "+displayRef(b.upstream))
	}
	return strings.Join(parts, ", ")
}

// delete deletes the local branch b with `git branch -d`, marking row with
// the outcome. A branch that moved since it was judged is kept, judged again
// by the next run. A refusal of git's keeps the branch, naming git's own
// words, and is returned as the run's fault.
func (bj *branchJudge) delete(b branchRef, row *Row) error {
	name := strings.TrimPrefix(b.ref, "refs/heads/")
	if beforeBranchDelete != nil {
		beforeBranchDelete(name)
	}
	now, err := refTip(bj.ctx, bj.root, b.ref)
	switch {
	case err != nil:
		row.Reason = clean("its tip could not be read again before the delete, so it is kept: " + firstLine(err.Error()))
		return nil
	case now != b.tip:
		row.Reason = clean("it moved since it was judged, so it is kept for the next run to judge")
		return nil
	}
	if err := bj.ctx.Err(); err != nil {
		row.Reason = clean("the run was stopped before the delete, so it is kept")
		return err
	}
	// The delete runs the way the person's own `git branch -d` does, through
	// gitutil's runner for a local write (LocalGit): under the person's own
	// configuration, with the repository's hooks (a reference-transaction
	// hook) running. That runner takes no context; it is bounded by
	// gitutil.NetTimeout instead, and the run's context is checked just
	// before it starts.
	if _, err := gitutil.LocalGit(bj.root, "branch", "-d", "--", name); err != nil {
		row.Reason = clean("`git branch -d` refused to delete it, so it is kept: " + err.Error())
		return errors.New(clean("sweep: `git branch -d` refused to delete " + name + ": " + err.Error()))
	}
	row.Outcome = OutcomeDeleted
	row.Reason = clean("deleted with `git branch -d`; " + row.Reason)
	return nil
}
