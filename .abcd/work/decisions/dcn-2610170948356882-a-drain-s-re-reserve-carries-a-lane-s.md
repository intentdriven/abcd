---
id: dcn-2610170948356882
slug: a-drain-s-re-reserve-carries-a-lane-s
date: 2026-10-17
---
2026-10-17 — A drain's re-reserve carries a lane's stage only for a concurrent build, and never under --take-over (facilitator-on-delegation, drain-many-accounts, spc-2610161044461448 step 18 fix round; corrects the premise of dcn-2610170809478944): a drain move passes over every issue with a live run in this checkout before it reserves, so its reservation reaches an issue whose lane here holds it live only when abcd build started that run after the move read the runs, and only then does the stage it names keep the lane's stage on origin; under --take-over the drain's reservation names no stage, since a take-over is never a renewal and claim refuses a stage beside it (grounds: the earlier record's ground that the drain re-reserves its lane's issue at each move does not hold; a take-over is consent to one act, which a stage would refuse for an unrelated reason; review finding 7 of the step 14 to 18 fix round)
