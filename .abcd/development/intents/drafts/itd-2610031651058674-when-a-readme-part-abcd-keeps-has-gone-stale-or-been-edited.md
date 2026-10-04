---
id: itd-2610031651058674
slug: when-a-readme-part-abcd-keeps-has-gone-stale-or-been-edited
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610031348087517]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# abcd's check reports README parts that have gone stale, and writes nothing

## Press Release

> When a README part abcd keeps has gone stale or been edited by hand since the last setup run, abcd's check says which part and why, and writes nothing; the owner decides whether to run setup again or keep their edit.

_Proposed by the facilitator on filing (2026-10-03): the product thinker chose, at itd-2610031348087517's interview (its decision 1), that keeping README parts current also means a separate check that reports a stale part and writes nothing; to be confirmed at this draft's own planning interview._

## Why This Matters

_Facilitator-written._

Setup rewrites the README parts an owner chose only when it runs; between runs a licence can change, a workflow can be renamed, or the owner can edit inside a marked part. This check says so without touching the file, so a person or a continuous-integration run sees the drift and the owner decides what happens. The research for the parent draft found no tool that reports drift outside its own markers; the cog tool's checksum guard is the model for telling a hand edit from a stale derivation.

Typed links: builds on itd-2610031348087517 (the README parts and their marked places).

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
