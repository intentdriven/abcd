---
schema_version: 1
id: "iss-2611152052517498"
slug: "testareadgrowthtestrefusesaquadraticread"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run, 2026-11-15 merge-queue removal"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/findings_test.go"
remedy: "Give the control test room below its ideal: time the control at more units so fixed per-call overhead weighs less, or judge the control by the largest of its rounds rather than the median, and keep the threshold the linear tests use unchanged; then run the control under -race a few hundred times to check it no longer fails."
---

TestAReadGrowthTestRefusesAQuadraticRead in internal/core/implement/loop failed once in the merge queue's race shard on 2026-11-15: the control quadratic read grew 9.9 times on an input four times larger, against a threshold of more than 10, and a records-only pull request was removed from the queue for it. The ideal growth is 16, and the median of up to five timing rounds came out under 10 on a loaded race-instrumented runner, so the control test sits close enough to its threshold to fail on noise.
