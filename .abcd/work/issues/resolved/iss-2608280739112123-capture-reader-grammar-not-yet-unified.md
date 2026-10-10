---
schema_version: 1
id: "iss-2608280739112123"
slug: "capture-reader-grammar-not-yet-unified"
severity: "nitpick"
category: "tech-debt"
source: "agent-finding"
found_during: "adversarial review of the record-lint parity batch (2026-08-28)"
found_at: "internal/core/capture/workflow.go"
resolution: "capture detects ledger records through the shared recordid.FilenameNumRe, so writer/reader/resolver/gate share one grammar"
impact: internal
resolved_by:
  commit: "97c6d6d7"
---

capture's ledger reader is stricter than the shared recordid grammar: after standardising lint and the resolver on recordid.FilenameNumRe, capture's scanLedger/alloc/validate still use their own reFilenameID (strict kebab, no double/leading/trailing hyphen), so a hand-crafted iss-5--x.md is lint-green and resolver-visible but silently absent from capture list. Harmless direction (lint too lenient, capture cannot mint such a slug, no committed record hits it) but it leaves the writer/reader/resolver/gate on two grammars instead of one. Route scanLedger, alloc and validate through recordid.FilenameNumRe("iss") so all four share one grammar. Follow-up to the batch that closed iss-2608270908346617.