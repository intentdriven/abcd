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
---

A drain's hand-backs and judgements live only in the checkout that made them: routeHandBack promotes the issue to an intent draft in that checkout, uncommitted, then the release pushes a tombstone so the issue reads free everywhere, and eligibility ignores related_intents, so every other account's drain judges or works the same issue again and promotes it to its own draft id.
