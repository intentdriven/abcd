---
id: itd-169
slug: an-agent-loop-that-stops-says-who-it-is
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

Escalation reaches the product thinker only for what they can answer: what should exist, whether what was delivered matches it, and trade offs that change the design. Everything else stops at the facilitator, including every question about evidence, technique, and whether a criterion was verifiable at all. A stop addressed to the product thinker that they cannot answer is a defect in the stop, not a failure of the reader ([adr-2609151528057260](../../decisions/adrs/2609151528057260-three-roles-who-each-artefact-addresses.md)).

Every stop carries its addressee as data, so a surface can show whose question it is before anyone reads the words. The addressee is not a prefix somebody remembers to type: it travels with the stop, survives serialisation, and is what a renderer keys on.

A stop that asks a conjectural question carries its options under `widen-options-never-recommend`: at most three, the null answer always among them, none marked as preferred, and the non-recommendation stated rather than left to be inferred. A stop that reports a settled fact carries no such marker, because hedging a fact is false balance.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [autonomous-loops SOTA note](../../research/notes/2026-10-23-autonomous-loops-sota.md).

- **A standard envelope exists for the question half.** MCP elicitation
  (specification 2025-11-25, which supersedes the 2025-06-18 shape) sends a
  `message` and, in form mode, a `requestedSchema` limited to flat primitives,
  with titled choices as `oneOf: [{const, title}]` (multi-select as `anyOf`
  inside an array; `enumNames` is gone), and takes a three-action answer:
  `accept` (chose and submitted), `decline` (explicitly refused), `cancel`
  (dismissed without choosing); servers "MUST NOT use form mode elicitation to
  request sensitive information", a URL mode sends the person to a page
  out of band with consent and a completion notification, and clients "MUST
  provide clear indication of which server is requesting information". It has
  no addressee, no judgement-versus-fact marker and no non-recommendation
  statement.
- **The host routes that envelope today.** An `Elicitation` hook receives
  `mcp_server_name`, `message`, `mode`, `url`, `elicitation_id` and
  `requested_schema` and may answer with the same three actions, so that "The
  server gets your answer and no dialog appears"; the host's `AskUserQuestion`
  offers options plus an `Other` row, its questions "stay open until you
  answer them", and an optional `askUserQuestionTimeout` closes one so that
  "Claude proceeds on its own judgment", a timer that "never starts" in a
  background session or while connected to remote control (Claude Code hooks
  and tools reference, re-read in this pass).
- **The waiting state is data elsewhere too.** A2A puts `input-required` and
  `auth-required` on the task as interrupted, non-terminal states, resumed on
  the same task id (A2A 1.0). LangGraph's interrupt payload is any
  JSON-serialisable value and the graph waits indefinitely; the OpenAI Agents
  SDK stores pending approvals as a durable run state, warns that deserialising
  one does not authenticate it, and advises a version marker (framework docs).
- **Addressing a role is incident practice.** An escalation policy's rules name
  users or schedules, not a person by hand, and acknowledgement does not
  resolve (PagerDuty docs). W3C PROV's `actedOnBehalfOf` names a facilitator
  relaying for a product thinker.

**Path 2**: a stop record of abcd's own whose question half is elicitation's
envelope (message, flat schema, a `oneOf` of at most three `const`/`title`
options with the null answer as a named option, `accept|decline|cancel`,
and URL mode as the standard hand-off to a page) and whose header carries the
addressee role, the kind (judgement or fact), the non-recommendation flag, the
record concerned and a schema version; `AskUserQuestion` is the first renderer
and the hook's three actions are the answer's values, so a web page or an MCP
client renders the same record later. "None of these" is an enum value, never
`cancel`; the answer is appended to the record, never loaded from an
unauthenticated snapshot. No Go dependency. The independent fit-challenge runs
at plan time.

## Open Questions

**Decided 2026-08-29.** The addressee is first-class data on the stop, not a rendering convention, because the terminal is not the surface that matters most and a convention does not survive the trip to a surface abcd owns.

**Decided 2026-08-29 (product thinker).** A stop nobody answers waits indefinitely by default and stays visible as waiting on the product thinker. Escalating to the facilitator after a period is available but is not the default, and it is configured where the rest of the facilitator's involvement is configured (itd-174), so a repository that wants the work to keep moving can say so and a repository that does not is never overridden by a timer it did not set.

- (SOTA 2026-10-23) Whether `decline` (an explicit refusal to answer) and
  `cancel` (a dismissal that leaves the stop open) are both kept as answer
  values, or only one, given that the null option already covers "none of
  these".
- (SOTA 2026-10-23) Whether the host's answer-for-the-user path (the
  `Elicitation` hook, the question timeout) is ever allowed on a stop of kind
  judgement, or only on a stop that reports a fact.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
