---
schema_version: 1
id: "iss-2610170948520608"
slug: "in-a-drain-a-lane-opens-on-the"
severity: "nitpick"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Push one renewal naming the lane's first stage as the drain opens a lane on its judgement hold, so peers see the stage from the lane's start."
---

In a drain, a lane opens on the reservation the drain took to ask its judgement and pushes nothing at its start, so the reservation names no lane stage, and peers see none for the lane until its first renewal, about half a lease later.
