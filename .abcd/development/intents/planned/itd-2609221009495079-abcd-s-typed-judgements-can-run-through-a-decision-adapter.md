---
id: itd-2609221009495079
supersedes: [itd-17]
slug: abcd-s-typed-judgements-can-run-through-a-decision-adapter
spec_id: spc-2609221011151661
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609212137128014, itd-2609081951381895, itd-2609170822093401]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-17, itd-2609212137116617, itd-82]
related_adrs: [adr-2609221009491186]
---

# abcd's typed judgements can run through a decision adapter, measured in a lab before it decides anything

## Press Release

> **A decision adapter answers abcd's closed-option judgements with a value and a calibrated probability; it runs in shadow as a lab first, and a research note turns it on.**
>
> "Half of what my agents ask a frontier model is a yes-or-no about one record," said a product thinker reading a run's cost. "A decision model answers those in half a second for a fraction of a cent. I do not trust it yet, so it runs beside the host, the lab records every pair, and when the note shows it agrees, I turn that judgement over."

## Why This Matters

The research pass of 2026-09-22 found Jev (TypeSafe AI) to be a typed-decision model, not a text model and not a router: choose one of N, score against a rubric, yes or no, each with a calibrated probability, in under a second at $0.042 per million tokens, 32K context, and unable to abstain. Independent evidence is one forty-case test; the vendor's numbers are vendor-graded. abcd's judgement-shaped steps are many: the remedy read, duplicate matching, unmeetable criteria, the eligibility residual, source classification and selection in the library, memory checks, ledger category and severity, the pre-pass's overlap, the docs lint's present-tense advisory, embark's lesson ranking. Ruled: an adapter behind one interface, the host as default, shadow mode as an abcd lab, a research note as the evidence a ruling turns a judgement type on with.

## Mechanism

We expect calibrated typed answers to agree with the host on closed-option judgements at a rate a lab can measure, because those judgements have a small option set and a state that fits the model's window; shown wrong if the shadow run's agreement is below what the note's threshold demands.

## Scope Conditions

- Holds for judgements whose option set is closed and whose state fits 32K tokens after retrieval; a judgement needing the whole corpus in view is out. <!-- cond: cond-2609221011150901 -->
- Holds where a refusal is not the right answer: the model cannot abstain, so a judgement type whose right answer is often "unknown" is not turned over. <!-- cond: cond-2609221011151505 -->
- Holds under adr-2609221009491186: the adapter's model is on its provider's allowlist. <!-- cond: cond-2609221011150032 -->

## What's In Scope

- **One interface**: `(state, typed question) → (value, probability)` with three question kinds (choose, score, yes/no); the host's own judgement implements it too and stays the default.
- **The judgement types** it may be configured for: the remedy read, duplicate matching, unmeetable criteria, the eligibility residual, source classification and selection over a shortlist, the memory checks, ledger category and severity, the pre-pass's overlap, the docs lint's present-tense advisory, embark's lesson ranking.
- **Shadow mode as a lab**: `abcd lab mint` opens a lab home; the adapter runs beside the host on the configured types, decides nothing, and each pair (host's answer, adapter's answer, probability, the state's hash) is recorded under the lab home in the machine-scoped store, never the repository.
- **The evidence**: `abcd lab harvest` produces agreement rates per type with counts and probability bands; the research note filed from it (dated, under the record's research notes) is the evidence a ruling turns a type on with.
- **Turning on**: per judgement type in configuration; unreachable falls back to the host with a receipt; the run record names which route judged.
- **The first candidate**: `typesafe/jev-1.13` through the API adapter, on OpenRouter's allowlist; a small text model scoring options by log-probability is the second, behind the same interface.

## What's Out of Scope

- Any generative step.
- A learned per-request router for lane models (itd-17, superseded).
- What to build next (a computed score, itd-2609211116005482; a rubric score may become one component later).

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609221009491186 records the vocabulary rulings it rests on):

1. Shadow mode only, as an abcd lab; a research note is the evidence; the host stays the default (ruled 2026-09-22).
2. The interface is the host's too, so the two are compared like for like.
3. Turned on per judgement type, never all at once.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** a judgement type configured for shadow and a lab home minted, **when** the judgement runs, **then** the host decides, the adapter answers beside it, and the pair with its probability and the state's hash is written under the lab home; nothing is written into the repository.
- **Given** a lab home with recorded pairs, **when** `abcd lab harvest` runs, **then** it reports agreement per judgement type with counts and probability bands, and the note filed from it names the lab home and the count.
- **Given** a judgement type turned on in configuration, **when** it runs, **then** the adapter decides, the run record names the route, and an unreachable adapter falls back to the host with a receipt saying so.
- **Given** a configured model not on its provider's allowlist, **when** the configuration is read, **then** it is refused before any call (adr-2609221009491186).
- **Given** a judgement whose state exceeds the model's window, **when** it runs in shadow, **then** the pair is recorded as skipped with the size, and the host decides alone.
- **Given** the host judgement and the adapter, **when** either is called, **then** both go through the one interface, so a type can be switched without a caller changing.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: cost per judgement is a hundredfold lower and the evidence is vendor-graded; we expect the shadow lab to show agreement the note can put a number on; shown wrong if the agreement is below the note's threshold
