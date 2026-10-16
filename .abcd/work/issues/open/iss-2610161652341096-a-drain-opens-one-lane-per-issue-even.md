---
schema_version: 1
id: "iss-2610161652341096"
slug: "a-drain-opens-one-lane-per-issue-even"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "product thinker, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain.go"
remedy: "When the drain plans a move, group the eligible issues whose found_at paths overlap (same file, or a file and its directory) into one lane: one branch, one reservation per issue with a shared footprint, one review pair, one preflight and one pull request carrying a Resolves line per issue. Cap a bundle's size (for example five issues or one package) so one bad fix cannot stall many, and hand back an issue that fails its own fix round without failing the rest. Order bundles as single issues are ordered. Measure lanes and preflights per issue before and after."
---

A drain opens one lane per issue, even when several eligible issues touch the same files. Each lane pays its own branch, review, preflight and pull request, and the lanes then contend for the same footprint, so issues in one file are fixed one after another, each waiting on the last one's merge. The product thinker asked on 2026-10-16 that a drain bundle the issues that touch the same files and fix them in one go.
