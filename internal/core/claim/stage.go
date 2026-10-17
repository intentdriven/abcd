package claim

import "slices"

// keyStage is the trailer a drain lane's renewal carries: the stage the lane is
// in as it renews, one of LaneStages, so another holder's abcd peers can say
// where that lane stands (spc-2610161044461448, step 18).
const keyStage = "Abcd-Stage"

// The stages a drain lane's renewal may name: a lane's stages from its worktree
// to its land, spelled as the implement loop spells its own. The claim package
// cannot import the loop, which imports it, so the loop's tests hold the loop's
// names to this list.
const (
	LaneStageWorktree  = "worktree"
	LaneStageBrief     = "brief"
	LaneStageImplement = "implement"
	LaneStageValidate  = "validate"
	LaneStageLand      = "land"
)

// LaneStages is every stage a renewal may name, in the order a lane performs
// them.
var LaneStages = []string{LaneStageWorktree, LaneStageBrief, LaneStageImplement, LaneStageValidate, LaneStageLand}

// ValidLaneStage reports whether s is a stage a renewal may name.
func ValidLaneStage(s string) bool { return slices.Contains(LaneStages, s) }
