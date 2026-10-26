---
id: itd-2610262127138979
slug: abcd-works-out-for-itself-how-many
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# abcd sizes its own test admission from what it measures

## Press Release

> abcd works out for itself how many preflights a machine can run at once. Today a person writes a fixed number, one or two, and the gate only waits until the load is low enough; it never learns whether the number was right. With this change abcd measures each run it admits: how long each stage waited and ran, the load and memory it drew, and whether a test timed out. From that record it raises the number while runs finish well inside their limits and lowers it when tests time out or the machine stays overloaded. It meters every heavy stage, not only the tests, so queued runs stop feeding the load they wait on; it admits a run on the headroom the whole run will need, not on one instant's load; it lets a short targeted test go ahead of whole preflights without starving either; and it picks up a changed setting while runs are already waiting. The person's own setting stays the ceiling, a repository can never raise it, and each change abcd makes is written down with its grounds where the person can read and override it.

## Why This Matters

> On a shared machine, the number of preflights at once decides how fast everything lands, and a fixed number is wrong in both directions. In the autonomous run of 24–26 October 2026 on a 10-core laptop, two at once drove the load to 42 and tests timed out, so the person set it back to one. Then the right number changed when an endpoint scanner was switched off, which no written setting tracks. The load still reached 124 when a race stage admitted at low load spawned work faster than the gate could see, and a short package test timed out twice at the gate behind whole preflights. Sessions spent hours waiting and the person spent attention tuning a number by hand. The product thinker asked for the setting to follow the evidence, adjusted up or down as more comes in.
>
> The intent gathers what is captured piecemeal in the ledger: iss-2610161756384596 (admission derived from measured resources), iss-2610241214563687 (the slot count follows logged evidence), iss-2610241414124940 and iss-2610241911052191 (both major: every heavy stage behind one admission, a setting re-read while waiting, admission on the headroom a run will need), iss-2610241214065079 (lint, eval, build and vet stages run ungated), iss-2610260845422800 (a short-run lane) and iss-2610161639025702 (preflight throughput sets the pace of the whole run). The wider self-optimisation the person also asked for, iss-2610241215547722, stays its own record: this intent is its first, narrowest case.

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
