---
schema_version: 1
id: "iss-2609252047459767"
slug: "abcd-build-itd-n-run-again-while-its-run"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "Start looks up a live run for the key under the lock first and resumes it; the checks run only when a run is created. TestStartAgainResumesTheRunItsOwnLaneChanged plays a lane worktree that ships the intent and the lane's own claim."
impact: fix
resolved_by:
  commit: "aac2da42"
---

abcd build <itd-N> run again while its run is in progress re-runs the pre-start checks before looking up the live run (loop.Start, internal/core/implement/loop/loop.go), so once the run's own lane changes the tree the resume the verb documents is refused: a worktree or branch that moves the intent to shipped/ (the Delivers-trailer definition of done), or the lane's own claim in the shared run store, makes the peers check exit 3 'held by a peer', and after a land the ready row refuses. build.go's Long text, 31-build.md and commands/build.md all promise that starting again resumes. Remedy: under the lock, look up a live run for the key first and return it resumed; run the checks only when creating.

## Grounds

- pursued: we expect build run again during a run to resume whatever its lanes did to the tree; shown wrong if any lane-made change (worktree, branch, claim) still refuses the resume
