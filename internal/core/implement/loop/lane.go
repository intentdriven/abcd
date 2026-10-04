package loop

// lane.go is the lane's worktree (spec piece 6): the checkout a lane's
// implementer works in, made in abcd's form. It lives in the machine-scoped
// worktree store, ~/.abcd.noindex/worktrees/<root-sha>/<run-id>-<lane-id>, keyed on the
// repository's root commit in the full form the sibling stores use, on a branch
// cut from the default branch. The store's own verbs are a draft
// (itd-2609091014076309), so the lane is a plain `git worktree add` into the
// store's path until they ship.
//
// The path is derived, never taken: the run id and the lane id are each held to
// their own shape and the name they compose to a single safe path segment, so
// no component can walk out of the store, and nothing outside
// ~/.abcd.noindex/worktrees/<root-sha>/ is ever created (the user's directory is
// theirs; adr-2609091248200336). Every level of the store is made one at a time
// and proved a real directory that is the caller's alone, and git runs through
// the isolated environment.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// WorktreeStoreRel is the machine-scoped worktree store, relative to the
// caller's home.
var WorktreeStoreRel = abcdhome.Rel("worktrees")

// BranchPrefix is the namespace every lane branch is cut under.
const BranchPrefix = "build/"

// storeDirPerm is the mode a store level is created with.
const storeDirPerm fs.FileMode = 0o700

// maxWorktreeListing caps `git worktree list`.
const maxWorktreeListing = 16 << 20

// laneIDRe is the shape of a lane id the loop opens (openNextLane).
var laneIDRe = regexp.MustCompile(`^lane-[1-9][0-9]{0,3}$`)

// ValidLaneID reports whether id is a lane id the loop opens.
func ValidLaneID(id string) bool { return laneIDRe.MatchString(id) }

// safeSegment reports whether s is one path segment that cannot escape the
// directory it is joined onto and cannot be read as an option: letters,
// digits, '.', '_' and '-', not led by '-' or '.', and holding no "..".
func safeSegment(s string) bool {
	if s == "" || s[0] == '-' || s[0] == '.' || strings.Contains(s, "..") {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '.', c == '_', c == '-':
		default:
			return false
		}
	}
	return true
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
}

