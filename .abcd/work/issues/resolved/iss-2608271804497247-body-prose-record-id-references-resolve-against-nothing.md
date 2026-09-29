---
schema_version: 1
id: "iss-2608271804497247"
slug: "body-prose-record-id-references-resolve-against-nothing"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: ".abcd/record-lint.json"
resolution: "prose_citation_resolves reads its extra_roots as well as the ten record stores, and the shipped config names .abcd/development whole, so a record id in the brief, a principle, the roadmap, a plan or a research note is gated like one in a record; the write-path check agrees. The first run found one unresolvable id (spc-82, the retired predecessor store's own triage note), added to the baseline. The existing record-lint gate was widened rather than the site export the record proposed, so there is still one resolver; the export stays on typed edges."
impact: internal
resolved_by:
  commit: "0e8ab1c66"
---

body-prose record-id references resolve against nothing: the context_citation_currency rule's record_stores mapping covers one file, and the site record export extracts only the eight typed frontmatter edges, so adr-N/itd-N/spc-N/iss-N tokens in body prose across the durable record are checked by no gate (the mechanism behind this review's dangling-id findings). Widen the existing gate rather than adding a second: extend the export's reference extraction to body-prose tokens, flow them into the same Unresolved set and site-baseline ratchet, and seed the baseline with the first run's backlog.

## Grounds

- pursued: every markdown file under .abcd/development is read for prose citations; an invented id in a brief chapter or principle that record-lint does not report would show it wrong
