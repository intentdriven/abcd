---
schema_version: 1
id: "iss-2610150616434848"
slug: "releasehandedback-a-drain-retry-of-a"
severity: "nitpick"
category: "tech-debt"
source: "impl-review"
found_during: "final fix round of spc-2610031753359849 steps 12, 13 and 15 (feat/reserve-build)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Record the drain's successful release on the run, as a reservation_released entry in its record."
---

releaseHandedBack: a drain retry of a failed hand-back release that succeeds is not written back to the run record, so the run's last reservation entry stays reservation_release_failed (saying the reservation lapses with its lease) although it was released, and any later releaseHandedBack over the run, finding no reservation_released entry, pushes the release again: idempotent, but it costs a fetch.
