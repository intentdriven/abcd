---
schema_version: 1
id: "iss-2608301203521317"
slug: "record-schema-s-store-walk-reads-every-record-with-os-readdi"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "itd-189 implementation, 2026-08-30"
found_at: "internal/core/lint/schema.go (scanRecordStores)"
---

record_schema's store walk reads every record with os.ReadDir and os.ReadFile, with no symlink refusal and no size cap, while the sibling reading walk in readingoutstanding.go reads the same trees through fsutil.ReadGuarded — so a symlinked or oversized record is followed by the gate and declined by the report, and a symlink's target frontmatter can surface in lint output. Pre-existing across all four original stores; noted during the itd-189 security review and left out of that branch's scope. Read the store walk through the same guarded read the report uses.
