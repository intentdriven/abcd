---
term: intent
bounded_context: core
definition: A press-release-shaped description of a feature written before implementation begins, capturing the user problem, proposed solution, and success criteria.
aliases: ["feature intent", "intent document"]
forbidden_synonyms: ["ticket", "story", "issue", "requirement"]
status: stable
introduced_in: phase-1
starts_when: null
ends_when: null
not_to_be_confused_with: core/spec
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# intent

An **intent** is the highest-leverage authoring moment in abcd's workflow. Written in a
press-release style (present tense, customer-voice), it commits to the outcome before any
technical solution is chosen. The intent is frozen at promotion time and is an immutable input
artefact — it is never edited after `/abcd:intent plan` is run.

## When to use

Use "intent" when referring to the human-authored document that precedes implementation. Intent
files live under `.abcd/development/intents/` and carry a slugged identifier (e.g.,
`itd-27`).

## When NOT to use

Do not call an intent a "ticket", "story", or "requirement" — these terms carry Jira/Scrum
connotations that conflict with abcd's press-release framing. Do not use "spec" — a spec is the
specced *implementation* block, not the human-authored *intention*.

## Examples

- "The intent for `itd-27` describes the grill sub-verb from the user's perspective."
- "`/abcd:intent grill itd-27` interrogates that intent before it becomes a spec."

## Related terms

- [brief](brief.md) — the project-level root document (intent is feature-level)
- [spec](spec.md) — the implementation spec derived from an intent
- [oracle](oracle.md) — used to review intents at promotion time
