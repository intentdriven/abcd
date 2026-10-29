package sweep

// deleteproof.go is the shell guard's delete proof (iss-2610241239492159):
// the guard reads a branch delete or a worktree removal off a command line,
// and this proves its target safe with the judgements the sweep already
// makes: the store's merged judgement (ancestry, rebase or squash by patch
// id, or a pull request merged at the tip) and the liveness probe. It never
// re-makes either. Every check that cannot answer fails the proof, and every
// failure says what would settle it.
//
// A branch is safe to delete when it is merged into origin's default branch,
// no open pull request is on it, no worktree has it checked out, and it is
// not one of abcd's own coordination branches (a reservation, or a status
// branch). A remote branch is judged at the tip origin holds now, read with
// a bounded ls-remote, since `git push --delete` deletes whatever that tip
// is; a local one is judged offline, against origin's default branch as last
// fetched, so a local cleanup needs no network. A worktree is safe to remove
// when it holds no file git has not committed (ignored ones included: the
// local tier lives there), no live process or reservation holds it, and its
// branch is merged.

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
	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The delete proof's bounds. A read past its bound, or one that does not end
// in time, is no answer, and no answer fails the proof.
const (
	// proofNetLimit bounds the one ls-remote a remote branch's proof makes.
	proofNetLimit = 10 * time.Second
	// proofProcessLimit bounds the process table's reading.
	proofProcessLimit = 5 * time.Second
	// proofListCap bounds the worktree listing, the ls-remote and the forge's
	// listing.
	proofListCap = 1 << 20
	// proofPRLimit is how many pull requests from the branch the forge is
	// asked for.
	proofPRLimit = 100
)

// DeleteProofOptions is what the delete proof reads beyond git. Every field
// may be nil, and then the default reads the host.
type DeleteProofOptions struct {
	// Forge is the forge client the open and merged pull requests are asked
	// of; nil fails every remote branch's proof, which cannot show none open.
	Forge store.GH
	// RemoteTip reads the tip a remote holds for a branch now, and whether it
	// holds the branch at all; nil is a bounded `git ls-remote`.
	RemoteTip func(dir, remote, branch string) (tip string, found bool, err error)
	// Probe is what the liveness probe reads for a worktree removal; its
	// Reservations and Runs are read from the repository when left zero.
	Probe ProbeOptions
}

// deleteProver is the guard.DeleteProver the front doors attach.
type deleteProver struct {
	o DeleteProofOptions
}

// NewDeleteProver returns the prover the shell guard's front doors attach to
// the registry (guard.Registry.ProvingDeletes).
func NewDeleteProver(o DeleteProofOptions) guard.DeleteProver {
	if o.RemoteTip == nil {
		o.RemoteTip = lsRemoteTip
	}
	return deleteProver{o: o}
}

// unproven is a failed proof.
func unproven(failed, settle string) guard.DeleteProof {
	return guard.DeleteProof{Failed: clean(failed), Settle: clean(settle)}
}

// ProveDelete proves one target safe, or names the check that failed.
func (p deleteProver) ProveDelete(t guard.DeleteTarget) guard.DeleteProof {
	root, err := gitutil.Run(t.Dir, "rev-parse", "--show-toplevel")
	if err != nil || root == "" {
		return unproven("the directory the command runs in is not a git checkout abcd can read", "")
	}
	def := gitutil.TrackedDefaultBranch(root, claim.Remote)
	if def == "" {
		return unproven("origin's default branch has not been fetched here, so nothing can be shown merged into it",
			"Fetch origin (`git fetch origin`) and run the delete again.")
	}
	wts, err := gitutil.ListWorktrees(root, proofListCap)
	if err != nil {
		return unproven("the repository's worktrees could not be listed ("+err.Error()+")", "")
	}
	switch t.Kind {
	case guard.DeleteRemoteBranch:
		return p.remoteBranch(root, def, wts, t)
	case guard.DeleteLocalBranch:
		return p.localBranch(root, def, wts, t)
	case guard.DeleteWorktree:
		return p.worktree(root, def, wts, t)
	}
	return unproven("the delete is of a kind the proof does not know", "")
}

