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
remedy: "have the build loop record the receipt from a pair that read the pushed head, with the landing's records commit taking a review round of its own before anything is pushed"
resolution: "a lane's records commit takes a short review round of its own where the mode is auto, and the pair that read the pushed head is the receipt the merge gate reads, so a lane that passed arms itself and one that did not parks naming the finding"
impact: fix
---

A drive's lane parks at its pull request where facilitator.mode is auto: the merge gate arms only a head a review receipt names, and only the land verb records one, so an unattended drive pushes every lane and leaves it open, holding its reservation until the reservation cap stops the run.

## Grounds

- pursued: a pair that read the pushed head is enough for an unattended lane to arm; shown wrong if a lane arms on a head no pair read
