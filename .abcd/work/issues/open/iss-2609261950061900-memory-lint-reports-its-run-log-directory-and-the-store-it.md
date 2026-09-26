---
schema_version: 1
id: "iss-2609261950061900"
slug: "memory-lint-reports-its-run-log-directory-and-the-store-it"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainSite"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/lint.go"
---

memory lint reports its run-log directory and the store it read as absolute paths: LintResult.ReportDir and LintResult.StorePath (internal/core/memory/lint.go) are joined onto the repository root, so memory lint --json carries report_dir and store_path naming the developer's home whenever the checkout sits under it, against the iss-81 rule; the text report prints ReportDir too.
