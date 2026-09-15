---
schema_version: 1
id: "iss-2608300320001985"
slug: "known-prefix-store-root-still-hides-a-bucket"
severity: "major"
category: "security"
source: "impl-review"
found_during: "itd-180 third-round security review, 2026-08-30"
found_at: "internal/core/lint/schema.go (nestedStoreRoots), internal/core/lint/config.go (validateRecordStores)"
resolution: "record_schema grants the nested-root exemption only when the child is not a bucket the parent declares, and parseConfig refuses a store root that is or sits inside another store's declared bucket (by list or by grammar) and two prefixes on one path."
impact: fix
---

The nested-store-root exemption still blinds record_schema through a known prefix: a committed record_stores line pointing rdi (or dsp, or itd) at another store's declared bucket — .abcd/work/issues/open, or a readings run directory — passes validateRecordStores, marks that bucket a nested root so the parent store skips it, and the misdirected store ignores every file not matching its own filename grammar, so a malformed issue record with five missing required fields yields zero findings; reopens the class iss-2608300227224016 records as closed, whose resolution text is not true as written. Refuse a configured store root that equals or sits inside a declared bucket of any other store, refuse two prefixes resolving to one path, and grant the exemption only when the child is not a bucket the parent declares.
