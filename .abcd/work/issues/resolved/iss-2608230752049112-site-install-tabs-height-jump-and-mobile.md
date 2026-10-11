---
schema_version: 1
id: "iss-2608230752049112"
slug: "site-install-tabs-height-jump-and-mobile"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "user-observation"
found_at: "site-src/site.css"
resolution: "Fixed in the 2026-08-23 manual-test triage pass; verified by rebuild, the seven site-check gates, the overflow audit, and a screenshot of the affected route compared against the report that raised it."
impact: fix
---

Install tab panel takes the height of the active tab, so switching to a thin tab (Windows) collapses the layout; on iPhone the tab strip wraps into three broken rows (Plugin / CLI+OS / orphaned Build). Stack panels in one grid cell so the tallest wins; give the strip a deliberate narrow-width layout per the existing <=700px nav pattern (reports 9, 14b).