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
resolution: "abcd peers names each reservation's copy: this copy, the other worktree of this repository on this machine that made it (its directory's name, read from the copy key kept in its local tier, never the key), or elsewhere; the cap refusal groups the held records the same way. The host gives abcd no session name, so the worktree stands for the session."
impact: additive
---

A person's reservations do not say which of their sessions or checkouts holds them. Several sessions run under one account, and every hold reads holder <the product thinker's login> with an opaque copy key. On 2026-10-16 two live intent holds (the messaging and cloud-sessions intents) sat at the ten-reservation cap while no session could tell whose they were. Two peers had to ask each other in turn before one was released.

## Grounds

- pursued: a person with several sessions on one account can tell which hold is whose and release the stale one; shown wrong if a sibling worktree's hold reads as another machine's
