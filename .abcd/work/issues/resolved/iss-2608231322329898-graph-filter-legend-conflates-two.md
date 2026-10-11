---
schema_version: 1
id: "iss-2608231322329898"
slug: "graph-filter-legend-conflates-two"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "user-observation"
found_at: "site-src/record.js"
resolution: "Fixed in the 2026-08-23 manual-test triage pass; verified by rebuild, the seven site-check gates, the overflow audit, and a screenshot of the affected route compared against the report that raised it."
impact: fix
---

The relationship chart's filter legend is unreadable: it mixes two different encodings in one flowing list, so 'closed' appears to belong to specs and 'disciplines' sits between two lifecycle words. Two encodings are in play — COLOUR means the record's store (decisions, intents, specs, issues, principles) and BORDER STYLE means its lifecycle (open/planned hollow, drafts dashed, shipped/resolved/closed filled, superseded/wontfix faded) — and the legend runs them together as one wrapping row. Split it into two labelled sections: the coloured store dots, then the border styles beneath, so each encoding is read on its own. Disciplines currently share the principles colour and need their own, as a discipline is a distinct store to a reader even though the record files it under intents. Reported 2026-08-23.