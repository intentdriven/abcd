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
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// BranchPrefix is the namespace every lane branch is cut under.
const BranchPrefix = "build/"

// addWorktree is the store's Add, the one way the worktree stage makes a lane.
// It is a variable only so a test can watch the lane go through it.
var addWorktree = worktree.Add

// LaneOpened, when set, is called once the worktree stage has made a lane's
// worktree, for a build's lane and a drain's alike, and not when it adopts one it
// made before. The front door sets it to ask for the Go build cache's trim, since
// a lane is what fills the cache, and it must return at once: it runs in the
// stage, under the run's lock. Nil calls nothing, so the loop's own tests start
// no trim.
var LaneOpened func()

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

// isLaneWorktree reports whether path is named as the worktree of a lane is
// (laneName).
func isLaneWorktree(path string) bool {
	name := filepath.Base(path)
	i := strings.LastIndex(name, "-lane-")
	return i > 0 && ValidRunID(name[:i]) && ValidLaneID(name[i+1:])
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

// originAskTimeout bounds the worktree stage's ask of origin for its default
// branch's tip. It is a variable only so a test can shorten it.
var originAskTimeout = time.Minute

// originTip is the commit a lane in a checkout with an origin opens on:
// origin's own answer for its default branch's tip, asked of the remote and
// never read from a ref here, since anything that runs in the checkout can
// move a ref here (gitutil.RemoteHeadTip, bounded by originAskTimeout). The
// default branch as read here, defRef at base, must be that tip or an
// ancestor of it, so a default branch here the remote never had is refused
// rather than passed over; and the tip must already be in the repository, as
// a fetch leaves it, or nothing here can show base descends to it. A remote
// that cannot be asked is never read as one that agrees: a lost network (or
// no answer within the bound) is the stage's network failure (netFailure), a
// contention refusal, and any other failure the stage's plain refusal; either
// way nothing is made, and the step is taken again as any refused step is.
// The remote's default branch must be the one read here by its short name
// (main for refs/remotes/origin/main), or the lane would open on a branch
// this checkout does not take as its default. Every read of the repository
// here is as stored (gitutil.RunCappedAsStored), so no replace ref or graft
// file changes what the tip holds or descends from. It returns the remote's
// default branch and its tip.
func originTip(c Context, laneID, defRef, base string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), originAskTimeout)
	defer cancel()
	what := "git ls-remote " + Remote
	branch, tip, err := remoteHeadTip(ctx, nil, c.RepoRoot, Remote)
	if err != nil {
		if ctx.Err() != nil {
			err = fmt.Errorf("%s did not answer within %s: operation timed out", Remote, originAskTimeout)
		}
		if r := netFailure(StageWorktree, laneID, what, err); r != nil {
			r.Remedy = "keep to offline work; run `abcd implement step` again once the network is back; nothing was made"
			return "", "", r
		}
		return "", "", refuse(string(StageWorktree), "", laneID,
			"could not ask "+Remote+" for its default branch's tip, so the commit the lane opens on cannot be shown to be the reviewed default branch: "+gitutil.ScrubURLCredentials(fsutil.RedactHome(err.Error())),
			"settle what git reports (`git ls-remote --symref "+Remote+" HEAD`), then run `abcd implement step` again; nothing was made")
	}
	if here := defaultBranchName(defRef); branch != here {
		return "", "", refuse(string(StageWorktree), "", laneID,
			fmt.Sprintf("%s's default branch is %s, and the default branch here is %s, so the lane would open on a branch this checkout does not take as its default", Remote, branch, here),
			"set the default branch here to "+Remote+"'s (`git remote set-head "+Remote+" --auto`), or "+Remote+"'s to this one, then run `abcd implement step` again; nothing was made")
	}
	if _, err := gitutil.RunCappedAsStored(c.RepoRoot, 4096, "cat-file", "-e", tip+"^{commit}"); err != nil {
		return "", "", refuse(string(StageWorktree), "", laneID,
			fmt.Sprintf("%s reports its %s at %s, which this repository does not hold yet, so the lane's commit cannot be shown to be on it", Remote, branch, tip[:12]),
			"fetch "+Remote+" (`git fetch "+Remote+"`), then run `abcd implement step` again; nothing was made")
	}
	if base != tip {
		if _, err := gitutil.RunCappedAsStored(c.RepoRoot, 4096, "merge-base", "--is-ancestor", base, tip); err != nil {
			return "", "", refuse(string(StageWorktree), "", laneID,
				fmt.Sprintf("the default branch here, %s at %s, is not the remote's: it is neither %s's %s tip %s nor an ancestor of it", gitutil.ShortRef(defRef), base[:12], Remote, branch, tip[:12]),
				"look at what moved "+gitutil.ShortRef(defRef)+" here, set it back to "+Remote+"'s (`git fetch "+Remote+"`), then run `abcd implement step` again; nothing was made")
		}
	}
	return branch, tip, nil
}

