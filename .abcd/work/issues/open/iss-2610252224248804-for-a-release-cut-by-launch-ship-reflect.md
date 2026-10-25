---
schema_version: 1
id: "iss-2610252224248804"
slug: "for-a-release-cut-by-launch-ship-reflect"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "gap audit of the owed fidelity reviews, 2026-10-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reflect/seed.go"
remedy: "Have the reflect seed read the targets the cut moved for that release when it lists the intents targeted at it that did not ship, with a test over a cut that moved one; or drop the warning and its write refusal from reflect."
---

For a release cut by launch ship, reflect's warning about intents targeted at the release that did not ship can never fire. The cut moves those targets to next before a tag exists (internal/core/release/ingest.go:455, PlanTargetMoves), and the reflect seed (internal/core/reflect/seed.go:173) reads only the intents' current target_release, not the cut's record of the moves (fidelity review of itd-24).
