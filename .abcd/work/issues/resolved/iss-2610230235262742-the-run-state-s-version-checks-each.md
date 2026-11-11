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
resolution: "Every stored route record goes through one visitor (State.eachRoute) that the route checks share, each a function of one route, and the read's refusals come from one table of highest version and check (versionChecks). The walk the old code missed is closed: effort_asked on a receipt's route in a file of version 24 or lower is now refused. The detector test is not on the branch: its file is outside the lane's hold and the widen was refused for want of a git credential (report.md); its source is in the lane directory as state_routes_test.go.detector."
impact: fix
resolved_by:
  commit: "57385549ecef8f1a6565109fb77cb3276764751b"
---

The run state's version checks (metered, pinned, overridden and rounded in internal/core/implement/loop/state.go) each repeat their own walk over the lanes' receipts and validation rounds, and readStateIn repeats one case per check and version bound. A place that stores a route added later (a third slot holding a RouteRecord) has to be added to each walk by hand, and one missed walk silently accepts an older file carrying a newer field instead of refusing it.

## Grounds

- pursued: a file of version 24 or lower carrying effort_asked on a receipt's route is refused naming it, and every other refusal keeps its wording and order; a read that accepts such a file, a changed refusal text in the loop package's version tests, or a route slot on State that eachRoute does not visit would show it wrong
