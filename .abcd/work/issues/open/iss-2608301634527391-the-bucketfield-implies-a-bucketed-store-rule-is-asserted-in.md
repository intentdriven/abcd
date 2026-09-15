---
schema_version: 1
id: "iss-2608301634527391"
slug: "the-bucketfield-implies-a-bucketed-store-rule-is-asserted-in"
severity: "nitpick"
category: "tech-debt"
source: "user-observation"
found_during: "itd-189-round-4-builder"
found_at: "internal/core/lint/schema.go"
---

the bucketField implies a bucketed store rule is asserted in prose only so a flat store declaring one would report filed under empty

Reported by the round-4 builder alongside its deletion of the `r.bucket == ""`
guard in `checkRecordBucketField`. The deletion is sound TODAY: `bucketField` is
declared only by the bucketed `adm` store, so the empty-bucket branch is one no
fixture can enter, and the round had already ruled that such a branch is how a
guard comes to look tested.

What is missing is the pin. "A store declaring `bucketField` is a bucketed
store" is asserted in prose and by present fact, not by a test. A future flat
store declaring one would emit a finding reading `filed under ''` and nothing
would be red. The sibling assertion in the same diff IS pinned, by
`TestEveryJoinFamilyNamesADeclaredStore`; this one is not, which is the same
asymmetry iss-2608301519254240 was raised about one leg away.

Cheap close: a test walking the declared stores and asserting that every one
declaring a `bucketField` also declares buckets.
