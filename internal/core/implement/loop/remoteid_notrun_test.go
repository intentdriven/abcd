package loop

// remoteid_notrun_test.go holds rulings 2 and 4 of fix round 5 of
// iss-2611131308266006: a git the drive's comparison did not run is
// recognised by its type wherever a git error is classified, before any text
// is read; and a read cut at the comparison's deadline says the comparison
// did not finish.

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// networkWordsStop is the drive's stop for a hooks file whose name reads as
// git's words for a lost network.
func networkWordsStop() error {
	return &gitutil.NotRunError{Verb: "push", Err: refuse(StageConfig, "", "",
		"what the drive holds changed since it started (hooks/connection refused (added))", "look at it")}
}

// TestAGitNotRunIsNeverALostNetwork: a stop whose text holds git's words for
// a lost network, as a hooks file can be named, is not taken for one by
// netFailure, bare or wrapped, nor by the network probe, which says it was
// not run. Watched failing before the type was read first: netFailure took
// it for a lost network and the probe returned its text as an outage's.
func TestAGitNotRunIsNeverALostNetwork(t *testing.T) {
	stop := networkWordsStop()
	if !implement.IsNetworkFailure(stop.Error()) {
		t.Fatal("fixture: the stop's text reads as a lost network")
	}
	for _, err := range []error{stop, fmt.Errorf("git push: %w", stop)} {
		if r := netFailure(StageLand, "lane-1", "git push", err); r != nil {
			t.Fatalf("a git not run is not a lost network: %+v", r)
		}
	}
	repo, _, _ := hookedOrigin(t)
	unset, err := gitutil.SetBeforeHookedGit(func([]string) error { return errors.Unwrap(stop) })
	if err != nil {
		t.Fatal(err)
	}
	defer unset()
	if up, why := implement.RemoteProbe(repo.Root())(); up || !strings.HasPrefix(why, "the probe of origin was not run: git ls-remote was not run") {
		t.Fatalf("the probe says it was not run: %v %q", up, why)
	}
}

// TestASyncsReadsAfterAFailedMergeSayTheMergeIsLeft: the land step merges
// origin's moved default branch into an issue lane and the merge conflicts;
// a held setting changed while it ran stops the read of the conflicted paths
// that follows. The step stops at the config stage saying git diff was not
// run, that the comparison stopped it, and that a merge may be left in the
// lane's worktree, with how to abort it; the merge is indeed left there.
// Watched failing before the reads recognised the stop: the step said only
// that git could not merge, at the land stage.
func TestASyncsReadsAfterAFailedMergeSayTheMergeIsLeft(t *testing.T) {
	f := syncFixture(t)
	f.passToLandOnce(t, "fix.txt", "plain.txt")
	f.moveMain(t, "rewritten on main\n", "plain.txt")
	id, err := readRemoteIdentity(f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	asked := heldCheckAt(t, f.repo.Root(), id, "merge", false, func() {
		f.repo.Git("config", "core.editor", "/bin/true")
	})
	_, err = advance(f.repo.Root(), f.runID, f.stages, Options{remote: id})
	r := configRefused(t, err, "git diff was not run", "core.editor", "a merge may be left in progress in the lane's worktree")
	if !*asked || !strings.Contains(r.Remedy, "git merge --abort") {
		t.Fatalf("the merge ran and the remedy says how to abort what it left: %+v", r)
	}
	l := currentLane(t, f.repo, f.runID)
	if out := f.repo.Git("-C", l.Worktree, "rev-parse", "-q", "--verify", "MERGE_HEAD"); strings.TrimSpace(out) == "" {
		t.Fatal("the merge is left in progress, as the step says")
	}
}

// TestAReadCutInsideItsRetryWaitAtTheDeadlineIsTheComparisonNotFinishing: a
// read whose first ask runs out with less than the retry wait left before
// the comparison's deadline is not asked again, and is the comparison not
// finishing, not a read failure of its own. Watched failing before: it was a
// read failure naming its first ask's bound.
func TestAReadCutInsideItsRetryWaitAtTheDeadlineIsTheComparisonNotFinishing(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	standInGit(t, `*"var GIT_ATTR_GLOBAL"*) exec /bin/sleep 600 ;;`)
	bound, wait := heldReadTimeout, heldReadRetryWait
	heldReadTimeout, heldReadRetryWait = 4*time.Second, 2*time.Second
	t.Cleanup(func() { heldReadTimeout, heldReadRetryWait = bound, wait })
	_, err := isolatedEnv.readBy(time.Now().Add(2500*time.Millisecond), repo.Root(), "the global attributes file", 4096, "var", "GIT_ATTR_GLOBAL")
	if !errors.Is(err, errCompareUnfinished) {
		t.Fatalf("a read the comparison's deadline left no retry for is the comparison not finishing: %v", err)
	}
}
