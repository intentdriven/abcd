---
id: itd-2609020625402518
slug: a-reframe-occasioned-by-a-reading-is-recorded-as-a-reframe-j
spec_id: spc-2609020626048705
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-180, itd-189]
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# A reframe occasioned by a reading is recorded as a reframe, joined to what occasioned it, without carrying the construal it replaced

Typed links: `builds_on` [itd-180](../shipped/itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md) (record families in the issue tier), [itd-189](../shipped/itd-189-what-the-widening-reading-proposes-is-admitted-or-declined-o.md) (the surprise entry as its own act); `refines` [adr-55](../../decisions/adrs/0055-the-construal-stands-in-the-record-its-history-does-not.md) (a reframe record beside the construal, adopted as [adr-2609021016288378](../../decisions/adrs/2609021016288378-a-reframe-occasioned-by-a-reading-is-a-committed-pointer-to.md) in its three-surface form).

## Press Release

> **A change to the frame is a record, not a diff.** `abcd capture reframe --occasioned-by <rdi-N|dsp-N|srp-N> --grounds "<why>"` writes one frame-level revision record when the frame is rewritten because of a reading, naming what occasioned it, the content fingerprint of each of the frame's three surfaces before and after (the construal section, the committed glossary terms and the committed scope), which of them changed, and the grounds. The prior text of any surface passes to ledger content on the local side, as adr-55 requires, so the committed record shows that a reframe happened, when, why and where, without committing the framing it abandoned. `abcd <id>` on a reframe reports its occasion, the fingerprints and which surfaces moved.

> "When a detection sends me back to the frame rather than to the artefact, I need the record to say that is what happened," said an AI/agent researcher who keeps their own design record. "Otherwise a reframe looks like an edit to a paragraph, and the reading that caused it gets no credit and no blame."

## Why This Matters

The cold-reading design lists where an accepted detection may land: an intent, a discipline, an ADR, a brief passage, the construal section, and the frame. Every landing exists except the last, which the design schedules for Iteration 2 as a frame-level revision record, "without which a reframe occasioned by a reading cannot be recorded as a reframe".

[adr-55](../../decisions/adrs/0055-the-construal-stands-in-the-record-its-history-does-not.md) rules that the construal as it presently stands is committed record and that declined construals, superseded terms and the reasoning that settled a dispute stay on the local side and are read by nothing automated. The brief's framing chapter keeps no history in the section. A frame-level record therefore cannot carry the prior construal's text, and it need not: what is wanted is the join, from a detection to the reframe it occasioned and from a reframe to the reading that caused it, and the fact that the frame moved rather than the artefact.

Without it, Iteration 2's closing run cannot distinguish a tension that was answered by a reframe from one that was answered by a build, and the design's purpose-durability and convergence readings both depend on knowing which.

## Decisions flagged for the maintainer

Both were adopted by the maintainer on 2026-09-02 as [adr-2609021016288378](../../decisions/adrs/2609021016288378-a-reframe-occasioned-by-a-reading-is-a-committed-pointer-to.md), in the three-surface form, after the single-section draft was found to contradict the design's two landings and adr-55's enumeration of the framing's committed surfaces.

- **The frame is the framing as it presently stands, which adr-55 enumerates as three committed surfaces.** The narrowing to the construal section alone, which this intent's first draft flagged, is withdrawn. The record fingerprints the framing chapter's construal section, the committed glossary terms and the committed scope, before and after, and shows which of the three changed; a construal-only rewrite is one instance, not the definition.
- **A reframe record beside the construal refines adr-55.** adr-55 is silent on recording that a rewrite happened; this record adds a committed pointer to an event whose content stays local. Only a reframe a reading occasioned carries a record; a researcher's own rewrite of any surface stays as adr-55 leaves it, and a rule over every frame edit is adoptable later without changing the record's shape.

## What's In Scope

- **A reframe record family** in the working tier beside the other reading families, one record per reframe, minted on the timestamp seam, carrying: the occasion (a reading item, a disposition or a surprise), the content fingerprint of each of the three frame surfaces before and after, which of them changed, and the grounds. It carries no construal text, no term text and no scope text. It is warm and is excluded from every reading by positive inclusion, and the manifest asserts the exclusion.
- **`abcd capture reframe`** as the only writer, refusing an occasion that does not resolve, a ground below the substance floor, and a before-fingerprint it cannot find in the surfaces' history. The verb reads all three surfaces' committed content itself; the operator supplies no hash. The join to the occasion is operator-asserted, with one check: the occasion predates the rewrite.
- **Dispatch** on the reframe id, reporting the occasion, the fingerprints and which surfaces changed.
- The capture surface page documents the verb.

## What's Out of Scope

- The prior text of any surface. It passes to the local side under adr-55 and this record does not carry it.
- Reframes not occasioned by a reading, and any lint over frame edits. adr-2609021016288378 adds a pointer for the reading-occasioned case and no rule over every edit.
- The fourth audit verdict. A reframe changes what is built next; nothing here puts a delivered promise in question, which the design defers by decision.

## Mechanism

We expect a record keyed to the committed fingerprints of the three frame surfaces to make reframes countable without committing framing traces because the fingerprints identify which frame a reading saw and which replaced it, and all three surfaces are already committed content, while the reasoning between them is not. It fails if a rewrite and its record cannot be paired by fingerprint, which happens when a surface is rewritten twice between records; the verb refuses a before-fingerprint it cannot find in the surfaces' history, so the failure is loud.

## Scope Conditions

- The frame is three committed surfaces at fixed paths: the construal section of the framing chapter, the glossary directory's committed content, and the scope chapter of the brief. A repository whose frame lives elsewhere is outside this scope. <!-- cond: cond-2609020626047674 -->
- The verb reads the surfaces at HEAD. A reframe recorded before the rewrite is committed carries the pre-rewrite fingerprints as its before-fingerprints and is completed by a second write after the commit; the verb says which half it wrote. <!-- cond: cond-2609020626049770 -->

## Acceptance Criteria

- **Given** a reading item and any of the three frame surfaces rewritten and committed, **when** `capture reframe` runs with the item as occasion and a ground, **then** one reframe record exists carrying the occasion, a before-fingerprint of each surface matching its previously committed state, an after-fingerprint of each matching its current state, which surfaces changed, and the ground.
- **Given** a frame whose committed surfaces match no known prior state, **when** the verb runs, **then** it refuses and names the mismatch.
- **Given** an occasion that does not resolve to a reading item, a disposition or a surprise, **when** the verb runs, **then** it refuses.
- **Given** a repository holding reframe records, **when** any reading is assembled, **then** no reframe record reaches the bundle and the manifest asserts the family's exclusion.
- **Given** `abcd <reframe-id>`, **when** it runs, **then** it reports the occasion, the fingerprints and which surfaces changed.

## Prior Art

- [adr-55](../../decisions/adrs/0055-the-construal-stands-in-the-record-its-history-does-not.md); the brief's framing chapter; [itd-180](../shipped/itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md) and [itd-189](../shipped/itd-189-what-the-widening-reading-proposes-is-admitted-or-declined-o.md).
- The cold-reading rulings of 2026-08-28 in the decision log.

## Open Questions

None. The flagged decisions are adopted as adr-2609021016288378; the family's identifier prefix is the spec's to fix.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect a reframe record keyed to the committed fingerprints of the three frame surfaces, the construal section, the committed glossary terms and the committed scope, to make reframes countable and joinable to the reading that occasioned them without committing framing traces; a reframe the record cannot pair with its before and after fingerprints of the three surfaces, or a prior surface's text reaching the committed record, would show it wrong
