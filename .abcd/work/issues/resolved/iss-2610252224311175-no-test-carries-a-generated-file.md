---
schema_version: 1
id: "iss-2610252224311175"
slug: "no-test-carries-a-generated-file"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "gap audit of the owed fidelity reviews, 2026-10-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/sync_default_test.go"
remedy: "Add a loop test in which the default branch moves a generated file under a live lane, the implementer regenerates it, and the lane pushes a pull request that merges cleanly."
resolution: "A loop test now carries a generated-file conflict from the default branch's move under a live issue lane through the implementer's regeneration in a merge commit, a fresh round, the records, push, pull request and arming, to a fast-forward merge onto the default branch that brings the regenerated file."
impact: internal
resolved_by:
  commit: "0f6b270b18112e4541a92f00e37800c5d71ebd94"
---

No test carries a generated-file conflict through the implementer's regeneration to a pushed, mergeable pull request: the sync tests stop at the sync brief (internal/core/implement/loop/sync_default_test.go:140), and re-landing is pinned only for a conflict-free resync (:241). This is the half of itd-2610161016391855 its fidelity review found unproven.

## Grounds

- pursued: TestARegeneratedFileIsPushedAsAPullRequestThatMergesCleanly passes on the current loop and fails when the resolved sync stops recording its merge head; shown wrong if a regression in the sync-resolution-to-landing path (a pending sync left after a verified receipt, a second sync, a pushed head missing the default branch's tip, or a non-fast-forward merge) still lets it pass
