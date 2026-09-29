---
schema_version: 1
id: "iss-371"
slug: "the-persona-registry-lint-checks-names-not-roles-itd-114-shi"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "manual-capture"
found_at: ".abcd/development/personas.json"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (autonomous run A, 2026-09-29): a role-agreement check fails about fifteen existing quotes in shipped and planned intents whose role sits outside the named persona's role_hints (Kira and Alice as a maintainer, Carol as a facilitator, Iris as a technical facilitator), and quotes have since been ruled to attribute by role (itd-2609212137129937). Whether the existing quotes are rewritten, the roster widened to the roles the corpus uses, or the check only warns is a choice about the roster, not an implementer's."
---

The persona_registry lint checks names, not roles: itd-114 shipped 'Bob, a maintainer' (Bob is registered staff engineer; the maintainer role is Kira's) and itd-115 has 'Carol, a facilitator' (Nia's role) — both passed lint. The selection-by-role rule (personas.json, itd-79) has no detector for role-name agreement