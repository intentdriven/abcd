---
id: itd-2611031342011261
slug: every-agent-abcd-launches-for-a-session
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

# abcd keeps a per-session ledger of every agent it launches

## Press Release

> Every agent abcd launches for a session (a build or drain lane's implementer and reviewers, a review pair, any runner launch) is recorded against the session that orchestrated it: its billing route (the key service it was handed, or the subscription), model, spend, tokens, API time and outcome (finished, failed, stopped), and whether it is still running. The run state keeps spend and tokens per launch today but no API time, no route and no orchestrating session, so a session cannot sum or count what it started (lab lab-261103114820-bcba06f, findings F-2, F-3, F-12, F-13). The status line's counts, spend and API figures read this ledger, spend only for launches billed to a key, and the ledger is the record a fully autonomous run accounts for itself by.

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
