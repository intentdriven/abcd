---
schema_version: 1
id: "iss-2609251734061160"
slug: "consolidation-docs-residue"
severity: "minor"
category: "documentation"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/surface.md"
resolution: "The surface glossary's example names abcd lint identity as the report, and the createIntentFromText comment names only the quoted-text create, with its splice into resolveProductionMode's doc undone."
impact: internal
resolved_by:
  commit: "d5921418"
---

Two passages still describe spellings the verb consolidation retired: the surface glossary's example says bare abcd identity reports each surface's verdict and writes nothing, though bare identity now refuses and the report is abcd lint identity; and a cli.go comment calls abcd intent new the deprecated alias, though it is removed

## Grounds

- pursued: no passage in the brief, docs or code comments presents a retired spelling as live; shown wrong by a grep for the retired forms outside the stub tests
