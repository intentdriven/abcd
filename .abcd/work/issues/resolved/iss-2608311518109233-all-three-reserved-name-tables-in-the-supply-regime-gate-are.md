---
schema_version: 1
id: "iss-2608311518109233"
slug: "all-three-reserved-name-tables-in-the-supply-regime-gate-are"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "itd-185 fidelity audit rcp-fe3450ca55ff"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/ingest_regime.go"
resolution: "Pinned by content in ingest_tables_test.go: each reserved-name table is held to its criterion's literal enumeration with a count floor, presence, no extras, and each literal pushed through the gate; the closed vocabulary, the per-position body sets and the licence table are pinned the same way. Proven on scratch copies: emptying each row leaves the pre-existing suite green and fails the new pin (e5cca8ff)."
impact: internal
resolved_by:
  commit: "e5cca8ff7c230ceb4190bb75131ab583bad9f92c"
---

All three reserved-name tables in the supply-regime gate are mutation-vacuous: emptying any of them leaves the whole package green, because every test iterates the table under test rather than naming what it should contain. No test anywhere pins the literal field names that three acceptance criteria enumerate, so the criteria are satisfied by a gate that would still pass if the names it refuses were deleted. The signature registry has a floor asserting a minimum count; the reserved tables have none. The earlier vacuity sweep on this branch swept code BRANCHES and never data tables, which is why the shape was invisible to it: a guard whose behaviour is driven by a table is only as bound as the table is pinned. Also unpinned by the same reasoning: the per-position body field sets.

## Grounds

- pursued: a table-driven guard is only as bound as its table is pinned, so naming the criteria's literals in a test closes the vacuity; a later run that empties a row and stays green would show it wrong
