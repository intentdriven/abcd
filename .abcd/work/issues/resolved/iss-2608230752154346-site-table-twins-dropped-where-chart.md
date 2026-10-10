---
schema_version: 1
id: "iss-2608230752154346"
slug: "site-table-twins-dropped-where-chart"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "user-observation"
found_at: ".abcd/development/specs/closed/spc-38-alice-opens-the-record-explorer-and.md"
resolution: "Fixed in the 2026-08-23 manual-test triage pass; verified by rebuild, the seven site-check gates, the overflow audit, and a screenshot of the affected route compared against the report that raised it."
impact: fix
---

Maintainer re-rule 2026-08-23 amends spc-38's 'every visual has a table twin': a table twin is dropped wherever the visual already presents the same labels and numbers as text. Drop the twins on the dashboard seg bars, Assisted-by trailers, and the Genealogy release-cadence list; the cadence chart absorbs its dates; the relationship chart's Browse-as-a-list twin stays (spc-39 AC 4 keyboard path). Spec amendment to be recorded with the change (reports 11, 15, 20).