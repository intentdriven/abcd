---
schema_version: 1
id: "iss-2609020833531987"
slug: "docs-reference-writing-style-md-gained-a-structure-table-row"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "release-v0.7.1-docs-currency"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/reference/writing-style.md"
resolution: "The Escapes section of docs/reference/writing-style.md names harness_leak's own escape: abcd-lint:allow (abcd-audit:allow honoured too), the docs-lint allow comment not read, a fenced block never flagged — matching checkHarnessLeak in internal/core/lint/harnessleak.go. No gate would have caught it: docs-lint checks tokens and links, not whether a reference page's claims match the rule set."
impact: fix
resolved_by:
  commit: "120d11000"
---

docs/reference/writing-style.md gained a Structure-table row for harness_leak at v0.7.1 but its Escapes section (lines 81 to 90) still lists the docs-lint allow families as present_tense, punctuation, spelling, harness and names, and says the other machine-enforced rules have no line escape; harness_leak is in neither list, and it does take a line escape under a different token: checkHarnessLeak skips a line carrying abcd-lint:allow (or the pre-spc-29 abcd-audit:allow) and does not honour the docs-lint allow comment, and a fenced block is never flagged. A reader applies the wrong escape token or believes none exists. One sentence after line 88 closes it. Found by the v0.7.1 docs-currency pass on the release content commit; deferred from the receipt to this record.

## Grounds

- pursued: the paragraph states what checkHarnessLeak does (waiver tokens, mask for fences, no docs-lint allow read); a harness_leak escape the paragraph does not name, or one it names that the rule ignores, would show it wrong
