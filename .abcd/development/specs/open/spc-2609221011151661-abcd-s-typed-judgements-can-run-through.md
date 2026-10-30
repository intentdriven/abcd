---
id: spc-2609221011151661
slug: abcd-s-typed-judgements-can-run-through
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
3. **Shadow**: the router reads the configuration per type (`host` | `shadow:<provider/model>` | `<provider/model>`); shadow runs both and appends the pair to `~/.abcd.noindex/lab/<lab-id>/pairs.jsonl` (criteria 1, 5).
4. **Harvest**: `abcd lab harvest` gains the agreement report for a pairs file (criterion 2).
5. **Turned on**: the router calls the adapter and records the route; fallback with a receipt (criterion 3).
6. **Allowlist**: the resolver of adr-2609221009491186 runs before any call (criterion 4).

## Out of scope

- Generative steps; learned routers; the pick.

## Approach

One interface, one router with three modes, one adapter over the API adapter's client; the lab verb records and harvests, so the evidence path is the lab's, not a new one.

## Footprint

- packages: internal/core/decide, internal/core/lab, internal/adapter/openaiapi, internal/core/oracle/config.go, internal/core/oracle/judgement_mode_test.go, internal/core/oracle/connect_guide.go, internal/core/implement/loop/drain_decide.go, internal/core/implement/loop/drain_decide_test.go, internal/surface/cli/lab.go, internal/surface/cli/lab_surface_test.go, internal/README.md, commands/lab.md, .abcd/development/brief/04-surfaces/31-lab.md, .abcd/development/brief/05-internals/02-adapters.md, .abcd/development/brief/05-internals/03-configuration.md, .abcd/development/intents/planned/itd-2609221009495079-abcd-s-typed-judgements-can-run-through.md, .abcd/work/issues/open/iss-2610262033055327-the-decision-adapter-intent-treats.md, .abcd/work/issues/resolved/iss-2610262033055327-the-decision-adapter-intent-treats.md, .abcd/work/issues/open/iss-2610262037348735-the-decision-adapter-intent-needs-lab.md, .abcd/work/issues/resolved/iss-2610262037348735-the-decision-adapter-intent-needs-lab.md
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
