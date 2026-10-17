---
schema_version: 1
id: "iss-2610142247175819"
slug: "finishmerged-swallows-a-claim-view-error"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "the last review of spc-2610031753359849 steps 8 to 11 and 14"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain_finish.go"
remedy: "On a View error, lock each parked reserved run and call renewFailed, which logs it once."
---

finishMerged swallows a claim.View error, so every parked run's renewal is skipped on that drain move with no run-record entry and no answer naming it.
