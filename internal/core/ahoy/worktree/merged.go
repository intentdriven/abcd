package worktree

// merged.go is the merged judgement (spc-2609301811532881, decision 2): a
// worktree's branch is merged into the default branch when the forge records
// it, when its tip is the default branch's ancestor and holds commits of its
// own, or when its content is on the default branch commit by commit (a
// rebase merge) or as one change (a squash merge). Each test errs towards
// unmerged, because an unmerged worktree is kept and a merged one may be
// reclaimed. A worktree the store added has its own commits counted from the
// start Add recorded (start.go); any other is judged from its reflog alone.

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// GH runs the forge client with args from dir and returns its output: the
// front door hands the repository's own bounded runner. A failure of any kind
// (no client, no login, no GitHub remote, no answer in time) is no forge.
type GH func(dir string, args ...string) (string, error)

// The ways a merged branch was judged, as the row's judged_by column names them.
const (
	byForge    = "forge"
	byAncestry = "ancestry"
	byContent  = "content"
)

// The bounds of the judgement's reads. A read past its bound is no answer,
// and no answer is unmerged.
const (
	// prListLimit is how many pull requests the one forge listing asks for,
	// newest first.
	prListLimit = 500
	// reflogCap bounds a branch's reflog, an object name and a subject a
	// line.
	reflogCap = 1 << 20
	// cherryCap bounds `git cherry`, one object name a line.
	cherryCap = 1 << 20
	// diffCap bounds a branch's whole diff and its patch id's input.
	diffCap = 16 << 20
	// logCap bounds the default branch's patches since the merge base.
	logCap = 32 << 20
)

// judgement is one branch's merged judgement and how it was reached.
type judgement struct {
	merged bool
	by     string
}

// pullRequest is the one forge listing's record of a pull request, in gh's
// own field names.
type pullRequest struct {
	HeadRefName       string `json:"headRefName"`
	HeadRefOid        string `json:"headRefOid"`
	BaseRefName       string `json:"baseRefName"`
	IsCrossRepository bool   `json:"isCrossRepository"`
	State             string `json:"state"`
}

// askForge asks the forge, once, for the repository's pull requests, by head
// branch. A forge that does not answer is nil and a note saying so.
func askForge(forge GH, dir string) (map[string][]pullRequest, string) {
	if forge == nil {
		return nil, "no forge answered (none was asked); merged is judged from git alone"
	}
	out, err := forge(dir, "pr", "list", "--state=all", fmt.Sprintf("--limit=%d", prListLimit),
		"--json=headRefName,headRefOid,baseRefName,isCrossRepository,state")
	if err != nil {
		return nil, "no forge answered (" + termsafe.Sanitize(fsutil.RedactHome(firstLine(err.Error()))) + "); merged is judged from git alone"
	}
	var prs []pullRequest
	if json.Unmarshal([]byte(out), &prs) != nil {
		return nil, "no forge answered (its listing of pull requests does not parse); merged is judged from git alone"
	}
	by := map[string][]pullRequest{}
	for _, pr := range prs {
		by[pr.HeadRefName] = append(by[pr.HeadRefName], pr)
	}
	return by, ""
}

// openFrom reports whether the forge listed an open pull request from wt's
// branch. A pull request from a fork's branch of the same name counts too:
// this errs towards an active worktree, which is kept.
func openFrom(prs map[string][]pullRequest, wt gitutil.Worktree) bool {
	branch, ok := strings.CutPrefix(wt.Branch, "refs/heads/")
	if !ok {
		return false
	}
	for _, pr := range prs[branch] {
		if pr.State == "OPEN" {
			return true
		}
	}
	return false
}

