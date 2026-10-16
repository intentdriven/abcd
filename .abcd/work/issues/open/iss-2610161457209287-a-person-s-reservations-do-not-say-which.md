---
schema_version: 1
id: "iss-2610161457209287"
slug: "a-person-s-reservations-do-not-say-which"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "crash recovery on 2026-10-16 (abcd-22/abcd-a2)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim"
remedy: "Show each live hold's checkout (its worktree's display name) and, where the host gives one, the session name in abcd peers and in the cap refusal. Name a hold's copy as this checkout, another of yours here, or another machine, so the person can release the stale one."
---

A person's reservations do not say which of their sessions or checkouts holds them. Several sessions run under one account, and every hold reads holder REPPL with an opaque copy key. On 2026-10-16 two live intent holds (the messaging and cloud-sessions intents) sat at the ten-reservation cap while no session could tell whose they were. Two peers had to ask each other in turn before one was released.
