---
id: itd-2611011813162141
slug: a-repository-names-a-sequence-of-abcd
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# A repository names a sequence of abcd steps a person runs

## Press Release

> A repository names a sequence of abcd steps and a person runs it with one command: abcd run <name> reads a named list of abcd's own verbs and non-widening flags from the repository's config, judges each step through the guard, and runs them with the build loop's stage semantics, stopping on a refusal, a back-off or an open question and releasing what earlier steps reserved; for an agent the same list renders as step-by-step instructions, so each step is its own judged command line. No shell steps, no new top-level names, no shadowing of a built-in verb.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Decisions

Answered by the product thinker on 2026-11-01.

1. **Sequences live in `.abcd/config.json`**, under one key read only by `abcd run`, not in a rules domain: data and the rules agents are taught stay apart, and an agent learns a sequence's steps from the page `abcd run` renders for it.
2. **Steps fail closed, and `abcd run` passes nothing on** (adr-2611011813190527): only verbs and flags classified as non-widening may appear, a verb that widens with no flag (a push, `reserve`, `update`) is refused, and `abcd run <name>` takes no flag beyond the name, so a widening flag is only ever the person's own direct command.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
