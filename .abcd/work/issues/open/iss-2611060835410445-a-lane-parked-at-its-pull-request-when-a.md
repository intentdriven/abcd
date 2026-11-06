---
schema_version: 1
id: "iss-2611060835410445"
slug: "a-lane-parked-at-its-pull-request-when-a"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/terminal.go"
remedy: "Have the killed drive's end say what moves such a lane on, and name the lane in a later drive's first lines."
---

A lane parked at its pull request when a drive is killed, with no stop's mark on its run, gets no resume command: the drive's end names the lane and its run as left on its branch (`internal/surface/cli/drain_drive.go:599`) and stops there, because a later drive takes up no parked lane (`terminal.go:1806`). A later drive still reads that lane through finishMerged on its first move. It says so when the lane has finished, and otherwise only in the line of a wait, if the drive waits. A person reading how the run ended is not told what moves the lane on, or that another drive will touch it.
