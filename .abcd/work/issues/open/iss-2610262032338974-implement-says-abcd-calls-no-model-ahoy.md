---
schema_version: 1
id: "iss-2610262032338974"
slug: "implement-says-abcd-calls-no-model-ahoy"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 285 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/27-implement.md:507"
remedy: "none (filed automatically)"
finding_ref: cfk-8464a591409084be
---

Implement says abcd calls no model; ahoy has a delegating verb send its step to a configured provider itself

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/27-implement.md:507` — "the model service only by the verdict of a canary agent the lead runs, since abcd calls no model"
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:204` — "A delegating verb whose agent's `oracle.roles` entry points at a configured provider sends the step there itself"

27-implement probes the model service only through a canary agent the lead runs, and gives the reason that abcd calls no model. 01-ahoy says a delegating verb whose agent's oracle.roles entry points at a configured provider sends the step there itself, so abcd does call a model whenever a provider is configured. The premise the outage probe rests on is false in that case.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 285.
