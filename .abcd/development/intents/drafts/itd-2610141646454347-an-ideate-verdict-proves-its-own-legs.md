---
id: itd-2610141646454347
slug: an-ideate-verdict-proves-its-own-legs
spec_id: null
kind: bundle-member
bundle: ideate-librarian
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
impact: breaking
origin: researcher-authored
production_mode: hand-written
---

# An ideate verdict proves its own legs

## Press Release

> **An ideate verdict proves every leg it records, not just the record grill.**
> Every record id in every leg resolves, and a primary source must be a web
> address, a filed source or a record. The grill says what it searched and
> where, so an empty grill can be told from a broken search. A reframing has its
> own field. A reviewer pair is recorded reviewer by reviewer, with a split
> settled by an advisor. A claim checked only as a quotation says so.
>
> "I read a verdict that killed an idea," said Bob, a staff engineer. "I could
> see which searches came back empty, that both reviewers agreed, and that the
> claim it leaned on was only checked as quoted. I trusted the kill because I
> could check it."

## Why This Matters

The lab of 2026-10-14 recorded, with exit 0, a verdict that cited a
non-existent intent in leg 3 and another in leg 1 and named "a remark heard in
passing" as a verified primary source (iss-2610141647304982). In the real run, a reviewer's
surviving attempt rested on a draft presented as shipped, caught only by an
advisor's premise check. The same lab found that a broken grill and an empty one
write the same record (iss-2610141647307807), that leg 2 cannot read or cite the private
verdict store (iss-2610141647309012), that leg 3's single evaluator conflicts with the
standing review-pair rule (iss-2610141647308358), that nothing defines how a pair merges or
where a reframing goes (iss-2610141647315880), and that "verified" cannot say "verified as
quoted" (iss-2610141647313018).

Decided in the product thinker's planning interview of 2026-10-14 (session abcd-66), which followed lab lab-261014142726-9c4d759, a hand-run of /abcd:ideate on one external write-up.

## Mechanism

We expect verdicts to become checkable because each weakness the lab found is a
missing field or a missing check in the record, and the binary already proves
the grill's citations: extending the same proof to every leg, and adding the
fields a reader needs, closes each gap without the binary judging anything.

## Scope Conditions

- A schema change (version 2). Verdicts already recorded under version 1 keep
  their shape and are read as they are; they are never backfilled.
- The binary checks shapes and citations; it forms no opinion of an idea.

## Acceptance Criteria

- **Given** a verdict citing a record id that does not resolve in any leg,
  **when** it is recorded, **then** it is refused, naming the leg and the id.
- **Given** a claim whose primary source is not a web address, a filed source key
  or a record id, **when** the verdict is recorded, **then** it is refused.
- **Given** a grill leg with no hits and no record of what was searched, **when**
  the verdict is recorded, **then** it is refused.
- **Given** a prior verdict in the private verdict store, **when** a grill cites
  it by its key, **then** the citation resolves.
- **Given** a reframed verdict, **when** it is recorded, **then** the reframing
  sits in its own field and a reframed verdict without one is refused.
- **Given** two reviewers, **when** the verdict is recorded, **then** each
  reviewer's attempts are kept apart, and a split pair carries the advisor's
  ruling.
- **Given** a claim checked only against its own source's wording, **when** it is
  recorded, **then** its status reads quoted, not verified.

## Open Questions

- The key a private verdict is cited by (technical facilitator, at planning).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
