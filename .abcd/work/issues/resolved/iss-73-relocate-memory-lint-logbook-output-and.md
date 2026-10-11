---
schema_version: 1
id: "iss-73"
slug: "relocate-memory-lint-logbook-output-and"
severity: "minor"
impact: internal
category: "tech-debt"
source: "user-observation"
found_during: "abcd-run-design"
found_at: "internal/core/memory/lint.go"
resolution: "relocated memory-lint report dir + scanner skip fragments from .abcd/logbook to .abcd/.work.local/logs; source-grep detector armed; plugin doc updated. Record-lint markdown ban-arming deferred (needs research-note reconciliation)"
---

Relocate memory-lint logbook output and scanner skip-paths from .abcd/logbook/ (a retired location per iss-36) to .abcd/.work.local/logs/, the gitignored runtime-artefact tier. Maintainer adjudication of iss-56 (2026-07-12): runtime artefacts belong in .work.local/logs/, not a tracked dir. Sites: internal/core/memory/lint.go writes .abcd/logbook/memory/lint dirs; internal/adapter/scanner/scanner.go defaultSkipFragments references .abcd/logbook/pii-scan/ and audit-history/. Fix both plus tests; once the binary no longer writes there the .abcd/logbook retired-location ban (iss-36/iss-56) can be armed. Actionable fix behind iss-56 adjudication.