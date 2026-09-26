---
schema_version: 1
id: "iss-2609260057117838"
slug: "abcd-update-s-foreign-refusal-for-a-path-entry-that-is-a"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/update/update.go"
---

abcd update's foreign refusal, for a PATH entry that is a symlink, reads: the entry at X leads to Y, which is not ... a regular file abcd can verify. But Y is a regular file (the classifier Lstat'd the entry, not Y), so the person is told something false about the link's target. The negatives belong to the entry.
