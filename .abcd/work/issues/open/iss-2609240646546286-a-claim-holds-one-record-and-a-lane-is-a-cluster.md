---
schema_version: 1
id: "iss-2609240646546286"
slug: "a-claim-holds-one-record-and-a-lane-is-a-cluster"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/claim.go"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (lapsed-deferral triage, run A 2026-09-29): May an implement claim hold a cluster of records under one lane key, refused whole when any member is held elsewhere, or does one claim per record stand? implement claim (internal/core/implement/claim.go) holds one record at v0.11.1 and refuses the second session a second live claim."
---

An `abcd implement claim` holds one record, and a lane is often a cluster of records. In autonomous run A the second session's first lane fixed three issues and captured and fixed a fourth under one claim; the other records were unguarded against the first session for the lane's whole life, and only the two orchestrators' agreement kept a second lane off them. The second session cannot close the gap by claiming each record either, because it is refused a second live claim. Wanted: a cluster form of the claim, several record ids in one claim or a lane key the records are declared under, refused whole when any member is held elsewhere.
