---
term: phase
bounded_context: core
definition: An ordered stretch of development work that bundles a set of intents and brief plumbing-phases and ends in a milestone; abcd's sequencing layer, recorded as a document in roadmap/phases/. Unqualified it always carries that sense, the brief's own numbered build milestones being plumbing-phases.
aliases: ["roadmap phase"]
forbidden_synonyms: ["version", "release", "milestone", "sprint", "iteration"]
status: stable
introduced_in: adr-9
starts_when: null
ends_when: null
not_to_be_confused_with: core/spec
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# phase

A **phase** is abcd's sequencing layer — an ordered stretch of work that ends in a
**milestone** (a concrete, checkable end condition). Phases replace plugin-version language
(`v1`, `v2`) as the way the project organises what ships together and in what order. Each
phase is a document in `.abcd/development/roadmap/phases/` that opens with a product
`## Expectation`: a working-backwards re-statement, at phase granularity, of what the phase is
expected to make true for the user.

A phase sits between the [brief](../../README.md) (the whole project) and the
[intent](intent.md) (one user-facing capability) on the question it answers — see the
four-layer mental model in `brief/01-product/03-mental-model.md` and
[adr-9](../../../decisions/adrs/0009-phase-as-product-layer.md).

## Senses

The word carries two senses in the record, and the glossary fixes one spelling for each
(iss-2609012245352480).

| Sense | The one spelling | Where it lives |
|---|---|---|
| The sequencing unit: an ordered stretch of work bundling intents and plumbing-phases, ending in a milestone | **phase**, unqualified | [`roadmap/phases/`](../../../roadmap/phases/README.md), one `phase-N-<slug>.md` per phase |
| One numbered build milestone of the brief's dependency DAG, which a phase bundles from | **plumbing-phase**, hyphenated and qualified "brief plumbing-phase" on first use | [`06-delivery/01-build-sequence.md`](../../06-delivery/01-build-sequence.md) |

**The two numberings are not a mapping.** Roadmap phases run 0–7 and the brief's
plumbing-phases run 0–6; the low numbers coincide, and Phase 7 — Provenance ledger and cold
reading has no plumbing-phase of that number at all. A number alone therefore never says which
sense is meant, and the qualifier does.

**Where the record disagrees.** [adr-9](../../../decisions/adrs/0009-phase-as-product-layer.md)
writes both "plumbing phases" and "brief plumbing-phases" in one document, and the mental-model
chapter's diagram edge is labelled `plumbing phase`. The glossary fixes the hyphenated
**plumbing-phase**, which is the form
[`roadmap/phases/README.md`](../../../roadmap/phases/README.md) and every phase doc's `## Scope`
already use; the unhyphenated instances are the disagreement, not a third sense.

## When to use

Use "phase" for an ordered stretch of work that bundles intents and plumbing into a coherent
milestone-ending unit. Use it when discussing *what order* work happens in, or *what
expectation* a stretch of work is measured against. Use "brief plumbing-phase" for a numbered
section of the build sequence.

## When NOT to use

Do not use "phase" for an individual work block (that is a [spec](spec.md)), for a plugin
release number (a release version is an *output* of completing a phase, never the organising
unit), or interchangeably with "milestone" — the milestone is the *end condition* of a phase,
not the phase itself. Do not write bare "phase" for a build-sequence milestone, and do not
write "build phase" at all: it reads as either sense.

## Examples

- "Phase 1 — Substrate ends when the oracle backend ships and project state is reconciled."
- "Phase 1's `## Scope` bundles the brief plumbing-phases 0 and 1."
- "Spec `spc-5` carries `phase: phase-1-substrate` in its frontmatter." (The `phase:` anchor's tooling is specified in spc-66 (predecessor store) — the phase-audit reviewer that reads it and the `PA001` verify-exists lint are a design target for the Go binary, not yet shipped; also deferred is the *corpus backfill* that would make the anchor a standing convention. Phase membership today is still editorial, via each phase doc's `## Scope`.)
- "The phase audit reviews delivered reality against the phase's structured `## Phase Acceptance`." (Per the adr-9 amendment: the reviewable cut is the structured `## Phase Acceptance` bullets, NOT the prose `## Expectation` — prose is the narrative re-statement, not the audit target.)

## Related terms

- [spec](spec.md) — the implementation unit; many specs belong to one phase
- [intent](intent.md) — the user-facing capability; a phase bundles a set of intents
- [roadmap](roadmap.md) — the folder the phase docs live in, and the dashboard that renders their progress
- [plan](plan.md) — the phase docs are the ordered build plan, a different sense from the `intent plan` verb
- [voyage](voyage.md) — the operations namespace recording what abcd did to produce a lifeboat; not a lifecycle arc (that sense is retired per adr-35)
