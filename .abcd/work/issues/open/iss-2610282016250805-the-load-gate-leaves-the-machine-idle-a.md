---
schema_version: 1
id: "iss-2610282016250805"
slug: "the-load-gate-leaves-the-machine-idle-a"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "coordinator check abcd-f5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/machineload/queue.go"
remedy: "A preflight waiting at its race gate takes no first-gate slot, and MaxPreflightSlots rises so a large machine admits more test stages, keeping the load wait limit, the jitter and one race run at a time as the safety bounds."
---

The load gate leaves the machine idle: a make preflight queued at its race gate still counts as Busy at the first gate, so with load.preflights=2 one preflight running its race stage and a second waiting idle at the race gate fill both slots and every other preflight waits. Observed 2026-10-28 20:02-20:15Z: four preflights queued, one test stage running, load 13 on 16 cores. MaxPreflightSlots also caps the setting at 2.
