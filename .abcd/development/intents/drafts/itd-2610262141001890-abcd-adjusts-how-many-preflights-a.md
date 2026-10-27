---
id: itd-2610262141001890
slug: abcd-adjusts-how-many-preflights-a
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610262140595217]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
blocked_by: [itd-2610262140595217]
---

# abcd adjusts how many preflights run at once from what it recorded

## Press Release

> abcd adjusts how many preflights a machine runs at once from its record of how earlier ones behaved: it lowers the number when tests time out or runs stretch well past their usual time, and raises it again, a step at a time, while runs finish comfortably inside their limits. Nobody adjusts the number by hand: a person may name a starting default, and abcd uses even that only once it has measured what the machine can take. abcd keeps the number it works out in a file of its own, never in the person's settings, and a repository can never raise it. Each change is written down with the runs it rests on, so the person can read why and override it.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Decisions

- The number allowed at once is adjusted automatically, never by hand; a person may name a starting default, used only once the machine's capacity has been measured (product thinker, 26 October 2026, at the planning interview of itd-2610262127138979). This reverses the standing rule that the person writes the number (the LOAD rules, dcn-2610161814279204), and the cap at 2 is lifted only on measured evidence that 2 still leaves room; both are settled at this intent's planning.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
