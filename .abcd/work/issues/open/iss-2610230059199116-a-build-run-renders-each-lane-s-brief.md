---
schema_version: 1
id: "iss-2610230059199116"
slug: "a-build-run-renders-each-lane-s-brief"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "lane D merge-gate build, lane 1 review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "Render a lane's brief from the run's own checkout when its record differs from the lane's base (naming both in the brief), or refuse to open a lane while the run's checkout holds uncommitted-to-main refinements of the spec or intent, naming them, so a refinement made during a run is never silently missing from the lanes after it."
---

A build run renders each lane's brief from the intent and spec as they stand at the lane's base on the default branch, so a refinement recorded during the run (a narrowed footprint, a new requirement or test line on a later step, a decision citing the spec) never reaches the run's own later lanes until it is merged to the default branch by some other change. On 2026-10-23 the merge gate's run (run-2610222359421510, itd-2610150602547271) recorded step-4 security requirements and step-wise footprint decisions on the orchestrating branch; lanes 2-6 would have been briefed without them, and a separate records pull request had to be cut to carry them to main ahead of lane 2.
