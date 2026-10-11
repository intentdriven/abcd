---
schema_version: 1
id: "iss-2608211146562243"
slug: "rd001-doc-omits-sha-keyed-receipt"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-b-round-4"
found_at: ".abcd/development/brief/05-internals/06-lint.md"
resolution: "Documented the sha-keyed receipt-directory exemption at both RD001 definition sites (06-lint.md and reviews/README.md), naming the receipt_gate integrity check that governs that class. The script is unchanged — it already implements the exemption correctly."
impact: internal
---

RD001 is documented as unconditional in both definition sites (.abcd/work/reviews/README.md and brief/05-internals/06-lint.md name only the root README as exempt), but scripts/check-reviews.sh exempts 40-hex sha-keyed semantic-gate receipt directories — 7 of the 10 live review directories. The exemption is principled and recorded elsewhere (DECISIONS, spc-14, itd-93 acceptance criterion), but a reader auditing the folder against either charter sees 7 apparent blockers, and 06-lint.md says the rule 'ports into internal/core/lint', so an implementer working from the record alone would port it without the exemption and turn the gate permanently red.