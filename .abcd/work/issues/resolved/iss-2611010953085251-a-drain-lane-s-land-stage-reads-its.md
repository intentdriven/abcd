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
remedy: "When the checkout's ledger lacks the record, have runFootprint read the issue's found_at from origin's default branch as last fetched (the ledger the drain judged it from), and keep a refusal of its own for a record origin no longer holds; never fall back to the run's live reservation, which the landing has narrowed and which reserveOnLeavingLand exists to widen back."
resolution: "When the checkout's ledger lacks an issue's record, runFootprint now reads the issue's found_at from origin's default branch as last fetched (lastFetchedOrigin, the ref the drain judged it from), with no network call, from whichever status folder holds it. A drain lane that leaves its landing for a fresh round (reserveOnLeavingLand) therefore widens its narrowed reservation back to the found_at instead of refusing. A record that origin no longer holds is refused on its own terms ('nor of origin/main at <sha> as last fetched'), and the run's live reservation is never read as a fall-back."
impact: fix
resolved_by:
  commit: "93954a74428645da04c45cda776611f421fb46dd"
---

A drain lane's land stage reads its issue's found_at from the ledger of the checkout the run lives in, while the drain judged and opened the issue from origin's default branch. When that checkout is behind origin and lacks the record, a landing that must sync with a moved main refuses at the reservation ('in no status folder of this checkout's ledger') after its review round has passed, because reserveOnLeavingLand asks runFootprint for the found_at to widen the narrowed reservation back to (internal/core/implement/loop/reservation.go). Attended, the only way past is to fast-forward the checkout; unattended, the stuck rule hands the lane back after three identical refusals or an hour at the stage and releases its reservation, discarding the validated work. Seen on 2026-11-01: run-2611010841189220 for iss-2610312044569253 stopped at landing after round 1 until the main checkout, 135 commits behind, was fast-forwarded.

## Grounds

- pursued: a drain lane at its landing, in a checkout whose ledger lacks the issue's record, goes back to a fresh round holding the found_at again when origin's default branch moves (TestADrainLaneLeavingItsLandingReadsItsFoundAtFromOrigin, which fails with the issue's own 'in no status folder of this checkout's ledger' refusal when origin's read is switched off). It would be shown wrong by that refusal returning, or by a footprint taken from the live reservation rather than from origin's record.
