---
schema_version: 1
id: "iss-2611101317363963"
slug: "an-automatic-reservation-renewal-can"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/renewdue.go"
remedy: "Never write a renewal whose end is earlier than the live tip's, and on a refused push refetch and report success when this copy's live reservation already covers the footprint and term asked for; a test races a hand renewal against a waiter's renewal and finds both exit 0, and a test renews with a shorter term and keeps the later end."
---

An automatic reservation renewal can move a lease's end earlier, and a renewal that loses a race reads as an error though the hold it wanted stands. On 7 Nov at 20:16 the load gate's automatic renewal set a reservation to end at 21:17:52Z, though a hand renewal at 19:41 had set it to end at 21:41:07Z. On 7 Nov at 11:12 a hand renewal and the run's own renewal from the same copy, 11 seconds apart, raced: the second push was refused as non-fast-forward and exited 2 with a raw git error. iss-2610270659564789 fixed the race between two load-gate waiters in one copy with a per-copy lock; a hand reserve does not take that lock.
