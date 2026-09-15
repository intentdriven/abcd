---
id: itd-143
slug: the-brief-gains-a-framing-chapter-under-01-product-the-macro
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# The brief gains a framing chapter under 01-product/ — the macro-why home: one section holds the project's framing, the why behind the why, and receives the brief-creation interview's committed framing products; the brief-lifeboat mapping gains the matching row so a lifeboat carries framing like any other section

Typed links: `refines` the brief skeleton's `01-product/` chapter (a new
section, not a rework of an existing one). Receives the committed framing
products of
[itd-142](itd-142-the-brief-creation-interview-abcd-elicits-a-repository-s-bri.md);
only committed products land here — framing traces stay local per
[adr-50](../../decisions/adrs/0050-framing-traces-never-enter-the-record.md),
as refined by adr-55: the construal as it presently stands is committed
record and readable by automated readers (the cold reading included); its
history is not.

## Press Release

> **A brief now says why the why holds.** The product chapter gains a framing
> section — the macro-why home: the frame the project reasons inside, stated
> where a reader can find it instead of reconstructed from scattered scope
> and context notes. It is a first-class brief section: the brief↔lifeboat
> mapping carries a matching row, `disembark probe` measures it against real
> repositories like any other section, and the brief-creation interview's
> committed framing products land in it with the same confirmation
> discipline as everything else in the brief.

## Why This Matters

The brief records what the project is and what constrains it, but the frame
those answers were formed inside has no home: today it leaks into
`02-context.md` and `04-scope.md` as asides, or stays in humans' heads. A
named framing section makes the macro-why a maintained artefact — and gives
the itd-142 interview a committed destination, so elicited framing lands in
the record rather than in a transcript that is discarded.

## What's In Scope

- The new section under `01-product/` (position and filename fixed at the
  planning interview), with its reading guide in the chapter README.
- The matching row in the brief↔lifeboat mapping — a change to
  `internal/core/lifeboat/mapping.go`, whose generated table and round-trip
  tests hold `00-meta.md` in agreement.
- The tier hypothesis for the new row (expected: blank at tier 0, partial at
  best from conventions, grounded only abcd-native — like `product/personas`,
  framing is a question for a human, not an extraction).
- This repository's own framing section, written as the first instance.
- The construal statement the section opens with: what the situation is
  treated as, in one or two sentences — the surface a widening reading reads
  against; where it describes an intention rather than a commitment it
  carries the not-yet-real marker.
- The not-yet-real marker's form (ratified 2026-08-30): a blockquote line
  holding exactly `**Status: NOT YET REAL.**`, a blank line, then the
  statement as the first paragraph — the file-level `Status: LIVE`
  convention carried to passage level. One exact token, reader-visible in
  the rendered prose, lintable later without new machinery.
- **Ruled (facilitator, 2026-08-28; decision log):** the construal is sited as a brief
  section — this section — and adr-50 is extended to distinguish construal
  from revision history, the extension filed as an ADR in Iteration 1
  *before* the construal surface is built (it files first in the build sequence; adr-55 is that extension). Ground: dropping the Step 2 reading
  would forfeit the position of greatest exposure; a glossary-only reading
  is a defensible fallback but must be stated as a bound, never passed off
  as the full reading.
- Construal revision passes the prior construal to the local ledger side
  rather than retaining it in the section — the brief holds the present
  (adr-5) and version control serves history; no in-section version trail.

## What's Out of Scope

- **The elicitation** — itd-142 owns how framing is asked for; this intent
  owns where it lives.
- **Framing traces** — adr-50: only committed products enter the section.
- **Glossary curation** — the glossary README already names committed terms
  as the framing's machine-visible fingerprint; no new machinery here.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline. Initial set; refined at planning._

- **Given** the brief skeleton, **when** the section ships, **then**
  `01-product/` contains the framing section, the chapter README indexes it,
  and `00-meta.md`'s generated mapping table carries its row — with the
  round-trip test proving code and document agree.
- **Given** a repository probed by `disembark probe`, **when** coverage is
  reported, **then** the framing section is measured and reported with the
  same three-valued status as every other section.
- **Given** an itd-142 interview that commits framing products, **when** they
  land, **then** they land in this section with confirmation provenance, and
  nothing uncommitted lands at all.
- **Given** the framing section, **when** it is read, **then** it opens with
  a one-or-two-sentence construal statement, and any passage describing an
  intention rather than a commitment carries the not-yet-real marker.
- **Given** a construal revision, **when** the section is rewritten, **then**
  the prior construal exists on the local ledger side and not in the brief.
- **Given** the framing section, **when** it is projected, **then** the
  construal statement and its load-bearing vocabulary are
  machine-extractable — fixed section heading, statement-first layout, the
  exact marker token — so the framing vocabulary can be pulled from the
  brief retrospectively (facilitator review requirement, 2026-08-28).

## Open Questions

- Section position and filename within `01-product/` (the chapter is
  numbered; inserting versus appending changes existing links).
- Whether the tier hypothesis for framing matches personas (blank below
  abcd-native) once probed against the corpus — where they disagree, the
  mapping table loses.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
