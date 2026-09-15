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
not_to_be_confused_with: null
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# persona

A **persona** is a named, role-typed placeholder character from `.abcd/development/personas.json`.
Personas (Alice, Bob, Carol, etc.) represent real user archetypes in press releases and design
documents without embedding real names or PII. Each persona has a role hint (e.g., "product lead",
"solo founder") and is selected by matching the role the document needs.

## When to use

Use a persona whenever a press release, intent document, or design fiction needs a human voice or
quote. Always pick from the registry — never invent names. Selection is by role, never by
name: the document's audience determines the role, and the role's registered name is used.

## When NOT to use

Do not call a persona a "user" (too generic and erases the named-character framing) or an "actor"
(UML connotations). Do not use real names or team member names.

## Examples

- "Said Iris, product lead: 'Grill caught the ambiguity in three minutes.'"
- "Alice (product engineer) would use `/abcd:intent grill` before every plan submission."

## Related terms

- [brief](brief.md) — personas appear in briefs as the primary audience
- [intent](intent.md) — persona quotes anchor the press-release section of intents
