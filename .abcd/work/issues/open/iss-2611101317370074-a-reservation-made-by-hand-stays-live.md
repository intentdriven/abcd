---
schema_version: 1
id: "iss-2611101317370074"
slug: "a-reservation-made-by-hand-stays-live"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/release.go"
remedy: "Have land, or the next peers scan, release a reservation whose branch's pull request has merged into the default branch, logging the release; a test merges a reserved branch's pull request and finds the reservation released at the next scan."
---

A reservation made by hand stays live after the pull request carrying its work merges, until someone releases it or it lapses. On 9 Nov a lane's hand reservation ran until 01:54Z on 10 Nov after its pull request merged at 20:25Z, holding peers off its files for more than five hours; it was noticed at 20:50.
