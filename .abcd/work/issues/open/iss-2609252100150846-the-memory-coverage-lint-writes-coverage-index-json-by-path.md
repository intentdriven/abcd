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
---

The memory coverage lint writes .coverage_index.json by path (writeCoverageIndex over CoverageIndexPath) after openStore vetted the store, so a store directory swapped for a symlink between the open and the write lands the index outside the repository: the handle's containment covers its reads, not this write. It is not behind the store lock or validatedMemoryDir the way the page writer is, so the writer residual iss-2608291814572914 declares does not cover it. Fix: write the index through the store handle's os.Root (fsutil.WriteFileAtomicInRoot).
