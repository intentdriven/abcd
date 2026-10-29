---
schema_version: 1
id: "iss-2610230235262742"
slug: "the-run-state-s-version-checks-each"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "review pair of the run-state schema-15 bump, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/state.go"
remedy: "Route every stored RouteRecord through one visitor the checks share (each check a function of a route), and drive readStateIn's refusals from a table of {highest version, check}, so a new route slot or a new field is added in one place"
---

The run state's version checks (metered, pinned, overridden and rounded in internal/core/implement/loop/state.go) each repeat their own walk over the lanes' receipts and validation rounds, and readStateIn repeats one case per check and version bound. A place that stores a route added later (a third slot holding a RouteRecord) has to be added to each walk by hand, and one missed walk silently accepts an older file carrying a newer field instead of refusing it.
