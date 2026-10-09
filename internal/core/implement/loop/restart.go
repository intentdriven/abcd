package loop

// restart.go restarts a lane whose implementer is gone (iss-2610080620372731,
// the product thinker's interview of 2026-10-09): an agent that died when the
// network or the model service dropped, or one that yielded mid-task on a
// network failure, handing back a `NETWORK: <cmd>` line and no receipt. The
// lane restarts as a FRESH agent from its last commit. What the gone agent
// left uncommitted is saved aside for the product thinker's review and never
// built on: the run record names where it is, and the implementer's brief does
// not.
//
// The order is what makes it safe. Everything uncommitted (staged, unstaged and
// untracked, not ignored) is captured through a scratch index, so the lane's
// own index is never touched; the patch is proved to apply to the lane's head,
// again on a scratch index and never the working tree; only then is it saved
// aside and the lane's worktree reset to its head and cleaned. A patch that
// does not prove out refuses the restart with nothing changed, because a reset
// after a save that cannot be replayed would lose the work. The reset runs in
// the lane's worktree alone, at the path the loop derives for the lane, never
// at a path the state file merely names. A restart is refused while the run's
// outage is open: a fresh agent started into the same outage dies the same way.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The aside a restart saves, under the lane's directory of the run:
// <lane-dir>/aside/<UTC stamp>/.
const (
	AsideDirName   = "aside"
	AsideFileName  = "aside.json"
	AsidePatchName = "changes.patch"
)

// The reasons a restart records.
const (
	WhyDied       = "agent died"
	whyYieldedPre = "agent yielded: "
	// NetworkLinePrefix opens the line an agent yielding on a network failure
	// hands back.
	NetworkLinePrefix = "NETWORK:"
)

// stageRestart is the refusal stage and the record entry's stage.
const stageRestart = "restart"

const (
	// asideStampLayout names an aside by the second it was saved, in UTC.
	asideStampLayout = "20060102T150405Z"
	// maxAsidePatchBytes bounds the patch: a larger one is refused rather than
	// saved cut short, since a truncated patch replays as a wrong one.
	maxAsidePatchBytes = 64 << 20
	// maxPartialReceiptBytes bounds the partial receipt copied aside.
	maxPartialReceiptBytes = 1 << 20
	// maxYieldBytes bounds the NETWORK: line a yield names.
	maxYieldBytes = 512
	// maxAsidesPerSecond bounds the asides one lane takes in one second.
	maxAsidesPerSecond = 9
)

// RestartAside is what a restart saved aside, as aside.json holds it.
type RestartAside struct {
	RunID string    `json:"run_id"`
	Lane  string    `json:"lane"`
	At    time.Time `json:"at"`
	// Path is the aside's directory, relative to the checkout root.
	Path string `json:"path"`
	// Head is the lane's last commit, the one the patch applies to and the one
	// the fresh agent starts from.
	Head string `json:"head"`
	// Files are the paths the patch changes; empty when nothing was left
	// uncommitted, and Patch is then empty too.
	Files []string `json:"files"`
	Patch string   `json:"patch,omitempty"`
	// Why is WhyDied, or "agent yielded: NETWORK: <cmd>".
	Why string `json:"why"`
	// Receipt is the partial receipt copied aside, by its name there; empty
	// when the gone agent wrote none.
	Receipt string `json:"receipt,omitempty"`
}

// RestartResult is what Restart returns: the lane's step result, re-telling
// the implementer await, and the aside.
type RestartResult struct {
	StepResult
	Aside RestartAside `json:"aside"`
}

// checkAsidePatch proves a patch applies to the worktree's HEAD without
// touching the worktree or its index. Tests replace it to stand for a patch
// git will not apply.
var checkAsidePatch = applyCheck

