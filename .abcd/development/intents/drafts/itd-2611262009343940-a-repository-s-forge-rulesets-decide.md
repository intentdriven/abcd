---
id: itd-2611262009343940
slug: a-repository-s-forge-rulesets-decide
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

# An agent may apply a forge ruleset change that only tightens protection

## Press Release

> A repository's forge rulesets decide what an agent may push or merge, so today every ruleset change is reserved to the person: an agent that could change them could loosen its own limits. Many changes only add protection, such as a new ruleset that forbids deleting or force-pushing a branch abcd itself writes. The person wants such a change to be one an agent may apply in an autonomous run, so a landed feature's protection does not wait days for the person's return. The idea: abcd proves mechanically that a change only tightens (the new rule set is a superset of the old for every ref it touches, no bypass actor is added, enforcement only moves towards active, no ruleset is removed or narrowed), applies it through the existing confirmed remote-apply path when the proof holds, and refuses and queues it for the person otherwise. Decomposition, for the person to confirm: the capability is this intent; the trust rule (an agent never loosens a rule that binds it; a tighten-only change is proved, never judged) is an ADR and a brief invariant; the stance refines the evaluator-outside-the-loop principle; the remote-apply plumbing is the brief's. Verdict: SPLIT, the ADR to be minted when this is planned. Raised by the person on 2026-11-26 during the surge run, after the spot-check ruleset had to wait for them.

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
