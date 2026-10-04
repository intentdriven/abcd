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
related_intents: [itd-2609091034175565]
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (lapsed-deferral triage, run A 2026-09-29): May an implement claim hold a cluster of records under one lane key, refused whole when any member is held elsewhere, or does one claim per record stand? implement claim (internal/core/implement/claim.go) holds one record at v0.11.1 and refuses the second session a second live claim."
remedy: "Waits on the claim-mode ruling: if clusters, implement claim takes several record ids under one --lane and, inside the run lock it already holds, checks every member and creates every claim file or none (removing those it made on a refusal), logging one claim naming all members, with the second session's cap counting lanes rather than records; if one claim per record stands, move the record to wontfix naming the ruling. Prove the cluster form with a test where one member held by another session refuses the whole claim and leaves no claim file behind."
---

An `abcd implement claim` holds one record, and a lane is often a cluster of records. In autonomous run A the second session's first lane fixed three issues and captured and fixed a fourth under one claim; the other records were unguarded against the first session for the lane's whole life, and only the two orchestrators' agreement kept a second lane off them. The second session cannot close the gap by claiming each record either, because it is refused a second live claim. Wanted: a cluster form of the claim, several record ids in one claim or a lane key the records are declared under, refused whole when any member is held elsewhere.

## Remedy grounds (2026-09-29)

- Why: acquiring every member at once removes hold-and-wait, and claim.go's withLock already serialises claimers, so no new lock or dependency is needed.
- Sources (consulted 2026-09-29): Coffman, Elphick and Shoshani, System Deadlocks, ACM Computing Surveys 3(2), 1971 (https://doi.org/10.1145/356586.356588), hold-and-wait as a necessary deadlock condition; etcd's Txn, a conjunction of per-key guards applied atomically (https://etcd.io/docs/v3.5/learning/api/).
- Rejected: claiming members one by one, which can leave a partial cluster held when a later member is refused.
