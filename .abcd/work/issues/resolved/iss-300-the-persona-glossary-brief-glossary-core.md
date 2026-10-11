---
schema_version: 1
id: "iss-300"
slug: "the-persona-glossary-brief-glossary-core"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: ".abcd/development/brief/glossary/core/persona.md"
resolution: "Rewrote persona.md to the role-first selection rule and re-roled its worked examples (Iris for product lead; Alice as product engineer)."
impact: fix
---

The persona glossary (brief/glossary/core/persona.md) and brief/01-product/05-personas.md say the persona picker 'chooses at random', contradicting personas.json and itd-79 whose rule is 'selection is by role, never by name'; the glossary's worked examples also assign off-role names (Carol as product lead, Alice as developer)
## Evidence

- `.abcd/development/personas.json` convention: "the audience picks the ROLE; the role
  determines the name — never pick a name directly", echoed by
  `.abcd/development/intents/disciplines/itd-79-persona-registry.md` ("by role, never by name").
  The `persona_registry` lint (`internal/core/lint/persona.go`, blocker) and the 2026-07-08 /
  2026-08 DECISIONS entries confirm this is the shipped, reconciled rule; there is no
  random-picker code in the tree.
- `.abcd/development/brief/glossary/core/persona.md` says "Match the role hint when the role
  matters; pick randomly otherwise", and its worked examples assign `Carol, product lead`
  (registry: product lead → Iris; Carol → engineering manager) and `Alice (developer)`
  (registry: Alice → solo founder / product engineer).
- The `brief/01-product/05-personas.md` "chooses at random" half is already filed as open
  iss-49 (record-cosmetics batch); the novel finding is the glossary file.

## Adversarial review

CONFIRMED by an independent refuter (glossary half is novel; personas.md half is prior art
iss-49). Fix confined to `persona.md`: state the role-first rule and re-role the two examples
(Iris for product lead; Alice with a registered hint).