// remoteHeadTip is the ask originTip makes of origin, a variable so a test
// can act while it runs.
var remoteHeadTip = gitutil.RemoteHeadTip

// defaultBranchName is the branch name a default ref names: main for
// refs/remotes/origin/main or refs/heads/main.
func defaultBranchName(defRef string) string {
	if s, ok := strings.CutPrefix(defRef, "refs/remotes/"+Remote+"/"); ok {
		return s
	}
	return strings.TrimPrefix(defRef, "refs/heads/")
}

// worktreeStage is the worktree stage's body. It finds what it made last time
// before making anything: a worktree git already lists at the lane's path on
// the lane's branch is the lane's, and is adopted; one on another branch, or
// anything else occupying the path, is refused and left where it is. Otherwise
// the lane's branch is cut into the store from the default branch (where the
// checkout has an origin, from the tip origin reports for it: originTip), or,
// in a trial, from the trial's frozen commit (trialBase). Where origin reported
// the tip, the lane's hooks are pinned before any commit in it (pinLaneHooks).
func worktreeStage(c Context, lane *Lane) (Outcome, error) {
	lw, err := laneWorktree(c.RepoRoot, c.State.RunID, lane.ID)
	if err != nil {
		return Outcome{}, err
	}
	var defRef, base, from, tip string
	if c.State.Trial {
		if base, err = trialBase(c, lane.ID); err != nil {
			return Outcome{}, err
		}
		from = "the trial's frozen commit"
	} else {
		defRef = gitutil.DefaultRef(c.RepoRoot)
		if defRef == "" {
			return Outcome{}, refuse(string(StageWorktree), "", lane.ID, "the repository has no default branch to cut the lane from (no origin/HEAD, and no main, master, trunk or develop)",
				"fetch the remote, or create the default branch, then run `abcd implement step` again")
		}
		base, err = gitutil.RunCappedAsStored(c.RepoRoot, 4096, "rev-parse", "--verify", "--quiet", defRef+"^{commit}", "--")
		if err != nil || !gitutil.IsFullSHA(base) {
			return Outcome{}, fmt.Errorf("resolving the default branch %s: %v", defRef, err)
		}
		from = gitutil.ShortRef(defRef)
		if err := remoteUnchanged(c, lane.ID); err != nil {
			return Outcome{}, err
		}
		has, err := claim.HasOrigin(c.RepoRoot)
		if err != nil {
			return Outcome{}, err
		}
		if has {
			branch, reported, err := originTip(c, lane.ID, defRef, base)
			if err != nil {
				return Outcome{}, err
			}
			// Read again once origin has answered: a change made while the
			// ask ran and left in place is refused too (remoteUnchanged).
			if err := remoteUnchanged(c, lane.ID); err != nil {
				return Outcome{}, err
			}
			tip, base = reported, reported
			from = Remote + "/" + branch + " as " + Remote + " reports it"
		}
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
	// Where origin reported the tip, the hooks directory git takes at the
	// checkout is read once: a standing branch is judged by it and the add
	// is told it (laneAddGit). git's add runs in the checkout, so it takes
	// the hooks directory there and, for a relative core.hooksPath, runs the
	// checkout's own working-tree copy of the post-checkout hook, never the
	// copy of the commit it checks out. The standing-branch check is
	// defensive: it refuses a branch whose hooks differ before a worktree is
	// made rather than at the proof after it. The value on the add's command
	// line refuses nothing: it pins the directory the check judged. Neither
	// stops that hook from running.
	hooksPath, judged := "", ""
	var added worktree.Added
	if tip != "" && !adopted {
		if hooksPath, err = checkoutHooksPath(c.RepoRoot); err != nil {
			return Outcome{}, refuse(string(StageWorktree), "", lane.ID,
				"the hooks directory git takes at the checkout cannot be read, so the hooks the lane's checkout would run cannot be told: "+gitutil.ScrubURLCredentials(fsutil.RedactHome(err.Error())),
				"look at the repository's git configuration (`git config --show-origin --get-all core.hooksPath`), then run `abcd implement step` again; nothing was made")
		}
	}
	switch {
	case adopted:
		// Made last time; the base is where the branch left the default branch.
		mb, err := gitutil.RunCappedAsStored(c.RepoRoot, 4096, "merge-base", base, branchRef)
		if err != nil || !gitutil.IsFullSHA(mb) {
			return Outcome{}, fmt.Errorf("finding where %s left %s: %v", lw.Branch, from, err)
		}
		base = mb
	default:
		opts := worktree.AddOptions{Name: name, Branch: lw.Branch, Base: base, Git: laneAddGit(hooksPath)}
		if _, err := gitutil.Run(c.RepoRoot, "rev-parse", "--verify", "--quiet", branchRef, "--"); err == nil {
			// A process killed between git making the branch and the worktree
			// left the branch: it is the lane's, checked out rather than remade.
			// Where origin reported the tip, its head is resolved once, as
			// stored, and that commit is judged: its hooks must be the tip's
			// (standingBranchHooks), so a lane whose hooks differ is refused
			// before a worktree is made rather than after. If the branch holds
			// another commit after the add, the lane is refused and the
			// worktree this call added removed, so the next step judges the
			// branch again.
			opts.Existing = true
			at := branchRef
			if tip != "" {
				judged, err = gitutil.RunCappedAsStored(c.RepoRoot, 4096, "rev-parse", "--verify", "--quiet", branchRef+"^{commit}", "--")
				if err != nil || !gitutil.IsFullSHA(judged) {
					return Outcome{}, fmt.Errorf("resolving the lane branch %s: %v", lw.Branch, err)
				}
				if err := standingBranchHooks(c, lane.ID, judged, lw.Branch, tip, hooksPath); err != nil {
					return Outcome{}, err
				}
				at = judged
			}
			if mb, err := gitutil.RunCappedAsStored(c.RepoRoot, 4096, "merge-base", base, at); err == nil && gitutil.IsFullSHA(mb) {
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
		//
		// Just before it, a drive's lane has the checkout's hooks directory
		// compared with what the drive read as it started
		// (checkoutHooksUnchanged), so a top-level hook file changed there
		// between two lanes and still changed at this read is refused before
		// this add. Its limits are checkoutHooksUnchanged's: a change made
		// after this read and before git runs the hook is not seen, since
		// nothing holds the directory through the add; the directory read is
		// the one the repository's own settings name; and only its top-level
		// entries are compared.
		if err := checkoutHooksUnchanged(c, lane.ID); err != nil {
			return Outcome{}, err
		}
		added, err = addWorktree(c.RepoRoot, opts)
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
		if LaneOpened != nil {
			LaneOpened()
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
	head, err := gitutil.RunCappedAsStored(c.RepoRoot, 4096, "rev-parse", "--verify", "--quiet", branchRef+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(head) {
		return Outcome{}, fmt.Errorf("resolving the lane branch %s: %v", lw.Branch, err)
	}
	if judged != "" && head != judged {
		// The worktree this call added is removed (worktree.RemoveAdded, which
		// requires a real directory at the lane's path that git lists there on
		// the lane's branch), and the branch left, so the next step meets the
		// standing branch and judges it again before any add. If the removal
		// is refused or fails, the step is refused and what stands at the path
		// is left; a step taken again while the lane's own worktree stands
		// there adopts it, and an adopted lane is judged by its hooks only
		// (pinLaneHooks; TestAStandingBranchMovedWhoseWorktreeCannotBeRemovedIsAdoptedNext).
		why := fmt.Sprintf("the branch %s moved from %s, the commit judged, to %s while git added the lane's worktree, so what was checked out is not what was judged; nothing is pinned", lw.Branch, judged[:12], head[:12])
		if added.Path != filepath.Clean(lw.Path) {
			return Outcome{}, fmt.Errorf("%s, and the worktree made is not at the lane's path", why)
		}
		if err := worktree.RemoveAdded(c.RepoRoot, added); err != nil {
			return Outcome{}, refuse(string(StageWorktree), "", lane.ID,
				why+"; the worktree the step added could not be removed: "+fsutil.RedactHome(gitutil.ScrubURLCredentials(err.Error())),
				"look at what moved the branch and at what stands at "+fsutil.RedactHome(lw.Path)+"; if a real directory, not a link, stands there and git lists it on the lane's branch (`git worktree list`), remove it (`git worktree remove --force "+fsutil.RedactHome(lw.Path)+"`) so the next step judges the branch again, then run `abcd implement step` again; a step taken while that worktree stands adopts it, judged by its hooks only; nothing was committed in the lane")
		}
		return Outcome{}, refuse(string(StageWorktree), "", lane.ID,
			why+"; the worktree the step added was removed and the branch left, so the next step judges the branch again before any add",
			"look at what moved the branch; remove it (`git branch -D "+lw.Branch+"`) so the stage cuts the lane afresh from the tip, or leave it to be judged again, then run `abcd implement step` again; nothing was committed in the lane")
	}
	// The lane's hooks are pinned before any commit in it (pinLaneHooks),
	// made or adopted alike, from the commit it opens on. One hook has run
	// there already: git's add ran its post-checkout hook with the new
	// worktree as its directory. The add runs in the checkout, so that hook
	// is from the hooks directory git takes at the checkout: for a relative
	// core.hooksPath, the checkout's own working-tree copy, never the copy of
	// the commit checked out (TestTheAddRunsTheCheckoutsOwnCopyOfThePostCheckoutHook);
	// otherwise the directory the setting names. Where the repository's own
	// settings name the hooks directory, the run's fingerprint records that
	// copy at the run's first check, normally this very step, and within a run
	// a later change to it is refused at the next step. Across the lanes of one
	// drive the drive holds it, within limits (checkoutHooksUnchanged, just
	// before the add): a change after that read and before git runs the hook
	// is not seen, the directory read is the one the repository's own settings
	// name, and only its top-level entries are compared.
	// Elsewhere (a step the host runs, a single build, a drive started again)
	// nothing does: a copy changed before the run's first check is that run's
	// starting point and runs in the add, so it must not change between runs,
	// and nothing enforces it. One named only in the person's global settings
	// is not watched at all. The add reads
	// objects as stored, so the lane holds the tree its commit stores. An
	// adopted worktree ran its hook when it was made. With no origin, or in a
	// trial, nothing is pinned and the lane is compared with the checkout as
	// the run recorded it.
	if tip != "" {
		if err := pinLaneHooks(c, lane.ID, lw.Path, base, tip); err != nil {
			return Outcome{}, err
		}
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
		verb, lw.StoreRel, filepath.Base(lw.Path), lw.Branch, from, base[:12], picked)}, nil
}
