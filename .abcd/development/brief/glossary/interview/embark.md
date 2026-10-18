---
term: embark
bounded_context: interview
definition: The opening move of a grill session, a design target that does not ship, in which the oracle reads the target intent, identifies the primary ambiguities, and poses the first round of Socratic questions.
aliases: ["session opening", "grill opening"]
forbidden_synonyms: ["start", "begin", "introduction", "onboarding"]
status: stable
introduced_in: phase-1
starts_when: null
ends_when: null
not_to_be_confused_with: core/record-families
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# embark

**Embark** is the structured opening phase of a grill session. The grill is a **design target**,
not shipped: `abcd intent` registers no grill sub-verb, so nothing that ships today performs an
embark. The [glossary README](../README.md)'s grill section records the design. Two things ship
instead. One is the planning interview: the `/abcd:intent` plugin page runs it with the person
present, and the `interview` sub-verb runs it in a plain Terminal. On a draft, the plugin page's
interview opens with its pre-pass, which reads the draft against the brief's invariants, the
principles and the other intents before the first question; the pre-pass refuses any record that
is not a draft. The plain-Terminal interview reads the planning brief that pass writes, when there
is one. Both forms ask one thing at a time, not in rounds. The other is the record-grill leg of
`/abcd:ideate`'s admission gauntlet, which reads the record and asks no one anything. In the
design, when a human invokes `/abcd:intent grill`, the oracle does not ask a freeform question
immediately. Instead, it:

1. Reads the full intent (press release + scope + constraints).
2. Identifies the top 2–3 areas of vagueness or hidden assumption.
3. Poses the first round of Socratic questions (max 3) tagged with move names
   (Definition / Elenchus / Dialectic / Maieutics / Counterfactual / Generalization).

This structured embark prevents the session from starting with a generic "tell me more"
prompt and ensures the oracle's first questions are grounded in the actual intent content.

## When to use

Use "embark" when describing the oracle's initialisation behaviour at the start of a designed grill
session. The term captures the intentional, navigational quality of beginning — the oracle sets
a heading before asking anything.

## When NOT to use

Do not use "start" or "begin" — these are too generic and do not convey the structured nature of
the opening. Do not use "onboarding" (user-product connotations) or "introduction"
(informal/social connotations).

## Examples

- "In the design, the embark phase identifies three vague nouns in an intent: 'session', 'context', and 'transport'."
- "In the design, after embark, the human has three questions to answer before round 2 begins."

## Related terms

- [record families](../core/record-families.md) — the one page that maps the record families and how they relate
- [session](session.md) — the full grill lifecycle that begins with embark
- [intent](../core/intent.md) — the artefact the oracle reads during embark