// branchHazard is the check every branch delete passes first: not the default
// branch, not one of abcd's coordination branches, and checked out in no
// worktree.
func branchHazard(def string, wts []gitutil.Worktree, branch string) (guard.DeleteProof, bool) {
	switch {
	case branch == strings.TrimPrefix(def, "refs/remotes/"+claim.Remote+"/"):
		return unproven("it is the default branch", "Leave the default branch in place."), true
	case strings.HasPrefix("refs/heads/"+branch, claim.RefPrefix):
		return unproven("it is a reservation branch, which says a record is being worked on",
			"Clear spent reservations with `abcd reserve --sweep --apply`, which deletes only the spent ones."), true
	case strings.HasPrefix("refs/heads/"+branch, claim.StatusRefPrefix):
		return unproven("it is a status branch, which holds a person's announcements",
			"Leave status branches to abcd."), true
	}
	for _, wt := range wts {
		if wt.Branch == "refs/heads/"+branch {
			return unproven("it is checked out in the worktree "+fsutil.RedactHome(wt.Path),
				"Remove that worktree, or switch it to another branch, first."), true
		}
	}
	return guard.DeleteProof{}, false
}

// remoteBranch proves a branch delete on origin: at the tip origin holds
// now, merged, with no open pull request.
func (p deleteProver) remoteBranch(root, def string, wts []gitutil.Worktree, t guard.DeleteTarget) guard.DeleteProof {
	if t.Remote != claim.Remote {
		return unproven("the delete is on the remote "+t.Remote+", and only a delete on "+claim.Remote+" can be proven",
			"Delete it on "+claim.Remote+", or keep it.")
	}
	if pr, bad := branchHazard(def, wts, t.Branch); bad {
		return pr
	}
	tip, found, err := p.o.RemoteTip(root, t.Remote, t.Branch)
	if err != nil {
		return unproven("origin's tip of it could not be read ("+err.Error()+")",
			"Run the delete again once origin answers here.")
	}
	if !found {
		// Origin holds no such branch: the delete removes nothing.
		return guard.DeleteProof{Safe: true}
	}
	if _, err := gitutil.Run(root, "cat-file", "-e", tip+"^{commit}"); err != nil {
		return unproven("origin's tip of it, "+short(tip)+", is a commit this checkout has not fetched",
			"Fetch it (`git fetch origin "+t.Branch+"`) and run the delete again.")
	}
	if p.o.Forge == nil {
		return unproven("no forge was asked whether a pull request is open on it",
			"Run the delete where gh is signed in to the repository's forge.")
	}
	prs, err := p.o.Forge(root, "pr", "list", "--head", t.Branch, "--state", "all",
		"--limit", strconv.Itoa(proofPRLimit),
		"--json", "number,headRefName,headRefOid,baseRefName,isCrossRepository,state")
	if err != nil {
		return unproven("the forge could not be asked whether a pull request is open on it ("+firstLine(err.Error())+")",
			"Run the delete again once gh answers here.")
	}
	open, err := openPullRequests(prs, t.Branch)
	if err != nil {
		return unproven("the forge's listing of its pull requests does not parse", "")
	}
	if len(open) > 0 {
		return unproven("pull request #"+strings.Join(open, ", #")+" is open on it",
			"Merge or close the pull request first.")
	}
	if ok, _ := store.Merged(root, def, "refs/remotes/"+claim.Remote+"/"+t.Branch, tip, []byte(prs)); !ok {
		return unproven("its tip "+short(tip)+" holds work "+displayRef(def)+" does not "+
			"(it is merged neither by ancestry, nor by a rebase or squash, nor by a pull request merged at that tip)",
			"Merge it first, or keep it.")
	}
	return guard.DeleteProof{Safe: true}
}

// localBranch proves a forced local branch delete, offline: merged into
// origin's default branch as last fetched.
func (p deleteProver) localBranch(root, def string, wts []gitutil.Worktree, t guard.DeleteTarget) guard.DeleteProof {
	if pr, bad := branchHazard(def, wts, t.Branch); bad {
		return pr
	}
	ref := "refs/heads/" + t.Branch
	tip, err := gitutil.Run(root, "rev-parse", "--verify", "--quiet", ref+"^{commit}", "--")
	if err != nil || tip == "" {
		// No such branch: the delete removes nothing.
		return guard.DeleteProof{Safe: true}
	}
	if ok, _ := store.Merged(root, def, ref, tip, nil); !ok {
		return unproven("its tip "+short(tip)+" holds work "+displayRef(def)+" does not, as last fetched "+
			"(it is merged neither by ancestry nor by a rebase or squash)",
			"Merge it first, or fetch origin if it was merged since, or keep it.")
	}
	return guard.DeleteProof{Safe: true}
}

