package loop

// sync.go is how a lane lands after a sibling lane of the same run landed
// first (spc-2609202134341288, "Two lanes that touch the same files"). Before a
// lane's landing begins, the loop checks whether a sibling has landed since
// this lane's base; if one has, it syncs the lane: it merges the default
// branch into the lane's branch with a merge commit in the lane's worktree. It
// never rebases, so no commit a validator judged is rewritten and every sha the
// record cites stays reachable. A clean merge moves the lane's head, so a fresh
// round judges the new head. A merge that conflicts is aborted, leaving the
// branch where it was, and the conflict goes to a fresh implementer with a sync
// brief naming each conflicting path and the sibling lanes whose landing
// brought the other side; its receipt must carry the merged sha as an ancestor
// of the new head, and a fresh round judges it. A sync does not count against
// the run's fix rounds.

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// SyncDirName is the lane's directory a sync's brief and receipt live in.
const SyncDirName = "sync"

// syncSubject is a sync's merge commit subject.
func syncSubject(lane Lane, def string, siblings []string) string {
	return "merge: sync " + lane.Branch + " with " + def + " after " + strings.Join(siblings, ", ") + " landed"
}

// syncMessage is the sync's whole merge message. Its text is computed by abcd
// from the run's state, so its trailer is the abcd label (ruling PC1).
func syncMessage(lane Lane, runID, def, merged string, siblings []string) string {
	return syncSubject(lane, def, siblings) + "\n\n" +
		fmt.Sprintf("The implement loop merges %s at %s into %s of %s, because %s landed since the lane's base; it never rebases, so every judged commit stays reachable.\n\n",
			def, shortSHA(merged), lane.ID, runID, strings.Join(siblings, ", ")) +
		composedAssistedBy() + "\n"
}

// landedSiblings names the run's lanes that landed with a head this lane does
// not hold.
func landedSiblings(c Context, lane Lane) ([]string, error) {
	var out []string
	for _, l := range c.State.Lanes {
		if l.ID == lane.ID || l.Stage != StageDone || l.Landing == nil || !gitutil.IsFullSHA(l.Landing.Pushed) {
			continue
		}
		on, err := gitutil.IsAncestor(c.RepoRoot, l.Landing.Pushed, lane.HeadSHA)
		if err != nil {
			return nil, fmt.Errorf("placing %s's landed head on %s: %v", l.ID, lane.ID, err)
		}
		if !on {
			out = append(out, l.ID)
		}
	}
	return out, nil
}

