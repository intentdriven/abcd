---
term: persona
bounded_context: core
definition: A placeholder stakeholder character drawn from the abcd personas registry, used in press releases, intents, and design documents to represent a real user archetype without using real names.
aliases: ["placeholder persona", "stakeholder persona"]
forbidden_synonyms: ["user", "customer", "actor", "role"]
status: stable
introduced_in: phase-1
starts_when: null
ends_when: null
not_to_be_confused_with: core/record-families
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# persona

A **persona** is a named, role-typed placeholder character from `.abcd/development/personas.json`.
Personas (Alice, Bob, Carol, etc.) represent real user archetypes in press releases and design
documents without embedding real names or PII. Each persona has a role hint (e.g., "product lead",
"solo founder") and is selected by matching the role the document needs.

## When to use

Use a persona whenever a press release, intent document, or design fiction needs the voice or
quote of a modelled user. Always pick from the registry — never invent names. Selection is by
role, never by name: the document's audience determines the role, and the role's registered
name is used.

A real role abcd addresses is not a modelled user and needs no persona: the
[product thinker](product-thinker.md) may be quoted by role, unnamed ("said a product thinker
who ran the first ledger drains by hand"), as many intents' press releases already do. A role
alone invents no name; a name, where one is used, is still a registered persona's.

## When NOT to use

Do not call a persona a "user" (too generic and erases the named-character framing) or an "actor"
(UML connotations). Do not use real names or team member names.

## Examples

- "Said Iris, product lead: 'The planning interview caught the ambiguity in three minutes.'"
- "Alice (product engineer) would run the planning interview (`abcd intent interview`) before every plan submission."

## Related terms

- [record families](record-families.md) — the one page that maps the record families and how they relate
- [brief](brief.md) — personas appear in briefs as the primary audience
- [intent](intent.md) — persona quotes anchor the press-release section of intents
