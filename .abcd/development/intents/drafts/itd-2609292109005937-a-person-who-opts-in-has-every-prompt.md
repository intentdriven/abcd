---
id: itd-2609292109005937
slug: a-person-who-opts-in-has-every-prompt
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

Typed links: `related_issues` [iss-2608261543489261](../../../work/issues/open/iss-2608261543489261-local-model-prompt-sanitiser-before.md) (the record this draft plans); `related_adrs` [adr-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md) (an external tool as an opt-in adapter over a native default).

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

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [security-isolation SOTA note](../../research/notes/2026-10-23-security-isolation-sota.md).

- **The prompt seam can hold and cannot rewrite**: a `UserPromptSubmit` hook "Blocks the prompt, so it never reaches Claude" on exit 2 or `decision: "block"`, and "can't replace the prompt; it only injects `additionalContext` alongside it"; the default hook timeout there is 30 seconds (Claude Code hooks). On this host "redact" is unavailable; "hold and tell" is.
- **The prompt is the smaller channel**: "Your prompts and the files Claude reads are transmitted to the Anthropic API or your configured provider with or without a sandbox"; a prompt-only check leaves file reads and tool results unchecked, so "nothing leaves this machine unchecked" should read "no prompt leaves unchecked" (Claude Code sandbox environments).
- **Rule-based detection misses what matters most, and small models do not close the gap**: REDACT (June 2026) recall on HIGH-sensitivity categories Presidio (a candidate opt-in adapter) 0.07, GLiNER 0.42, OpenAI's Privacy Filter 0.57, GPT-4.1 0.74, Claude Sonnet 4.6 0.77; partial micro-F1 0.195 / 0.320 / 0.512 / 0.597 / 0.636. PIIBench (April 2026) put every detector under F1 0.14. The two rank differently and agree that a local-only gate cannot carry the promise (arXiv 2606.19881; arXiv 2604.15776).
- **The earlier local survey's hybrid figure stands** (deterministic ~69%, hybrid ~83%, a practitioner benchmark): deterministic layer as the gate, local model as the assist (2026-08-22 note).
- **Presidio**, a candidate opt-in adapter, is regex, checksum and NER recognisers in Python, "transitioning to a community-owned project", with "no guarantee that Presidio will find all sensitive information" (Presidio docs).
- **Fail-open at the hook is a configuration choice, not a host constraint**: the default `UserPromptSubmit` timeout is 30 seconds and settable per hook; a hook that reaches it is cancelled and the prompt proceeds unless the hook sets `onFailure: "block"` (host v2.1.295 or later), which blocks the prompt on failure or timeout. A blocked prompt's text is still written to the local transcript (the block message ends with "Original prompt:" unless the hook suppresses it), which a privacy-motivated hold should disclose (Claude Code hooks).

**Path 2: native, deterministic-first, hold-and-tell at the hook, with the local model as the opt-in adapter behind the existing scanner seam** (adr-22's shape). The independent fit-challenge runs at plan time.

## Open Questions

- (SOTA 2026-10-23) **The host answers "redact or hold" with hold**: the record should say so by capability ("where the host lets a hook hold a prompt") rather than by harness.
- (SOTA 2026-10-23) **Whether the promise is narrowed to prompts**, since file reads and tool results leave unchecked on every host surveyed.
- (SOTA 2026-10-23) **Fail-closed or fail-open at the hook**: the host offers both (`onFailure: "block"`, or the default that lets the prompt through); the owed ADR chooses, and hold-within-budget is a latency discipline rather than the only fail-closed posture. A held prompt's text still reaches the local transcript either way.
- **Redact or hold**: does a finding rewrite the prompt, or stop it and ask the
  person?
- **Latency**: what delay per prompt the person accepts before the check is
  worth switching off.
- **Which local runtimes the adapter speaks to**, and whether the scanner-only
  half ships first without a model.
- **The owed ADR on failing open**: letting a prompt through when the local
  check is down is a trust posture at a privacy boundary, not a detail of this
  intent. The decomposition calibration note predicts a SPLIT, this intent plus
  an ADR on that posture, and flags it for the human; the loud-degradation scope
  item and the last criterion above hold only once that ADR rules fail-open
  rather than hold-and-ask.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
