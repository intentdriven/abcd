---
id: itd-2609081951381895
slug: abcd-ships-an-openai-compatible-api-oracle-adapter-the-first
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

# abcd ships an OpenAI-compatible api oracle adapter — the first concrete implementation of adr-25's api oracle-backend shape. An operator configures a baseURL (and optional key) in .abcd/config.json and abcd routes oracle calls — reviews, audits — to that model over plain HTTP. The first wired provider is a local OpenAI-compatible server (Gropius MLX on localhost, no key needed); cloud aggregators such as OpenRouter ride the same adapter as pure configuration, not code. The host-delegated default is untouched: with no adapter configured nothing changes and no network call is attempted.

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

abcd ships an OpenAI-compatible api oracle adapter — the first concrete implementation of adr-25's api oracle-backend shape. An operator configures a baseURL (and optional key) in .abcd/config.json and abcd routes oracle calls — reviews, audits — to that model over plain HTTP. The first wired provider is a local OpenAI-compatible server (Gropius MLX on localhost, no key needed); cloud aggregators such as OpenRouter ride the same adapter as pure configuration, not code. The host-delegated default is untouched: with no adapter configured nothing changes and no network call is attempted.

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
