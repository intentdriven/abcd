---
schema_version: 1
id: "iss-2609252100150846"
slug: "the-memory-coverage-lint-writes-coverage-index-json-by-path"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/coverage.go"
resolution: "The coverage lint writes .coverage_index.json through the store handle's os.Root (fsutil.WriteFileAtomicInRoot), so the write resolves inside the directory the handle vetted; Lint holds that one handle for the page lint and the coverage lint."
impact: fix
resolved_by:
  commit: "113f8ab0"
---

The memory coverage lint writes .coverage_index.json by path (writeCoverageIndex over CoverageIndexPath) after openStore vetted the store, so a store directory swapped for a symlink between the open and the write lands the index outside the repository: the handle's containment covers its reads, not this write. It is not behind the store lock or validatedMemoryDir the way the page writer is, so the writer residual iss-2608291814572914 declares does not cover it. Fix: write the index through the store handle's os.Root (fsutil.WriteFileAtomicInRoot).

## Grounds

- pursued: a store swapped for a symlink after openStore leaves the index write inside the vetted directory, pinned by TestCoverageLintWritesOnlyThroughTheStoreHandle; a coverage-index write by path reappearing, or the test passing with the write reverted, would show it wrong
