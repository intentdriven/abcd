---
id: itd-169
slug: an-agent-loop-that-stops-says-who-it-is-asking-and-what-it-n
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
---

# An agent loop that stops says who it is asking and what it needs, and never simply halts

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

Without a rule about when agents may proceed alone, every gate is a candidate for a human wait, and the framework's premise reverses: the human becomes the bottleneck it exists to remove. The agents run autonomously and stop only to obtain a verdict.

A stop is an event with an addressee and a question, never an unexplained halt. It names the role it is asking, states what it needs, and carries the options that would answer it. Work that is a step in the loop rather than a judgement is performed, not queued for somebody.

Escalation reaches the product thinker only for what they can answer: what should exist, whether what was delivered matches it, and trade offs that change the design. Everything else stops at the facilitator, including every question about evidence, technique, and whether a criterion was verifiable at all. A stop addressed to the product thinker that they cannot answer is a defect in the stop, not a failure of the reader ([adr-2609151528057260](../../decisions/adrs/2609151528057260-three-roles-who-each-artefact-addresses-and-when-the-loop-st.md)).

Every stop carries its addressee as data, so a surface can show whose question it is before anyone reads the words. The addressee is not a prefix somebody remembers to type: it travels with the stop, survives serialisation, and is what a renderer keys on.

A stop that asks a conjectural question carries its options under `widen-options-never-recommend`: at most three, the null answer always among them, none marked as preferred, and the non-recommendation stated rather than left to be inferred. A stop that reports a settled fact carries no such marker, because hedging a fact is false balance.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

**Decided 2026-08-29.** The addressee is first-class data on the stop, not a rendering convention, because the terminal is not the surface that matters most and a convention does not survive the trip to a surface abcd owns.

**Decided 2026-08-29 (product thinker).** A stop nobody answers waits indefinitely by default and stays visible as waiting on the product thinker. Escalating to the facilitator after a period is available but is not the default, and it is configured where the rest of the facilitator's involvement is configured (itd-174), so a repository that wants the work to keep moving can say so and a repository that does not is never overridden by a timer it did not set.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
