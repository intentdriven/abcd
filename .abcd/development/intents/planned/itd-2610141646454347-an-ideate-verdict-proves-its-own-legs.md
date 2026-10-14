---
id: itd-2610141646454347
slug: an-ideate-verdict-proves-its-own-legs
spec_id: spc-2610142138047122
kind: standalone
bundle: null
suggested_kind: null
reclassification_history:
  - { date: 2026-10-14, from: bundle-member, to: standalone, reason: "The product thinker ruled in the planning interview that this fix ships as soon as it is ready, outside bundle ideate-librarian: it needs nothing from the librarian, and its major issue blocks every release cut while it is open." }
builds_on: []
related_issues: [iss-2610141647304982, iss-2610141647307807, iss-2610141647308358, iss-2610141647315880, iss-2610141647313018]
severity: major
impact: breaking
origin: researcher-authored
production_mode: hand-written
target_release: next
---

# An ideate verdict proves its own legs

## Press Release

> **An ideate verdict proves every leg it records, not just the record grill.**
> Every record id in every leg resolves, and a primary source must be a web
> address, a filed source or a record. The grill says what it searched and
> where, so an empty grill can be told from a broken search. A reframing has its
> own field. Two reviewers are recorded reviewer by reviewer, and a split
> between them carries the ruling that settled it. A claim checked only against
> the idea's own source reads "quoted", never "verified".
>
> "I read a verdict that killed an idea," said Bob, a staff engineer. "I could
> see which searches came back empty, that both reviewers agreed, and that the
> claim it leaned on was only checked as quoted. I trusted the kill because I
> could check it."

## Why This Matters

`abcd ideate record` checks citations only in the grill leg. A lab hand-run of
2026-10-14 recorded, with exit 0, a verdict that cited a non-existent intent in
leg 3 and another in leg 1 and named "a remark heard in passing" as a verified
primary source (iss-2610141647304982). The same lab found that a broken grill
and an empty one write the same record (iss-2610141647307807); that two
reviewers and a disagreement between them have no place in the record
(iss-2610141647308358, iss-2610141647315880); and that "verified" cannot say
"verified as quoted" (iss-2610141647313018). Each is a missing field or a
missing check, and the binary already checks the grill leg's ids, so the same
kind of check extends to the rest.

Breaking, for anything that records a verdict: a payload with an unresolved id
in any leg, a free-text primary source, a reframed verdict without its
reframing, or schema version 1 is refused where it was accepted.

## What's In Scope

- Schema version 2 of the ideate verdict, its validation, and its rendering.
- The command page `commands/ideate.md` updated to the new shape in the same
  change.

## What's Out of Scope

- Private verdicts and citing them (itd-2610141646456128).
- Ideate starting from a source, and citing pack items (itd-2610141646452213).
- Judging any claim: the binary checks shapes and citations, never the truth of
  a claim.

## Mechanism

We expect no verdict recorded after this change to carry an unresolved id, an
unlabelled self-quotation or an unrecorded split, because each is refused at
record time by a check on the payload's shape and citations. Shown wrong if a
version 2 verdict on record carries an id that does not resolve, or a claim
marked verified whose only primary source is one of the idea's own origin
sources.

## Scope Conditions

- The writer accepts version 2 only; a version 1 payload is refused with a <!-- cond: cond-2610142138048878 -->
  message naming version 2. Verdicts already on record are rendered Markdown
  and are not parsed or backfilled.
- The command page and the binary ship together; a host reading an older page <!-- cond: cond-2610142138047607 -->
  gets the version-1 refusal message.
- The binary checks shapes and citations. Whether a claim is true, or how a <!-- cond: cond-2610142138040936 -->
  reviewer reached a verdict, is the host's and the person's judgement.

## Acceptance Criteria

- **Given** a verdict citing a record id that does not resolve, in any leg,
  **when** it is recorded, **then** it is refused, naming the leg and the id.
- **Given** a claim whose primary source is not a web address, a filed source
  key or a record id, **when** the verdict is recorded, **then** it is refused.
- **Given** a grill leg with no hits and no search record (the terms searched,
  the places searched, and the count each returned), **when** the verdict is
  recorded, **then** it is refused.
- **Given** a verdict marked reframed with no reframing field, **when** it is
  recorded, **then** it is refused.
- **Given** two reviewers, **when** the verdict is recorded, **then** each
  reviewer's kill attempts, verdict and consulted sources are kept under that
  reviewer.
- **Given** two reviewers whose verdicts differ, **when** the verdict is
  recorded without a ruling (who ruled, and the ruling), **then** it is
  refused.
- **Given** a claim marked verified whose only primary source is listed among
  the verdict's origin sources, **when** it is recorded, **then** it is
  refused; marked quoted, it is accepted.
- **Given** a version 1 payload, **when** it is recorded, **then** it is
  refused with a message naming version 2.
- **Given** the updated command page, **when** it is read, **then** it tells the
  runner to give each reviewer one idea per context.

## Prior Art

- itd-104 (shipped): the ideate admission gauntlet and `ideate record`, whose
  grill-leg citation check this extends to every leg.
- itd-1 (discipline): every audit verifies acceptance criteria; the verdict's
  own legs get the same treatment.
- iss-2608230748418054 (open): the grill's structural blind spot over the
  research notes, which the search record makes visible.

## SOTA

- Structured-output validation of model-produced records (JSON Schema
  validators, Pydantic-style models): mature; validates shape, not truth.
- Citation resolution in research tooling (DOI and URL resolvers): mature for
  external references; abcd's record ids need abcd's own resolver.

**Path:** 2. A native floor: abcd's own resolver and schema check, since the ids
are abcd's own; an external validator adds nothing the Go check lacks.

## Open Questions

_None._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: an ideate verdict on record today can cite things that do not exist and still pass; we expect that after this change no verdict on record carries an unresolved citation or an unlabelled self-quotation, and the fix also unblocks release cuts, which refuse while its serious issue is open. Shown wrong if a verdict recorded after it still carries one.
