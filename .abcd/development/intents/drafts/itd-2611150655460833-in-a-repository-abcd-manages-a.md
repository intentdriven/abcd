---
id: itd-2611150655460833
slug: in-a-repository-abcd-manages-a
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# A security finding not fixed before landing becomes a private draft advisory by default

## Press Release

> In a repository abcd manages, a security-relevant finding raised during development or review that is not fixed before the change that raised it lands is recorded by default as a private draft security advisory on the forge, rather than only in a file on one machine. The advisory says only what the code does, is tracked by abcd's embargo store like a report someone else filed, and is published only with the release that holds its fix, so the finding is kept off the public ledger and is not lost with one machine or one session.

## Why This Matters

itd-2610112239022766 keeps reports others file privately until their fix ships. Findings abcd's own development and review raise and do not fix before landing have no such route yet: they sit in a file on one machine, where one lost machine or session loses them.

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
