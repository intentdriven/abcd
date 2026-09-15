---
term: transport
bounded_context: core
definition: The mechanism by which curated context and artefacts are packaged and delivered to an oracle for review or reasoning.
aliases: ["context transport", "review transport"]
forbidden_synonyms: ["API call", "request", "prompt", "chat"]
status: stable
introduced_in: phase-1
starts_when: null
ends_when: null
not_to_be_confused_with: core/oracle
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# transport

The **transport** is the layer between the abcd pipeline and the oracle. It
handles packaging (selecting which files, diffs, and context snippets to
include), delivery (handing the material to the host's subagent dispatch when the
oracle is host-delegated, or to a wired oracle adapter — native, CLI, API, or
MCP), and receipt (capturing the structured response for parsing). The transport
is distinct from the oracle itself — different transports can deliver to the same
oracle model, and the same transport can route to different oracle models.

## When to use

Use "transport" when referring to the mechanism used to send context to an
oracle, whether that is the host's subagent dispatch (the default) or a wired
oracle adapter.

## When NOT to use

Do not call the transport an "API call" (too implementation-level), "prompt"
(conflates content with delivery), or "chat" (elides the structured review
framing).

## Examples

- "The impl-review skill hands the scoped diff to the oracle through the host's
  subagent dispatch."
- "The transport packages the frontmatter, diff, and schema into a single
  context window."

## Related terms

- [oracle](oracle.md) — the AI model that receives context through the transport
- [intent](intent.md) — one of the artefact types transported for review