// applyCheck is `git apply --check --cached` of patch on a scratch index read
// from the worktree's HEAD.
func applyCheck(worktree, patch string) error {
	tmp, err := os.MkdirTemp("", "abcd-restart-check-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	index := filepath.Join(tmp, "index")
	if _, err := gitutil.RunWithIndex(worktree, index, 4096, "read-tree", "HEAD"); err != nil {
		return err
	}
	_, err = gitutil.RunWithIndex(worktree, index, 4096, "apply", "--check", "--cached", "--", patch)
	return err
}

// whyOf reads the restart's reason: WhyDied when no yield is named, and
// otherwise the agent's NETWORK: line, which must be one line naming a command.
func whyOf(yielded string) (string, error) {
	line := strings.TrimSpace(yielded)
	if line == "" {
		return WhyDied, nil
	}
	remedy := "pass the agent's own line, `" + NetworkLinePrefix + " <cmd>`, or no --yielded for an agent that died; nothing was changed"
	if len(line) > maxYieldBytes {
		return "", refuse(stageRestart, "", "", fmt.Sprintf("the yield is %d bytes, over %d", len(line), maxYieldBytes), remedy)
	}
	if strings.ContainsFunc(line, unicode.IsControl) {
		return "", refuse(stageRestart, "", "", "the yield is not one line of text", remedy)
	}
	cmd, ok := strings.CutPrefix(line, NetworkLinePrefix)
	if !ok || strings.TrimSpace(cmd) == "" {
		return "", refuse(stageRestart, "", "", "a restart after a yield takes the agent's "+NetworkLinePrefix+" line naming the command the network failed", remedy)
	}
	return whyYieldedPre + NetworkLinePrefix + " " + strings.TrimSpace(cmd), nil
}

// restartWorktree is the worktree the loop derives for the lane, refusing a
// lane whose state names any other path or branch: the reset runs only where
// the loop itself made the lane.
//
// TODO(loopWorktree): fold this into the peer's loopWorktree helper in lane.go
// once it lands, so one function answers where a lane's worktree is.
func restartWorktree(repoRoot, runID string, lane Lane) (LaneWorktree, error) {
	lw, err := laneWorktree(repoRoot, runID, lane.ID)
	if err != nil {
		return LaneWorktree{}, err
	}
	if lane.Worktree == "" || filepath.Clean(lane.Worktree) != filepath.Clean(lw.Path) || lane.Branch != lw.Branch {
		return LaneWorktree{}, refuse(stageRestart, "", lane.ID,
			fmt.Sprintf("the state names %s's worktree as %s on %s, not the loop's own %s on %s, so nothing there is saved or reset",
				lane.ID, fsutil.RedactHome(lane.Worktree), lane.Branch, fsutil.RedactHome(lw.Path), lw.Branch),
			"restore the run's state file; nothing was changed")
	}
	if !fsutil.IsRealDir(lw.Path) {
		return LaneWorktree{}, refuse(stageRestart, "", lane.ID, fmt.Sprintf("%s's worktree %s is not a real directory", lane.ID, fsutil.RedactHome(lw.Path)),
			"restore the lane's worktree, or discard the run; nothing was changed")
	}
	ref, err := gitutil.Run(lw.Path, "symbolic-ref", "--quiet", "HEAD")
	if err != nil || ref != "refs/heads/"+lw.Branch {
		return LaneWorktree{}, refuse(stageRestart, "", lane.ID, fmt.Sprintf("%s's worktree is not on its branch %s", lane.ID, lw.Branch),
			"check the lane's branch out in its worktree again; nothing was changed")
	}
	return lw, nil
}

// snapshot captures everything uncommitted in worktree, against head, through
// a scratch index in tmp: the binary patch and the paths it changes. The
// worktree's own index is never written.
func snapshot(worktree, head, tmp string) ([]byte, []string, error) {
	index := filepath.Join(tmp, "index")
	for _, args := range [][]string{{"read-tree", head}, {"add", "-A"}} {
		if _, err := gitutil.RunWithIndex(worktree, index, 4096, args...); err != nil {
			return nil, nil, err
		}
	}
	patch, err := gitutil.RunWithIndex(worktree, index, maxAsidePatchBytes,
		"diff", "--cached", "--binary", "--full-index", "--no-renames", "--no-ext-diff", "--no-textconv", "--no-color", head, "--")
	if err != nil {
		return nil, nil, err
	}
	names, err := gitutil.RunWithIndex(worktree, index, maxAsidePatchBytes,
		"diff", "--cached", "--name-only", "-z", "--no-renames", head, "--")
	if err != nil {
		return nil, nil, err
	}
	files := []string{}
	for _, n := range strings.Split(string(names), "\x00") {
		if n != "" {
			files = append(files, n)
		}
	}
	slices.Sort(files)
	return patch, files, nil
}

// Restart restarts a lane whose implementer died, or yielded on a network
// failure (yielded is then its `NETWORK: <cmd>` line): what the agent left
// uncommitted is saved aside under the lane's directory, the lane's worktree
// is reset to its last commit and cleaned, the run record names the aside, and
// the implementer await is re-told so the lead starts a fresh agent from the
// same brief. It is refused, changing nothing, while the run's outage is open,
// for a lane with no implementer out, for a lane whose worktree is not the
// one the loop derives for it, and when the saved patch would not apply to
// the lane's head.
func Restart(repoRoot, runID, laneID, yielded string, o Options) (RestartResult, error) {
	why, err := whyOf(yielded)
	if err != nil {
		return RestartResult{}, err
	}
	var res RestartResult
	err = mutate(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
		i := slices.IndexFunc(st.Lanes, func(l Lane) bool { return l.ID == laneID })
		if i < 0 {
			return false, refuse(stageRestart, "", "", fmt.Sprintf("%s has no lane %q", st.RunID, laneID),
				"name a lane `abcd implement status` lists; nothing was changed")
		}
		lane := st.Lanes[i]
		if err := outageOpen(repoRoot, laneID); err != nil {
			return false, err
		}
		k := slices.IndexFunc(lane.Awaits, func(a Await) bool { return a.Role == RoleImplementer })
		if k < 0 {
			return false, refuse(stageRestart, "", laneID,
				fmt.Sprintf("%s has no implementer out (it is at %s), so there is no gone agent to restart", laneID, lane.Stage),
				"run `abcd implement step` to take the lane's next stage; nothing was changed")
		}
		aw := lane.Awaits[k]
		lw, err := restartWorktree(repoRoot, st.RunID, lane)
		if err != nil {
			return false, err
		}
		head, err := gitutil.Run(lw.Path, "rev-parse", "--verify", "--quiet", "HEAD^{commit}")
		if err != nil || !gitutil.IsFullSHA(head) {
			return false, fmt.Errorf("resolving %s's last commit: %v", laneID, err)
		}

		tmp, err := os.MkdirTemp("", "abcd-restart-")
		if err != nil {
			return false, err
		}
		defer os.RemoveAll(tmp)
		patch, files, err := snapshot(lw.Path, head, tmp)
		if err != nil {
			return false, refuse(stageRestart, "", laneID, "git could not capture the lane's uncommitted work: "+fsutil.RedactHome(err.Error()),
				"settle what git reports in the lane's worktree, then run `abcd implement step --restart "+laneID+"` again; nothing was changed")
		}
		if len(patch) > 0 {
			tmpPatch := filepath.Join(tmp, AsidePatchName)
			if err := os.WriteFile(tmpPatch, patch, filePerm); err != nil {
				return false, err
			}
			if err := checkAsidePatch(lw.Path, tmpPatch); err != nil {
				return false, refuse(stageRestart, "", laneID,
					"the uncommitted work's patch does not apply to the lane's last commit "+shortSHA(head)+", so it is not saved and the worktree is not reset: "+fsutil.RedactHome(err.Error()),
					"save the lane's uncommitted work by hand, then run `abcd implement step --restart "+laneID+"` again; nothing was changed")
			}
		}

		var partial []byte
		receiptName := ""
		data, err := fsutil.ReadGuardedInRoot(root, aw.Receipt, maxPartialReceiptBytes)
		switch {
		case err == nil:
			partial, receiptName = data, filepath.Base(aw.Receipt)
		case !errors.Is(err, os.ErrNotExist):
			return false, refuse(stageRestart, "", laneID, "the partial receipt at "+aw.Receipt+" cannot be read: "+fsutil.RedactHome(err.Error()),
				"move it out of the lane's directory yourself, then run `abcd implement step --restart "+laneID+"` again; nothing was changed")
		}

		now := o.now()
		laneDir, err := laneRel(st.RunID, laneID, stageRestart)
		if err != nil {
			return false, err
		}
		parent := laneDir + "/" + AsideDirName
		if err := fsutil.EnsureRealDirAll(repoRoot, parent, dirPerm); err != nil {
			return false, err
		}
		dir, err := asideDir(root, parent, now)
		if err != nil {
			return false, err
		}
		aside := RestartAside{RunID: st.RunID, Lane: laneID, At: now, Path: dir, Head: head, Files: files, Why: why, Receipt: receiptName}
		if len(patch) > 0 {
			aside.Patch = AsidePatchName
			if err := fsutil.WriteFileAtomicInRoot(root, dir+"/"+AsidePatchName, patch, filePerm); err != nil {
				return false, err
			}
		}
		if partial != nil {
			if err := fsutil.WriteFileAtomicInRoot(root, dir+"/"+receiptName, partial, filePerm); err != nil {
				return false, err
			}
		}
		meta, err := json.MarshalIndent(aside, "", "  ")
		if err != nil {
			return false, err
		}
		if err := fsutil.WriteFileAtomicInRoot(root, dir+"/"+AsideFileName, append(meta, '\n'), filePerm); err != nil {
			return false, err
		}
		// The partial receipt leaves the await's path, so the fresh agent
		// finds none there and nothing of the gone one's is handed back.
		if partial != nil {
			if err := root.Remove(aw.Receipt); err != nil && !errors.Is(err, os.ErrNotExist) {
				return false, err
			}
		}

		// Saved and proved: only now is the lane's own worktree reset.
		for _, args := range [][]string{{"reset", "--hard", "--quiet", head}, {"clean", "-f", "-d", "--quiet", "--"}} {
			if _, err := gitutil.Run(lw.Path, args...); err != nil {
				return false, refuse(stageRestart, "", laneID,
					"the uncommitted work is saved aside at "+dir+", but git could not reset the lane's worktree: "+fsutil.RedactHome(err.Error()),
					"settle what git reports in the lane's worktree, then run `abcd implement step --restart "+laneID+"` again")
			}
		}

		aw.Since = now
		lane.Awaits[k] = aw
		st.Lanes[i] = lane
		saved := "nothing was left uncommitted"
		if len(files) > 0 {
			saved = fmt.Sprintf("%d uncommitted file(s) saved aside for review at %s", len(files), dir)
		}
		if receiptName != "" {
			saved += ", with its partial receipt"
		}
		st.Record = append(st.Record, Entry{At: now, Lane: laneID, Stage: stageRestart,
			Note: fmt.Sprintf("restarted %s from its last commit %s (%s): %s (aside %s); a fresh implementer takes the brief again",
				laneID, shortSHA(head), why, saved, dir)})
		st.UpdatedAt = now
		res = RestartResult{StepResult: laneResult(*st, lane, "", &aw), Aside: aside}
		return true, nil
	})
	return res, err
}

