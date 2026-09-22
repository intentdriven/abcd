---
id: spc-2609221011151661
slug: abcd-s-typed-judgements-can-run-through-a-decision-adapter
intent: itd-2609221009495079
origin: researcher-authored
production_mode: hand-written
---
# abcd-s-typed-judgements-can-run-through-a-decision-adapter

## Summary

The design record for itd-2609221009495079: the decision interface, the shadow lab, the harvest and the per-type switch.

## Scope

1. **The interface** (`internal/core/decide/judge.go`, distinct from the ADR verb's package): `Judge(state, question) (value, prob)`, three question kinds; the host implementation renders the question as a prompt and parses the answer with probability 1 (criteria 1, 6).
2. **The callers**: each judgement-shaped step names its type and calls the interface; the registry of types is the closed list in scope (criterion 1).
3. **Shadow**: the router reads the configuration per type (`host` | `shadow:<provider/model>` | `<provider/model>`); shadow runs both and appends the pair to `~/.abcd/lab/<lab-id>/pairs.jsonl` (criteria 1, 5).
4. **Harvest**: `abcd lab harvest` gains the agreement report for a pairs file (criterion 2).
5. **Turned on**: the router calls the adapter and records the route; fallback with a receipt (criterion 3).
6. **Allowlist**: the resolver of adr-2609221009491186 runs before any call (criterion 4).

## Out of scope

- Generative steps; learned routers; the pick.

## Approach

One interface, one router with three modes, one adapter over the API adapter's client; the lab verb records and harvests, so the evidence path is the lab's, not a new one.

## Footprint

- packages: internal/core/decide, internal/core/lab, internal/adapter/openaiapi, callers across capture, intent, memory, lint
- tests: the router's three modes with a fake adapter; the pairs file; the harvest's rates; the window skip; the fallback receipt

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 shadow records pairs | scope 3 |
| 2 harvest and the note | scope 4 |
| 3 turned on, fallback | scope 5 |
| 4 allowlist | scope 6 |
| 5 window skip | scope 3 |
| 6 one interface | scope 1, 2 |
