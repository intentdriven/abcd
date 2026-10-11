---
schema_version: 1
id: "iss-2608270908340925"
slug: "ahoy-receipt-classification-s-under-is-a"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "issue-sweep-2026-08-27"
found_at: "internal/core/ahoy/receipt.go"
resolution: "ahoy under() classifies a case-variant path via fsutil.PathWithin on a case-folding FS"
impact: internal
resolved_by:
  commit: "aca9d57a"
---

ahoy receipt classification's under() is a third verbatim copy of the lexical containment compare and is case-sensitive, so a case-variant path is mis-classified for receipt redaction on a case-folding filesystem — route through fsutil.PathWithin