---
schema_version: 1
id: "iss-2608301747006182"
slug: "no-gate-requires-a-terminal-folder-record-to-carry-a-grounds"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "itd-179-round-5-builder"
found_at: "internal/core/lint"
---

no gate requires a terminal folder record to carry a grounds entry so fourteen resolved records have none

Reported by the round-5 builder. PRE-EXISTING in substance -- the old rule
checked the grammar of a `grounds` value only if one was present -- but far
more visible now that grounds is a section rather than an optional key.

A gate demanding one would be red today: the fourteen records that reached
`resolved/` before grounds existed carry none, and nothing backfills them. So
the gate cannot simply be armed; it needs a cutover rule saying from when the
obligation runs.
