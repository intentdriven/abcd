---
schema_version: 1
id: "iss-2609252243284279"
slug: "capture-reframe-s-whole-write-picks-the-previous-distinct"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/reframe.go"
---

capture reframe's whole write picks the previous distinct frame state in git log date order, so across a --no-ff merge where both parents moved the frame the before state is whichever parent's commit is newer, and changed depends on timestamps rather than topology: a record can under-report the surfaces the merged rewrite moved
