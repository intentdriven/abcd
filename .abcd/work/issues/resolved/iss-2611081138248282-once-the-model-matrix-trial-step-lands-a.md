---
schema_version: 1
id: "iss-2611081138248282"
slug: "once-the-model-matrix-trial-step-lands-a"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "run-2611080653222827 lane-1 validate round 4 (ruthless-reviewer note)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/record.go"
remedy: "pickFalsifiedBy skips a hand-back whose verdict is cell-failed, matching handBackLane, with a test that a cell-failed trial lane leaves the pick unfalsified"
resolution: "pickFalsifiedBy, and seriesOutcome with it, skips a cell-failed hand-back as handBackLane does, so a cell-failed trial lane leaves the pick unfalsified."
impact: internal
resolved_by:
  intent: "itd-2610161733316518"
  spec: "spc-2610161949559050"
---

Once the model-matrix trial step lands, a trial lane handed back as cell-failed is still counted as falsifying the run's pick: pickFalsifiedBy in record.go treats every hand-back with no kind as falsifying, while handBackLane leaves the pick's record line out for a cell-failed verdict, so the run record says Falsified true for a line it never wrote. Nothing starts a trial outside tests yet, so no run reaches it until the trial is wired.
