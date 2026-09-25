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
resolution: "capture reframe's whole write walks the surfaces' history along first parents, so across a --no-ff merge the before state is the first parent's and changed names what the merge brought, in either timestamp order"
impact: fix
resolved_by:
  commit: "b534f1c6"
---

capture reframe's whole write picks the previous distinct frame state in git log date order, so across a --no-ff merge where both parents moved the frame the before state is whichever parent's commit is newer, and changed depends on timestamps rather than topology: a record can under-report the surfaces the merged rewrite moved

## Grounds

- pursued: a merge where the branch moves two surfaces and main a third records the branch's two in both timestamp orders, as a squash would; a before state or changed that moved with commit dates would show it wrong
