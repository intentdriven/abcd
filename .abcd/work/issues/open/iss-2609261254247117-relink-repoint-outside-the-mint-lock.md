---
schema_version: 1
id: "iss-2609261254247117"
slug: "relink-repoint-outside-the-mint-lock"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review2-itd34 low note b"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/relink/relink.go"
---

relink.Repoint rewrites the link text in every record naming a moved path as a read-modify-write outside the intent mint lock, on every verb that moves a record (intent plan, spec close, intent reclassify): a concurrent edit to a linking record between the read and the rename is lost. The move itself is judged under the lock; only the repointing that follows it is not.
