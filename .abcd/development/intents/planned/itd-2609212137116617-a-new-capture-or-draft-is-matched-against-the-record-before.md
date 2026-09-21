---
id: itd-2609212137116617
slug: a-new-capture-or-draft-is-matched-against-the-record-before
spec_id: spc-2609212141417782
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-84, itd-42]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-87, itd-48]
related_adrs: [adr-2609212115255771]
---

# A new capture or draft is matched against the record before it is written

## Press Release

> **A new issue or draft is matched against the record at filing; a likely double is linked, named, and never dropped.**
>
> "I filed the same finding twice a month apart and nobody noticed until a consistency pass," said a technical facilitator. "Now the capture tells me at filing that it looks like iss-N, writes the link, and leaves it to me to confirm. Nothing is refused: a wrong match is a link I remove, not a finding I lost."

## Why This Matters

Three places look for a match today, none at filing: the pre-pass at planning (itd-42), the drain before it captures, and itd-87's recurrence rule after closure. The itd-84 discipline names a capture-time validator as its next rung. The research of 2026-09-21 on unattended triage is plain about the failure to avoid: a matcher that refuses is the one that loses findings silently. Ruled: file it, link it, say so.

## Mechanism

We expect a typed link written at filing to make doubles visible where a later triage never finds them, because the moment of filing is the only moment both records are in one hand; shown wrong if the next consistency pass still finds unlinked doubles filed after this ships.

## Scope Conditions

- Holds for text long enough to compare; a one-line capture below the declared minimum is filed without matching and says so. <!-- cond: cond-2609212141415326 -->

## What's In Scope

- **The match**: `capture` and the quoted-text `intent` create compare the new text with every open and resolved issue and every intent's title and press release, by the term-overlap heuristic the embark ranking uses, declared a heuristic.
- **The link**: a likely match is written onto the new record as `duplicates:` (near-identical) or `refines:` (narrower), naming the candidate; the verb prints the match and the link; nothing is refused or dropped.
- **The confirmation**: a person or a later pass confirms or removes the link; a record whose link is removed is ordinary.
- **The configuration**: threshold and compared fields declared with a bundled default; below the threshold nothing is written and `--json` lists the near misses.
- **The rung**: the itd-84 discipline's capture-time validator is marked delivered by this record.

## What's Out of Scope

- Refusing or merging records.
- Matching across repositories.
- Semantic matching by a model (the heuristic is lexical; a host pass is a later rung).

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. File it, link it, say so; never refuse, never drop (ruled 2026-09-21).
2. A lexical heuristic, declared as one; the threshold is configuration.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** a new capture whose text overlaps an existing issue above the threshold, **when** it is filed, **then** the record is written with a `duplicates:` or `refines:` link naming the candidate, and the verb prints the match.
- **Given** a new draft intent whose text overlaps an existing intent's title or press release, **when** it is created, **then** the same link is written and printed.
- **Given** any match, **when** filing completes, **then** no record was refused or dropped; removing the link leaves an ordinary record.
- **Given** overlap below the threshold, **when** filing completes, **then** nothing is written for it and `--json` lists the near misses with their scores.
- **Given** the itd-84 discipline record, **when** it is read after this ships, **then** the capture-time validator rung reads delivered by this record.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the ledger holds 480 open issues and the run and the drain will file more unattended; we expect a link at filing to make doubles visible; shown wrong if the next consistency pass still finds unlinked doubles filed after this ships
