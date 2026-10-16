---
schema_version: 1
id: "iss-2610160924048542"
slug: "no-test-covers-launch-s-hook-compliance"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "v0.15.0 deferred findings, PR1 confirmation review, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/gates_test.go"
remedy: "Add a TestHookRowJudgesModeOnlyWhereAFileIsExecuted case with an inline manifest hook handing go.sh to sh and hooks.json executing it directly, in both orders."
---

No test covers launch's hook-compliance gate across two hook documents (the manifest's inline hooks and hooks.json reaching the same file), so a regression that rebuilt the executed-reference set per document instead of as one union would pass; the code handles the case today.
