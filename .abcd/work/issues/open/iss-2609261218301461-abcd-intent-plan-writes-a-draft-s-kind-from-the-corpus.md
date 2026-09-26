---
schema_version: 1
id: "iss-2609261218301461"
slug: "abcd-intent-plan-writes-a-draft-s-kind-from-the-corpus"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-itd34"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/lifecycle.go"
---

abcd intent plan writes a draft's kind from the corpus loaded before the intent mint lock (Plan in internal/core/intent/lifecycle.go reads it.Kind and it.SpecID from the pre-lock corpus while the bytes it rewrites are read under the lock), so a reclassify landing in the window that makes the draft a bundle-member is overwritten: the planned record carries kind: standalone beside bundle: <name>, a pairing no verb otherwise writes.
