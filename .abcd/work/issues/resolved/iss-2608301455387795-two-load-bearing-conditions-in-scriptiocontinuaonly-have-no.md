---
schema_version: 1
id: "iss-2608301455387795"
slug: "two-load-bearing-conditions-in-scriptiocontinuaonly-have-no"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-179-round-4-recheck"
found_at: "internal/core/grounds/grounds.go"
resolution: "the two load-bearing conditions in scriptioContinuaOnly - the empty-unit guard and the letter test - each have a test that fails when it is deleted, so a text whose script does have word breaks is never told the floor asks for letters where it has none"
impact: internal
resolved_by:
  intent: "itd-179"
---

two load-bearing conditions in scriptioContinuaOnly have no test so deleting either leaves the whole suite green while the refusal names a property the text lacks

## Grounds

- pursued: we expect an untested condition to be deleted by a later reader who cannot see what holds it, and a mutation-killing test is the only record of what it holds