// worktree proves a worktree removal: a worktree git lists, not the main
// one, holding no uncommitted, untracked or ignored file, held by no live
// process, run or reservation, and on a merged branch.
func (p deleteProver) worktree(root, def string, wts []gitutil.Worktree, t guard.DeleteTarget) guard.DeleteProof {
	want := fsutil.RealExistingPath(t.Path)
	var wt *gitutil.Worktree
	for i := range wts {
		if fsutil.RealExistingPath(wts[i].Path) == want {
			wt = &wts[i]
			break
		}
	}
	switch {
	case wt == nil:
		return unproven("it is not a worktree git lists for this repository", "")
	case wt == &wts[0]:
		return unproven("it is the repository's main working tree", "Leave the main working tree in place.")
	case wt.Locked:
		return unproven("git has it locked", "Leave it to whatever locked it.")
	}
	st, err := gitutil.Status(wt.Path, changesCap, gitutil.StatusOptions{Ignored: true})
	if err != nil {
		return unproven("its files could not be read ("+firstLine(err.Error())+")", "")
	}
	if len(st) > 0 {
		return unproven("it holds "+strconv.Itoa(len(st))+" file(s) git has not committed, among them "+st[0].Path+
			" (uncommitted, untracked or ignored)",
			"Commit what belongs in the repository, move what is worth keeping to the main checkout's .abcd/.work.local/, and delete the rest, first.")
	}
	if live, held := p.held(root, def, wt.Path); held {
		return unproven("it may be in use: "+live, "Let whatever holds it finish, or end it, first.")
	}
	if wt.Branch == "" {
		return unproven("its HEAD is detached, so the commits it made are on no branch that can be shown merged",
			"Put its commits on a branch and merge them, or keep it.")
	}
	if ok, _ := store.Merged(root, def, wt.Branch, wt.Head, nil); !ok {
		return unproven("its branch "+strings.TrimPrefix(wt.Branch, "refs/heads/")+" holds work "+displayRef(def)+
			" does not, as last fetched", "Merge its branch first, or keep it.")
	}
	return guard.DeleteProof{Safe: true}
}

// held asks the liveness probe whether anything holds the worktree. Every
// proof and every failed check holds it; the recent-write hint alone does
// not, since it proves no holder, and a worktree is most often removed soon
// after its last write.
func (p deleteProver) held(root, def, path string) (string, bool) {
	o := p.o.Probe
	if o.Reservations.FetchedAt.IsZero() {
		if snap, err := claim.View(root, time.Now()); err == nil {
			o.Reservations = snap
		}
	}
	if o.Runs == nil && o.RunsUnread == "" {
		runs, err := loop.Runs(root)
		if err != nil {
			o.RunsUnread = err.Error()
		}
		o.Runs = runs
	}
	ctx, cancel := context.WithTimeout(context.Background(), proofProcessLimit)
	defer cancel()
	probe := newProbe(ctx, o)
	w := Worktree{Path: path}
	if _, foreign, err := probe.owner(fsutil.RealExistingPath(path)); !foreign && err == nil {
		w.Changed = changedPaths(path, def)
	}
	l := probe.Judge(w)
	if !l.Held {
		return "", false
	}
	var why []string
	for _, e := range l.Evidence {
		if e.Kind != KindRecentWrite {
			why = append(why, e.Detail)
		}
	}
	if len(why) == 0 {
		return "", false
	}
	return strings.Join(why, "; "), true
}

// openPullRequests is the numbers of the open pull requests from branch in a
// forge listing. One from a fork's branch of the same name counts too: this
// errs towards refusing the delete.
func openPullRequests(listing, branch string) ([]string, error) {
	var prs []struct {
		Number      int    `json:"number"`
		HeadRefName string `json:"headRefName"`
		State       string `json:"state"`
	}
	if err := json.Unmarshal([]byte(listing), &prs); err != nil {
		return nil, err
	}
	var open []string
	for _, pr := range prs {
		if pr.HeadRefName == branch && pr.State == "OPEN" {
			open = append(open, strconv.Itoa(pr.Number))
		}
	}
	return open, nil
}

// lsRemoteTip is the default RemoteTip: a bounded `git ls-remote` of the one
// branch.
func lsRemoteTip(dir, remote, branch string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), proofNetLimit)
	defer cancel()
	ref := "refs/heads/" + branch
	out, err := gitutil.RunLimitedNetContext(ctx, dir, proofListCap, "ls-remote", "--", remote, ref)
	if err != nil {
		return "", false, errors.New(firstLine(err.Error()))
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		sha, name, ok := strings.Cut(line, "\t")
		if ok && name == ref {
			if !gitutil.IsFullSHA(sha) {
				return "", false, fmt.Errorf("its answer for %s is not an object name", ref)
			}
			return sha, true, nil
		}
	}
	return "", false, nil
}

// displayRef names a default-branch ref the way a person reads it:
// origin/main.
func displayRef(ref string) string {
	return strings.TrimPrefix(strings.TrimPrefix(ref, "refs/remotes/"), "refs/heads/")
}

// short is an object name cut to twelve hex digits for a refusal.
func short(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// firstLine is an error's first line.
func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}
