---
schema_version: 1
id: "iss-2610020718369838"
slug: "the-release-cut-s-dry-run-does-not-say-which-targeted"
severity: "minor"
category: "ux"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/ship.go"
remedy: "Compute launch.MissedTargets(cut.Targets, cut.NextTag) where the dry run's Cut is assembled, carry it on release.Cut as a moves list in the JSON, and have renderCut mark each targeted line the cut moves (for example 'targeted: itd-N targets v0.11.1, moves to next') while a target above the derived tag keeps its line unchanged; a refused cut derives no tag and marks nothing, as MissedTargets already returns nil there. It is a launch-surface change, so the brief launch chapter and commands/launch.md move with it in the same change; test both a moved and a standing target in one dry run."
---

The release cut's dry run does not say which targeted intents the cut will move to next. renderCut (internal/surface/cli/ship.go) prints one 'targeted:' line per targeted intent, and the line is the same for a target the cut leaves standing (a tag above the derived next tag) and one it rewrites to next (next itself, or a tag at or below the derived tag); launch.MissedTargets, which decides the move, has one caller, the cut's ingest (internal/core/release/ingest.go), so the move is first visible in the changelog section the cut writes. A reader of the dry run has to compare each target with the derived tag by hand. Not a breach: the intent's criterion 3 is worded to the cut and commands/launch.md matches the code (review-targetNext, MINOR 3).
