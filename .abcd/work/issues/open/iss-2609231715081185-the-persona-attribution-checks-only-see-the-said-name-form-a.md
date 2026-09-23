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
deferred_after: "v0.9.0"
deferral_reason: "Deferred out loud by the pressbuild lane, fix round 3 of the 2026-09-23 run (review 3 nit). The fix is to widen personaAttrRe to match `says` as well as `said`, but that one regex is shared: lint.PersonaAttribution also drives record-lint's persona_registry rule over the whole committed record, so widening it for the release page widens that rule everywhere at once. The false positives it would raise across the record (prose that reports what someone says, not a quoted persona) need their own look before the change lands, which is next cycle's work, not this cut's."
---

The persona-attribution checks only see the `said <Name>,` form. A headline in a release page that attributes a quote to a persona with `says <Name>,` is not refused as an unverified quote, and record-lint's persona_registry rule does not see a `says` attribution either, so an unregistered persona quoted with `says` passes both. The page's `blockquote` refusal and `lint.PersonaAttribution` share one regex (personaAttrRe) that matches `said` only, while the verbatim-quote check accepts both verbs.
