---
schema_version: 1
id: "iss-2609260057117838"
slug: "abcd-update-s-foreign-refusal-for-a-path"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/update/update.go"
resolution: "The foreign refusal attaches every negative to the PATH entry the classifier examined; the resolved path is named as where the entry resolves, not as the subject of 'is not a regular file'."
impact: fix
resolved_by:
  commit: "8f1441de"
---

abcd update's foreign refusal, for a PATH entry that is a symlink, reads: the entry at X leads to Y, which is not ... a regular file abcd can verify. But Y is a regular file (the classifier Lstat'd the entry, not Y), so the person is told something false about the link's target. The negatives belong to the entry.

## Grounds

- pursued: the sentence is true of a link whose target is a regular file; TestPlanForeignRefusalJudgesTheLinkNotItsTarget would fail if the negatives were attached to the resolved path again
