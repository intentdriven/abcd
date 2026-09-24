---
schema_version: 1
id: "iss-2609231006554434"
slug: "the-user-scope-inventory-omits-the-run-store-abcd-implement"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md"
resolution: "Both user-scope inventories name ~/.abcd/runs/<root-sha>/."
impact: internal
resolved_by:
  commit: "6fcf0e3f"
---

The user-scope inventory omits the run store: abcd implement keeps its run state in ~/.abcd/runs/<root-sha>/, but neither the tree in 04-surfaces/01-ahoy.md nor the paragraph in 05-internals/03-configuration.md, which say they are one list that must agree, names it.

## Grounds

- pursued: we expect the two inventories to list every store abcd keeps under ~/.abcd; shown wrong if a store abcd writes there is absent from either
