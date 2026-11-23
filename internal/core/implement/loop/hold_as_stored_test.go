package loop

import (
	"strings"
	"testing"
)

// TestAnArmedSiblingIsPlacedOnTheDefaultBranchAsStored: the hold reads an armed
// sibling as landed only when the default branch's tracking ref stores its
// pushed head; a replace ref or a graft giving that tip the pushed head as a
// parent leaves the sibling disarmed and held before arming, as one the
// default branch does not hold.
func TestAnArmedSiblingIsPlacedOnTheDefaultBranchAsStored(t *testing.T) {
	for name, graftFile := range map[string]bool{"a replace ref": false, "a graft file": true} {
		t.Run(name, func(t *testing.T) {
			f := armedSibling(t, false)
			f.handedBack(t)
			l2 := f.lane(t, "lane-2")
			tip := f.repo.Git("rev-parse", "refs/remotes/origin/main")
			parents := strings.Fields(f.repo.Git("rev-list", "--parents", "-n", "1", tip))[1:]
			standParents(t, f.repo, graftFile, tip, append(parents, l2.Landing.Pushed)...)
			if !presentedHolds(f.repo, l2.Landing.Pushed, tip) {
				t.Fatal("git presents the pushed head on the default branch")
			}
			f.step(t)
			l2 = f.lane(t, "lane-2")
			if l2.Stage != StageHeld || l2.Hold == nil || l2.Hold.Before != HoldBeforeArm || l2.Landing.Armed || l2.Landing.Merged != "" {
				t.Fatalf("the sibling is held before arming, not read as landed: %+v", l2)
			}
			if !strings.Contains(f.ghLog(t), "pr merge 7 --disable-auto") {
				t.Fatalf("the sibling's arming is withdrawn:\n%s", f.ghLog(t))
			}
		})
	}
}
