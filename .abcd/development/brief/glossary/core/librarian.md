---
term: librarian
bounded_context: core
definition: The engine an agent runs to read the sources a person gives for a subject, search around them along the nine lenses, and record a private reading pack whose every quote matches the text filed for its source.
aliases: []
forbidden_synonyms: []
status: draft
introduced_in: itd-2610141646126492
starts_when: null
ends_when: null
not_to_be_confused_with: [core/library, core/record-families]
versions: null
---

# librarian

The **librarian** does the reading so that ideate and the research command start from evidence, not from memory. An agent runs it (`abcd library pack`); the person never types it. It keeps what it fetches privately, reads a private source alone, and hands back a reading pack.

## When to use

For the agent-driven reading step behind ideate and the research command.

## When NOT to use

For the person's own collection of sources: that is the [library](library.md). For the shared note a question produces: that is a research note, written by the research command.

## Examples

- You give ideate a web address; the agent runs the librarian, which records a reading pack of the ideas, principles, claims and references in it.

## Related terms

[reading-pack](reading-pack.md), [lens](lens.md), [library](library.md)
