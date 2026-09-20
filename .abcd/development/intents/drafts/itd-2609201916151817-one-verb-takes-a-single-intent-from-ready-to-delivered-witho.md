---
id: itd-2609201916151817
slug: one-verb-takes-a-single-intent-from-ready-to-delivered-witho
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201916056194]
related_intents: [itd-29, itd-50, itd-2, itd-2609170822093401]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# One verb takes a single intent from READY to delivered without a human in the loop, and refuses to start when a design decision is still open. abcd implement <itd-N> checks the readiness gate and, beyond it, that every decision the intent needs is recorded: no open question, no unanswered claim section, no hold on the record. It then runs the intent's spec through configurable sub-agent roles, an orchestrator that sequences the work, one or more implementers that build test-first in their own worktree, and validators that are never the implementer (the security and ruthless reviewers, and the fidelity auditor at the end), each role launched through the host sub-agent by default or through a configured runner. It lands the work on a branch with the spec closed in the same change, arms the merge, and writes a run record naming every agent, model, budget and verdict. By default it refuses an intent that is not in planned/; an explicit --plan flag lets it plan a draft whose decisions are all recorded, which reverses the never-unattended planning rule for that path alone and is ruled on in its own decision record before that path ships. This is the verb the managed-repository sessions have been hand-building as a coordinator, implementer, reviewer and adjudicator hierarchy, with a pause-and-resume budget pattern, and it is the shape the Dessau pilot is measuring.

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Open Questions

- **Planning path (settled 2026-09-20, product thinker):** the verb refuses an
  unplanned intent by default; `--plan` is allowed for a draft whose decisions
  are all recorded. That flagged path reverses the never-unattended rule on the
  intent surface page, so an ADR is minted before it ships. Owed, not yet minted.
- **Provider routing (settled 2026-09-20):** its own draft, itd-2609201916056194,
  which this intent builds on; the default remains the host sub-agent.

## Typed Links

Refines itd-29 (the autonomous spec run with its budget check and resume, keyed
on a spec; this verb is keyed on the intent and owns the roles), itd-50 (the
audit loop to acceptance becomes the verb's last stage), itd-2 (in-session
sub-agent dispatch stays the default runner) and itd-2609170822093401 (the
per-agent model tier decides which runner a role gets). Answers the
recipe question the sub-agent-lane session raised (its capture lands with PR #644). Field evidence: the sub-agent-lane experiment and the
Dessau pilot in a managed repository, both September 2026.

## Why This Matters

One verb takes a single intent from READY to delivered without a human in the loop, and refuses to start when a design decision is still open. abcd implement <itd-N> checks the readiness gate and, beyond it, that every decision the intent needs is recorded: no open question, no unanswered claim section, no hold on the record. It then runs the intent's spec through configurable sub-agent roles, an orchestrator that sequences the work, one or more implementers that build test-first in their own worktree, and validators that are never the implementer (the security and ruthless reviewers, and the fidelity auditor at the end), each role launched through the host sub-agent by default or through a configured runner. It lands the work on a branch with the spec closed in the same change, arms the merge, and writes a run record naming every agent, model, budget and verdict. By default it refuses an intent that is not in planned/; an explicit --plan flag lets it plan a draft whose decisions are all recorded, which reverses the never-unattended planning rule for that path alone and is ruled on in its own decision record before that path ships. This is the verb the managed-repository sessions have been hand-building as a coordinator, implementer, reviewer and adjudicator hierarchy, with a pause-and-resume budget pattern, and it is the shape the Dessau pilot is measuring.

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