// laneWorktree derives the lane's worktree and branch from the run and lane
// ids and the repository's root commit, building no path from anything else.
func laneWorktree(repoRoot, runID, laneID string) (LaneWorktree, error) {
	name, err := laneName(runID, laneID)
	if err != nil {
		return LaneWorktree{}, err
	}
	sha := gitutil.RootCommit(repoRoot)
	if !gitutil.IsFullSHA(sha) {
		return LaneWorktree{}, refuse(string(StageWorktree), "", laneID, "the repository has no root commit to key the worktree store on",
			"commit to the repository first; the store is keyed on its root commit")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return LaneWorktree{}, fmt.Errorf("cannot resolve the caller's home directory for the worktree store: %v", err)
	}
	storeRel := WorktreeStoreRel + "/" + sha
	return LaneWorktree{
		Home:     home,
		RootSHA:  sha,
		StoreRel: storeRel,
		Path:     filepath.Join(home, filepath.FromSlash(storeRel), name),
		Branch:   BranchPrefix + name,
	}, nil
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

	if err := ensureStore(lw.Home, lw.StoreRel, lane.ID); err != nil {
		return Outcome{}, err
	}

	adopted, err := adoptWorktree(c.RepoRoot, lw, lane.ID)
	if err != nil {
		return Outcome{}, err
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
		if _, err := os.Lstat(lw.Path); err == nil {
			return Outcome{}, refuse(string(StageWorktree), "", lane.ID,
				fmt.Sprintf("%s is occupied by something git does not list as this lane's worktree", fsutil.RedactHome(lw.Path)),
				"the loop never adopts what it cannot prove it made; move it aside, then run `abcd implement step` again")
		} else if !errors.Is(err, fs.ErrNotExist) {
			return Outcome{}, fmt.Errorf("checking %s: %w", fsutil.RedactHome(lw.Path), err)
		}
		args := []string{"worktree", "add", "-b", lw.Branch, "--", lw.Path, base}
		if _, err := gitutil.Run(c.RepoRoot, "rev-parse", "--verify", "--quiet", branchRef, "--"); err == nil {
			// A process killed between git making the branch and the worktree
			// left the branch: it is the lane's, checked out rather than remade.
			args = []string{"worktree", "add", "--", lw.Path, lw.Branch}
			if mb, err := gitutil.Run(c.RepoRoot, "merge-base", base, branchRef); err == nil && gitutil.IsFullSHA(mb) {
				base = mb
			}
		}
		if _, err := gitutil.Run(c.RepoRoot, args...); err != nil {
			return Outcome{}, refuse(string(StageWorktree), "", lane.ID,
				"git could not add the lane's worktree: "+fsutil.RedactHome(err.Error()),
				"settle what git reports, then run `abcd implement step` again")
		}
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

// ensureStore makes the store's levels under home one at a time, from the top
// down, and holds each to two tests before the next is made: it is a real
// directory (a symlink or a file anywhere in the chain refuses), and it is the
// caller's alone (fsutil.CallersAlone: owned by this uid, no group or other
// write bit), so no other account can replace the checkout a lane's
// implementer works in. A level made here is made storeDirPerm; one that
// already exists keeps its mode and is judged as it stands, and nothing is made
// inside a level that fails.
func ensureStore(home, rel, laneID string) error {
	if !fsutil.IsRealDir(home) || !fsutil.ValidRelPath(rel) {
		return refuse(string(StageWorktree), "", laneID,
			"the home directory is not a real directory to make the worktree store under",
			"run `abcd implement step` from an account whose home is a real directory")
	}
	dir, shown := home, "~"
	for _, seg := range strings.Split(rel, "/") {
		dir, shown = filepath.Join(dir, seg), shown+"/"+seg
		if err := fsutil.EnsureRealDir(dir, storeDirPerm); err != nil {
			return refuse(string(StageWorktree), "", laneID,
				fmt.Sprintf("the worktree store ~/%s cannot be made as real directories: %v", rel, fsutil.RedactHome(err.Error())),
				"remove what stands in for the store (a symlink or a file), then run `abcd implement step` again")
		}
		fi, err := os.Lstat(dir)
		if err != nil {
			return fmt.Errorf("checking the worktree store level %s: %w", shown, err)
		}
		switch err := fsutil.CallersAlone(dir, fi); {
		case errors.Is(err, fsutil.ErrDeclarationWritable):
			return refuse(string(StageWorktree), "", laneID,
				fmt.Sprintf("the worktree store level %s is writable by its group or by every user, so another account could replace the lane's checkout", shown),
				"make it yours alone (`chmod go-w` it), then run `abcd implement step` again")
		case err != nil:
			return refuse(string(StageWorktree), "", laneID,
				fmt.Sprintf("the worktree store level %s is owned by another account, or its owner could not be read, so that account could replace the lane's checkout", shown),
				"move it aside and let the worktree stage make the store, then run `abcd implement step` again")
		}
	}
	return nil
}

// adoptWorktree reports whether git already lists a worktree at the lane's
// path on the lane's branch. One at that path on any other branch is refused.
func adoptWorktree(repoRoot string, lw LaneWorktree, laneID string) (bool, error) {
	wts, err := gitutil.ListWorktrees(repoRoot, maxWorktreeListing)
	if err != nil {
		return false, fmt.Errorf("listing the repository's worktrees: %w", err)
	}
	want := fsutil.RealExistingPath(lw.Path)
	for _, wt := range wts {
		if fsutil.RealExistingPath(wt.Path) != want {
			continue
		}
		if wt.Branch != "refs/heads/"+lw.Branch {
			return false, refuse(string(StageWorktree), "", laneID,
				fmt.Sprintf("git lists a worktree at %s on %q, not on the lane's branch %s", fsutil.RedactHome(lw.Path), wt.Branch, lw.Branch),
				"the loop never adopts what it cannot prove it made; remove that worktree (`git worktree remove`), then run `abcd implement step` again")
		}
		fi, err := os.Lstat(lw.Path)
		if err != nil || !fi.IsDir() {
			return false, refuse(string(StageWorktree), "", laneID,
				fmt.Sprintf("git lists the lane's worktree at %s, but no real directory stands there", fsutil.RedactHome(lw.Path)),
				"run `git worktree prune`, then run `abcd implement step` again")
		}
		return true, nil
	}
	return false, nil
}
