---
schema_version: 1
id: "iss-2608301411017768"
slug: "the-samebucket-join-message-asserts-a-per-bucket-id-mechanis"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "itd-189-round-3-ruthless"
found_at: "internal/core/lint/schema.go"
resolution: "The join's bucket obligation now names the FAMILY it holds for (sameBucketAs) rather than being a bare flag, and the comparison runs only against a target of that family. An admission naming a record of any other family returns to the silence it had before the leg existed, so the per-run-minting message is emitted only where that mechanism is real. The empty-bucket stand-downs go with it: a record outside every bucket is reported by the walk and never indexed, so it reaches the join as a target that is not in the corpus."
impact: fix
resolved_by:
  intent: "itd-189"
  spec: "spc-67"
---

the sameBucket join message asserts a per-bucket id mechanism that exists only for reading items, so a cross-family target gets a confidently false diagnosis

Found by the round-3 adversarial ruthless review of build/itd-189.
INTRODUCED BY b977b77a, which added the leg.

`checkRecordJoins`' sameBucket leg compares buckets without constraining the
target's FAMILY, and its message names a mechanism that belongs to one family
only. With the live store configuration an admission carrying
`proposal: iss-42`, where `iss-42` sits in `open/`, resolves through the index,
compares `open` against `rdg-1`, and draws a blocker reading "ids in that family
are minted per bucket and collide across them ... the issue it names goes on
being reported as unanswered". Both halves are false of the issue ledger: issue
ids are timestamp-minted globally and survive a move between status directories,
and the outstanding report walks reading items only and never reports an issue
at all.

The operator is sent to look for an id collision that cannot exist, when the
actual defect is that `proposal` must name a reading item. Before the leg landed
the case was silent, so the false statement is new — and it is the same class as
iss-2608301308369559: a gate stating a consequence it has not established.

Remedy: give `recordJoin` a target-family field, set it to the reading-item
family on the admission's `proposal` join, and run the bucket comparison only
when the target is of that family. That returns the cross-family case to the
silence it had, adding no new claim. Reporting a cross-family target on its own
terms is a separate and larger decision.
