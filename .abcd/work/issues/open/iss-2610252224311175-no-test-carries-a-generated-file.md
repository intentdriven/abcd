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
---

No test carries a generated-file conflict through the implementer's regeneration to a pushed, mergeable pull request: the sync tests stop at the sync brief (internal/core/implement/loop/sync_default_test.go:140), and re-landing is pinned only for a conflict-free resync (:241). This is the half of itd-2610161016391855 its fidelity review found unproven.