// judge is the merged judgement of wt's branch against the default branch
// def, read from dir, a checkout of the repository. start is the commit the
// store recorded the branch starting at when it added the worktree, or "" for
// none. A detached worktree, a repository with no default branch and a tip git
// cannot name are unmerged.
func judge(dir, def string, wt gitutil.Worktree, start string, prs map[string][]pullRequest) judgement {
	branch, ok := strings.CutPrefix(wt.Branch, "refs/heads/")
	if !ok || def == "" {
		return judgement{}
	}
	tip, err := gitutil.Run(dir, "rev-parse", "--verify", "--quiet", wt.Branch+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(tip) {
		return judgement{}
	}
	return judgeTip(dir, def, wt.Branch, branch, tip, start, prs[branch])
}

// Merged is the merged judgement for a caller outside the store: whether the
// commit tip of the branch ref (refs/heads/<b> or refs/remotes/<r>/<b>) is
// merged into the default branch def, read from dir, and how. prs is the
// forge's listing of pull requests from that branch as JSON in gh's field
// names (headRefName, headRefOid, baseRefName, isCrossRepository, state), or
// nil when the forge was not asked; a listing that does not parse counts as
// none. It is the one judgement the worktree store's reclaim and the shell
// guard's delete proof share (iss-2610241239492159), erring towards unmerged
// the same way for both.
func Merged(dir, def, ref, tip string, prs []byte) (merged bool, by string) {
	if def == "" || !gitutil.IsFullSHA(tip) {
		return false, ""
	}
	var listed []pullRequest
	if prs != nil && json.Unmarshal(prs, &listed) != nil {
		listed = nil
	}
	branch := branchName(ref)
	var own []pullRequest
	for _, pr := range listed {
		if pr.HeadRefName == branch {
			own = append(own, pr)
		}
	}
	j := judgeTip(dir, def, ref, branch, tip, "", own)
	return j.merged, j.by
}

// judgeTip is judge past resolving the tip: ref is the branch's full ref,
// branch its name, and prs the forge's pull requests from it.
func judgeTip(dir, def, ref, branch, tip, start string, prs []pullRequest) judgement {
	// The forge's answer counts only at the local tip: a branch that gained
	// commits after its pull request merged has work the merge does not hold.
	for _, pr := range prs {
		if pr.State == "MERGED" && !pr.IsCrossRepository && pr.HeadRefOid == tip && pr.BaseRefName == branchName(def) {
			return judgement{true, byForge}
		}
	}
	// The default branch, checked out in a worktree of its own, holds every
	// commit of the default branch and none merged into it.
	if ref == def || branch == branchName(def) {
		return judgement{}
	}
	if ok, err := gitutil.IsAncestor(dir, tip, def); err == nil && ok {
		// A branch cut and never committed to is an ancestor of its base too,
		// and so is one fast-forwarded along it since; it is merged only when
		// it holds a commit of its own.
		if ownCommit(dir, ref, tip, start) {
			return judgement{true, byAncestry}
		}
	}
	if rebased(dir, def, tip) || squashed(dir, def, tip) {
		return judgement{true, byContent}
	}
	return judgement{}
}

// branchName is the default branch's name on the forge: ref with its local or
// remote-tracking prefix taken off.
func branchName(ref string) string {
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

// ownCommit reports whether ref's reflog proves the branch made a commit of
// its own that its tip still holds: an entry written by a commit, a
// cherry-pick or a merge that made a merge commit, newer than the entry that
// records where the branch began. A fast-forward, a reset and the branch's
// creation make no commit. A branch with no reflog to prove a commit is judged
// by the content tests alone.
//
// Where the reflog begins is not enough on its own: a reflog whose oldest
// entry is not that beginning has lost part of the branch's history to expiry,
// and a copied branch's can keep the source's beginning and commits once the
// copy's own entries expire (iss-2610231011465325). So with start, the commit
// the store recorded the branch starting at, a commit counts only when it is
// not already in start, which no other branch's history can satisfy; without
// it, the reflog must still reach the branch's beginning.
func ownCommit(dir, ref, tip, start string) bool {
	out, err := gitutil.RunCappedAsStored(dir, reflogCap, "log", "-g", "--format=%H%x00%gs", ref, "--")
	if err != nil {
		return false
	}
	// Newest first. Every command that creates a branch (git branch,
	// checkout -b, switch -c, worktree add -b) writes "branch: Created from
	// <start>" as its first entry. `git branch -c` copies the source branch's
	// reflog with it, so every entry older than the copy is the other branch's
	// history, and the copy is where this branch began. `git branch -m`
	// carries the branch's own reflog forward under its new name, so the scan
	// goes on past a rename.
	var made []string
	began := false
	for _, line := range strings.Split(out, "\n") {
		sha, subject, ok := strings.Cut(line, "\x00")
		if !ok || !gitutil.IsFullSHA(sha) {
			continue
		}
		if strings.HasPrefix(subject, "Branch: copied ") || strings.HasPrefix(subject, "branch: Created from ") {
			began = true
			break
		}
		if madeACommit(subject) {
			made = append(made, sha)
		}
	}
	if !began && start == "" {
		return false
	}
	for _, sha := range made {
		if start != "" {
			if before, err := gitutil.IsAncestor(dir, sha, start); err != nil || before {
				continue
			}
		}
		if held, err := gitutil.IsAncestor(dir, sha, tip); err == nil && held {
			return true
		}
	}
	return false
}

// madeACommit reports whether a reflog subject is one git writes when the
// branch gains a commit it made: "commit: …", "commit (amend): …",
// "commit (merge): …", "cherry-pick: …", or a merge that was no fast-forward.
func madeACommit(subject string) bool {
	switch {
	case strings.HasPrefix(subject, "commit:"), strings.HasPrefix(subject, "commit ("),
		strings.HasPrefix(subject, "cherry-pick:"):
		return true
	case strings.HasPrefix(subject, "merge "):
		return strings.Contains(subject, ": Merge made by")
	}
	return false
}

// rebased reports whether every commit tip holds that def does not has an
// equivalent on def, and there is at least one: `git cherry` marks each.
func rebased(dir, def, tip string) bool {
	out, err := gitutil.RunCappedAsStored(dir, cherryCap, "cherry", def, tip)
	if err != nil || out == "" {
		return false
	}
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "- ") {
			return false
		}
	}
	return true
}

