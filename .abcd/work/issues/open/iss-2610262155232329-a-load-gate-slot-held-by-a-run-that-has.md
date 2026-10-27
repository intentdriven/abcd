---
schema_version: 1
id: "iss-2610262155232329"
slug: "a-load-gate-slot-held-by-a-run-that-has"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "planning interview of itd-2610262127138979"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement"
remedy: "Decide, as its own recorded trust decision, when a waiter may treat a live holder as idle (little CPU for a set time, measured from the run record of itd-2610262140595217), and then admit the next waiter without signalling or killing the holder; never by pattern, never across accounts by force."
---

A load-gate slot held by a run that has stopped using the machine is never released: a holder that hangs or idles keeps its slot until its own timeout, and every other waiter waits behind it (part of the remedy of iss-2610241414124940, split out of itd-2610262127138979 at its planning on 2026-10-26, because whole-run admission does not decide it). Admitting a run on the headroom it will need rather than the load at one instant (part of iss-2610241911052191) moves with it to the adaptive count, itd-2610262141001890.
