---
schema_version: 1
id: "iss-2608230751437480"
slug: "site-dot-separator-design-pass"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "user-observation"
found_at: "site-src/site.css"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker: should the separator and typography design pass ruled on 2026-08-23 become an intent this cycle?"
remedy: "Waits on the separator design-pass ruling (an intent this cycle, or not): if planned: replace the literal ' · ' in the header pill, datelines and panel summaries with one separator class whose CSS generated content carries empty alternative text, and show multi-field metadata as a description list, proven by a site-render test that those three places emit no literal middle dot; if not planned: re-defer naming the cycle that takes it."
---

Site-wide ' · ' dot separators (header pill, datelines, panel summaries) read poorly; a deliberate separator/typography design pass is PLANNED per maintainer ruling 2026-08-23 (report 3b).

## Remedy grounds (2026-09-29)

- A separator drawn by CSS with empty alternative text (content: '·' / '') keeps a decorative mark out of speech output (https://developer.mozilla.org/en-US/docs/Web/CSS/content, checked 2026-09-29); key-and-value metadata reads better as a description list, the pattern the GOV.UK summary list uses (https://design-system.service.gov.uk/components/summary-list/, checked 2026-09-29).
- Scope stays on the three places the record names: the dots inside the relationship chart's canvas labels (site-src/record.js) are drawn text, not markup.
- Rejected: swapping the dot for another glyph, which leaves the reading problem and the screen-reader noise in place.
