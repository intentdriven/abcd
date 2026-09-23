---
schema_version: 1
id: "iss-2608231024022535"
slug: "site-print-nodelist-scroll-window-slices"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "agent-verification"
found_at: "site-src/site.css"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (design call owed: cap, hide or accept the printed relationship list). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

The relationship page's record list is a scroll window (.nodelist max-height:520px, overflow:auto), so printing it captures only the first 520px and slices the entry at that boundary: 3 of 778 entries reach paper, the third cut mid-sentence. Lifting the cap in print prints all 778 and turns a 1-page document into 96 — precisely the outcome the stylesheet already rejects on the record. This is a design call, not a defect fix: cap the printed list at a stated number of entries, hide it in print and say so, or accept the length. Measured 2026-08-23.