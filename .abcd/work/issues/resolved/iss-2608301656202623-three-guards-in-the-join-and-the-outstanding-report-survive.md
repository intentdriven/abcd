---
schema_version: 1
id: "iss-2608301656202623"
slug: "three-guards-in-the-join-and-the-outstanding-report-survive"
severity: "nitpick"
category: "tech-debt"
source: "user-observation"
found_during: "itd-189-round-5-ruthless"
found_at: "internal/core/lint/schema.go"
resolution: "Each of the three stand-downs now has a test that fails when it is deleted, watched failing: spellsHandleOf's rest check is killed by carrying 'rdi-' into the spelling set (the finding becomes silence and the exact-count assertion drops from eight to seven); checkRecordBucketField's absence check is killed by a blank run drawing a second, weaker finding beside the required-property one; and the Unadmitted leg's wellFormed check is killed by one item being named both Unreadable and Unadmitted. admittedProposals' proposal check is JUDGED, not deleted: it is the set's definition rather than defensive scaffolding, because the set is the proposals the store admits and an admission naming nothing admits nothing. Its invisibility rests on readingItemFileRe at the CALLER, one call site away, not on anything this function does, so it is pinned on the set itself and the reasoning is carried beside it."
impact: internal
resolved_by:
  intent: "itd-189"
---

three guards in the join and the outstanding report survive their own deletion and one equality check is inert by construction

Found by the round-5 ruthless review, reported as notes rather than findings.
Recorded so they are not rediscovered as new.

Three guards survive their own deletion against the full lint and capture
suite:

- `spellsHandleOf`'s `rest == ""`: deleting it turns the `proposal: rdi-`
  finding into silence.
- `checkRecordBucketField`'s `isAbsentValue(got)`: deleting it adds a second,
  weaker finding on `run: ""`.
- the Unadmitted leg's `answer.standing.wellFormed`: deleting it reports one
  item as both `Unreadable` and `Unadmitted`.

Separately, `admittedProposals`' `proposal == ""` check is INERT rather than
defensive: `readingItemFileRe` guarantees the queried item is never empty, so
the map lookup it guards cannot be reached. That is the same class round 4 swept
in `31107a30`, with one instance missed.

The reviewer also confirmed three sets of negative assertions are pinned to
strings that occur nowhere in production code and so pass unconditionally. It
did NOT report them as defects, and neither does this record: each sits beside a
positive assertion on the current wording and each commit states the
anti-regression intent. They are deliberate reintroduction pins. Noted only so a
later reader does not mistake them for live coverage.
