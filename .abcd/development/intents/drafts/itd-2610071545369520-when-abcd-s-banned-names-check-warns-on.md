---
id: itd-2610071545369520
slug: when-abcd-s-banned-names-check-warns-on
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
refines: [iss-2610071538075997]
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A person confirms the assistant's verdict on each warned banned-name line instead of judging it alone

## Press Release

> When abcd's banned-names check warns on a line (a listed name that is also an ordinary word, such as a game title like Portal), the assistant in the person's session proposes for each warned line whether it uses the protected name or the ordinary word, with a one-line reason, and the person approves or overrides. An approved protected use stays blocked; an approved ordinary use gets the allow marker, so the next run passes. Product thinker's ruling 2026-10-07 (interview on iss-2610071538075997, promoted from a managed repository's report): add the judge; the managed repository's further asks, a free local model judging silently into a private log and a review of that log's miss rate, are not taken on now. Open for planning: what identifies a line so a verdict still matches after the line moves; whether the allow marker records who decided and when.

## Why This Matters

> _Why this matters to the user — replace before planning._

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
