package loop

// lane.go is the lane's worktree (spec piece 6): the checkout a lane's
// implementer works in, made in abcd's form. It lives in the machine-scoped
// worktree store, ~/.abcd.noindex/worktrees/<root-sha>/<run-id>-<lane-id>, keyed on the
// repository's root commit in the full form the sibling stores use, on a branch
// cut from the default branch. The store is the worktree package's
// (internal/core/ahoy/worktree, itd-2609091014076309): the lane is keyed,
// made, adopted and added through its LaneFor, Lane.Ensure, Adopt and Add, so
// a lane enters the store the one way every worktree does. What stays here is
// the loop's own: the run-id and lane-id shapes, the build/ branch prefix, the
// base cut from the default branch, the checkout's time bound, and the pick
// commit.
//
// The path is derived, never taken: the run id and the lane id are each held to
// their own shape and the name they compose to a single safe path segment, so
// no component can walk out of the store, and nothing outside
// ~/.abcd.noindex/worktrees/<root-sha>/ is ever created (the user's directory is
// theirs; adr-2609091248200336).

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// BranchPrefix is the namespace every lane branch is cut under.
const BranchPrefix = "build/"

// addWorktree is the store's Add, the one way the worktree stage makes a lane.
// It is a variable only so a test can watch the lane go through it.
var addWorktree = worktree.Add

// laneIDRe is the shape of a lane id the loop opens (openNextLane).
var laneIDRe = regexp.MustCompile(`^lane-[1-9][0-9]{0,3}$`)

// ValidLaneID reports whether id is a lane id the loop opens.
func ValidLaneID(id string) bool { return laneIDRe.MatchString(id) }

// safeSegment reports whether s is one path segment that cannot escape the
// directory it is joined onto and cannot be read as an option: the store's
// own test of a worktree name (worktree.ValidName).
func safeSegment(s string) bool { return worktree.ValidName(s) }

// storeRefusal is a refusal of the worktree store as the loop's own, at the
// worktree stage for laneID, its remedy ending in how to retry here. Any other
// error is returned as it is.
func storeRefusal(err error, laneID string) error {
	var r *worktree.Refusal
	if !errors.As(err, &r) {
		return err
	}
	return refuse(string(StageWorktree), "", laneID, r.Reason, r.Remedy+", then run `abcd implement step` again")
}

// laneName composes a lane's store name, <run-id>-<lane-id>, refusing any
// component that is not the loop's own shape. Each refusal names, quoted, the id
// it refuses and carries no lane: the lane id is not yet proved when the run id
// fails, and is the thing refused when it fails itself.
func laneName(runID, laneID string) (string, error) {
	if !ValidRunID(runID) {
		return "", refuse(string(StageWorktree), "", "", fmt.Sprintf("%q is not a run id, so no lane path is built from it", runID),
			"the loop names its runs; restore the run's state file")
	}
	if !ValidLaneID(laneID) {
		return "", refuse(string(StageWorktree), "", "", fmt.Sprintf("%q is not a lane id (lane-<n>), so no lane path is built from it", laneID),
			"the loop names its lanes; restore the run's state file")
	}
	name := runID + "-" + laneID
	if !safeSegment(name) {
		return "", refuse(string(StageWorktree), "", laneID, fmt.Sprintf("%q is not a single safe path segment", name),
			"the loop names its lanes; restore the run's state file")
	}
	return name, nil
}

// LaneWorktree is where a lane's worktree lives and what it is called.
type LaneWorktree struct {
	// Home is the caller's home, the base the store is proved real from.
	Home string
	// RootSHA keys the store on the repository.
	RootSHA string
	// StoreRel is the lane's store directory, relative to Home.
	StoreRel string
	// Path is the worktree, Home/StoreRel/<name>.
	Path string
	// Branch is the lane's branch, build/<name>.
	Branch string

	// lane is the store's lane the worktree lives in.
	lane worktree.Lane
}

// laneWorktree derives the lane's worktree and branch from the run and lane
// ids and the repository's root commit, building no path from anything else.
func laneWorktree(repoRoot, runID, laneID string) (LaneWorktree, error) {
	name, err := laneName(runID, laneID)
	if err != nil {
		return LaneWorktree{}, err
	}
	// A repository with no root commit is the store's refusal; git not
	// answering is an error, no evidence of none, and no path is built.
	l, err := worktree.LaneFor(repoRoot)
	if err != nil {
		return LaneWorktree{}, storeRefusal(err, laneID)
	}
	return LaneWorktree{
		Home:     l.Home,
		RootSHA:  l.RootSHA,
		StoreRel: l.Rel,
		Path:     l.Path(name),
		Branch:   BranchPrefix + name,
		lane:     l,
	}, nil
}

