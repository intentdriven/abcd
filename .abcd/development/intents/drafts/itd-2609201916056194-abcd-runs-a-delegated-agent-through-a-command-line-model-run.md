---
id: itd-2609201916056194
slug: abcd-runs-a-delegated-agent-through-a-command-line-model-run
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
related_intents: [itd-2609170822093401, itd-2]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# abcd runs a delegated agent through a command-line model runner the operator chose, not only through the host session. A managed repository declares, per agent role, whether the role is launched in the host's own sub-agent (the default, unchanged) or through an opt-in command-line adapter: the claude CLI, or an opencode runner reaching an openrouter model. The adapter is the cli rung the oracle backend already names and nothing reads. It launches the role with the same prompt, the same inputs and the same output contract the host sub-agent gets, captures the transcript into the same store, and reports which runner and model answered. A role whose adapter is configured but unreachable falls back to the host and says so, never silently. This is the runner the implement verb builds on, and it refines the per-agent model tier intent by giving the tier a second way to be satisfied.

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Typed Links

Refines itd-2609170822093401 (the per-agent model tier gains a runner that can
satisfy it outside the host) and itd-2 (the host sub-agent stays the default;
this is the opt-in cli rung the boundary rules already name). Built on by
itd-2609201916151817.

## Why This Matters

abcd runs a delegated agent through a command-line model runner the operator chose, not only through the host session. A managed repository declares, per agent role, whether the role is launched in the host's own sub-agent (the default, unchanged) or through an opt-in command-line adapter: the claude CLI, or an opencode runner reaching an openrouter model. The adapter is the cli rung the oracle backend already names and nothing reads. It launches the role with the same prompt, the same inputs and the same output contract the host sub-agent gets, captures the transcript into the same store, and reports which runner and model answered. A role whose adapter is configured but unreachable falls back to the host and says so, never silently. This is the runner the implement verb builds on, and it refines the per-agent model tier intent by giving the tier a second way to be satisfied.

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
