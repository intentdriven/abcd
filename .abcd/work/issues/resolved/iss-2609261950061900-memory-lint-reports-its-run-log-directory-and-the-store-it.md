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
resolution: "Lint names report_dir, store_path, coverage_index.path and every finding's file relative to the repository through fsutil.DisplayPath, in the --json result and in the run log's report.json and report.md; the run log is still written to the absolute directory. The finding files and the coverage index path were in the same class and are fixed with the two fields the record names."
impact: fix
resolved_by:
  commit: "dc6ff0c53"
---

memory lint reports its run-log directory and the store it read as absolute paths: LintResult.ReportDir and LintResult.StorePath (internal/core/memory/lint.go) are joined onto the repository root, so memory lint --json carries report_dir and store_path naming the developer's home whenever the checkout sits under it, against the iss-81 rule; the text report prints ReportDir too.

## Grounds

- pursued: a lint of a checkout under HOME reports store_path .abcd/memory, a repository-relative report_dir under which the run log exists, and no absolute finding file, and neither the JSON nor the run log carries the home (TestLintReportsItsPathsRelativeToTheRepository, TestMemoryLintFromSubdirectoryReadsAndReportsInsideTheCheckout); an absolute path in any of them would show it wrong
