---
schema_version: 1
id: "iss-2611042320148804"
slug: "land-s-stacked-check-guards-only-a-fresh"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/land.go"
remedy: "Run the same check before a re-arm and in the build loop's land stage, stopping unarmed there as land does; a test re-arms a stacked branch through each path."
---

abcd land's stacked-branch check runs only before it makes a fresh arming: #1398's stackedOn sits in the arming branch of internal/core/landing/land.go, so it guards neither the 30-minute toggle that re-arms an arming land already made nor the build loop's own land stage (toggleArming in internal/core/implement/loop/queue.go), as the session that wrote it reported. iss-2611031719581061 holds the fresh-arming case.
