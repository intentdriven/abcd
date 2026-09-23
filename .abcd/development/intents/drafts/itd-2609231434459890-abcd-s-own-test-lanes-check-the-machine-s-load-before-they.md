---
id: itd-2609231434459890
slug: abcd-s-own-test-lanes-check-the-machine-s-load-before-they
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# abcd's own test lanes refuse to start under excessive machine load

## Press Release

> abcd's own test lanes check the machine's load before they start and refuse when it is too high, naming what is running. The product thinker ruled on iss-2609210828122412 (2026-09-23, DECISIONS.md) that a load experiment is trusted only under four conditions; the first three (one owned process group killed together, clean proven by what is running, explicit consent and a cap below the core count on a live development machine) hold as rules now and ship in the bundled LOAD rule domain. This fourth is the automatic check: before make preflight or a test lane begins, the gate reads the load average and refuses above a declared ceiling, and the refusal names the processes that are burning (after the 2026-09-21 panic, abcd's own reading.test and cli.test were found running under eight orphaned burners). To plan: the ceiling and where it is declared, which lanes run the check (preflight, CI, the eval lanes), how a refusal names what is running on each host, and whether an explicit override exists.

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
