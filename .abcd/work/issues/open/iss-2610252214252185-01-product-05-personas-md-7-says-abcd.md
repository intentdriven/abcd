---
schema_version: 1
id: "iss-2610252214252185"
slug: "01-product-05-personas-md-7-says-abcd"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S02"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/05-personas.md"
remedy: "Drop the clause and point at § 4 for the persona_registry lint only."
---

01-product/05-personas.md:7 says '/abcd:intent '<text>' picks a persona for the customer quote'; the create has no persona logic (internal/core/intent/create.go), and 05-intent.md § 4 claims nothing of the kind.
