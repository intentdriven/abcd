---
schema_version: 1
id: "iss-2608301203521317"
slug: "record-schema-s-store-walk-reads-every-record-with-os-readdi"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "itd-189 implementation, 2026-08-30"
found_at: "internal/core/lint/schema.go (scanRecordStores)"
resolution: "record_schema's store walk reads each record with fsutil.ReadGuarded on the unresolved leaf, as readingoutstanding.go does, and reports a symlinked, non-regular or oversized record as a finding on the file instead of following or reading it; configured store paths are contained. TestRecordSchemaDeclinesSymlinkedAndOversizedRecords pins both shapes and that the link target's frontmatter never reaches the output."
impact: fix
resolved_by:
  commit: "b48fd584"
---

record_schema's store walk reads every record with os.ReadDir and os.ReadFile, with no symlink refusal and no size cap, while the sibling reading walk in readingoutstanding.go reads the same trees through fsutil.ReadGuarded — so a symlinked or oversized record is followed by the gate and declined by the report, and a symlink's target frontmatter can surface in lint output. Pre-existing across all four original stores; noted during the itd-189 security review and left out of that branch's scope. Read the store walk through the same guarded read the report uses.

## Grounds

- pursued: the gate and the reading report decline the same records; a symlinked record whose target frontmatter appears in record-lint output would show it wrong
