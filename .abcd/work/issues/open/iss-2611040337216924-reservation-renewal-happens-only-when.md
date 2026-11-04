---
schema_version: 1
id: "iss-2611040337216924"
slug: "reservation-renewal-happens-only-when"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "the v0.17.0 cut and the overnight lanes, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/renewruns.go"
remedy: "Have the load gate's wait renew the reservation of the run it waits for, as its own renewal reads the run already; a test holds a waiter past half a lease."
---

Reservation renewal happens only when something calls status or step, and a preflight outlasts half a lease on a busy day, so a waiting lane can lapse: a lane's reservation lapsed during a stop on 2026-11-03 and a peer's narrow file then blocked its renewal for 90 minutes.