// HeldWorktrees is what the build loop holds in the worktree store, for the
// store's prune (spc-2609301811532881): every lane of a run whose state is not
// complete, by the name laneName gives it in the store, read across the local
// tier of every checkout git lists for the repository, because a run's state
// lives in the checkout that started it. A state file that cannot be read
// holds every worktree named after its run; a run directory that cannot be
// listed fails the read, so prune removes nothing. It is a
// worktree.HeldReader.
func HeldWorktrees(repoRoot string) (worktree.Held, error) {
	wts, err := gitutil.ListWorktrees(repoRoot, gitutil.MaxWorktreeListing)
	if err != nil {
		return worktree.Held{}, fmt.Errorf("listing the repository's worktrees: %w", err)
	}
	held := worktree.Held{Names: map[string]string{}}
	for _, wt := range wts {
		// A checkout whose directory is gone holds no state to read.
		if wt.Bare || wt.Prunable || !fsutil.IsRealDir(wt.Path) {
			continue
		}
		if err := heldIn(wt.Path, &held); err != nil {
			return worktree.Held{}, err
		}
	}
	return held, nil
}

// heldIn adds what the runs in the checkout at dir hold to held.
func heldIn(dir string, held *worktree.Held) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("opening %s to read its runs: %w", fsutil.RedactHome(dir), err)
	}
	defer root.Close()
	ids, err := runIDs(root)
	if err != nil {
		return err
	}
	for _, id := range ids {
		st, err := readStateIn(root, id)
		if err != nil {
			held.Runs = append(held.Runs, id)
			continue
		}
		if st.Complete() {
			continue
		}
		for _, l := range st.Lanes {
			held.Names[id+"-"+l.ID] = id
		}
	}
	return nil
}

// loopWorktree refuses a lane whose recorded worktree is not the path
// laneWorktree derives for the run and lane. The run's state file sits in the
// gitignored local tier, where an archive of the checkout can carry a
// hand-written one, so a stage that makes, runs or removes anything in the
// lane's worktree, or hands it to an agent, takes the path from the
// derivation and never from the state alone: the land stage trusted any
// non-empty string and laid the close's local tier in a directory the state
// named (iss-2610090821552801). stage labels the refusal.
func loopWorktree(c Context, lane Lane, stage string) error {
	_, err := derivedLaneWorktree(c.RepoRoot, c.State.RunID, lane, stage)
	return err
}

// derivedLaneWorktree is the worktree the loop derives for lane, refusing a
// lane whose state names any other path: the one answer to where a lane's
// worktree is, which every stage that hands the path on, and the restart,
// asks (loopWorktree, restartWorktree).
func derivedLaneWorktree(repoRoot, runID string, lane Lane, stage string) (LaneWorktree, error) {
	lw, err := laneWorktree(repoRoot, runID, lane.ID)
	if err != nil {
		return LaneWorktree{}, relabel(err, Stage(stage))
	}
	if lane.Worktree == "" || filepath.Clean(lane.Worktree) != filepath.Clean(lw.Path) {
		return LaneWorktree{}, refuse(stage, "", lane.ID,
			"the lane has no worktree the loop made (its state names "+quoteOrNone(fsutil.RedactHome(lane.Worktree))+")",
			"the worktree stage makes it; restore the run's state file")
	}
	return lw, nil
}

