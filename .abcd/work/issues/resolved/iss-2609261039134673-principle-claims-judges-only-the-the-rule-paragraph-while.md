---
schema_version: 1
id: "iss-2609261039134673"
slug: "principle-claims-judges-only-the-the-rule-paragraph-while"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-principles"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/principles.go"
resolution: "principle_claims judges the H1 title as well as the paragraph, over the same derivation the projection uses, so a title citing a record is reported by record-lint."
impact: fix
resolved_by:
  commit: "a5f5a266662ed258f549a49a4847a391e68fd9f6"
---

principle_claims judges only the **The rule.** paragraph while the projection sends the H1 title above it and verifyPrincipleItem scans both, so a typed principle whose title cites a record (# Title citing itd-79) passes record-lint and then refuses the whole reading assembly: the two readers of a principle's statement disagree about what it is.

## Grounds

- pursued: TestPrincipleTitleMayNotCite reports a handle, a link and a URL in the title, and TestPrincipleReadersAgreeOnTheStatement finds no typed principle the assembler refuses or drops that record-lint passes; a title citation passing the lint while the assembly refuses would show it wrong
