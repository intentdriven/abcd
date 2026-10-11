---
schema_version: 1
id: "iss-2608230751270193"
slug: "site-health-panel-wrap-and-unlabelled"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "user-observation"
found_at: "internal/core/site/explorer.go"
resolution: "Fixed in the 2026-08-23 manual-test triage pass; verified by rebuild, the seven site-check gates, the overflow audit, and a screenshot of the affected route compared against the report that raised it."
impact: fix
---

Record health panel entries wrap mid-phrase and the summary line '8 / 8 · 148' is three unlabelled numbers. Restructure to two lines per entry (fact, then explanation) and label the summary (report 3a).