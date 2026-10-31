---
id: itd-2610300955026531
slug: agents-can-work-under-a-forge-account-of
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Agents can work under a forge account of their own

## Press Release

> Agents can work under a forge account of their own, as an optional setup. Today an agent acts with the person's own forge login, so the forge cannot tell an agent's push, pull request or review from the person's, and the person cannot approve on the forge a pull request their agent opened under their name. With an account of its own, an agent pushes and opens pull requests as itself, and the person's approval is a forge review from a different account, which the merge gate already counts. The design is chosen from a primary-source review of current practice for machine identities on a forge, and works the same in a cloud session and on a local machine, whether the agent reaches its model through an API key or a subscription. It stays optional: a repository that does not set it up works as it does now.

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
