---
schema_version: 1
id: "iss-2609020239068243"
slug: "the-memory-lint-s-degraded-scanner-finding-carries-a-per-rep"
severity: "nitpick"
category: "observation"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/lint.go"
resolution: "renderLintReportMD passes the store path and each finding's file, message and suggestion through termsafe.Sanitize, the primitive the CLI render uses, so a pii.json pattern name carried by the degraded-scanner MR001 message reaches report.md masked"
impact: fix
resolved_by:
  commit: "4713e995e386ecb2b312ad66a6f6f235f252256f"
---

the memory lint's degraded-scanner finding carries a per-repo pattern name into report.md unsanitised. Lint builds an MR001 message from openStoreRedactor's error, whose text includes the pattern name read from .abcd/config/pii.json, and renderLintReportMD writes that message into the local-tier report.md with no termsafe pass; the CLI render sanitises, the file render does not. A hostile or careless pattern name therefore reaches a file the operator opens in a pager. Recorded from a security review of the memory-store lane; not fixed there, and the file report is local-tier rather than committed, so the exposure is one terminal read.

## Grounds

- pursued: sanitising at the one render that writes report.md makes the file and the CLI agree on the same finding; a control, bidi or zero-width rune from a finding reaching report.md raw would show it wrong
