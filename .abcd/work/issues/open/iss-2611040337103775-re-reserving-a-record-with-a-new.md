---
schema_version: 1
id: "iss-2611040337103775"
slug: "re-reserving-a-record-with-a-new"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "the v0.17.0 cut and the overnight lanes, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim"
remedy: "Make a repeated footprint on a live reservation add to it, with an explicit form to narrow, or refuse a narrowing without that form; a test re-reserves and asserts nothing was dropped."
---

Re-reserving a record with a new footprint replaces the old footprint, with only a warning: on 2026-11-04 adding .github/workflows/ci.yml to iss-2610181757540736 dropped three directories other live work relied on, until a coordinator noticed the warning and reserved all four again.
