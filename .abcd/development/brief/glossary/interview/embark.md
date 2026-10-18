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
embark, and the [glossary README](../README.md)'s grill section records the design. What ships is
the planning interview, which the `/abcd:intent` plugin page runs with the person present and the
`interview` sub-verb runs in a plain Terminal, and the record-grill leg of `/abcd:ideate`'s
admission gauntlet. The plugin page's planning interview opens with its pre-pass instead, which
reads the draft against the brief's invariants, the principles and the other intents before the
first question (the plain-Terminal interview reads the planning brief that pass writes, when there
is one), and both ask one question at a time rather than in rounds. In the design, when a human
invokes `/abcd:intent grill`, the oracle does not ask a freeform question immediately. Instead, it:

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

- "The embark phase identified three vague nouns: 'session', 'context', and 'transport'."
- "After embark, the human had three questions to answer before round 2 began."

## Related terms

- [record families](../core/record-families.md) — the one page that maps the record families and how they relate
- [session](session.md) — the full grill lifecycle that begins with embark
- [intent](../core/intent.md) — the artefact the oracle reads during embark
