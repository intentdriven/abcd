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
technical solution is chosen. Planning (`/abcd:intent plan`) is the product thinker's sign-off on
that outcome, not a freeze of the file, and a planned intent's record still changes: its planning
interview may edit any part of it, and the intent sub-verbs write a hold, a target release,
dependency edges, the spec link, its kind and its impact judgement into the frontmatter, stamp
identities onto its scope conditions and append to its grounds.

## When to use

Use "intent" when referring to the human-authored document that precedes implementation. Intent
files live under `.abcd/development/intents/` and carry a slugged identifier (e.g.,
`itd-27`).

## When NOT to use

Do not call an intent a "ticket", "story", or "requirement" — these terms carry Jira/Scrum
connotations that conflict with abcd's press-release framing. Do not use "spec" — a spec is the
specced *implementation* block, not the human-authored *intention*.

## Examples

- "The intent for `itd-10` describes removing abcd from a project entirely, from the user's perspective."
- "`abcd intent interview itd-10` runs that intent's planning interview before it becomes a spec."

## Related terms

- [record families](record-families.md) — the one page that maps the record families and how they relate
- [brief](brief.md) — the project-level root document (intent is feature-level)
- [spec](spec.md) — the implementation spec derived from an intent
- [oracle](oracle.md) — used to review intents at promotion time
