---
term: oracle
bounded_context: core
definition: An AI model invoked to review, reason over, or validate a project's artefacts — host-delegated by default, or reached through an opt-in oracle adapter.
aliases: ["reviewer", "AI reviewer"]
forbidden_synonyms: ["LLM", "model", "bot", "AI", "chatbot"]
status: stable
introduced_in: phase-1
starts_when: null
ends_when: null
not_to_be_confused_with: core/transport
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# oracle

An **oracle** is abcd's abstraction for an AI model used in a review or reasoning
role. By default the oracle is **host-delegated**
([adr-25](../../../decisions/adrs/0025-host-delegated-llm-default.md)): abcd's
core does the deterministic work, hands a prompt to the host's subagent dispatch,
and consumes the structured verdict (SHIP / NEEDS_WORK / MAJOR_RETHINK). When an
operator wants abcd to reach a model directly, a concrete **oracle adapter** —
native, CLI, API, or MCP — is wired behind the same seam. Referring to it as
"oracle" rather than by a generic model name emphasises its role as a decider
rather than a generator.

## When to use

Use "oracle" when referring to the AI model invoked for a review step in the
abcd pipeline. The oracle is host-delegated by default; a wired oracle adapter
reaches a model directly over the same seam.

## When NOT to use

Do not reduce the oracle to a generic model or bot in workflow documentation —
that elides its review-and-decide role. Do not confuse the oracle with the
transport that delivers context to it.

## Examples

- "The impl-review skill hands the diff to the host-delegated oracle and reads
  back its verdict."
- "The oracle returned NEEDS_WORK on round 6."

## Related terms

- [transport](transport.md) — the mechanism that delivers context to the oracle
- [intent](intent.md) — one of the artefact types that oracles review
