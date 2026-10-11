---
schema_version: 1
id: "iss-2609252007434356"
slug: "abcd-source-cite-check-reports-a-finding"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/source/guard.go"
resolution: "cite-check renders the offset as a text offset ('line L, at byte B of the text') and Hit, Finding, the help, the command page and the surface chapter document it as the start of the matched span counted from the start of the whole text"
impact: fix
resolved_by:
  commit: "a5510f1b"
---

abcd source cite-check reports a finding as 'line L, byte B' where B is grep -b's FILE offset of the matched span, which also includes the leading boundary byte, so a reader takes it for a column in line L. The offset must be documented and rendered as what it is, consistently in the type, the help and the surface pages.

## Grounds

- pursued: a reader of a cite-check finding locates it by counting bytes from the start of the text; a finding whose number is a column, or a surface that still calls it a line position, would show it wrong
