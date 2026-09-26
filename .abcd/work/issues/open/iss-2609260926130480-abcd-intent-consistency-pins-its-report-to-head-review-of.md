---
schema_version: 1
id: "iss-2609260926130480"
slug: "abcd-intent-consistency-pins-its-report-to-head-review-of"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/consistency.go"
---

abcd intent consistency pins its report to HEAD (review_of_commit) but reads the brief and the intents from the working tree, so an uncommitted or untracked edit to a brief page or an intent is quoted in an append-only report under a commit that does not hold the quoted text at the cited path:line, and nothing in the report says so. itd-28's dirty-tree policy for review pins is to tag dirty: true and not block; the consistency emit neither tags nor refuses.
