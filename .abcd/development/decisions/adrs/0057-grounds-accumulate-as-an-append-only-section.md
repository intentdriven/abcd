---
id: adr-57
slug: grounds-accumulate-as-an-append-only-section
status: accepted
date: 2026-08-30
supersedes: null
superseded_by: null
related_intents: [itd-179]
related_rfcs: []
related_adrs: []
---

# ADR-57: Grounds accumulate as an append-only section, in one shared form

## Context

itd-179 records the conjecture behind a decision. It grew two writers. The
intent half wrote an append-only `## Grounds` section and argued for it in
terms: the earlier conjecture is precisely what a later reader checks the
outcome against, so rewriting it would leave the record saying only what was
believed last. The ledger half wrote a single `grounds:` frontmatter scalar and
set it, so `promote` recorded a conjecture and a later `resolve` or `wontfix`
destroyed it. Nothing warned; the result reported success.

The obvious remedy — refuse a transition that would overwrite a non-empty
`grounds` — was checked and does not work. `promote`, `resolve` and `wontfix`
all REQUIRE `--grounds`, so refusing the overwrite makes every promoted issue
impossible to resolve. Fourteen records in `resolved/` already carry
`promoted_to`.

So the two halves held opposite rules for one concept, and the ledger's rule
was the one that lost data.

## Decision

**Grounds accumulate. One entry per grounds-bearing act, appended, never
replaced — and in ONE form across both halves.**

1. The ledger's grounds move from a frontmatter scalar to a `## Grounds` body
   section, the shape the intent half already used.
2. The reading and writing primitive lives once, in `core/mdrecord` (masking,
   section bounds, bullet blocks, link-reference peel) with the vocabulary and
   section operations in `core/grounds`. `core/intent` and `core/capture` both
   consume it. Neither owns the other's parser.
3. One asymmetry between the halves is real and is preserved explicitly rather
   than flattened: the intent reader applies the substance floor because its
   gate claims it, and the ledger reader stops at the grammar because a
   wontfix's grounds derive from a reason whose contract is only non-emptiness.
   That is expressed as `ParseSectionAboveFloor` layered ON TOP OF
   `ParseSection`, so what an entry IS stays one definition and only the
   admission bar differs.

## Alternatives Considered

**Refuse the overwrite.** The reviewer's proposal, and the reason this ADR
exists rather than a one-line fix: it deadlocks the ledger's mainline sequence,
because every route into a terminal folder requires grounds.

**Last-write-wins, documented.** Smallest change, and honest — the loss becomes
a stated tradeoff rather than an accident. Rejected because the ledger would
then record the opposite rule to the intent half for the same concept, and the
thing lost is exactly what the intent half argues a later reader most needs.

**An append-only list in frontmatter.** Keeps the data where the ledger reader
already looks. Rejected as a second shape for one concept: it would differ from
the intent half's bullets, so the repository would hold two definitions of how
grounds accumulate, which is what one-canonical-primitive forbids.

## Consequences

- Sixteen committed records carrying the frontmatter scalar were migrated
  through the real writer, so their shape is byte-identical to what the verbs
  produce. Expensive to reverse: that is half of why this is an ADR.
- The retired `grounds` key stays in the issue schema's known set rather than
  being refused. A refusal makes capture SKIP the record, and a record invisible
  to every surface while it sits in the ledger is worse than a key nothing
  reads; `record_schema` blocks the misplacement and names the section instead.
- Coverage moves, and not only forward. A malformed `grounds:` VALUE was
  something the schema gate could judge; a malformed bullet under `## Grounds`
  reads as prose, so nothing is wrong to any gate and nothing says so
  (iss-2608301747001641). That is the price of the shared form and it is
  recorded rather than absorbed.
- `Issue.Grounds` becomes a list in the JSON envelope. Unreleased field on an
  unreleased branch, so nothing external moves.
- `core/mdrecord` is a new package in the core DAG, which is the other half of
  why this is an ADR: a future session asking why the two halves do not each
  keep their own reader would otherwise re-derive the whole argument.
