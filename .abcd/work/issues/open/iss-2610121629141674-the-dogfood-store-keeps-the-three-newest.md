---
schema_version: 1
id: "iss-2610121629141674"
slug: "the-dogfood-store-keeps-the-three-newest"
severity: "nitpick"
category: "bug"
source: "impl-review"
found_during: "re-review of the dogfood store fix 4c0555851, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/dogfood/dogfood.go"
remedy: "never prune a build that a verified-tip record in the store still names, and keep keepBuilds as the cap on unnamed builds only"
---

The dogfood store keeps the three newest builds per repository (keepBuilds, internal/core/dogfood/dogfood.go), and since 4c0555851 each declaration keeps its own verified tip. Four or more declarations of one repository at four distinct tips therefore never settle: each build's prune evicts another clone's build, whose next Resolve finds no ready binary and asks for a rebuild, which evicts the next. Two or three declarations are fine.
