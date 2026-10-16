---
schema_version: 1
id: "iss-2610161015370582"
slug: "a-drain-s-hand-backs-and-judgements-live"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "multi-account drain investigation, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain.go"
remedy: "Publish the hand-back on origin (a tombstone carrying its reason that a fresh reserve refuses) and make an issue that names an intent draft ineligible (grounds: reasoned tombstones, as Cassandra and Kafka keep)"
related_intents: [itd-2610161016394967]
resolution: "Step 3 publishes a drain's hand-back on origin, as Abcd-Handed-Back and Abcd-Record trailers on the reservation's tombstone, and every account's drain passes over a hand-back that stands until a person changes the record; step 4 hands back an issue whose related_intents names an intent as promoted, so no drain judges or promotes it again"
impact: fix
resolved_by:
  commit: "7a23c958c6c14034a02a0fd419329ae89c707019"
---

A drain's hand-backs and judgements live only in the checkout that made them: routeHandBack promotes the issue to an intent draft in that checkout, uncommitted, then the release pushes a tombstone so the issue reads free everywhere, and eligibility ignores related_intents, so every other account's drain judges or works the same issue again and promotes it to its own draft id.

## Grounds

- pursued: no other account's drain judges, works or promotes an issue once one drain has handed it back or promoted it into an intent; a second account's drain reserving or asking a judgement over such an issue while its record is unchanged on origin's default branch would show it wrong