// squashed reports whether the branch's whole change since its merge base
// with def is one commit on def since that base, by stable patch id.
func squashed(dir, def, tip string) bool {
	base, err := gitutil.Run(dir, "merge-base", def, tip)
	if err != nil || !gitutil.IsFullSHA(base) {
		return false
	}
	diff, err := asStored(dir, "", diffCap, "diff", "--no-ext-diff", "--no-textconv", "--no-color", base, tip, "--")
	if err != nil || len(strings.TrimSpace(diff)) == 0 {
		return false
	}
	want := patchIDs(dir, diff)
	if len(want) != 1 {
		return false
	}
	log, err := asStored(dir, "", logCap, "log", "-p", "--no-ext-diff", "--no-textconv", "--no-color", "--no-decorate",
		"--format=commit %H", base+".."+def, "--")
	if err != nil || log == "" {
		return false
	}
	for _, id := range patchIDs(dir, log) {
		if id == want[0] {
			return true
		}
	}
	return false
}

// patchIDs is `git patch-id --stable` over patch: one id per change in it.
func patchIDs(dir, patch string) []string {
	out, err := asStored(dir, patch, diffCap+logCap, "patch-id", "--stable")
	if err != nil {
		return nil
	}
	var ids []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if id, _, ok := strings.Cut(line, " "); ok && id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// asStored is one bounded git read of history as stored, its output verbatim.
func asStored(dir, stdin string, maxBytes int, args ...string) (string, error) {
	out, err := gitutil.RunCappedBytesAsStoredContext(context.Background(), dir, stdin, maxBytes, args...)
	return string(out), err
}
