---
schema_version: 1
id: "iss-2611051821551531"
slug: "a-drive-s-lane-parks-at-its-pull-request"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "reading how a drive ends"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "have the build loop record the receipt itself from the passing pair of a lane's latest round, in process, with the pushed head resting on that pair while it adds only the landing's records"
resolution: "the build loop records its own review receipt from a lane's latest passing round, so a lane that passed arms itself in auto"
impact: fix
---

A drive's lane parks at its pull request where facilitator.mode is auto: the merge gate arms only a head a review receipt names, and only the land verb records one, so an unattended drive pushes every lane and leaves it open, holding its reservation until the reservation cap stops the run.

## Grounds

- pursued: a passing round is enough for an unattended lane to arm; shown wrong if a lane with no passing pair ever arms