// syncLane merges the default branch into the lane when a sibling landed since
// its base, and reports whether it did anything. A clean merge sends the lane
// to a fresh round over the merge head; a conflicting one is aborted and sends
// the lane to a fresh implementer with a sync brief.
func syncLane(c Context, lane *Lane) (Outcome, bool, error) {
	siblings, err := landedSiblings(c, *lane)
	if err != nil || len(siblings) == 0 {
		return Outcome{}, false, err
	}
	def, err := defaultBranch(c, *lane)
	if err != nil {
		return Outcome{}, false, err
	}
	merged, err := gitutil.Run(c.RepoRoot, "rev-parse", "--verify", "--quiet", "refs/remotes/"+Remote+"/"+def+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(merged) {
		return Outcome{}, false, fmt.Errorf("resolving %s/%s to sync %s: %v", Remote, def, lane.ID, err)
	}
	tip, err := branchTip(c, *lane)
	if err != nil {
		return Outcome{}, false, err
	}
	s := Sync{At: c.Now, Siblings: siblings, Merged: merged}
	if tip != lane.HeadSHA {
		// A call killed after its merge commit and before its state write:
		// found, when the tip is exactly that merge.
		parents, _ := gitutil.Run(c.RepoRoot, "rev-list", "--parents", "-n", "1", tip, "--")
		if f := strings.Fields(parents); len(f) != 3 || f[1] != lane.HeadSHA || f[2] != merged {
			return Outcome{}, false, refuse(string(StageLand), "", lane.ID,
				fmt.Sprintf("%s moved to %s, which is not a sync of the judged head %s", lane.Branch, shortSHA(tip), shortSHA(lane.HeadSHA)),
				"restore the branch to the judged head, then run `abcd implement step` again")
		}
	} else {
		msg := syncMessage(*lane, c.State.RunID, def, merged, siblings)
		if err := syncMerge(lane.Worktree, msg, merged); err != nil {
			conflicted, _ := pickGit(lane.Worktree, "diff", "--name-only", "--diff-filter=U", "-z")
			if _, aerr := pickGit(lane.Worktree, "merge", "--abort"); aerr != nil {
				return Outcome{}, false, fmt.Errorf("aborting the sync's merge in the lane's worktree: %w", aerr)
			}
			paths := strings.FieldsFunc(conflicted, func(r rune) bool { return r == 0 })
			if len(paths) == 0 {
				return Outcome{}, false, refuse(string(StageLand), "", lane.ID, "git could not merge "+def+" into "+lane.Branch+": "+fsutil.RedactHome(err.Error()),
					"settle what git reports in the lane's worktree, then run `abcd implement step` again")
			}
			s.Conflicted, s.Paths = true, paths
			if err := writeSyncBrief(c, *lane, len(lane.Syncs)+1, def, &s); err != nil {
				return Outcome{}, false, err
			}
			lane.Syncs = append(append([]Sync{}, lane.Syncs...), s)
			return Outcome{Goto: StageValidate, Note: fmt.Sprintf("the sync of %s with %s at %s conflicted in %s (%s landed first); the merge is aborted with the branch unchanged, and a fresh implementer resolves it from %s",
				lane.ID, def, shortSHA(merged), strings.Join(paths, ", "), strings.Join(siblings, ", "), s.Brief)}, true, nil
		}
		if tip, err = branchTip(c, *lane); err != nil {
			return Outcome{}, false, err
		}
	}
	s.Head = tip
	lane.HeadSHA = tip
	lane.Syncs = append(append([]Sync{}, lane.Syncs...), s)
	return Outcome{Goto: StageValidate, Note: fmt.Sprintf("synced %s with %s at %s after %s landed: merge commit %s; a fresh round judges it, and counts no fix round",
		lane.ID, def, shortSHA(merged), strings.Join(siblings, ", "), shortSHA(tip))}, true, nil
}

// syncMerge merges merged into the lane's worktree with a merge commit, with
// git's built-in merge on every path: each merge driver the repository
// configures, whether an attribute or merge.default selects it, is replaced
// by the built-in text merge (gitutil.MergeDriverOverrides), so the merge
// starts no program the repository names and its result is git's own
// (iss-2610090821510097). A driver name the override cannot carry refuses the
// merge before it starts.
func syncMerge(dir, msg, merged string) error {
	drivers, err := gitutil.MergeDriverOverrides(dir)
	if err != nil {
		return fmt.Errorf("switching the lane's merge drivers to git's built-in merge: %w", err)
	}
	_, err = pickGit(dir, append(drivers, "merge", "--no-ff", "--no-edit", "-m", msg, merged)...)
	return err
}

// writeSyncBrief renders the brief of the fresh implementer a conflicting sync
// goes to, and names it and its receipt on s.
func writeSyncBrief(c Context, lane Lane, n int, def string, s *Sync) error {
	dir, err := laneFile(c.State.RunID, lane.ID, StageValidate, fmt.Sprintf("%s/%d", SyncDirName, n))
	if err != nil {
		return err
	}
	laneDir, err := laneRel(c.State.RunID, lane.ID, StageValidate)
	if err != nil {
		return err
	}
	inLane := strings.TrimPrefix(dir, laneDir+"/")
	s.Brief, s.Receipt = dir+"/"+BriefFileName, dir+"/"+ReceiptFileName
	var b bytes.Buffer
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	p("# Sync brief: %s of %s\n\n", lane.ID, c.State.RunID)
	p("You are a fresh implementer. %s landed on %s since this lane's base, and merging %s at %s into\n", strings.Join(s.Siblings, ", "), def, def, s.Merged)
	p("`%s` conflicted. The loop aborted that merge, so the branch is where its validators left it.\n\n", lane.Branch)
	p("The conflicting paths:\n\n")
	for _, path := range s.Paths {
		p("- `%s`\n", path)
	}
	p("\nIn the worktree `%s`, merge %s into `%s` with a merge commit (`git merge %s`; never a\n", lane.Worktree, s.Merged, lane.Branch, s.Merged)
	p("rebase), resolve each conflict keeping both lanes' intent, and commit the merge. The merged sha must be an\n")
	p("ancestor of your head; a fresh round of validators judges it.\n\n")
	p("Your brief as the lane's implementer: `%s`\n\n", abs(c.RepoRoot, lane.Brief))
	p("## What you hand back\n\n")
	p("- your report, at `%s` (in the receipt: `%s/%s`)\n", abs(c.RepoRoot, dir+"/"+ReportFileName), inLane, ReportFileName)
	p("- the definition of done's whole output, at `%s` (in the receipt: `%s/%s`)\n", abs(c.RepoRoot, dir+"/"+DoDFileName), inLane, DoDFileName)
	p("- the receipt, at `%s`: the lane receipt's fields — `schema_version` %d, `run_id` %q, `lane` %q,\n", abs(c.RepoRoot, s.Receipt), ReceiptSchemaVersion, c.State.RunID, lane.ID)
	p("  `branch` %q, `commits` (the merge commit and any you made), `definition_of_done`, `report`, and an optional `model`.\n", lane.Branch)
	return writeRoundFile(c.RepoRoot, s.Brief, b.Bytes())
}

// verifySync verifies the receipt of the implementer a conflicting sync went
// to: as a fix receipt is, and with the merged sha an ancestor of the new head.
func verifySync(c Context, lane *Lane, receiptRel string) error {
	s := lane.pendingSync()
	if err := verifyLaneReceipt(c, lane, receiptRel, s.Receipt); err != nil {
		return err
	}
	on, err := gitutil.IsAncestor(c.RepoRoot, s.Merged, lane.HeadSHA)
	if err != nil || !on {
		return refuse("receipt", "", lane.ID,
			fmt.Sprintf("%s's head %s does not contain %s, the default branch's sha the sync merges in", lane.Branch, shortSHA(lane.HeadSHA), shortSHA(s.Merged)),
			"merge "+s.Merged+" into "+lane.Branch+" with a merge commit (never a rebase), then hand the receipt back")
	}
	syncs := append([]Sync{}, lane.Syncs...)
	syncs[len(syncs)-1].Head = lane.HeadSHA
	lane.Syncs = syncs
	return nil
}
