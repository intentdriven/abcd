---
term: brief
bounded_context: core
definition: The living root document that holds a project's purpose, constraints, and success criteria — always the project's current state, revised in place as the project moves.
aliases: ["project brief", "brief doc"]
forbidden_synonyms: []
status: stable
introduced_in: phase-1
starts_when: null
ends_when: null
not_to_be_confused_with: core/intent
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# brief

The **brief** is the authoritative root document for a project. It establishes what the project
exists to do, the constraints it operates under, and the success criteria by which completion is
judged. It is written by a human stakeholder and kept as a living record: the brief is the project's
*current* state, carrying no version label and no archive directory, so every section is revised in
place as the project moves ([adr-5](../../../decisions/adrs/0005-brief-is-current-state.md)). History
lives in `git log`; inflection-point rationale lives in the ADRs.

## When to use

Use "brief" when referring to the top-level project specification document that lives at
`.abcd/development/brief/`. A project has exactly one brief.

## When NOT to use

Do not use "brief" to describe an intent (which is feature-scoped) or a spec (which is
implementation-scoped). The brief is project-wide; intents and specs are narrower.

## Examples

- "The brief says the target persona is Carol, not Alice."
- "This intent is out of scope for the brief's Phase 1 boundary."

## Related terms

- [intent](intent.md) — a press-release-shaped feature description within the project scope
- [voyage](voyage.md) — the operations namespace recording what abcd did to produce a lifeboat; a
  [disembark](disembark.md) run grounds the brief's structure section by section
