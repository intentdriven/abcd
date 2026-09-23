---
schema_version: 1
id: "iss-2608301203525338"
slug: "record-schema-declares-a-bucketfield-only-for-the-admission"
severity: "nitpick"
category: "inconsistency"
source: "impl-review"
found_during: "itd-189 implementation, 2026-08-30"
found_at: "internal/core/lint/schema.go (recordStores, checkRecordBucketField)"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (deferred by its own terms until the disposition store gains a required set (spc-58 gap, still absent); planning owed). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

record_schema declares a bucketField only for the admission store, so an admission whose run field contradicts its bucket is named while a disposition whose item field contradicts its item-keyed directory is not — the same double claim, checked in one store and not its sibling. The disposition store declares no frontmatter schema this cycle (spc-58's stated gap), so the field was left undeclared rather than half-stating a schema in a second place. Declare it when the disposition store gains its required set.
