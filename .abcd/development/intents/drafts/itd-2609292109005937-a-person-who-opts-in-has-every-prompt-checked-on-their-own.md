---
id: itd-2609292109005937
slug: a-person-who-opts-in-has-every-prompt-checked-on-their-own
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
related_issues: [iss-2608261543489261]
related_adrs: [adr-22]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# An opt-in local model checks every prompt for secrets and personal data before it leaves the machine

Typed links: `related_issues` [iss-2608261543489261](../../../work/issues/open/iss-2608261543489261-local-model-prompt-sanitiser-before-foundation-model.md) (the record this draft plans); `related_adrs` [adr-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md) (an external tool as an opt-in adapter over a native default).

## Press Release

> A person who opts in has every prompt checked on their own machine before it reaches the hosted model: a small local model flags or redacts secrets, private names and personal data at the moment the prompt is submitted, alongside the scanner and the banned-names lists abcd already runs. When the local model is missing or fails, abcd says so and passes the prompt through unchanged, never silently. "I wanted one plain promise: nothing leaves this machine unchecked," said Dave, a security engineer. "Now I can switch it on, see what it caught, and know when it is not running."

## Why This Matters

abcd redacts on write: the history store scans every transcript before it is
stored, and the banned-names lists guard what is committed. Nothing checks a
live prompt before it goes to the hosted model, so a secret or a private name
typed into a prompt leaves the machine as typed. iss-2608261543489261 proposes a
small on-machine model at the prompt seam (the `UserPromptSubmit` hook) that
flags or redacts before the prompt is sent, composing with the scanner and the
banned-names layers rather than replacing them. The product thinker ruled on
2026-09-29 to plan it as an opt-in feature (ruling J14).

## What's In Scope

- **An opt-in check at prompt submission** that runs the existing scanner and
  banned-names layers and, when configured, a local model over the prompt.
- **The outcome shown to the person**: what was flagged or redacted, before the
  prompt is sent.
- **Loud degradation**: a missing or failing local model passes the prompt
  through and says the check did not run.

## What's Out of Scope

- Shipping a model: the local model is the person's own, an opt-in adapter.
- Checking what the hosted model sends back.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

Seeded by the run from the record and the ruling, unconfirmed: the planning
interview walks each one.

- **Given** the check is off, **when** a prompt is submitted, **then** nothing
  changes and no local model is started.
- **Given** the check is on and a prompt carries a token-shaped secret, **when**
  it is submitted, **then** the secret is redacted or the prompt is held, and
  the person is told which.
- **Given** the check is on and a prompt carries a name on the banned-names
  lists, **when** it is submitted, **then** it is flagged the same way.
- **Given** the check is on and the local model is missing or fails, **when** a
  prompt is submitted, **then** the prompt passes through and the person is
  told the local-model check did not run.

## Open Questions

- **Redact or hold**: does a finding rewrite the prompt, or stop it and ask the
  person?
- **Latency**: what delay per prompt the person accepts before the check is
  worth switching off.
- **Which local runtimes the adapter speaks to**, and whether the scanner-only
  half ships first without a model.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
