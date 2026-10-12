---
schema_version: 1
id: "iss-2610120138028442"
slug: "the-implement-loop-s-merged-check"
severity: "nitpick"
category: "tech-debt"
source: "impl-review"
found_during: "ruthless review of the loop's conflict refusal (iss-2610110908443341, PR #960)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "read state, mergeable and mergeStateStatus in one gh pr view call and pass the parsed verdict to the conflict check"
---

The implement loop's merged check (internal/core/implement/loop/land.go, landMerged) asks the forge twice per poll while a pull request is not merged: `gh pr view --json state`, then prMergeability's `gh pr view --json mergeable,mergeStateStatus`. Both fields come from one view, so each poll spends two forge calls where one would answer, against the same rate limit the loop's other waits share.
