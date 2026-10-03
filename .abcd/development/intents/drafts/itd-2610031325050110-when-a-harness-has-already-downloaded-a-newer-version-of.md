---
id: itd-2610031325050110
slug: when-a-harness-has-already-downloaded-a-newer-version-of
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610031026190632]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# abcd notices a harness update downloaded but not yet running

## Press Release

> When a harness has already downloaded a newer version of itself that a restart would start, abcd says so, reading only this computer: the person learns a restart is pending without abcd asking the internet anything.

_Proposed by the facilitator on filing (2026-10-03): the product thinker asked, at itd-2610031026190632's interview (its decision 6), that this signal be filed as its own draft; to be confirmed at its own planning interview._

## Why This Matters

A harness can download an update while a session runs; the update starts only when the harness is restarted. On the facilitator's test machine the native installer keeps each downloaded version in a versions folder beside the one the launcher points at, so a newer folder than the running one would mean a restart is pending. No such case was seen on 2026-10-03: the running version was the newest download. Both reviews of itd-2610031026190632 found this a second user moment (the running session, not the installed harness) with no evidence yet that it occurs, so it waits here until a case is seen.

Typed links: builds on itd-2610031026190632 (the offline harness line in abcd's update check) and its installed-version reading.

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
