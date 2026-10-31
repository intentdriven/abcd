---
id: itd-2609180517121254
slug: every-payload-a-host-hands-back-from-a
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

# Every payload a host hands back from a delegated step names the model that produced it and the number of agents that ran, and the ingesting verb refuses one that does not

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

Every payload a host hands back from a delegated step names the model that produced it and the number of agents that ran, and the ingesting verb refuses one that does not

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [autonomous-loops SOTA note](../../research/notes/2026-10-23-autonomous-loops-sota.md).

- **The field names exist, on different spans.** OpenTelemetry's GenAI
  conventions (status: Development) define `gen_ai.request.model` (on the
  create-agent span, "the exact name of the model requested"; on the
  invoke-agent span Recommended only where one model applies),
  `gen_ai.response.model` ("the name of the model that generated the
  response", defined on the model-call spans and not listed on the
  invoke-agent span at all), `gen_ai.provider.name`, `gen_ai.agent.id`,
  `gen_ai.agent.name`, `gen_ai.conversation.id` and `gen_ai.usage.input_tokens`
  / `output_tokens`, with input tokens including cached ones (OTel GenAI spans
  and agent spans). A per-agent `gen_ai.response.model`, and the agents list
  itself, are abcd's own composition of OTel names, not an OTel structure.
- **The host's run already carries the facts.** `system/init` reports the
  session's `model`; the result's `modelUsage` is keyed by model name and
  counts sub-agent requests where `usage` does not; every sub-agent message
  carries a `parent_tool_use_id` from which the nesting tree is rebuilt;
  `SubagentStop` carries `agent_id` and `agent_type` but no model; cost
  figures are client-side estimates (Claude Code and Agent SDK docs). So the
  per-model totals come from the final result, but the per-agent served model
  and tokens can only be rebuilt from the stream (`stream-json` messages with
  `parent_tool_use_id`, each carrying its model and usage), not from the
  result alone; and after a process crash the final `error_during_execution`
  result "may carry zeroed `usage`, `total_cost_usd`, and `modelUsage`", so a
  run that did work can hand back an empty agent list, and the refusal must
  say what it does with a zeroed result rather than treat it as no work.
- **Provenance words.** W3C PROV-O: an entity `wasAttributedTo` an agent, an
  activity `wasAssociatedWith` an agent, an agent `actedOnBehalfOf` another,
  and `SoftwareAgent` is a kind of `Agent`; the human the run acts on behalf of
  is the attribution rule's author.
- **Served is not asked.** The 2026-10-16 note measured the spread between
  provider pairings; the build verb's scope condition already says the model is
  recorded as the runner reported it and cannot be verified by the binary.

**Path 1 for the names, Path 2 for the mechanism**: the payload carries the
OTel attribute names verbatim (`gen_ai.provider.name`, `gen_ai.request.model`,
`gen_ai.response.model`) and an `agents` list keyed by `gen_ai.agent.id` with
each agent's served model and tokens, the per-agent `gen_ai.response.model`
being abcd's extension of a convention whose agent span does not define it;
the agent count is the number of distinct agent ids that ran, the top-level
one included; the ingesting verb refuses a payload with no
`gen_ai.response.model` or an empty agent list, and a zeroed crash result is
recorded as "unknown" with the crash named rather than refused as no work; the
per-agent fields are read from the stream, the per-model totals from the
result; the payload pins the convention version because the status is
Development. Names only, no OTel SDK, no Go dependency. The independent
fit-challenge runs at plan time.

## Open Questions

- (SOTA 2026-10-23) Whether "the number of agents that ran" counts distinct
  agent ids (the top-level one included) or only sub-agents, and whether a
  forked skill with a `forked-command-` parent counts as an agent.
- (SOTA 2026-10-23) Whether the payload records both the requested and the
  served model, and refuses only when the served one is missing.
- (SOTA 2026-10-23) Which convention version the payload pins, and what the
  ingesting verb does with a payload pinned to a newer one.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
