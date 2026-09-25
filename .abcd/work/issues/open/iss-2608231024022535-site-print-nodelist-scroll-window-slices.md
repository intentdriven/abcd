---
schema_version: 1
id: "iss-2608231024022535"
slug: "site-print-nodelist-scroll-window-slices"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "agent-verification"
found_at: "site-src/site.css"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): For the printed relationship list: cap it, hide it, or accept it?"
---

The relationship page's record list is a scroll window (.nodelist max-height:520px, overflow:auto), so printing it captures only the first 520px and slices the entry at that boundary: 3 of 778 entries reach paper, the third cut mid-sentence. Lifting the cap in print prints all 778 and turns a 1-page document into 96 — precisely the outcome the stylesheet already rejects on the record. This is a design call, not a defect fix: cap the printed list at a stated number of entries, hide it in print and say so, or accept the length. Measured 2026-08-23.