---
schema_version: 1
id: "iss-2609231715081185"
slug: "the-persona-attribution-checks-only-see-the-said-name-form-a"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A, pressbuild fix round 2 (review2-pressbuild)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/persona.go"
resolution: "personaAttrRe matches said and says, so persona_registry and the release page headline refusal both see a says attribution."
impact: fix
resolved_by:
  commit: "ee0cce36"
---

The persona-attribution checks only see the `said <Name>,` form. A headline in a release page that attributes a quote to a persona with `says <Name>,` is not refused as an unverified quote, and record-lint's persona_registry rule does not see a `says` attribution either, so an unregistered persona quoted with `says` passes both. The page's `blockquote` refusal and `lint.PersonaAttribution` share one regex (personaAttrRe) that matches `said` only, while the verbatim-quote check accepts both verbs.

## Grounds

- pursued: an unregistered persona quoted with says is refused by persona_registry and a says headline attribution by the page ingest; either passing would show it wrong
