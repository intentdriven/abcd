---
schema_version: 1
id: "iss-2608301649339636"
slug: "the-proposal-join-constrains-the-target-bucket-but-not-its-p"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-189-round-5-security"
found_at: "internal/core/lint/schema.go"
resolution: "The proposal join now declares the reading POSITION its target must hold, and checkRecordJoins refuses an admission naming an item at any other. ReadReadingOutstanding consults the admissions tree only inside if position == PositionWidening, so an admission naming a detection, an entailment or a comparative is keyed on a pair nothing ever queries; the finding says that and stops there rather than claiming the item goes on being reported unanswered, which depends on the dispositions this leg has not read. The leg stands down where the target's filename is not a bare handle, on the padding leg's grounds: what reads the family never opens such a file."
impact: fix
resolved_by:
  intent: "itd-189"
---

the proposal join constrains the target bucket but not its position so an admission naming a non widening item passes clean and admits nothing

Found by the round-5 security review. Branch-introduced, and the FIFTH
recurrence of this branch's standing class.

The `adm` store's `proposal` join constrains the target's FAMILY and its BUCKET
and says nothing about its POSITION. So a fully well-formed admission naming an
item at `position: "detection"` (or `entailment`, or `comparative`) resolves,
matches the bucket, spells correctly, and draws zero findings — while
`ReadReadingOutstanding` consults the admissions tree only behind
`if position == issueschema.PositionWidening`. The admitted pair is never
queried. The report goes on saying the item carries no disposition.

That consequence is verbatim the one iss-2608301327013320 and
iss-2608301519255871 were opened and closed for: it admits nothing, the
proposal it names goes on being reported as unadmitted, and no line says an
answer was written. The branch closed that class along the RUN axis (round 3)
and along the SPELLING axis (round 4) and left the POSITION axis, which is the
third coordinate of the same pair.

`AdmissionRequired`'s own doc already says `proposal` names the WIDENING item
admitted, so the schema states the constraint the join does not enforce.

Remedy: a leg refusing an admission whose target is not at
`PositionWidening` — or, if that is deferred, a record saying so, the way the
other two axes were handled rather than left silent.
