---
schema_version: 1
id: "iss-2609252007434356"
slug: "abcd-source-cite-check-reports-a-finding-as-line-l-byte-b"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/source/guard.go"
---

abcd source cite-check reports a finding as 'line L, byte B' where B is grep -b's FILE offset of the matched span, which also includes the leading boundary byte, so a reader takes it for a column in line L. The offset must be documented and rendered as what it is, consistently in the type, the help and the surface pages.
