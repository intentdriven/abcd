---
id: itd-2609091034175565
slug: nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-2609020716570699
origin: extracted-from-record
production_mode: hand-written
---

# Nothing tells an agent that a record it is about to fix has been claimed or resolved by another session until the resolution gate refuses the push. In one night a peer session re-fixed two issues a paused branch also fixed, and two of its open PRs duplicate merged work. The claim signal that worked in every published multi-agent run is the repository itself: a claim written into the open record (claimed_by: account and harness, branch) and pushed alone through the queue before any fix starts, so a losing race is a push rejection; plus a duplicate-guard required check that fails a PR whose Resolves trailer names a record already resolved on origin/main; plus capture resolve refusing a record that is already terminal on the fetched origin/main. Folder membership is already the status signal, so the claim extends the one canonical primitive rather than adding a lock file that rots. Refines iss-2608220750029993.

## Press Release

> _Seeded by promotion from iss-2609020716570699. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2609020716570699`: Nothing tells an agent that a record it is about to fix has been claimed or resolved by another session until the resolution gate refuses the push. In one night a peer session re-fixed two issues a paused branch also fixed, and two of its open PRs duplicate merged work. The claim signal that worked in every published multi-agent run is the repository itself: a claim written into the open record (claimed_by: account and harness, branch) and pushed alone through the queue before any fix starts, so a losing race is a push rejection; plus a duplicate-guard required check that fails a PR whose Resolves trailer names a record already resolved on origin/main; plus capture resolve refusing a record that is already terminal on the fetched origin/main. Folder membership is already the status signal, so the claim extends the one canonical primitive rather than adding a lock file that rots. Refines iss-2608220750029993.. Read that issue record for the source observation.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
