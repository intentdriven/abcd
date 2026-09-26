---
schema_version: 1
id: "iss-2609252159470046"
slug: "the-lint-chapter-s-section-1-1-says-prose-citation-resolves"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "The chapter names ten stores and lists rfm among the non-citation families."
impact: internal
resolved_by:
  commit: "49b5400f0c3579eca929aaf7f2015ee2600a2290"
---

The lint chapter's section 1.1 says prose_citation_resolves scans nine record stores and lists rdi, dsp, rdg, adm and srp as the non-citation stores, while .abcd/record-lint.json names ten under that rule: the reframe store rfm was added to the config and not to the sentence.

## Grounds

- pursued: the chapter's count and list agree with prose_citation_resolves' record_stores in .abcd/record-lint.json; a store added to that rule without the sentence moving would show it wrong again
