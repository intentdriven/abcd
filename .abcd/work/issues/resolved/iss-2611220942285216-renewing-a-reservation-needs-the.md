---
schema_version: 1
id: "iss-2611220942285216"
slug: "renewing-a-reservation-needs-the"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "autonomous fix run 2026-11-21/22"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim"
remedy: "Add abcd reserve --renew <id>, runnable from any worktree of the checkout, that reuses the footprint the reservation already records; and have land keep the reservations its pull request resolves renewed while that pull request is open or queued, releasing them when it merges."
resolution: "abcd reserve --renew renews your own reservation with the footprint it records from any worktree of the checkout. The remedy's second half is served by abcd implement heartbeat (iss-2611220942554289) rather than by land, and only while heartbeats run: each one renews, half through its lease, a reservation of yours held from a worktree of this checkout whose branch has a pull request the heartbeat finds open, armed or queued, and whose latest reservation commit that worktree pushed. A reservation with no such pull request behind it is reported and not renewed, so it lapses on its lease. Nothing releases the reservations when the pull request merges beyond the derived release a record finished on the default branch already has."
impact: additive
---

Renewing a reservation needs the footprint given again and must run from the worktree that took it. An autonomous run renewed one by hand every half hour for twelve hours, and was refused twice: once for a missing --footprint and once for running from another worktree of the same checkout.
