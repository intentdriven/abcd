---
id: itd-165
slug: an-intent-that-fails-its-own-fidelity-audit-leaves-a-tracked
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
---

# A failed fidelity verdict becomes work somebody can see

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

When an audit judges an acceptance criterion not met, the ingest counts it into a rollup, renders it into the shipped intent's audit notes, and stops. No record is created, no exit code changes, and no gate reads it. An intent can ship with every criterion failed and the only consequence is prose nobody queries.

The verdict's consumer is the facilitator, which is a machine by default, so turning a failure into a tracked item is the loop's own bookkeeping rather than an automation somebody has to be persuaded to accept. Each not-met criterion becomes one ledger record stamped with the receipt identifier and deduplicated on it, so a re-ingest cannot mint a second record for the same failure, and the record identifier is written back into the audit notes so the edge is two-sided like every other link in the record.

An inconclusive verdict deliberately does not create a record. It means the auditor was under-fed, which is an input fault rather than a product defect, and minting records for it at volume would fill the ledger with noise and teach a reader to ignore it. It must still leave the receipt visibly outstanding, or a verdict that decided nothing becomes indistinguishable from one that passed.

What blocks depends on the facilitator's mode ([adr-2609151528057260](../../decisions/adrs/2609151528057260-three-roles-who-each-artefact-addresses-and-when-the-loop-st.md)): a machine facilitator works the queue, and an activated human facilitator is what the queue waits for. The ratchet that would make a new failure fail a gate is deliberately not part of this: it is sequenced behind a corpus of real verdicts, because there are none yet and a ratchet baselines whatever number it finds.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

**Decided 2026-08-29 (product thinker).** An unmet criterion goes on the facilitator's list. The product thinker hears about it only when the shortfall changes what they asked for.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
