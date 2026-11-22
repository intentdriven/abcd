---
id: itd-2611200216505384
slug: 17-guard-md-1038-lists-a-pid-list-a-kill
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2611060810012471]
origin: extracted-from-record
production_mode: hand-written
---

# `17-guard.md:1038` lists a pid list a kill reads through a variable or a file, or from a `ps | grep` chain, as unseen. `iss-2611040555239973` and `iss-2611040555220611` record the forms that block as `kill-by-search`. They do not record what stays true: `kill $(cat pidfile)` and a bare `kill $p` still allow, so removing the clause outright would leave the chapter implying those are seen. `p=$(ps aux | grep foo); kill $p` also blocks.

## Press Release

> _Seeded by promotion from iss-2611060810012471. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2611060810012471`: `17-guard.md:1038` lists a pid list a kill reads through a variable or a file, or from a `ps | grep` chain, as unseen. `iss-2611040555239973` and `iss-2611040555220611` record the forms that block as `kill-by-search`. They do not record what stays true: `kill $(cat pidfile)` and a bare `kill $p` still allow, so removing the clause outright would leave the chapter implying those are seen. `p=$(ps aux | grep foo); kill $p` also blocks.. Read that issue record for the source observation.

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
