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
---

the memory lint's degraded-scanner finding carries a per-repo pattern name into report.md unsanitised. Lint builds an MR001 message from openStoreRedactor's error, whose text includes the pattern name read from .abcd/config/pii.json, and renderLintReportMD writes that message into the local-tier report.md with no termsafe pass; the CLI render sanitises, the file render does not. A hostile or careless pattern name therefore reaches a file the operator opens in a pager. Recorded from a security review of the memory-store lane; not fixed there, and the file report is local-tier rather than committed, so the exposure is one terminal read.