// worktreeStage is the worktree stage's body. It finds what it made last time
// before making anything: a worktree git already lists at the lane's path on
// the lane's branch is the lane's, and is adopted; one on another branch, or
// anything else occupying the path, is refused and left where it is. Otherwise
// the lane's branch is cut from the default branch into the store.
func worktreeStage(c Context, lane *Lane) (Outcome, error) {
	lw, err := laneWorktree(c.RepoRoot, c.State.RunID, lane.ID)
	if err != nil {
		return Outcome{}, err
	}
	defRef := gitutil.DefaultRef(c.RepoRoot)
	if defRef == "" {
		return Outcome{}, refuse(string(StageWorktree), "", lane.ID, "the repository has no default branch to cut the lane from (no origin/HEAD, and no main, master, trunk or develop)",
			"fetch the remote, or create the default branch, then run `abcd implement step` again")
	}
	base, err := gitutil.Run(c.RepoRoot, "rev-parse", "--verify", "--quiet", defRef+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(base) {
		return Outcome{}, fmt.Errorf("resolving the default branch %s: %v", defRef, err)
	}

	if err := lw.lane.Ensure(); err != nil {
		return Outcome{}, storeRefusal(err, lane.ID)
	}

	name := filepath.Base(lw.Path)
	adopted, err := worktree.Adopt(c.RepoRoot, lw.lane, name, lw.Branch)
	if err != nil {
		return Outcome{}, storeRefusal(err, lane.ID)
	}
	branchRef := "refs/heads/" + lw.Branch
	switch {
	case adopted:
		// Made last time; the base is where the branch left the default branch.
		mb, err := gitutil.Run(c.RepoRoot, "merge-base", base, branchRef)
		if err != nil || !gitutil.IsFullSHA(mb) {
			return Outcome{}, fmt.Errorf("finding where %s left %s: %v", lw.Branch, defRef, err)
		}
		base = mb
	default:
		opts := worktree.AddOptions{Name: name, Branch: lw.Branch, Base: base, Git: worktreeAddGit}
		if _, err := gitutil.Run(c.RepoRoot, "rev-parse", "--verify", "--quiet", branchRef, "--"); err == nil {
			// A process killed between git making the branch and the worktree
			// left the branch: it is the lane's, checked out rather than remade.
			opts.Existing = true
			if mb, err := gitutil.Run(c.RepoRoot, "merge-base", base, branchRef); err == nil && gitutil.IsFullSHA(mb) {
				base = mb
			}
		}
		// The checkout writes the lane's files as the person's own git
		// would, through the filters their global config defines
		// (iss-2610111729375108), and runs the repository's post-checkout
		// hook (the one `git lfs install` installs), as the person's own
		// `git worktree add` does (iss-2610120640452939). Its time scales
		// with the tree, so it has its own bound (worktreeAddGit). The
		// store's Add refuses what stands at the path, and sets the
		// worktree to 0700 once git has made it.
		_, err := addWorktree(c.RepoRoot, opts)
		var gitFailed *worktree.Refusal
		if errors.As(err, &gitFailed) && gitFailed.Err != nil {
			err = gitFailed.Err
		} else if err != nil {
			return Outcome{}, storeRefusal(err, lane.ID)
		}
		if errors.Is(err, errCheckoutStalled) {
			return Outcome{}, refuse(string(StageWorktree), "", lane.ID,
				"git could not add the lane's worktree: "+fsutil.RedactHome(err.Error()),
				fmt.Sprintf("settle the repository's post-checkout hook if it waits; a checkout that needs longer than %s (a large Git LFS download) can be made by hand at %s on %s, which the worktree stage adopts; then run `abcd implement step` again",
					worktreeAddTimeout, fsutil.RedactHome(lw.Path), lw.Branch))
		} else if err != nil {
			return Outcome{}, refuse(string(StageWorktree), "", lane.ID,
				"git could not add the lane's worktree: "+fsutil.RedactHome(err.Error()),
				"settle what git reports, then run `abcd implement step` again")
		}
	}
	// An adopted worktree is narrowed to 0700 here, as Add narrows one it
	// makes, since a process killed between git's add and Add's narrowing left
	// it as git made it. A store on a filesystem that cannot change modes
	// (exFAT, a network share) keeps the lane as git made it rather than
	// refusing the step: the home above it is narrowed at every command, so
	// other accounts cannot reach the lane.
	if adopted {
		_, _ = fsutil.NarrowToOwner(lw.Path)
	}
	head, err := gitutil.Run(c.RepoRoot, "rev-parse", "--verify", "--quiet", branchRef+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(head) {
		return Outcome{}, fmt.Errorf("resolving the lane branch %s: %v", lw.Branch, err)
	}
	// A picked run's first lane carries the pick's reason as its first
	// commit, made here, before the brief and the implementer
	// (itd-2609211116005482; pickcommit.go).
	picked := ""
	if p := c.State.Pick; p != nil && p.Lane == lane.ID {
		sha, err := pickCommit(c, lane, lw.Path, lw.Branch, base)
		if err != nil {
			return Outcome{}, err
		}
		lane.PickSHA, head = sha, sha
		picked = fmt.Sprintf("; the pick's entry is its first commit, %s", sha[:12])
	}
	lane.Worktree = lw.Path
	lane.Branch = lw.Branch
	lane.BaseSHA = base
	lane.HeadSHA = head
	verb := "made"
	if adopted {
		verb = "found"
	}
	return Outcome{Note: fmt.Sprintf("%s the worktree ~/%s/%s on %s, cut from %s at %s%s",
		verb, lw.StoreRel, filepath.Base(lw.Path), lw.Branch, gitutil.ShortRef(defRef), base[:12], picked)}, nil
}
