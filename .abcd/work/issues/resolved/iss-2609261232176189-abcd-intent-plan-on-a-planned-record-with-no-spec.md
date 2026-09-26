---
schema_version: 1
id: "iss-2609261232176189"
slug: "abcd-intent-plan-on-a-planned-record-with-no-spec"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-itd34"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/lifecycle.go"
resolution: "linkPlannedSpec parses the record from the bytes it reads under the intent mint lock and keeps the kind those carry."
impact: fix
resolved_by:
  commit: "a82cc92e426f1403259fef418ebc090d2886b545"
---

abcd intent plan on a planned record with no spec (linkPlannedSpec in internal/core/intent/lifecycle.go) writes the record's kind from the corpus loaded before the intent mint lock while rewriting bytes read under it, so a reclassify landing in the window that makes the record a bundle-member is overwritten: the record carries kind: standalone beside bundle: <name>. The draft branch of Plan had the same shape (iss-2609261218301461); this is its in-place twin.

## Grounds

- pursued: we expect Plan's in-place link to keep a kind a reclassify wrote in the window; shown wrong if a planned record ever carries kind: standalone beside a bundle name after it
