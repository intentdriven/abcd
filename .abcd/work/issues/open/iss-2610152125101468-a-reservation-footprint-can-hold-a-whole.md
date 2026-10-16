---
schema_version: 1
id: "iss-2610152125101468"
slug: "a-reservation-footprint-can-hold-a-whole"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "messaging reservation renewal, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Treat generated files (those a go:generate directive writes) as shareable in the overlap check, and warn rather than refuse when the overlap is between two reservations of the same holder"
related_intents: [itd-2610161016394967]
---

A reservation footprint can hold a whole directory (docs, hooks, internal/surface/cli), and the overlap check then refuses every other reservation naming any file inside it. Generated files that every verb-adding branch regenerates, such as docs/reference/cli/commands.md, sit inside such directories, so two unrelated intents held by the same person from two copies could not both declare an honest footprint: on 2026-10-15 the messaging reservation's renewal was refused twice and had to leave out .githooks/pre-push, the ruleset mirror and the generated CLI reference, which it does change.

Measured 2026-10-16 for the multi-account drain: 14 eligible issues name a directory as found_at; while one account holds internal/core/implement/loop, every other account is blocked from all 32 eligible issues under it, and in the lab the reverse held too (a directory issue could not be reserved while a file inside it was held).
