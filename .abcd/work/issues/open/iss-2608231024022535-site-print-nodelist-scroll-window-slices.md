---
schema_version: 1
id: "iss-2608231024022535"
slug: "site-print-nodelist-scroll-window-slices"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "agent-verification"
found_at: "site-src/site.css"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker: for the printed relationship list, cap it at a stated number of entries, hide it in print and say so, or accept the length?"
remedy: "Waits on the printed-relationship-list ruling (cap, hide and say so, or accept the length); at this base the print rule lifts the .nodelist cap but the list sits in a closed details.blist, so it prints as its summary alone: if hidden: add a print-only line to that summary naming the entry count and the page URL; if capped: on beforeprint open the disclosure and hide entries past a stated N, the summary saying first N of M; if accepted: open details.blist in print as details.more is; each proven by a site-render test on the emitted markup and stylesheet."
---

The relationship page's record list is a scroll window (.nodelist max-height:520px, overflow:auto), so printing it captures only the first 520px and slices the entry at that boundary: 3 of 778 entries reach paper, the third cut mid-sentence. Lifting the cap in print prints all 778 and turns a 1-page document into 96 — precisely the outcome the stylesheet already rejects on the record. This is a design call, not a defect fix: cap the printed list at a stated number of entries, hide it in print and say so, or accept the length. Measured 2026-08-23.

## Remedy grounds (2026-09-29)

- Read at the base: site-src/site.css:663-665 keeps the list twin closed in print and :685-686 lifts the scroll cap, and internal/core/site/graphpage.go:115-117 wraps the list in details.blist, so the sliced three entries the record measured no longer reach paper; what stays open is saying so on the page, which is the ruling's middle option.
- Print styling and the beforeprint hook: https://developer.mozilla.org/en-US/docs/Web/CSS/Guides/Media_queries/Printing (checked 2026-09-29). Rejected: resolving the record on the base's behaviour, since a silent omission is not one of the three answers the record offers.
