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
resolution: "PlanBundle stages its members and checks the bundle name on a corpus loaded under the intent mint lock, and Reclassify runs its whole judgement in one lock acquisition, so a plan or reclassify landing in the window is seen before the write."
impact: fix
resolved_by:
  commit: "5a6c2f5ab3e357f1a70181ae94db26feb4e8b0b3"
---

abcd intent plan --bundle checks that no other record carries the bundle name against a corpus loaded before the intent mint lock and never re-checks it under the lock (internal/core/intent/bundle.go PlanBundle), so two concurrent plans of disjoint drafts under one name both succeed: four records name one bundle across two specs, the bundle lint passes, and each spec's close ships only its own pair. reclassify has the same shape at lower stakes: changeKind's 'named by another record' check and supersede's survivor set are computed from the pre-lock corpus, so a concurrent supersession of the only other member leaves a bundle of one with no history line.

## Grounds

- pursued: we expect a second plan of the same bundle name, or a reclassify that empties or thins a bundle, landing between a verb's early reads and its write to be refused or accounted for; shown wrong if two specs ever name one bundle, or a survivor left alone by concurrent supersessions carries no bundle-of-one line
