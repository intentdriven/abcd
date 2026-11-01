---
schema_version: 1
id: "iss-2611010953085251"
slug: "a-drain-lane-s-land-stage-reads-its"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "abcd drain, landing run-2611010841189220"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Have runFootprint fall back to the footprint the run recorded when it reserved the issue (the drain's judging hold, or the reservation the run already holds) when the checkout's ledger lacks the record, and refuse only when neither names one."
---

A drain lane's land stage reads its issue's found_at from the ledger of the checkout the run lives in, while the drain judged and opened the issue from origin's default branch. When that checkout is behind origin and lacks the record, the landing refuses at the reservation ('in no status folder of this checkout's ledger') after both review rounds have passed, and the only way past is to fast-forward the checkout (internal/core/implement/loop/reservation.go, runFootprint). Seen on 2026-11-01: run-2611010841189220 for iss-2610312044569253 stopped at landing until the main checkout, 135 commits behind, was fast-forwarded.
