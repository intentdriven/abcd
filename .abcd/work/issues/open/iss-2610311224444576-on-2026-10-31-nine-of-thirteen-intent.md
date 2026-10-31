---
schema_version: 1
id: "iss-2610311224444576"
slug: "on-2026-10-31-nine-of-thirteen-intent"
severity: "major"
category: "process"
source: "agent-observation"
found_during: "selecting intents to build from the board"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Per dcn-2610311251002713: an issue lane (build <iss-N>, drain) reserves its found_at alone and widens the footprint as the lane changes further paths; a hand reserve keeps the footprint it names."
---

On 2026-10-31 nine of thirteen intent builds the product thinker chose were refused at the reservation check by one hold: iss-2610252214522207, whose found_at is the single file internal/core/surface/examples.go, holds a live reservation on origin until 2026-10-31T18:42:23Z whose footprint names dozens of paths (brief chapters 01-product, 02-constraints and 04-surfaces, commands/*.md among them) and which carries no run stage, unlike the build lanes beside it. A hold many times wider than its record's own scope, kept for hours with no lane showing progress, starves every autonomous build that touches the brief or the command pages, and nothing flags it: abcd peers lists it like any other live hold.
