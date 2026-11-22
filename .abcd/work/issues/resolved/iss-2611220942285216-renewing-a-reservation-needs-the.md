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
resolution: "abcd reserve --renew renews your own reservation with the footprint it records from any worktree of the checkout; the remedy's second half, keeping a queued pull request's reservations renewed, is served by abcd implement heartbeat (iss-2611220942554289) rather than by land"
impact: additive
---

Renewing a reservation needs the footprint given again and must run from the worktree that took it. An autonomous run renewed one by hand every half hour for twelve hours, and was refused twice: once for a missing --footprint and once for running from another worktree of the same checkout.
