---
schema_version: 1
id: "iss-2608231339287073"
slug: "genealogy-chart-needs-a-design-rethink"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "user-observation"
found_at: "internal/core/site/timeline.go"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (design owed: what the genealogy chart should answer (future intent seed)). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

The genealogy chart is a draft and needs rethinking as a designed picture (future intent seed). Its five lanes over one axis carry real facts — releases, decisions, intents, specs, principles, and issues as a per-day histogram — but the drawing does not yet read: the supersession arcs and their dashed stubs cross the lanes and collide with each other and with the marks they point at (adr-4, adr-8 and adr-18 overprint), a crowded day collapses into a capsule whose count says nothing about what is inside it, the lanes are unlabelled beyond a store name and a total, and the first-commit marker cuts the whole picture without saying why. It reads as five separate strips sharing an axis rather than as one genealogy. The rethink is a design question, not a defect: what should this picture ANSWER, and what shape answers it. Docked folded at the top of the dashboard as of 2026-08-23.