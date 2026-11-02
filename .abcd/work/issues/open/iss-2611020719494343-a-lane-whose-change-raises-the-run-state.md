---
schema_version: 1
id: "iss-2611020719494343"
slug: "a-lane-whose-change-raises-the-run-state"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "the runner-retry lane's landing preflight, 2026-11-02"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Have a renewal made from inside a lane's checkout save only the reservation fields, or save the run state at the schema it read rather than the binary's own, so a lane's preflight never raises the version of the run that drives it; and test that a lane built with a newer schema leaves its driver able to step."
---

A lane whose change raises the run state's schema rewrites the live run's state at the new version: its own preflight's load gate (implement load --wait, run through the lane's freshly built binary) renews the run's reservation and saves the run state with the lane binary's schema, after which the driver, still on the default branch's older abcd, refuses every step and receipt ('schema version 18, which a newer abcd wrote'). On 2026-11-02 the lane for iss-2611012047534449, which moves the schema from 17 to 18, did this three times to its own run (about 04:56, 05:57 and 07:15 UTC, the last in its landing preflight), and the version was set back by hand each time, no version-18 field being in use.
