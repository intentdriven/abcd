---
schema_version: 1
id: "iss-2609231156260287"
slug: "abcd-capture-writes-a-record-with-an-empty-found-at-silently"
severity: "minor"
category: "ux"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/validate.go"
resolution: "A capture without --found-at is written as before and says on stderr that the record names no location in this checkout, with no_location in --json; a capture naming a location prints no such line."
impact: additive
resolved_by:
  commit: "a60cc4c5"
---

abcd capture writes a record with an empty found_at silently, and that is the shape all nine misfiled installer records of iss-2609120511058115 had: the guard that closed that record refuses a path-shaped found_at absent from the checkout, which none of the nine carried, so the batch that motivated it would still file today. A capture without found_at is legitimate (a conceptual finding, a process observation) and must stay legitimate, so the remedy the record floated is a nudge rather than a refusal: when --found-at is absent, the verb says the record names no location in this checkout and so nothing ties it to the repository it is filed into (on stderr, and as a field in --json), without changing the exit code or what is written. Acceptance: given a capture with no --found-at, when the verb runs, then the record is written as today and the output says no location was named; given one with a found_at, no such line appears.

## Grounds

- pursued: a finding filed with no location is surfaced to its author at write time; a location-less capture that prints nothing would show it wrong
