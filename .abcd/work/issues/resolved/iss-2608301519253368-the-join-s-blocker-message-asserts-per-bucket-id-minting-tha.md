---
schema_version: 1
id: "iss-2608301519253368"
slug: "the-join-s-blocker-message-asserts-per-bucket-id-minting-tha"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-189-round-3-security"
found_at: "internal/core/lint/schema.go"
resolution: "The blocker and the six comments that echoed it now state only the consequence the walk establishes: the reader keys the family on the (bucket, target) pair, so a record reaching across buckets is keyed on a pair nothing queries. The minting clause is gone from all seven sites rather than scoped to a family where it is equally false, which is how the previous spelling survived. The vacuous negative assertion the deletion would have created is re-anchored on the new wording."
impact: internal
resolved_by:
  intent: "itd-189"
---

the join's blocker message asserts per-bucket id minting that the same rule refuses as a fault and that the branch base already made false

Found by the round-3 security review. Must change before merge.

The bucket leg emits, as a BLOCKER, that reading ids "are minted per bucket and
collide across them, so the pair this record forms is what identifies it". They
are not. `mintUnusedItemID` (capture/reading.go:856-874) probes EVERY run
directory under the ledger lock and redraws on a hit — the fix recorded in
iss-2608300227228575, present in the branch base 932629f9.

The same rule contradicts itself 440 lines away: `record_schema`'s
duplicate-ordinal leg (schema.go:441-449) refuses a cross-run `rdi` collision
outright, and the reviewer fixtured one to confirm it fires. So the rule
simultaneously refuses the collision AND says it happens by construction.

Note iss-2608301411017768 (major) closed the PREVIOUS spelling of this defect by
SCOPING the claim to `rdi` — where it is equally false — rather than removing
it. That is the pattern to break here.

SEVEN sites carry it, not the four this record first named. The four found by
review: the message at schema.go:880, the doc comments at schema.go:210-216 and
:812-816, and readingoutstanding.go:57 ("Reading ids are minted per run and
collide"), which cites the very issue whose fix made it false.

The three more found by grepping the CLAIM rather than the line, and recorded
here so the fix that removes them is covered by a detector rather than widening
one silently: schema_test.go:1523 and :1588, and reading_outstanding_test.go:607
— all three test comments, none of them reaching an operator. They matter
because a test doc block is where the next author LEARNS the mechanism: leaving
them is how a corrected message gets re-broken in good faith, which is the route
by which this defect has now returned four times.

One of them carries a second fault. The negative assertion at schema_test.go:1610
is pinned to the substring "minted per bucket" — so deleting that phrase from the
message makes the assertion vacuous, passing forever against any message at all.
The sweep must re-anchor it, or closing this record silently disarms the guard on
the record next to it.

Remedy: state only what the walk establishes. `admittedProposals` keys the
admitted set on (directory run, proposal), so an admission filed under another
run is keyed on a pair nothing queries — that consequence is TRUE and
SUFFICIENT. Delete the minting clause from all four sites in one change, so one
rule holds one story.
