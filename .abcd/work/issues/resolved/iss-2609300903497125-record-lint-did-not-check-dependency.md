---
schema_version: 1
id: "iss-2609300903497125"
slug: "record-lint-did-not-check-dependency"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/schema.go"
remedy: "Add two record-lint rules over the record_schema scan: stale_edge (a planned or draft intent's builds_on or blocked_by names an intent in superseded/, reported per edge with the chain's live successor, followed through the intent package's one supersession chain reader) and edge_cycle (a cycle in builds_on and blocked_by together, naming every record on it); arm both at warn while the known edges await rulings, then promote to blocker"
resolution: "record-lint's stale_edge rule flags a planned or draft intent whose builds_on or blocked_by names a superseded intent, naming the successor its chain ends at through the intent package's chain reader, and edge_cycle flags a builds_on/blocked_by cycle naming every record on it; both armed at warn"
impact: additive
resolved_by:
  commit: "8771f0cf4"
---

record-lint did not check dependency edges against supersession or cycles: an intent's builds_on or blocked_by naming a superseded intent passed (record_schema checks existence only), and so did two intents building on each other; the base tree carried eleven such stale edges and the runner and loop cycle with record-lint green

## Grounds

- pursued: record-lint over this tree reports exactly the two itd-2 edges and the three two-intent cycles at warn, and the fixture tests pin each rule; shown wrong by a superseded target or a cycle the rules miss, or by a finding on a live edge
