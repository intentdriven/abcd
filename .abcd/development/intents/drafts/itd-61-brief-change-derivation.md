---
id: itd-61
slug: brief-change-derivation
spec_id: null
kind: null
suggested_kind: standalone
reclassification_history: []
related_adrs: []
prd_path: null
grill_session_id: 61d0f1de-0002-4a61-9c0d-000000000061
glossary_terms_used:
- core/brief
- core/intent
grilled_intent_hash: 11a0d2ede357a7c853ae7703b194b54431167e3d634b52426e09ed0db8088e9c
prd_grandfathered: false
builds_on: [itd-60]
severity: minor
---

# A Human Edit To The Brief Cannot Commit Until Its Implied Intents And Principles Are Drawn Out And Reconciled

## Press Release

> **abcd gains a brief-change derivation: when a human edits the brief directly, a grill-style pass draws out what that edit implies — new or changed intents, and new or changed cross-cutting principles (disciplines) — and the edit is blocked until each drawn-out item is accepted or dismissed.** abcd already carries built reality forward into the brief and docs ([[itd-60-doc-fidelity-anti-drift]]). This is the reverse: a person changes the brief — adds a capability, tightens a rule, shifts the mental map — and the framework refuses to let the roadmap silently lag behind. It interrogates the edit the way the grill interrogates an intent, surfaces the implied `itd-N` intents and the implied disciplines, and reconciles each. Any change fires the pass; if an implied item is already captured (because the forward path authored it), the dedup recognizes it and no-ops.

> "I edit the brief when my thinking moves — that's the whole point of it being a living canvas," said Iris, a product thinker steering abcd. "But an edit to the brief is a promise about the product, and promises imply work. If I write 'every surface must survive a restart' into the brief, that's a new discipline, and I shouldn't be able to walk away pretending it isn't. Make me reckon with what I just committed to — draw it out, let me accept or drop it — before the edit lands."

## Why This Matters

The brief is abcd's single source of truth and a living canvas a human edits directly. That power is also a gap: a brief edit can introduce a capability or a cross-cutting rule that never becomes an intent or a discipline, so the roadmap silently falls behind the stated design. The forward direction ([[itd-60-doc-fidelity-anti-drift]]) keeps the brief in step with *built* reality; it cannot catch a brief that runs *ahead* of reality by human authorship. Only a reverse pass can.

abcd already owns the right instrument: the grill — a Socratic interview that draws out what a stated intention actually entails. Pointing that instrument at a brief *edit* (rather than a fresh intent) turns an ungoverned edit into a reconciled one: the implied intents and the implied principles are made explicit and either accepted into the roadmap or consciously dismissed. Blocking until reconciliation mirrors abcd's existing tamper-evident, fail-closed posture — a brief edit is consequential enough to deserve the same "you cannot proceed past an unresolved state" treatment that provenance and the reviewers already enforce.

## What's In Scope

- A derivation pass triggered by **any** change to the brief: it analyses the edit (grill-style) and draws out the changes it implies.
- Two output kinds: **new or updated intents** (`itd-N` drafts — when the edit implies a capability or changes an existing one) and **new or updated principles** (disciplines — when the edit implies a cross-cutting rule or invariant).
- **Blocking-until-reconciled** semantics: the brief edit cannot be committed until each drawn-out item has been explicitly accepted (captured into the roadmap) or dismissed.
- **Dedup against the forward path:** because any brief change fires the pass — including brief edits the forward doc-fidelity pass itself auto-drafted ([[itd-60-doc-fidelity-anti-drift]]) — the derivation recognizes an already-captured implication and no-ops rather than producing a duplicate intent/principle.
- Reuse of the existing grill's interrogation machinery rather than a parallel re-implementation.

## What's Out of Scope

- **The forward direction.** Carrying built reality into the brief and the public docs is [[itd-60-doc-fidelity-anti-drift]]. This intent only handles human-authored brief edits flowing back into the roadmap.
- **Editing the brief's prose for the human.** The pass draws out implications and reconciles them; it does not rewrite the human's brief edit.
- **Auto-promoting drawn-out intents past draft.** Implied intents land as drafts for the normal lifecycle; the pass does not skip grilling/planning.
- **Sequencing/dependency decisions** relative to itd-60 and itd-62 — `/abcd:intent plan`'s job.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** a human edits the brief, **when** the derivation pass runs, **then** it draws out the intents and the principles (disciplines) that the edit implies, each as an explicit drawn-out item.
- **Given** the pass has drawn out implied items, **when** the human has not yet accepted or dismissed each one, **then** the brief edit is blocked from committing.
- **Given** an implied item is accepted, **when** the human accepts it, **then** it is recorded as a draft intent (`itd-N`) and/or a principle, per its kind.
- **Given** a brief change whose implications are already captured (e.g. authored by the forward doc-fidelity path), **when** the pass runs, **then** the dedup recognizes the existing item and the pass does not create a duplicate.
- **Given** the derivation cannot reliably run, **when** a brief edit is attempted, **then** the block holds (fail closed) rather than letting the edit commit unreconciled.

## Open Questions

- What counts as "the brief edit committing"? The brief is a directory of files — does the block live at commit time (a pre-commit hook), at a `/abcd` surface, or both? abcd's three-tier design (pre-commit → CI → runtime) is the natural template.
- How is "any brief change fires the pass; dedup later" scoped to avoid firing on trivial edits (typo fixes, reflows)? The dedup handles already-captured implications, but a cheap "is this load-bearing?" pre-filter may still be needed to avoid grilling on whitespace.
- How does dedup identify an "already-captured" implication robustly — provenance markers from the forward path, content matching against existing drafts/disciplines, or both?
- Does the grill run interactively (like the existing grill) on every brief edit, or single-pass-then-confirm? Interactive is truer to the grill but heavier per edit.
- Where do drawn-out **principles** land — as discipline drafts under `intents/disciplines/`, and does a framework-provided vs app-authored distinction apply (see [[itd-62-pluggable-safety-gate]])?

## Evidence 2026-10-04 (a downstream lab)

- **Tense** (recorded for the product thinker, not decided). A downstream private project's brief-authoring lab found that the present-tense-only convention makes a target-design brief read as a working product: 99 present-tense product claims against 9 "staged" markers. Its owner called this the biggest issue they face with abcd, and holds that the brief is the target design the product thinker owns. The lab named this intent with itd-142 and itd-143.
- **Brief to records.** The same lab wrote a 22-file brief over two days and drew no intent from it: no verb proposes intents, decisions or captures from a brief, and the status board did not notice a brief with no records beside it. This intent's derivation pass is the record for the first half, since a brief written from nothing is a run of brief edits; the board's silence on a brief with no records is outside its scope as written. The lab rated it major.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

- Originating assessment: `~/Desktop/abcd-assessment.html` (2026-06-26) — the
  brief-vs-reality coherence concern that motivates both directions.
- Reuses: `skills/abcd-intent-grill/` (the Socratic interrogation machinery),
  pointed at a brief edit rather than a fresh intent.
- Paired with: [[itd-60-doc-fidelity-anti-drift]] (the forward direction; its
  auto-drafted brief edits are the primary dedup case).
- Posture precedent: abcd's tamper-evident provenance and fail-closed reviewers
  — "cannot proceed past an unresolved state" applied to brief edits.
