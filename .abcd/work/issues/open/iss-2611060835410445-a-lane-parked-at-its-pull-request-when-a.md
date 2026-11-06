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
remedy: "Name such a lane in the later drive's first lines."
---

A lane parked at its pull request when a drive is killed, with no stop's mark on its run, gets no resume command and no line at the drive's end (`terminal.go:1813`), while a later drive still reads it through finishMerged. A person reading how the run ended cannot tell the lane exists or that another drive will touch it.
