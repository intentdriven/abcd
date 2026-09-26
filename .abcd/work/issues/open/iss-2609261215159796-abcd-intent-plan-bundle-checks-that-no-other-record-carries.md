---
schema_version: 1
id: "iss-2609261215159796"
slug: "abcd-intent-plan-bundle-checks-that-no-other-record-carries"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-itd34"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/bundle.go"
---

abcd intent plan --bundle checks that no other record carries the bundle name against a corpus loaded before the intent mint lock and never re-checks it under the lock (internal/core/intent/bundle.go PlanBundle), so two concurrent plans of disjoint drafts under one name both succeed: four records name one bundle across two specs, the bundle lint passes, and each spec's close ships only its own pair. reclassify has the same shape at lower stakes: changeKind's 'named by another record' check and supersede's survivor set are computed from the pre-lock corpus, so a concurrent supersession of the only other member leaves a bundle of one with no history line.
