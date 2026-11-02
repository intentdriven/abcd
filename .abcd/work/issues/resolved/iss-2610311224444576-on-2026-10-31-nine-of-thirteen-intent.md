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
remedy: "Add abcd implement widen, which an issue lane's implementer runs before its first edit of a path beyond the reservation, reserving the issue again with the held footprint and that path, judged for overlap, and backing off (exit 3, nothing pushed) a live hold another copy has there; refuse an implementer's receipt while the lane's branch changes a path the reservation does not cover; keep the paths a lane widened to when it leaves its landing (dcn-2610311544325397). The hand-reservation half is iss-2610311539091723, under dcn-2610311539024848."
resolution: "an issue lane widens its hold with abcd implement widen before it edits a path beyond the found_at, backing off another copy's live hold with exit 3, and a receipt is refused while the branch changes a path the hold does not cover"
impact: additive
---

On 2026-10-31 nine of thirteen intent builds the product thinker chose were refused at the reservation check by one hold: iss-2610252214522207, whose found_at is the single file internal/core/surface/examples.go, holds a live reservation on origin until 2026-10-31T18:42:23Z whose footprint names dozens of paths (brief chapters 01-product, 02-constraints and 04-surfaces, commands/*.md among them) and which carries no run stage, unlike the build lanes beside it. A hold many times wider than its record's own scope, kept for hours with no lane showing progress, starves every autonomous build that touches the brief or the command pages, and nothing flags it: abcd peers lists it like any other live hold.

## Grounds

- pursued: a lane needing a file beyond its found_at reserves it before the edit and meets a clash first; shown wrong if a lane's branch reaches its landing changing a path its reservation never held
