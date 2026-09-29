---
schema_version: 1
id: "iss-2608301203525338"
slug: "record-schema-declares-a-bucketfield-only-for-the-admission"
severity: "nitpick"
category: "inconsistency"
source: "impl-review"
found_during: "itd-189 implementation, 2026-08-30"
found_at: "internal/core/lint/schema.go (recordStores, checkRecordBucketField)"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Keep parked until the disposition store gains a required set (the spc-58 gap), or close?"
resolution: "record_schema now declares the disposition store's required set and allow-list from issueschema and its bucketField item, so a disposition whose item contradicts its item-keyed directory is a blocker, as the admission's run is; TestDispositionItemFieldMustAgreeWithItsBucket and TestDispositionRecordRequiresItsDeclaredFields"
impact: fix
resolved_by:
  commit: "d86930f0b"
---

record_schema declares a bucketField only for the admission store, so an admission whose run field contradicts its bucket is named while a disposition whose item field contradicts its item-keyed directory is not — the same double claim, checked in one store and not its sibling. The disposition store declares no frontmatter schema this cycle (spc-58's stated gap), so the field was left undeclared rather than half-stating a schema in a second place. Declare it when the disposition store gains its required set.

## Grounds

- pursued: a hand-written disposition filed under rdi-2 naming item rdi-9 is now a record_schema finding; a committed disposition the writer accepts turning red would show it wrong
