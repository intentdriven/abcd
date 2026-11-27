---
schema_version: 1
id: "iss-2611262132147121"
slug: "spot-check-outbound-reads-the-whole"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "spot-check PR-A fix round 2 (itd-2610301750162657)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/spotcheck/store.go"
remedy: "Bound outbound's read to the local records written since the last full carry (a checkpoint ref on the local copy, advanced commit by commit as origin is found to hold each), keeping the over-bound case a refusal and never a draw over a partial read."
deferred_after: v0.17.1
deferral_reason: "Capacity, not a fault that misleads: outbound refuses loudly over the bound and only on the checkout whose local copy is past it, about eight weeks away at the current rate, and the bounded read needs a design because a typed copy's name differs from the local file's (dcn-2611262132038983)."
---

The spot-check's outbound reader (internal/core/spotcheck/store.go, outbound) reads the whole local copy refs/abcd/spotcheck through read, which refuses a store holding more than 4,096 records. Every observation with an open window and every push of the store runs it, so once this checkout's local copy passes the bound, which it reaches in about eight weeks at about 60 records a day, nothing more is carried to origin and nothing is drawn from this checkout, with one line saying so at every landing and drain move. Writing records keeps working. The stop must stay a refusal: a fix bounds what outbound reads and must never turn the refusal into a draw over a partial read. Deferred with dcn-2611262132038983.

## Deferral 2026-11-26

Deferred past v0.17.1: Capacity, not a fault that misleads: outbound refuses loudly over the bound and only on the checkout whose local copy is past it, about eight weeks away at the current rate, and the bounded read needs a design because a typed copy's name differs from the local file's (dcn-2611262132038983).
