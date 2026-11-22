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
resolution: "TestHookRowJudgesModeOnlyWhereAFileIsExecuted gains two cases with a manifest inline hook and hooks/hooks.json reaching hooks/go.sh at 0644, one handing it to sh and the other executing it, in both orders; each expects the not-executable finding. Test only; the gate already unions the executed set."
impact: internal
resolved_by:
  commit: "057511ae6b926f7c79f7d9f67f03c0a574ef1e4b"
---

No test covers launch's hook-compliance gate across two hook documents (the manifest's inline hooks and hooks.json reaching the same file), so a regression that rebuilt the executed-reference set per document instead of as one union would pass; the code handles the case today.

## Grounds

- pursued: both cases pass on today's code; a per-document executed set fails one of them (last-document-wins fails the manifest-executes case, first-document-wins fails the hooks.json-executes case), both watched failing through the narrow gate before the code was restored