// asideDir makes the aside's directory under parent, named by the UTC second,
// with -2, -3, … after it when a restart of the same lane already took that
// second. It is made exclusively, so no aside is ever written into another's.
func asideDir(root *os.Root, parent string, now time.Time) (string, error) {
	stamp := parent + "/" + now.UTC().Format(asideStampLayout)
	for n := 1; n <= maxAsidesPerSecond; n++ {
		dir := stamp
		if n > 1 {
			dir = fmt.Sprintf("%s-%d", stamp, n)
		}
		err := root.Mkdir(dir, dirPerm)
		if err == nil {
			return dir, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return "", err
		}
	}
	return "", contend(stageRestart, "", "", fmt.Sprintf("%d asides were saved under %s this second", maxAsidesPerSecond, parent),
		"run the restart again in a second; nothing was changed")
}

// outageOpen refuses a restart while the run's outage is open: the fresh agent
// would meet the same lost connection.
func outageOpen(repoRoot, laneID string) error {
	run, err := implement.Peek(gitutil.RootCommit(repoRoot))
	if err != nil {
		return err
	}
	cur, err := run.Outage().Current()
	if err != nil {
		return err
	}
	if cur == nil || cur.Status != implement.OutageOpen {
		return nil
	}
	return contend(stageRestart, "", laneID,
		fmt.Sprintf("the run's %s outage is still open (next probe at %s), so a fresh agent would lose its connection too",
			strings.Join(cur.Down, " and "), cur.NextProbeAt.UTC().Format(time.RFC3339)),
		"wait for the shared probe to prove the connection back (`abcd implement outage`), then restart the lane; nothing was changed")
}
