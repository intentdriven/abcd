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
resolution: "A drain retry of a failed hand-back release that succeeds now appends a reservation_released entry, carrying its clause, to the run record through the run's mutate frame, so the run no longer reads as lapsing and a later routing reuses the entry instead of pushing the release again."
impact: internal
resolved_by:
  commit: "4f7feb7e7401f53837d0a8408347743d767585bc"
---

releaseHandedBack: a drain retry of a failed hand-back release that succeeds is not written back to the run record, so the run's last reservation entry stays reservation_release_failed (saying the reservation lapses with its lease) although it was released, and any later releaseHandedBack over the run, finding no reservation_released entry, pushes the release again: idempotent, but it costs a fetch.

## Grounds

- pursued: after the drain's retry releases the reservation, the run's last reservation entry is reservation_released and a further releaseHandedBack makes no claim call (TestADrainRetryThatReleasesIsLoggedOnTheRun); a run record still ending in reservation_release_failed after a successful retry, or a second claim push, would show it wrong.
